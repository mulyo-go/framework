package migration

import (
	"fmt"
	"time"

	"github.com/mulyo-go/framework/database/schema"
	"gorm.io/gorm"
)

type MigrationRecord struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Migration string `gorm:"size:255;not null" json:"migration"`
	Batch     int    `gorm:"not null" json:"batch"`
	CreatedAt int64  `gorm:"not null;default:0" json:"created_at"`
}

func (MigrationRecord) TableName() string {
	return "migrations"
}

type MigrationStatus struct {
	Name       string
	Connection string
	Ran        bool
	Batch      int
}

type Migrator struct {
	DB             *gorm.DB
	ConnectionName string
}

func New(db *gorm.DB, connName ...string) *Migrator {
	cName := "default"
	if len(connName) > 0 && connName[0] != "" {
		cName = connName[0]
	}
	return &Migrator{
		DB:             db,
		ConnectionName: cName,
	}
}

// EnsureMigrationTable creates the migrations table if it does not exist
func (m *Migrator) EnsureMigrationTable() error {
	if m.DB == nil {
		return fmt.Errorf("database connection '%s' is nil", m.ConnectionName)
	}

	if schema.HasTable(m.DB, "migrations") {
		return nil
	}

	return schema.Create(m.DB, "migrations", func(table *schema.Blueprint) {
		table.ID()
		table.String("migration", 255).NotNull()
		table.Integer("batch").NotNull()
		table.BigInteger("created_at").Default(0)
	})
}

// GetRanMigrations retrieves all completed migrations from the database
func (m *Migrator) GetRanMigrations() (map[string]MigrationRecord, error) {
	if err := m.EnsureMigrationTable(); err != nil {
		return nil, err
	}

	var records []MigrationRecord
	if err := m.DB.Table("migrations").Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}

	res := make(map[string]MigrationRecord, len(records))
	for _, r := range records {
		res[r.Migration] = r
	}
	return res, nil
}

// GetLastBatch returns the maximum batch number in the migrations table
func (m *Migrator) GetLastBatch() (int, error) {
	if err := m.EnsureMigrationTable(); err != nil {
		return 0, err
	}

	var maxBatch int
	row := m.DB.Table("migrations").Select("COALESCE(MAX(batch), 0)").Row()
	if err := row.Scan(&maxBatch); err != nil {
		return 0, err
	}
	return maxBatch, nil
}

// RunPending executes all pending migrations for this connection
func (m *Migrator) RunPending() error {
	if err := m.EnsureMigrationTable(); err != nil {
		return err
	}

	ran, err := m.GetRanMigrations()
	if err != nil {
		return err
	}

	all := GetRegisteredMigrations(m.ConnectionName)
	pending := make([]*Migration, 0)
	for _, mig := range all {
		if _, exists := ran[mig.Name]; !exists {
			pending = append(pending, mig)
		}
	}

	if len(pending) == 0 {
		fmt.Printf("[%s] Nothing to migrate.\n", m.ConnectionName)
		return nil
	}

	lastBatch, err := m.GetLastBatch()
	if err != nil {
		return err
	}
	nextBatch := lastBatch + 1

	fmt.Printf("[%s] Running %d pending migration(s) (Batch %d)...\n", m.ConnectionName, len(pending), nextBatch)

	for _, mig := range pending {
		fmt.Printf("  Migrating: %s\n", mig.Name)
		start := time.Now()

		err := m.DB.Transaction(func(tx *gorm.DB) error {
			if mig.Up != nil {
				if err := mig.Up(tx); err != nil {
					return err
				}
			}

			record := MigrationRecord{
				Migration: mig.Name,
				Batch:     nextBatch,
				CreatedAt: time.Now().Unix(),
			}
			return tx.Table("migrations").Create(&record).Error
		})

		if err != nil {
			fmt.Printf("  ❌ Error migrating %s: %v\n", mig.Name, err)
			return err
		}

		fmt.Printf("  ✅ Migrated:  %s (%v)\n", mig.Name, time.Since(start).Round(time.Millisecond))
	}

	fmt.Printf("[%s] Migration completed successfully!\n", m.ConnectionName)
	return nil
}

// Rollback rolls back the last migration batch
func (m *Migrator) Rollback() error {
	if err := m.EnsureMigrationTable(); err != nil {
		return err
	}

	lastBatch, err := m.GetLastBatch()
	if err != nil {
		return err
	}

	if lastBatch == 0 {
		fmt.Printf("[%s] Nothing to rollback.\n", m.ConnectionName)
		return nil
	}

	var records []MigrationRecord
	if err := m.DB.Table("migrations").Where("batch = ?", lastBatch).Order("id desc").Find(&records).Error; err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Printf("[%s] Nothing to rollback for batch %d.\n", m.ConnectionName, lastBatch)
		return nil
	}

	all := GetRegisteredMigrations(m.ConnectionName)
	mapMig := make(map[string]*Migration, len(all))
	for _, mig := range all {
		mapMig[mig.Name] = mig
	}

	fmt.Printf("[%s] Rolling back batch %d (%d migration(s))...\n", m.ConnectionName, lastBatch, len(records))

	for _, rec := range records {
		mig, exists := mapMig[rec.Migration]
		fmt.Printf("  Rolling back: %s\n", rec.Migration)
		start := time.Now()

		err := m.DB.Transaction(func(tx *gorm.DB) error {
			if exists && mig.Down != nil {
				if err := mig.Down(tx); err != nil {
					return err
				}
			}
			return tx.Table("migrations").Where("id = ?", rec.ID).Delete(&MigrationRecord{}).Error
		})

		if err != nil {
			fmt.Printf("  ❌ Error rolling back %s: %v\n", rec.Migration, err)
			return err
		}

		fmt.Printf("  ✅ Rolled back: %s (%v)\n", rec.Migration, time.Since(start).Round(time.Millisecond))
	}

	return nil
}

// Reset rolls back all migrations
func (m *Migrator) Reset() error {
	for {
		lastBatch, err := m.GetLastBatch()
		if err != nil || lastBatch == 0 {
			break
		}
		if err := m.Rollback(); err != nil {
			return err
		}
	}
	return nil
}

// Fresh drops all tables and reruns all migrations
func (m *Migrator) Fresh() error {
	fmt.Printf("[%s] Dropping all tables...\n", m.ConnectionName)
	if err := m.Reset(); err != nil {
		return err
	}
	_ = schema.DropIfExists(m.DB, "migrations")
	return m.RunPending()
}

// Status prints the migration status table
func (m *Migrator) Status() ([]MigrationStatus, error) {
	ran, err := m.GetRanMigrations()
	if err != nil {
		return nil, err
	}

	all := GetRegisteredMigrations(m.ConnectionName)
	res := make([]MigrationStatus, 0, len(all))

	for _, mig := range all {
		connDisplay := mig.Connection
		if connDisplay == "" {
			connDisplay = "default"
		}
		st := MigrationStatus{
			Name:       mig.Name,
			Connection: connDisplay,
		}
		if r, ok := ran[mig.Name]; ok {
			st.Ran = true
			st.Batch = r.Batch
		}
		res = append(res, st)
	}

	return res, nil
}
