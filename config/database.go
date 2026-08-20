package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbConnections = make(map[string]*gorm.DB)

var defaultDBName string

func InitDatabases() {
	defaultDBName = Env("DB_DEFAULT", "")
	connections := strings.Split(Env("DB_CONNECTIONS", ""), ",")
	for _, rawName := range connections {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		upper := strings.ToUpper(name)
		driver := Env("DB_"+upper+"_DRIVER", "")
		if driver == "" {
			continue
		}
		host := Env("DB_"+upper+"_HOST", "127.0.0.1")
		port := Env("DB_"+upper+"_PORT", "")
		user := Env("DB_"+upper+"_USER", "")
		pass := Env("DB_"+upper+"_PASS", "")
		dbName := Env("DB_"+upper+"_NAME", "")

		var dialector gorm.Dialector
		switch driver {
		case "mysql":
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, dbName)
			dialector = mysql.Open(dsn)
		case "postgres":
			dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbName)
			dialector = postgres.Open(dsn)
		default:
			continue
		}

		db, err := gorm.Open(dialector, &gorm.Config{
			Logger: GetGormLogger(name),
		})
		if err != nil {
			fmt.Printf("❌ Koneksi Database '%s' (%s %s:%s) GAGAL: %v\n", name, driver, host, port, err)
			continue
		}

		// Atur Connection Pool (Manajemen koneksi biar gak bocor)
		sqlDB, err := db.DB()
		if err != nil {
			fmt.Printf("❌ Gagal mengambil instance sql.DB untuk '%s': %v\n", name, err)
			continue
		}

		// 1. Max Idle Connections (default: 10)
		// Jumlah koneksi nganggur yg standby, siap dipake kapan aja.
		maxIdle := EnvInt("DB_"+upper+"_MAX_IDLE_CONNS", 10)
		sqlDB.SetMaxIdleConns(maxIdle)

		// 2. Max Open Connections (default: 100, 0 = unlimited)
		// Mentoknya berapa koneksi yg boleh dibuka barengan. Kalo 0 berarti gas terus tanpa rem.
		maxOpen := EnvInt("DB_"+upper+"_MAX_OPEN_CONNS", 100)
		sqlDB.SetMaxOpenConns(maxOpen)

		// 3. Connection Max Lifetime (default: 1h, 0 = unlimited/forever)
		// Umur maksimal koneksi boleh idup. Kalo udah tua, dimatiin & ganti baru.
		// Format: "1h", "30m", "300s". Kalo angka doang = detik.
		lifetimeStr := Env("DB_"+upper+"_CONN_MAX_LIFETIME", "1h")
		if lifetime, err := time.ParseDuration(lifetimeStr); err == nil {
			sqlDB.SetConnMaxLifetime(lifetime)
		} else {
			// Kalo gagal parse durasi, anggep aja integer (detik)
			if sec, err := strconv.Atoi(lifetimeStr); err == nil {
				sqlDB.SetConnMaxLifetime(time.Duration(sec) * time.Second)
			}
		}

		// 4. Connection Max Idle Time (default: 30m, 0 = unlimited/forever)
		// Lama maksimal koneksi boleh bengong. Kalo kelamaan gabut, tendang aja.
		idleTimeStr := Env("DB_"+upper+"_CONN_MAX_IDLE_TIME", "30m")
		if idleTime, err := time.ParseDuration(idleTimeStr); err == nil {
			sqlDB.SetConnMaxIdleTime(idleTime)
		} else {
			if sec, err := strconv.Atoi(idleTimeStr); err == nil {
				sqlDB.SetConnMaxIdleTime(time.Duration(sec) * time.Second)
			}
		}

		dbConnections[name] = db
	}
}

func DB(name string) *gorm.DB {
	return dbConnections[name]
}

func DefaultDB() *gorm.DB {
	if defaultDBName == "" {
		for _, db := range dbConnections {
			return db
		}
		return nil
	}
	return dbConnections[defaultDBName]
}

// EnvInt helper buat baca env variable tapi outputnya integer
func EnvInt(key string, defaultValue int) int {
	valStr := Env(key, "")
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}


func GetAllDBs() map[string]*gorm.DB {
	return dbConnections
}

func DefaultDBName() string {
	if defaultDBName != "" {
		return defaultDBName
	}
	return Env("DB_DEFAULT", "default")
}
