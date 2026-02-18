package app

import (
	"context"
	"database/sql"

	"iam-service/internal/config"
	"iam-service/internal/db"
	"iam-service/internal/logger"
	"iam-service/internal/redis"

	_ "github.com/lib/pq"
)

type Infra struct {
	DB    *db.DB
	Redis *redis.Client
}

func setupInfra(ctx context.Context, cfg config.Config) (*Infra, error) {
	sqlDB, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	if err := db.RunKeystoneMigration(ctx, sqlDB); err != nil {
		return nil, err
	}

	logger.Info("database ready", nil)

	redisClient, err := redis.New(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		return nil, err
	}

	logger.Info("redis ready", nil)

	return &Infra{
		DB:    &db.DB{DB: sqlDB},
		Redis: redisClient,
	}, nil
}
