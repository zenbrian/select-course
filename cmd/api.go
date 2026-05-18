package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zenbrian/select-course/internal/course"
	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
	redisinfra "github.com/zenbrian/select-course/internal/infrastructure/redis"
	"github.com/zenbrian/select-course/internal/user"
)

// mount
func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	//LOGGER THAT SHOWS THE REQUESTS COMING IN, AND HOW LONG THEY TAKE TO PROCESS

	// A good base middleware stack
	r.Use(middleware.RequestID) // for rate limiting
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	OrderService := course.NewService(repo.New(app.db), app.db, app.redis)
	UserService := user.NewService(repo.New(app.db), app.db, app.redis)

	// 在啟動服務之前，先執行預熱
	if err := OrderService.PreheatCoursesToRedis(context.Background()); err != nil {
		log.Fatalf("failed to preheat courses to redis: %v", err)
	}
	if err := UserService.PreheatUsersToRedis(context.Background()); err != nil {
		log.Fatalf("failed to preheat users to redis: %v", err)
	}

	OrderHandler := course.NewHandler(OrderService)
	r.Get("/courses/{id}", OrderHandler.GetCourse)
	r.Post("/courses/select", OrderHandler.SelectCourse)
	r.Post("/courses/back-course", OrderHandler.BackCourse)

	UserHandler := user.NewHandler(UserService)
	r.Post("/users/register", UserHandler.Register)
	r.Post("/users/login", UserHandler.Login)
	r.Post("/users/logout", UserHandler.Logout)
	r.Get("/users/{id}", UserHandler.GetUser)

	return r
}

// run
func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server is running on port", app.config.addr)
	return srv.ListenAndServe()
}

type application struct {
	config config
	//loger
	db    *pgxpool.Pool
	redis *redisinfra.Client
}

type config struct {
	addr  string
	db    dbconfig
	redis redisconfig
}

type dbconfig struct {
	dsn string //user= pass= dbname= sslmode=
}

type redisconfig struct {
	enabled      bool
	addr         string
	password     string
	db           int
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
}
