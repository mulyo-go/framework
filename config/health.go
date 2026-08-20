package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// CheckConnectionsStatus mengecek status koneksi Database (MySQL, PostgreSQL, dll) dan Redis
func CheckConnectionsStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Cek Database (MySQL / PostgreSQL / dll)
	connections := strings.Split(Env("DB_CONNECTIONS", ""), ",")
	hasDB := false
	for _, rawName := range connections {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		hasDB = true
		upper := strings.ToUpper(name)
		driver := Env("DB_"+upper+"_DRIVER", "")
		host := Env("DB_"+upper+"_HOST", "127.0.0.1")
		port := Env("DB_"+upper+"_PORT", "")

		db := DB(name)
		if db == nil {
			fmt.Printf("❌ Koneksi Database '%s' (%s %s:%s) GAGAL: DB instance nil / gorm open error\n", name, driver, host, port)
			continue
		}

		sqlDB, err := db.DB()
		if err != nil {
			fmt.Printf("❌ Koneksi Database '%s' (%s %s:%s) GAGAL: %v\n", name, driver, host, port, err)
			continue
		}

		if err := sqlDB.PingContext(ctx); err != nil {
			fmt.Printf("❌ Koneksi Database '%s' (%s %s:%s) GAGAL: %v\n", name, driver, host, port, err)
		} else {
			fmt.Printf("✅ Koneksi Database '%s' (%s %s:%s) BERHASIL\n", name, driver, host, port)
		}
	}
	if !hasDB {
		fmt.Println("ℹ️  Tidak ada konfigurasi DB_CONNECTIONS di .env")
	}

	// 2. Cek Redis
	redisConfigs := []struct {
		Name    string
		Client  *redis.Client
		HostKey string
		PortKey string
	}{
		{"Cache", RedisClient, "REDIS_HOST", "REDIS_PORT"},
		{"Session", RedisSessionClient, "REDIS_SESSION_HOST", "REDIS_SESSION_PORT"},
		{"Auth/Sec", RedisSecClient, "REDIS_SEC_HOST", "REDIS_SEC_PORT"},
		{"CSRF", RedisCSRFClient, "REDIS_CSRF_HOST", "REDIS_CSRF_PORT"},
	}

	for _, rc := range redisConfigs {
		host := Env(rc.HostKey, "")
		port := Env(rc.PortKey, "6379")
		if host == "" && rc.HostKey != "REDIS_HOST" {
			continue
		}
		if host == "" {
			host = "127.0.0.1"
		}

		if rc.Client == nil {
			fmt.Printf("❌ Koneksi Redis %s (%s:%s) GAGAL / non-aktif\n", rc.Name, host, port)
		} else if err := rc.Client.Ping(ctx).Err(); err != nil {
			fmt.Printf("❌ Koneksi Redis %s (%s:%s) GAGAL: %v\n", rc.Name, host, port, err)
		} else {
			fmt.Printf("✅ Koneksi Redis %s (%s:%s) BERHASIL\n", rc.Name, host, port)
		}
	}
}
