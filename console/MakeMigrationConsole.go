package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MakeMigrationConsole struct{}

func init() {
	Register(&MakeMigrationConsole{})
}

func (c *MakeMigrationConsole) Name() string {
	return "make:migration"
}

func (c *MakeMigrationConsole) Aliases() []string {
	return []string{"make-migration", "create:migration"}
}

func (c *MakeMigrationConsole) Usage() string {
	return "<nama_migration> [-db <connection_name>] (contoh: create_products_table -db mysql_dua)"
}

func (c *MakeMigrationConsole) Description() string {
	return "create a new database migration file in database/migrations"
}

func (c *MakeMigrationConsole) Run(args []string) error {
	var rawName string
	var targetDB string

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "-db" || arg == "--db" {
			if i+1 < len(args) {
				targetDB = strings.TrimSpace(args[i+1])
				i++
			}
		} else if strings.HasPrefix(arg, "-db=") || strings.HasPrefix(arg, "--db=") {
			parts := strings.SplitN(arg, "=", 2)
			targetDB = strings.TrimSpace(parts[1])
		} else if !strings.HasPrefix(arg, "-") && rawName == "" {
			rawName = arg
		}
	}

	if rawName == "" {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("migration name is required (contoh: app.exe make:migration create_products_table [-db mysql_dua])")
	}

	rawName = strings.ToLower(rawName)
	rawName = strings.ReplaceAll(rawName, "-", "_")
	rawName = strings.ReplaceAll(rawName, " ", "_")

	timestamp := time.Now().Format("2006_01_02_150405")
	migrationName := fmt.Sprintf("%s_%s", timestamp, rawName)
	fileName := migrationName + ".go"

	migrationsDir := filepath.Join("database", "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("gagal membuat folder database/migrations: %w", err)
	}

	filePath := filepath.Join(migrationsDir, fileName)

	// Guess table name from migration name (e.g. create_users_table -> users)
	tableName := "my_table"
	if strings.HasPrefix(rawName, "create_") && strings.HasSuffix(rawName, "_table") {
		tableName = strings.TrimSuffix(strings.TrimPrefix(rawName, "create_"), "_table")
	} else if strings.HasPrefix(rawName, "create_") {
		tableName = strings.TrimPrefix(rawName, "create_")
	}

	connArg := ""
	if targetDB != "" {
		connArg = fmt.Sprintf(",\n\t\t%q", targetDB)
	}

	templateContent := fmt.Sprintf(`package migrations

import (
	"github.com/mulyo-go/framework/database/migration"
	"github.com/mulyo-go/framework/database/schema"
	"gorm.io/gorm"
)

func init() {
	migration.Register("%s",
		// UP: Jalankan perubahan skema
		func(db *gorm.DB) error {
			return schema.Create(db, "%s", func(table *schema.Blueprint) {
				table.ID()
				table.String("name", 255).NotNull()
				table.Timestamps()
			})
		},
		// DOWN: Batalkan / rollback perubahan skema
		func(db *gorm.DB) error {
			return schema.DropIfExists(db, "%s")
		}%s,
	)
}
`, migrationName, tableName, tableName, connArg)

	if err := os.WriteFile(filePath, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("gagal menulis file migration: %w", err)
	}

	dbInfo := "default"
	if targetDB != "" {
		dbInfo = targetDB
	}
	fmt.Printf("✅ Berhasil membuat migration [%s]: %s\n", dbInfo, filePath)
	return nil
}
