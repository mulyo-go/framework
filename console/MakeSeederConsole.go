package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MakeSeederConsole struct{}

func init() {
	Register(&MakeSeederConsole{})
}

func (c *MakeSeederConsole) Name() string {
	return "make:seeder"
}

func (c *MakeSeederConsole) Aliases() []string {
	return []string{"make-seeder"}
}

func (c *MakeSeederConsole) Usage() string {
	return "<NamaSeeder> (contoh: ProductSeeder)"
}

func (c *MakeSeederConsole) Description() string {
	return "create a new database seeder file in database/seeders"
}

func (c *MakeSeederConsole) Run(args []string) error {
	if len(args) == 0 {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("seeder name is required (contoh: app.exe make:seeder ProductSeeder)")
	}

	name := strings.TrimSpace(args[0])
	if !strings.HasSuffix(name, "Seeder") {
		name += "Seeder"
	}

	seedersDir := filepath.Join("database", "seeders")
	_ = os.MkdirAll(seedersDir, 0755)

	filePath := filepath.Join(seedersDir, name+".go")

	template := fmt.Sprintf(`package seeders

import (
	"time"

	"github.com/mulyo-go/framework/database/seeder"
	"gorm.io/gorm"
)

func init() {
	seeder.Register("%s", func(db *gorm.DB) error {
		now := time.Now().Unix()
		data := []map[string]any{
			{
				"name":       "Sample Data 1",
				"created_at": now,
			},
			{
				"name":       "Sample Data 2",
				"created_at": now,
			},
		}

		// Ganti 'my_table' dengan nama tabel target Anda
		return db.Table("my_table").Create(&data).Error
	})
}
`, name)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("gagal menulis file seeder: %w", err)
	}

	fmt.Printf("✅ Successfully created seeder: %s\n", filePath)
	return nil
}
