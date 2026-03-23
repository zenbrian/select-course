package course

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
)

type Service interface {
	GetCourseByID(ctx context.Context, id int64) (repo.Course, error)
	SelectCourse(ctx context.Context, course_id int64, user_id int64) (repo.Course, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgx.Conn
}

const maxSlots = 32

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
		return repo.Course{}, errors.New("course is full")
	}

	// 3. 鎖 user（避免 flag race）
	user, err := qtx.GetUserByID(ctx, userID)
	if err != nil {
		return repo.Course{}, err
	}

	if !course.Week.Valid {
		return repo.Course{}, errors.New("invalid course week")
	}

	durationSlot, err := strconv.Atoi(course.Duration)
	if err != nil {
		return repo.Course{}, fmt.Errorf("invalid course duration: %w", err)
	}

	offset := int(course.Week.Int32)*3 + durationSlot
	if offset < 0 || offset >= maxSlots {
		return repo.Course{}, errors.New("invalid course time slot")
	}

	// 4. 用 testBit 檢查時間衝突
	occupied, err := testBit(user.Flag, offset)
	if err != nil {
		return repo.Course{}, err
	}
	if occupied {
		return repo.Course{}, errors.New("time conflict")
	}

	// 5. 扣減課程容量
	courseUpdated, err := qtx.UpdateCourseCapacity(ctx, repo.UpdateCourseCapacityParams{
		ID:       course.ID,
		Capacity: course.Capacity - 1,
	})
	if err != nil {
		return repo.Course{}, fmt.Errorf("failed to update course capacity: %w", err)
	}

	// 6. 建立選課紀錄（處理重複選課）
	err = qtx.CreateUserCourse(ctx, repo.CreateUserCourseParams{
		UserID:   userID,
		CourseID: courseID,
	})
	if err != nil {
		// duplicate key
		if strings.Contains(err.Error(), "duplicate key") {
			return repo.Course{}, errors.New("course already selected")
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
