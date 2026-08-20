package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MakeCommandConsole struct{}

func init() {
	Register(&MakeCommandConsole{})
}

func (c *MakeCommandConsole) Name() string {
	return "make:command"
}

func (c *MakeCommandConsole) Aliases() []string {
	return []string{"make-command"}
}

func (c *MakeCommandConsole) Usage() string {
	return "<NamaCommand> (contoh: SyncStock)"
}

func (c *MakeCommandConsole) Description() string {
	return "create a custom terminal CLI command in app/Console"
}

func (c *MakeCommandConsole) Run(args []string) error {
	if len(args) == 0 {
		PrintCommandHelp(c.Name())
		return fmt.Errorf("command name is required (contoh: app.exe make:command SyncStock)")
	}

	rawName := strings.TrimSpace(args[0])
	structName := rawName
	if !strings.HasSuffix(structName, "Console") {
		structName += "Console"
	}

	cmdName := strings.ToLower(rawName)
	cmdName = strings.ReplaceAll(cmdName, "console", "")
	cmdName = strings.ReplaceAll(cmdName, "_", "-")

	dir := filepath.Join("app", "Console")
	_ = os.MkdirAll(dir, 0755)

	filePath := filepath.Join(dir, structName+".go")

	template := fmt.Sprintf(`package console

import (
	"flag"
	"fmt"

	"github.com/mulyo-go/framework/console"
)

type %s struct{}

func init() {
	console.Register(&%s{})
}

func (c *%s) Name() string {
	return "%s"
}

func (c *%s) Aliases() []string {
	return []string{"%s"}
}

func (c *%s) Usage() string {
	return "[-flag <nilai>] [parameter]"
}

func (c *%s) Description() string {
	return "deskripsi command %s"
}

func (c *%s) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	testFlag := fs.String("test", "", "contoh flag")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("Menjalankan command %%s (flag: %%s)\n", c.Name(), *testFlag)
	return nil
}
`, structName, structName, structName, cmdName, structName, rawName, structName, structName, cmdName, structName)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("gagal menulis file command: %w", err)
	}

	fmt.Printf("✅ Berhasil membuat console command: %s\n", filePath)
	return nil
}
