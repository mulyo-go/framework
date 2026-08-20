package schema

import (
	"fmt"

	"gorm.io/gorm"
)

// Create executes a CREATE TABLE migration using the Blueprint
func Create(db *gorm.DB, table string, callback func(table *Blueprint)) error {
	b := NewBlueprint(table, false)
	callback(b)

	driver := db.Dialector.Name()
	statements := CompileCreate(driver, b)

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute sql [%s]: %w", stmt, err)
		}
	}
	return nil
}

// Table executes an ALTER TABLE migration using the Blueprint
func Table(db *gorm.DB, table string, callback func(table *Blueprint)) error {
	b := NewBlueprint(table, true)
	callback(b)

	driver := db.Dialector.Name()
	statements := CompileAlter(driver, b)

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute sql [%s]: %w", stmt, err)
		}
	}
	return nil
}

// DropIfExists drops a table if it exists
func DropIfExists(db *gorm.DB, table string) error {
	driver := db.Dialector.Name()
	quoted := quoteIdentifier(driver, table)
	stmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoted)
	return db.Exec(stmt).Error
}

// HasTable checks if a table exists in the database
func HasTable(db *gorm.DB, table string) bool {
	return db.Migrator().HasTable(table)
}

// HasColumn checks if a column exists in a table
func HasColumn(db *gorm.DB, table, column string) bool {
	return db.Migrator().HasColumn(table, column)
}
