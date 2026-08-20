package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MakeModuleConsole struct{}

func init() {
	Register(&MakeModuleConsole{})
}

func (c *MakeModuleConsole) Name() string {
	return "make:module"
}

func (c *MakeModuleConsole) Aliases() []string {
	return []string{"make-module", "make:plugin"}
}

func (c *MakeModuleConsole) Usage() string {
	return "<NamaModule> (contoh: Kasir)"
}

func (c *MakeModuleConsole) Description() string {
	return "buat struktur folder modul baru (Controller, Model, Views)"
}

func (c *MakeModuleConsole) Run(args []string) error {
	if len(args) == 0 {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("nama module wajib diisi (contoh: mulyo-go make:module Kasir)")
	}

	modName := strings.TrimSpace(args[0])
	modDir := filepath.Join("Module", modName)

	controllerDir := filepath.Join(modDir, "Controller")
	modelDir := filepath.Join(modDir, "Model")
	viewsDir := filepath.Join(modDir, "View")

	_ = os.MkdirAll(controllerDir, 0755)
	_ = os.MkdirAll(modelDir, 0755)
	_ = os.MkdirAll(viewsDir, 0755)

	// Buat .gitkeep agar folder kosong tetap ter-commit di Git
	_ = os.WriteFile(filepath.Join(controllerDir, ".gitkeep"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(modelDir, ".gitkeep"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(viewsDir, ".gitkeep"), []byte(""), 0644)

	fmt.Printf("✅ Modul '%s' berhasil dibuat!\n", modName)
	fmt.Printf("   📁 %s\n", controllerDir)
	fmt.Printf("   📁 %s\n", modelDir)
	fmt.Printf("   📁 %s\n\n", viewsDir)
	fmt.Printf("💡 Buat controller: mulyo-go make:controller %sController -m %s --resource\n", modName, modName)
	fmt.Printf("💡 Buat model:      mulyo-go make:model %s -m %s\n", modName, modName)
	return nil
}
