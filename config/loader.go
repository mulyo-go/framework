package config

import (
	"github.com/joho/godotenv"
)

var loaded bool

func Load() {
	if loaded {
		return
	}
	_ = godotenv.Load()
	InitDatabases()
	InitRedis()
	InitCache()
	InitSession()
	loaded = true
	// log.Println("configuration loaded")
}
