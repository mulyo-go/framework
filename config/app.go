package config

import (
	"os"
	"strings"
)

func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func RoutingMode() string {
	mode := strings.ToLower(Env("ROUTING_MODE", "auto"))
	if mode != "manual" {
		return "auto"
	}
	return "manual"
}

func AppPort() string {
	return Env("APP_PORT", "8080")
}

func UsePermMenu() bool {
	return strings.ToLower(Env("USE_PERM_MENU", "true")) == "true"
}

func UsePermRoute() bool {
	return strings.ToLower(Env("USE_PERM_ROUTE", "true")) == "true"
}
