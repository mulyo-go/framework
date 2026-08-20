package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MakeModelConsole struct{}

func init() {
	Register(&MakeModelConsole{})
}

func (c *MakeModelConsole) Name() string {
	return "make:model"
}

func (c *MakeModelConsole) Aliases() []string {
	return []string{"make-model"}
}

func (c *MakeModelConsole) Usage() string {
	return "<NamaModel> [-m <Module>] [-t <table_name>] [--migration]"
}

func (c *MakeModelConsole) Description() string {
	return "create a GORM struct model (optional: generate migration)"
}

func (c *MakeModelConsole) Run(args []string) error {
	var rawName string
	var moduleName string
	var tableName string
	var withMigration bool

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "-m" || arg == "--module" {
			if i+1 < len(args) {
				moduleName = strings.TrimSpace(args[i+1])
				i++
			}
		} else if strings.HasPrefix(arg, "-m=") || strings.HasPrefix(arg, "--module=") {
			moduleName = strings.TrimSpace(strings.SplitN(arg, "=", 2)[1])
		} else if arg == "-t" || arg == "--table" {
			if i+1 < len(args) {
				tableName = strings.TrimSpace(args[i+1])
				i++
			}
		} else if strings.HasPrefix(arg, "-t=") || strings.HasPrefix(arg, "--table=") {
			tableName = strings.TrimSpace(strings.SplitN(arg, "=", 2)[1])
		} else if arg == "--migration" || arg == "-mig" {
			withMigration = true
		} else if !strings.HasPrefix(arg, "-") && rawName == "" {
			rawName = arg
		}
	}

	if rawName == "" {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("model name is required (contoh: app.exe make:model Product -m Kasir -t products)")
	}

	modelName := rawName
	if tableName == "" {
		tableName = strings.ToLower(modelName) + "s"
	}

	var targetDir string
	var pkgName string
	if moduleName != "" {
		targetDir = filepath.Join("Module", moduleName, "Model")
		pkgName = "model"
	} else {
		targetDir = filepath.Join("app", "Model")
		pkgName = "model"
	}

	_ = os.MkdirAll(targetDir, 0755)
	filePath := filepath.Join(targetDir, modelName+".go")

	q := string(rune(96)) // backtick
	lines := []string{
		"package " + pkgName,
		"",
		"type " + modelName + " struct {",
		"\tID        int64  " + q + `gorm:"primaryKey;autoIncrement" json:"id"` + q,
		"\tName      string " + q + `gorm:"size:255;not null" json:"name"` + q,
		"\tCreatedAt int64  " + q + `gorm:"autoCreateTime" json:"created_at"` + q,
		"\tUpdatedAt int64  " + q + `gorm:"autoUpdateTime" json:"updated_at"` + q,
		"\tDeletedAt *int64 " + q + `gorm:"index" json:"deleted_at,omitempty"` + q,
		"}",
		"",
		"func (" + modelName + ") TableName() string {",
		"\treturn \"" + tableName + "\"",
		"}",
		"",
	}

	content := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("gagal menulis file model: %w", err)
	}

	fmt.Printf("✅ Successfully created model: %s\n", filePath)

	if withMigration {
		migCmd := &MakeMigrationConsole{}
		return migCmd.Run([]string{"create_" + tableName + "_table"})
	}

	return nil
}
