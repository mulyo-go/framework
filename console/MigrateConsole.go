package console

import (
	"flag"
	"fmt"
	"strings"

	"github.com/mulyo-go/framework/config"
	"github.com/mulyo-go/framework/database/migration"
	"gorm.io/gorm"
)

type MigrateConsole struct{}

func init() {
	Register(&MigrateConsole{})
}

func (c *MigrateConsole) Name() string {
	return "migrate"
}

func (c *MigrateConsole) Aliases() []string {
	return []string{"db:migrate", "migration"}
}

func (c *MigrateConsole) Usage() string {
	return "[-db <name>] [--rollback] [--reset] [--fresh] [--status]"
}

func (c *MigrateConsole) Description() string {
	return "run database migrations (supports multi-db)"
}

func (c *MigrateConsole) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	dbFlag := fs.String("db", "", "Nama koneksi database spesifik (dari .env DB_CONNECTIONS)")
	rollbackFlag := fs.Bool("rollback", false, "Rollback migration batch terakhir")
	resetFlag := fs.Bool("reset", false, "Rollback semua migration")
	freshFlag := fs.Bool("fresh", false, "Drop semua tabel dan jalankan ulang migration")
	statusFlag := fs.Bool("status", false, "Lihat status migration")

	if err := fs.Parse(args); err != nil {
		return err
	}

	config.Load()
	config.InitDatabases()

	var targetDBs map[string]*gorm.DB
	if *dbFlag != "" {
		db := config.DB(*dbFlag)
		if db == nil {
			// Cek apakah dbFlag adalah default DB
			if strings.EqualFold(*dbFlag, "default") || strings.EqualFold(*dbFlag, config.DefaultDBName()) {
				db = config.DefaultDB()
			}
		}
		if db == nil {
			return fmt.Errorf("koneksi database '%s' tidak ditemukan di .env", *dbFlag)
		}
		targetDBs = map[string]*gorm.DB{*dbFlag: db}
	} else {
		// Jika tanpa -db flag: jalankan untuk Default DB
		defDB := config.DefaultDB()
		if defDB == nil {
			return fmt.Errorf("tidak ada koneksi database aktif di .env")
		}
		defaultName := config.DefaultDBName()
		targetDBs = map[string]*gorm.DB{defaultName: defDB}
	}

	for connName, db := range targetDBs {
		migrator := migration.New(db, connName)

		if *statusFlag {
			statuses, err := migrator.Status()
			if err != nil {
				return err
			}
			fmt.Printf("\n=== Status Migration [%s] ===\n", connName)
			fmt.Printf("%-5s | %-12s | %-55s | %-6s\n", "Ran?", "Target DB", "Migration", "Batch")
			fmt.Println(strings.Repeat("-", 84))
			for _, st := range statuses {
				ranStr := "No"
				batchStr := "-"
				if st.Ran {
					ranStr = "Yes"
					batchStr = fmt.Sprintf("%d", st.Batch)
				}
				fmt.Printf("%-5s | %-12s | %-55s | %-6s\n", ranStr, st.Connection, st.Name, batchStr)
			}
			fmt.Println()
			continue
		}

		if *freshFlag {
			if err := migrator.Fresh(); err != nil {
				return err
			}
			continue
		}

		if *resetFlag {
			if err := migrator.Reset(); err != nil {
				return err
			}
			continue
		}

		if *rollbackFlag {
			if err := migrator.Rollback(); err != nil {
				return err
			}
			continue
		}

		// Default: Run Pending Migrations
		if err := migrator.RunPending(); err != nil {
			return err
		}
	}

	return nil
}
