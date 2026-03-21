package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/zenbrian/select-course/internal/course"
	repo "github.com/zenbrian/select-course/internal/infrastructure/postgresql/sqlc"
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
	OrderService := course.NewService(repo.New(app.db), app.db)
	OrderHandler := course.NewHandler(OrderService)
	r.Get("/course/{id}", OrderHandler.GetCourse)
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
	db *pgx.Conn
}

type config struct {
	addr string
	db   dbconfig
}

type dbconfig struct {
	dsn string //user= pass= dbname= sslmode=
}
