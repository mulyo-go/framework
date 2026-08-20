package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutoRegisterController(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(origWd) }()

	// Create dummy go.mod
	_ = os.WriteFile("go.mod", []byte("module myapp\n\ngo 1.26.5\n"), 0644)

	// Create dummy config/registry/controllers.go
	regDir := filepath.Join("config", "registry")
	_ = os.MkdirAll(regDir, 0755)
	initialContent := `package registry

import (
	dispatcher "github.com/mulyo-go/framework/http/dispatcher"

	sampleController "myapp/Module/Sample/Controller"
)

func RegisterControllers() {
	dispatcher.RegisterController(&sampleController.BladeController{})
}
`
	_ = os.WriteFile(filepath.Join(regDir, "controllers.go"), []byte(initialContent), 0644)

	// Run autoRegisterController
	autoRegisterController("Jamal", "AnjayController")

	// Read updated content
	updated, err := os.ReadFile(filepath.Join(regDir, "controllers.go"))
	if err != nil {
		t.Fatalf("failed to read updated controllers.go: %v", err)
	}

	res := string(updated)
	if !strings.Contains(res, `jamalController "myapp/Module/Jamal/Controller"`) {
		t.Errorf("expected import not found in:\n%s", res)
	}
	if !strings.Contains(res, `dispatcher.RegisterController(&jamalController.AnjayController{})`) {
		t.Errorf("expected dispatcher registration not found in:\n%s", res)
	}
}
