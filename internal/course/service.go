package course

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
)

type Service interface {
	GetCourseByID(ctx context.Context, id int64) (repo.Course, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgx.Conn
}

func NewService(repo *repo.Queries, db *pgx.Conn) Service {
	return &svc{
		repo: repo,
		db:   db,
	}
}

// func (s *svc) CreateCourse(ctx context.Context, course *repo.Course) (repo.Course, error) {

// }

func (s *svc) GetCourseByID(ctx context.Context, id int64) (repo.Course, error) {
	return s.repo.GetCourseByID(ctx, id)
}

func (s *svc) SelectCourse(ctx context.Context, course_id int64, user_id int64) (repo.Course, error) {
	//檢查課堂庫存
	course, err := s.repo.GetCourseByID(ctx, course_id)
	if err != nil {
		return repo.Course{}, err
	}

	if course.Capacity <= 0 {
		return repo.Course{}, errors.New("course is full")
	}
	user, err := s.repo.GetUserByID(ctx, user_id)
	if err != nil {
		return repo.Course{}, err
	}

	if err := s.checkUserTimeConflict(user, course); err != nil {
		return repo.Course{}, err
	}
	//扣減課程庫存
	//創建選課紀錄
	return course, nil
}

func (s *svc) checkUserTimeConflict(user repo.User, course repo.Course) error {
	if !course.Week.Valid {
		return errors.New("invalid course week")
	}

	durationSlot, err := strconv.Atoi(course.Duration)
	if err != nil {
		return fmt.Errorf("invalid course duration: %w", err)
	}

	offset := int(course.Week.Int32)*3 + durationSlot
	if offset < 0 || offset >= 32 {
		return errors.New("invalid course time slot")
	}

	if (user.Flag>>offset)&1 == 1 {
		return errors.New("time conflict")
	}

	return nil
}
