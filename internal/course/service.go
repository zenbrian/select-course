package course

import (
	"context"

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
	return s.repo.GetCourse(ctx, id)
}

func (s *svc) SelectCourse(ctx context.Context, course *repo.Course) (repo.Course, error) {
	//檢查課堂庫存
	//檢查是否有時間衝突
	//扣減課程庫存
	//創建選課紀錄
}
