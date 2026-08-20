package console

import (
	"flag"
	"fmt"
	"strings"

	"github.com/mulyo-go/framework/config"
	"github.com/mulyo-go/framework/database/seeder"
	"gorm.io/gorm"
)

type SeedConsole struct{}

func init() {
	Register(&SeedConsole{})
}

func (c *SeedConsole) Name() string {
	return "db:seed"
}

func (c *SeedConsole) Aliases() []string {
	return []string{"seed", "db-seed"}
}

func (c *SeedConsole) Usage() string {
	return "[-db <name>] [-class <SeederName>]"
}

func (c *SeedConsole) Description() string {
	return "run database seeders"
}

func (c *SeedConsole) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	dbFlag := fs.String("db", "", "Nama koneksi database")
	classFlag := fs.String("class", "", "Nama class seeder spesifik (opsional)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	config.Load()
	config.InitDatabases()

	var targetDB *gorm.DB
	connName := "default"
	if *dbFlag != "" {
		targetDB = config.DB(*dbFlag)
		if targetDB == nil && (strings.EqualFold(*dbFlag, "default") || strings.EqualFold(*dbFlag, config.DefaultDBName())) {
			targetDB = config.DefaultDB()
		}
		if targetDB == nil {
			return fmt.Errorf("koneksi database '%s' tidak ditemukan di .env", *dbFlag)
		}
		connName = *dbFlag
	} else {
		targetDB = config.DefaultDB()
		if targetDB == nil {
			return fmt.Errorf("tidak ada koneksi database aktif di .env")
		}
		connName = config.DefaultDBName()
	}

	fmt.Printf("=== Database Seeding [%s] ===\n", connName)
	return seeder.RunSeeders(targetDB, *classFlag)
}
