package seeder

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Seeder struct {
	Name string
	Run  func(db *gorm.DB) error
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]*Seeder)
)

// Register registers a new seeder to the framework registry
func Register(name string, run func(db *gorm.DB) error) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = &Seeder{
		Name: name,
		Run:  run,
	}
}

// GetRegisteredSeeders returns all registered seeders sorted by name
func GetRegisteredSeeders() []*Seeder {
	registryMu.RLock()
	defer registryMu.RUnlock()

	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := make([]*Seeder, 0, len(keys))
	for _, k := range keys {
		list = append(list, registry[k])
	}
	return list
}

// RunSeeders executes all or specific registered seeders
func RunSeeders(db *gorm.DB, specificName ...string) error {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var targets []*Seeder
	if len(specificName) > 0 && specificName[0] != "" {
		targetName := specificName[0]
		s, ok := registry[targetName]
		if !ok {
			return fmt.Errorf("seeder '%s' tidak ditemukan di registry", targetName)
		}
		targets = append(targets, s)
	} else {
		targets = GetRegisteredSeeders()
	}

	if len(targets) == 0 {
		fmt.Println("Nothing to seed.")
		return nil
	}

	for _, s := range targets {
		fmt.Printf("🌱 Seeding: %s...\n", s.Name)
		start := time.Now()
		err := db.Transaction(func(tx *gorm.DB) error {
			if s.Run != nil {
				return s.Run(tx)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("❌ Error seeding %s: %v\n", s.Name, err)
			return err
		}
		fmt.Printf("✅ Seeded:  %s (%v)\n", s.Name, time.Since(start).Round(time.Millisecond))
	}
	return nil
}
