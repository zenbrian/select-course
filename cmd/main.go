package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	redisinfra "github.com/zenbrian/select-course/internal/infrastructure/redis"
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
		redis: redisconfig{
			enabled:      getEnvBool("REDIS_ENABLED", false),
			addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			password:     getEnv("REDIS_PASSWORD", ""),
			db:           getEnvInt("REDIS_DB", 0),
			dialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			readTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			writeTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
	}

	//Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	slog.SetDefault(logger)

	//Database connection
	conn, err := pgxpool.New(ctx, cfg.db.dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	logger.Info("connected to database successfully", "dsn", cfg.db.dsn)

	var redisClient *redisinfra.Client
	if cfg.redis.enabled {
		redisClient, err = redisinfra.NewClient(redisinfra.Config{
			Addr:         cfg.redis.addr,
			Password:     cfg.redis.password,
			DB:           cfg.redis.db,
			DialTimeout:  cfg.redis.dialTimeout,
			ReadTimeout:  cfg.redis.readTimeout,
			WriteTimeout: cfg.redis.writeTimeout,
		})
		if err != nil {
			logger.Error("failed to connect to redis", "error", err)
			os.Exit(1)
		}
		defer redisClient.Close()
		logger.Info("connected to redis successfully", "addr", cfg.redis.addr, "db", cfg.redis.db)

		startupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := redisClient.Set(startupCtx, "server:startup", "booted", time.Minute); err != nil {
			logger.Warn("failed to set redis startup key", "error", err)
		} else {
			logger.Info("redis startup key set", "key", "server:startup", "ttl", "1m")
		}
	} else {
		logger.Info("redis is disabled via config")
	}

	//api
	api := application{
		config: cfg,
		db:     conn,
		redis:  redisClient,
	}

	// h := api.mount()

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

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
