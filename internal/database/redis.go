package database

import (
	"context"
	"fmt"
	"time"

	"github.com/apidet/go-api-service/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis เปิด client + ping (fail fast ตอน startup) — จุดเดียวที่เปิด Redis connection
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}
