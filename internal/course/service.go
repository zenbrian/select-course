package course

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
	"github.com/zenbrian/select-course/internal/infrastructure/redis"
)

type Service interface {
	GetCourseByID(ctx context.Context, id int64) (repo.Course, error)
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
	// 1. 開 transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Course{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.repo.WithTx(tx)

	// 2. 鎖 course（避免超賣）
	course, err := qtx.GetCourseByID(ctx, courseID)
	if err != nil {
		return repo.Course{}, err
	}

	if course.Capacity <= 0 {
		return repo.Course{}, ErrCourseFull
	}

	// 3. 鎖 user（避免 flag race）
	user, err := qtx.GetUserByID(ctx, userID)
	if err != nil {
		return repo.Course{}, err
	}

	if !course.Week.Valid {
		return repo.Course{}, ErrInvalidCourseWeek
	}

	// 4. 用 testBit 檢查時間衝突

	durationSlot, err := strconv.Atoi(course.Duration)
	if err != nil {
		return repo.Course{}, fmt.Errorf("invalid course duration: %w", err)
	}

	offset := int(course.Week.Int32)*3 + durationSlot
	if offset < 0 || offset >= maxSlots {
		return repo.Course{}, ErrInvalidTimeSlot
	}

	occupied, err := testBit(user.Flag, offset)
	if err != nil {
		return repo.Course{}, err
	}
	if occupied {
		return repo.Course{}, ErrTimeConflict
	}

	// 5. 原子扣減課程容量，避免併發下遺失更新
	courseUpdated, err := qtx.TryDecrementCapacity(ctx, course.ID)
	if err != nil {
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
		// duplicate key
		if strings.Contains(err.Error(), "duplicate key") {
			return repo.Course{}, ErrAlreadySelected
		}
		return repo.Course{}, fmt.Errorf("failed to create user course: %w", err)
	}

	// 7. 用 setBit 更新 user flag
	newFlag, err := setBit(user.Flag, offset)
	if err != nil {
		return repo.Course{}, err
	}

	err = qtx.UpdateUserFlag(ctx, repo.UpdateUserFlagParams{
		ID:   userID,
		Flag: newFlag,
	})
	if err != nil {
		return repo.Course{}, fmt.Errorf("failed to update user flag: %w", err)
	}

	// 8. commit
	if err := tx.Commit(ctx); err != nil {
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

	// 1. 鎖課程
	course, err := qtx.GetCourseByID(ctx, courseID)
	if err != nil {
		return repo.Course{}, err
	}

	// 2. 鎖 user
	user, err := qtx.GetUserByID(ctx, userID)
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
	//更新用戶選課紀錄()
	return courseUpdated, nil

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
