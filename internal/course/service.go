package course

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
	"github.com/zenbrian/select-course/internal/infrastructure/redis"
)

type Service interface {
	GetCourseByID(ctx context.Context, id int64) (repo.Course, error)
	ListCourses(ctx context.Context) ([]repo.Course, error)
	GetCoursesByUserID(ctx context.Context, userID int64) ([]repo.Course, error)
	SelectCourse(ctx context.Context, course_id int64, user_id int64) (repo.Course, error)
	BackCourse(ctx context.Context, course_id int64, user_id int64) (repo.Course, error)
	PreheatCoursesToRedis(ctx context.Context) error
}

type svc struct {
	repo  *repo.Queries
	db    *pgxpool.Pool
	redis *redis.Client
}

const maxSlots = 32

var (
	ErrCourseFull        = errors.New("course is full")
	ErrInvalidCourseWeek = errors.New("invalid course week")
	ErrInvalidTimeSlot   = errors.New("invalid course time slot")
	ErrTimeConflict      = errors.New("time conflict")
	ErrAlreadySelected   = errors.New("course already selected")
	ErrNotSelected       = errors.New("course not selected")
)

func NewService(repo *repo.Queries, db *pgxpool.Pool, redis *redis.Client) Service {
	return &svc{
		repo:  repo,
		db:    db,
		redis: redis,
	}
}

func (s *svc) GetCourseByID(ctx context.Context, id int64) (repo.Course, error) {
	return s.repo.GetCourseByID(ctx, id)
}

func (s *svc) ListCourses(ctx context.Context) ([]repo.Course, error) {
	return s.repo.ListCourses(ctx)
}

func (s *svc) GetCoursesByUserID(ctx context.Context, userID int64) ([]repo.Course, error) {
	return s.repo.GetCoursesByUserID(ctx, userID)
}

func (s *svc) PreheatCoursesToRedis(ctx context.Context) error {
	// 1. 從 MySQL/PostgreSQL 中把課程全部撈出來
	courses, err := s.repo.ListCourses(ctx)
	if err != nil {
		return fmt.Errorf("failed to list courses from db: %w", err)
	}

	// 2. 使用 Pipeline 批次寫入，避免迴圈內一直建立連線
	pipe := s.redis.Pipeline()

	for _, v := range courses {
		// 定義你在 Redis 的 Hash Key，例如 "course:info:1"
		hashKey := fmt.Sprintf("course:info:%d", v.ID)

		// 參考文章的做法，使用 HSet 把欄位一次寫入
		pipe.HSet(
			ctx,
			hashKey,
			"category_id", v.CategoryID,
			"duration", v.Duration, // 你的 schema 是 string，HSet 也吃
			"week", v.Week.Int32, // 你的 Week 是 pgtype.Int4，提取裡面的 Int32
			"capacity", v.Capacity,
		)
	}

	// 3. 一次發送給 Redis
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute redis pipeline for preheat: %w", err)
	}

	return nil
}

func (s *svc) SelectCourse(ctx context.Context, courseID int64, userID int64) (repo.Course, error) {
	// ── Redis 預檢：原子扣減 capacity，快速擋掉「已滿」的請求 ──
	redisKey := fmt.Sprintf("course:info:%d", courseID)
	redisPreChecked := false

	if s.redis != nil {
		remaining, err := s.redis.HIncrBy(ctx, redisKey, "capacity", -1)
		if err != nil {
			// Redis 異常時不阻擋選課，退化為純 DB 路徑
			slog.Warn("redis pre-check failed, falling back to DB", "course_id", courseID, "error", err)
		} else {
			redisPreChecked = true
			if remaining < 0 {
				// 名額已用完，補回 Redis 並直接拒絕
				s.redis.HIncrBy(ctx, redisKey, "capacity", 1)
				return repo.Course{}, ErrCourseFull
			}
		}
	}

	// ── 以下為原有 DB Transaction 邏輯 ──

	// 1. 開 transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	// 2. 鎖 course（FOR UPDATE 避免超賣）
	course, err := qtx.GetCourseByIDForUpdate(ctx, courseID)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, err
	}

	if course.Capacity <= 0 {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, ErrCourseFull
	}

	// 3. 鎖 user（FOR UPDATE 避免 flag race）
	user, err := qtx.GetUserByIDForUpdate(ctx, userID)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, err
	}

	if !course.Week.Valid {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, ErrInvalidCourseWeek
	}

	// 4. 用 testBit 檢查時間衝突
	durationSlot, err := strconv.Atoi(course.Duration)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, fmt.Errorf("invalid course duration: %w", err)
	}

	offset := int(course.Week.Int32)*3 + durationSlot
	if offset < 0 || offset >= maxSlots {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, ErrInvalidTimeSlot
	}

	occupied, err := testBit(user.Flag, offset)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, err
	}
	if occupied {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, ErrTimeConflict
	}

	// 5. 原子扣減課程容量，避免併發下遺失更新
	courseUpdated, err := qtx.TryDecrementCapacity(ctx, course.ID)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		if errors.Is(err, pgx.ErrNoRows) {
			return repo.Course{}, ErrCourseFull
		}
		return repo.Course{}, fmt.Errorf("failed to try decrement course capacity: %w", err)
	}

	// 6. 建立選課紀錄（處理重複選課）
	err = qtx.CreateUserCourse(ctx, repo.CreateUserCourseParams{
		UserID:   userID,
		CourseID: courseID,
	})
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		// unique_violation (SQLSTATE 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repo.Course{}, ErrAlreadySelected
		}
		return repo.Course{}, fmt.Errorf("failed to create user course: %w", err)
	}

	// 7. 用 setBit 更新 user flag
	newFlag, err := setBit(user.Flag, offset)
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, err
	}

	err = qtx.UpdateUserFlag(ctx, repo.UpdateUserFlagParams{
		ID:   userID,
		Flag: newFlag,
	})
	if err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, fmt.Errorf("failed to update user flag: %w", err)
	}

	// 8. commit
	if err := tx.Commit(ctx); err != nil {
		s.redisCompensate(ctx, redisKey, redisPreChecked)
		return repo.Course{}, err
	}

	return courseUpdated, nil
}

func (s *svc) BackCourse(ctx context.Context, courseID int64, userID int64) (repo.Course, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Course{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	// 1. 鎖課程（FOR UPDATE）
	course, err := qtx.GetCourseByIDForUpdate(ctx, courseID)
	if err != nil {
		return repo.Course{}, err
	}

	// 2. 鎖 user（FOR UPDATE）
	user, err := qtx.GetUserByIDForUpdate(ctx, userID)
	if err != nil {
		return repo.Course{}, err
	}
	// 3. 刪除選課紀錄
	rows, err := qtx.DeleteUserCourse(ctx, repo.DeleteUserCourseParams{
		UserID:   user.ID,
		CourseID: course.ID,
	})
	if err != nil {
		return repo.Course{}, err
	}

	if rows == 0 {
		return repo.Course{}, ErrNotSelected
	}

	// 原子回補課程容量，避免併發下遺失更新
	courseUpdated, err := qtx.IncrementCapacity(ctx, course.ID)
	if err != nil {
		return repo.Course{}, fmt.Errorf("failed to increment course capacity: %w", err)
	}
	if !course.Week.Valid {
		return repo.Course{}, ErrInvalidCourseWeek
	}

	durationSlot, err := strconv.Atoi(course.Duration)
	if err != nil {
		return repo.Course{}, fmt.Errorf("invalid course duration: %w", err)
	}

	offset := int(course.Week.Int32)*3 + durationSlot
	if offset < 0 || offset >= maxSlots {
		return repo.Course{}, ErrInvalidTimeSlot
	}
	// 用 clearBit 更新 user flag
	newFlag, err := clearBit(user.Flag, offset)
	if err != nil {
		return repo.Course{}, err
	}
	err = qtx.UpdateUserFlag(ctx, repo.UpdateUserFlagParams{
		ID:   user.ID,
		Flag: newFlag,
	})
	if err != nil {
		return repo.Course{}, fmt.Errorf("failed to update user flag: %w", err)
	}
	// 6. commit
	if err := tx.Commit(ctx); err != nil {
		return repo.Course{}, err
	}

	// 7. DB commit 成功後，回補 Redis capacity
	if s.redis != nil {
		redisKey := fmt.Sprintf("course:info:%d", courseID)
		if _, err := s.redis.HIncrBy(ctx, redisKey, "capacity", 1); err != nil {
			slog.Warn("failed to increment redis capacity after back-course", "course_id", courseID, "error", err)
		}
	}

	return courseUpdated, nil

}

// redisCompensate 在 DB 操作失敗時，將 Redis 預扣的 capacity 補回來。
func (s *svc) redisCompensate(ctx context.Context, redisKey string, preChecked bool) {
	if !preChecked || s.redis == nil {
		return
	}
	if _, err := s.redis.HIncrBy(ctx, redisKey, "capacity", 1); err != nil {
		slog.Error("failed to compensate redis capacity", "key", redisKey, "error", err)
	}
}

func setBit(flag int32, slot int) (int32, error) {
	if slot < 0 || slot >= maxSlots {
		return 0, errors.New("slot out of range")
	}

	return flag | (int32(1) << slot), nil
}

func clearBit(flag int32, slot int) (int32, error) {
	if slot < 0 || slot >= maxSlots {
		return 0, errors.New("slot out of range")
	}

	return flag & ^(int32(1) << slot), nil
}

func testBit(flag int32, slot int) (bool, error) {
	if slot < 0 || slot >= maxSlots {
		return false, errors.New("slot out of range")
	}

	return ((flag >> slot) & 1) == 1, nil
}
