package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if present
	_ = godotenv.Load()

	ctx := context.Background()
	cfg := config{
		addr: getEnv("SERVER_ADDR", ":8081"),
		db: dbconfig{
			dsn: getEnv("DB_DSN", getEnv("GOOSE_DBSTRING", "host=localhost user=postgres password=postgres dbname=select-course sslmode=disable")),
		},
	}

	//Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	slog.SetDefault(logger)

	//Database connection
	conn, err := pgx.Connect(ctx, cfg.db.dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)
	logger.Info("connected to database successfully", "dsn", cfg.db.dsn)

	//api
	api := application{
		config: cfg,
		db:     conn,
	}

	// h := api.mount()
	// api.run(h)

	if err := api.run(api.mount()); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
