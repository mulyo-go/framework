package migration

import (
	"sort"
	"strings"
	"sync"

	"github.com/mulyo-go/framework/config"
	"gorm.io/gorm"
)

type Migration struct {
	Name       string
	Connection string
	Up         func(db *gorm.DB) error
	Down       func(db *gorm.DB) error
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]*Migration)
)

// Register registers a new migration to the framework registry
// Optional connection param specifies target DB (e.g. "mysql_dua", empty = default)
func Register(name string, up func(db *gorm.DB) error, down func(db *gorm.DB) error, connection ...string) {
	conn := ""
	if len(connection) > 0 && connection[0] != "" {
		conn = strings.TrimSpace(connection[0])
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = &Migration{
		Name:       name,
		Connection: conn,
		Up:         up,
		Down:       down,
	}
}

// GetRegisteredMigrations returns registered migrations, optionally filtered by connection
func GetRegisteredMigrations(targetConnection ...string) []*Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	targetConn := ""
	if len(targetConnection) > 0 {
		targetConn = strings.TrimSpace(targetConnection[0])
	}

	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := make([]*Migration, 0)
	for _, k := range keys {
		mig := registry[k]
		if targetConn == "" || targetConn == "all" {
			list = append(list, mig)
		} else {
			if mig.Connection != "" && !strings.EqualFold(mig.Connection, "default") {
				if strings.EqualFold(mig.Connection, targetConn) {
					list = append(list, mig)
				}
			} else {
				// Default DB migration matches default, empty, or default DB name from .env
				if targetConn == "" || strings.EqualFold(targetConn, "default") || strings.EqualFold(targetConn, config.DefaultDBName()) {
					list = append(list, mig)
				}
			}
		}
	}
	return list
}
