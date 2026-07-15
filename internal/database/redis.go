package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"excel-crud-app/internal/config"

	"github.com/redis/go-redis/v9"
)

var (
	// RDB is the shared Redis client used by EmployeeService for the list
	// and per-record caches. Populated once by ConnectRedis at startup.
	RDB *redis.Client

	// Ctx is a base context for call sites without a request context handy.
	Ctx = context.Background()
)

// ConnectRedis initializes the shared Redis client and verifies
// connectivity with a PING.
func ConnectRedis(cfg *config.Config) error {
	RDB = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(Ctx, 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Connected to Redis successfully")
	return nil
}
