package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MakeMiddlewareConsole struct{}

func init() {
	Register(&MakeMiddlewareConsole{})
}

func (c *MakeMiddlewareConsole) Name() string {
	return "make:middleware"
}

func (c *MakeMiddlewareConsole) Aliases() []string {
	return []string{"make-middleware"}
}

func (c *MakeMiddlewareConsole) Usage() string {
	return "<NamaMiddleware> (contoh: CheckApiKey)"
}

func (c *MakeMiddlewareConsole) Description() string {
	return "create a new middleware file in app/Http/Middleware"
}

func (c *MakeMiddlewareConsole) Run(args []string) error {
	if len(args) == 0 {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("middleware name is required (contoh: app.exe make:middleware CheckRole)")
	}

	name := strings.TrimSpace(args[0])
	dir := filepath.Join("app", "Http", "Middleware")
	_ = os.MkdirAll(dir, 0755)

	filePath := filepath.Join(dir, strings.ToLower(name)+".go")

	template := fmt.Sprintf(`package middleware

import (
	"github.com/gin-gonic/gin"
)

// %s custom middleware
func %s() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Logika sebelum request diproses
		// Contoh: c.GetHeader("X-Api-Key")

		c.Next()

		// Logika setelah request selesai diproses
	}
}
`, name, name)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("gagal menulis file middleware: %w", err)
	}

	fmt.Printf("✅ Successfully created middleware: %s\n", filePath)
	return nil
}
