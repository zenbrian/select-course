package user

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
	"github.com/zenbrian/select-course/internal/infrastructure/redis"
)

type Service interface {
	Register(ctx context.Context, username, password string) (repo.User, error)
	Login(ctx context.Context, username, password string) (repo.User, error)
	GetUserByID(ctx context.Context, id int64) (repo.User, error)
	CreateSession(ctx context.Context, userID int64) (string, error)
	GetSessionUser(ctx context.Context, sessionID string) (int64, error)
	DestroySession(ctx context.Context, sessionID string) error
	PreheatUsersToRedis(ctx context.Context) error
}

type svc struct {
	repo       *repo.Queries
	db         *pgxpool.Pool
	redis      *redis.Client
	sessions   map[string]int64
	sessionsMu sync.RWMutex
}

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrWrongPassword   = errors.New("invalid password")
	ErrUsernameTaken   = errors.New("username already exists")
	ErrSessionNotFound = errors.New("session not found")
)

func NewService(repo *repo.Queries, db *pgxpool.Pool, redis *redis.Client) Service {
	return &svc{
		repo:     repo,
		db:       db,
		redis:    redis,
		sessions: make(map[string]int64),
	}
}

func (s *svc) Register(ctx context.Context, username, password string) (repo.User, error) {
	user, err := s.repo.CreateUser(ctx, repo.CreateUserParams{
		Username: username,
		Password: password, // Plain text password per user requirement
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repo.User{}, ErrUsernameTaken
		}
		return repo.User{}, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (s *svc) Login(ctx context.Context, username, password string) (repo.User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, ErrUserNotFound
		}
		return repo.User{}, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Password != password {
		return repo.User{}, ErrWrongPassword
	}

	return user, nil
}

func (s *svc) GetUserByID(ctx context.Context, id int64) (repo.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.User{}, ErrUserNotFound
		}
		return repo.User{}, fmt.Errorf("failed to get user by id: %w", err)
	}
	return user, nil
}

func (s *svc) CreateSession(ctx context.Context, userID int64) (string, error) {
	_ = ctx
	if userID <= 0 {
		return "", ErrUserNotFound
	}

	sessionID := uuid.NewString()
	s.sessionsMu.Lock()
	s.sessions[sessionID] = userID
	s.sessionsMu.Unlock()

	return sessionID, nil
}

func (s *svc) GetSessionUser(ctx context.Context, sessionID string) (int64, error) {
	_ = ctx
	if sessionID == "" {
		return 0, ErrSessionNotFound
	}

	s.sessionsMu.RLock()
	userID, ok := s.sessions[sessionID]
	s.sessionsMu.RUnlock()
	if !ok {
		return 0, ErrSessionNotFound
	}

	return userID, nil
}

func (s *svc) DestroySession(ctx context.Context, sessionID string) error {
	_ = ctx
	if sessionID == "" {
		return nil
	}

	s.sessionsMu.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMu.Unlock()

	return nil
}

func (s *svc) PreheatUsersToRedis(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}

	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users from db: %w", err)
	}

	pipe := s.redis.Pipeline()

	for _, user := range users {
		key := fmt.Sprintf("user:flag:%d", user.ID)
		pipe.Set(ctx, key, user.Flag, 0)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute redis pipeline for user preheat: %w", err)
	}

	return nil
}
