package config

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var RedisSessionClient *redis.Client
var RedisSecClient *redis.Client
var RedisCSRFClient *redis.Client

func InitRedis() {
	// Redis untuk cache
	RedisClient = initRedisClient("REDIS_HOST", "REDIS_PORT", "REDIS_PASS", "REDIS_DB", "0")

	// Redis untuk session (DB terpisah)
	RedisSessionClient = initRedisClient("REDIS_SESSION_HOST", "REDIS_SESSION_PORT", "REDIS_SESSION_PASS", "REDIS_SESSION_DB", "1")

	// Redis untuk auth token (DB terpisah)
	RedisSecClient = initRedisClient("REDIS_SEC_HOST", "REDIS_SEC_PORT", "REDIS_SEC_PASS", "REDIS_SEC_DB", "2")

	// Redis untuk CSRF token (DB terpisah)
	RedisCSRFClient = initRedisClient("REDIS_CSRF_HOST", "REDIS_CSRF_PORT", "REDIS_CSRF_PASS", "REDIS_CSRF_DB", "3")
}

func initRedisClient(hostKey, portKey, passKey, dbKey, defaultDB string) *redis.Client {
	host := Env(hostKey, "127.0.0.1")
	port := Env(portKey, "6379")
	pass := Env(passKey, "")
	dbStr := Env(dbKey, defaultDB)
	dbNum, err := strconv.Atoi(dbStr)
	if err != nil {
		dbNum = 0
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: pass,
		DB:       dbNum,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis %s:%s db=%d disabled: %v", host, port, dbNum, err)
		return nil
	}
	return client
}
