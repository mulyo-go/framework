package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const Version = "v1.0.0"

func main() {
	if len(os.Args) <= 1 {
		printBanner()
		printHelp()
		return
	}

	arg := strings.TrimSpace(os.Args[1])

	switch arg {
	case "version", "-v", "--version":
		fmt.Printf("Mulyo Go CLI %s\n", Version)
		return
	case "help", "-h", "--help", "-H":
		printBanner()
		printHelp()
		return
	case "self-update", "update", "upgrade":
		selfUpdate()
		return
	case "new":
		if len(os.Args) <= 2 {
			fmt.Println("Error: Project name is required.")
			fmt.Println("Example: mulyo-go new my-app")
			return
		}
		createNewProject(os.Args[2])
		return
	case "dev":
		if isMulyoProject() {
			runHotReload(os.Args[2:])
			return
		}
		printBanner()
		fmt.Println("Error: 'dev' command can only be executed inside a Mulyo Go project.")
		return
	default:
		if isMulyoProject() {
			if arg == "serve" || arg == "server" {
				if hasHotReloadFlag(os.Args[2:]) {
					runHotReload(filterHotReloadFlags(os.Args[2:]))
					return
				}
			}
			runInProject(os.Args[1:])
			return
		}

		printBanner()
		fmt.Printf("Command '%s' is unknown or executed outside a Mulyo Go project.\n\n", arg)
		printHelp()
	}
}

func printBanner() {
	fmt.Println("  __  __       _              ____        ")
	fmt.Println(" |  \\/  |_   _| |_   _  ___   / ___| ___  ")
	fmt.Println(" | |\\/| | | | | | | | |/ _ \\ | |  _ / _ \\ ")
	fmt.Println(" | |  | | |_| | | |_| | (_) || |_| | (_) |")
	fmt.Println(" |_|  |_|\\__,_|_|\\__, |\\___/  \\____|\\___/ ")
	fmt.Println("                 |___/                    ")
	fmt.Printf(" Mulyo Go Framework CLI Tool (%s)\n\n", Version)
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  mulyo-go <command> [options] [arguments]")
	fmt.Println()
	fmt.Println("CLI Commands:")
	fmt.Println("  self-update / update                                   - Update mulyo-go CLI to the latest version")
	fmt.Println("  version / -v                                           - Display mulyo-go CLI version")
	fmt.Println("  new <project-name>                                     - Create a new project from starter kit")
	fmt.Println()
	fmt.Println("Development Commands (inside project):")
	fmt.Println("  serve [-p <port>] [--watch|--debug|-w|-d]              - Start HTTP server (optional: hot-reload)")
	fmt.Println("  dev [-p <port>]                                        - Start server in hot-reload mode (Air)")
	fmt.Println("  make:module <ModuleName>                               - Scaffold a new module directory structure (Controller, Model, Views)")
	fmt.Println("  make:controller <Name> [-m <Module>] [--resource]      - Create a new controller")
	fmt.Println("  make:model <Name> [-m <Module>] [-t <table>] [-mig]    - Create a GORM struct model")
	fmt.Println("  make:migration <Name> [-db <conn>]                     - Create a new database migration file")
	fmt.Println("  make:middleware <Name>                                 - Create a new HTTP middleware")
	fmt.Println("  make:command <Name>                                    - Create a custom terminal CLI command")
	fmt.Println("  make:seeder <Name>                                     - Create a new database seeder")
	fmt.Println("  migrate [-db <name>] [--rollback] [--status] [--fresh] - Run database migrations")
	fmt.Println("  db:seed [-db <name>] [-class <SeederName>]             - Run database seeders")
	fmt.Println("  route:list                                             - Display all registered routes & controllers")
	fmt.Println("  route:generate                                         - Synchronize auto-routes to database")
	fmt.Println()
}

func hasHotReloadFlag(args []string) bool {
	for _, a := range args {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "--watch" || a == "-w" || a == "--debug" || a == "-debug" || a == "-d" || a == "--dev" || a == "-dev" {
			return true
		}
	}
	return false
}

func filterHotReloadFlags(args []string) []string {
	var clean []string
	for _, a := range args {
		lower := strings.ToLower(strings.TrimSpace(a))
		if lower != "--watch" && lower != "-w" && lower != "--debug" && lower != "-debug" && lower != "-d" && lower != "--dev" && lower != "-dev" {
			clean = append(clean, a)
		}
	}
	return clean
}

func extractPort(args []string) string {
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		if a == "-p" || a == "--port" {
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
		} else if strings.HasPrefix(a, "-p=") || strings.HasPrefix(a, "--port=") {
			return strings.TrimSpace(strings.SplitN(a, "=", 2)[1])
		}
	}
	return ""
}

func runHotReload(extraArgs []string) {
	fmt.Println("🔥 Starting server in hot-reload mode (live reloading)...")
	airPath, err := exec.LookPath("air")
	if err != nil {
		fmt.Println("📦 'air' live-reloader not found in PATH. Installing automatically...")
		installCmd := exec.Command("go", "install", "github.com/air-verse/air@latest")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install 'air': %v\n", err)
			fmt.Println("💡 Falling back to standard server runner without hot-reload...")
			runInProject(append([]string{"serve"}, extraArgs...))
			return
		}
		airPath, err = exec.LookPath("air")
		if err != nil {
			if gopath := os.Getenv("GOPATH"); gopath != "" {
				p := filepath.Join(gopath, "bin", "air.exe")
				if _, err := os.Stat(p); err == nil {
					airPath = p
				}
			}
			if airPath == "" {
				if home, err := os.UserHomeDir(); err == nil {
					p := filepath.Join(home, "go", "bin", "air.exe")
					if _, err := os.Stat(p); err == nil {
						airPath = p
					}
				}
			}
		}
	}
	ensureAirConfig()

	airExec := "air"
	if airPath != "" {
		airExec = airPath
	}

	airArgs := append([]string{"--", "serve"}, extraArgs...)
	cmd := exec.Command(airExec, airArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	portVal := extractPort(extraArgs)
	if portVal != "" {
		cleanPort := strings.TrimPrefix(portVal, ":")
		cmd.Env = append(os.Environ(), "APP_PORT="+cleanPort, "PORT="+cleanPort)
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func ensureAirConfig() {
	if _, err := os.Stat(".air.toml"); err == nil {
		return
	}
	defaultAirToml := "[build]\n  cmd = \"go build -o ./tmp/main.exe .\"\n  bin = \"./tmp/main.exe\"\n  args_bin = [\"serve\"]\n  delay = 1000\n  exclude_dir = [\"tmp\", \"storage\", \"vendor\", \"node_modules\", \".git\", \"Web\"]\n  include_ext = [\"go\", \"toml\", \"html\"]\n  kill_on_error = true\n"
	_ = os.WriteFile(".air.toml", []byte(defaultAirToml), 0644)
	fmt.Println("📄 Configuration file '.air.toml' created successfully.")
}

func selfUpdate() {
	fmt.Printf("🔄 Updating Mulyo Go CLI (current version: %s)...\n", Version)
	fmt.Println("📦 Running: go install github.com/mulyo-go/framework/cmd/mulyo-go@latest")
	cmd := exec.Command("go", "install", "github.com/mulyo-go/framework/cmd/mulyo-go@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Failed to update CLI: %v\n", err)
		return
	}
	fmt.Println("\n✅ Mulyo Go CLI updated successfully to the latest version!")
	fmt.Println("Type 'mulyo-go version' to check the version.")
}

func isMulyoProject() bool {
	if _, err := os.Stat("go.mod"); err == nil {
		if _, err := os.Stat("main.go"); err == nil {
			return true
		}
	}
	return false
}

func runInProject(args []string) {
	cmdArgs := append([]string{"run", "."}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func createNewProject(projectName string) {
	targetDir := filepath.Clean(projectName)
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("❌ Directory '%s' already exists. Please choose another name.\n", targetDir)
		return
	}
	fmt.Printf("🚀 Creating new project '%s' from Mulyo Starter Kit...\n", projectName)
	repoURL := "https://github.com/mulyo-go/framework.git"
	cmd := exec.Command("git", "clone", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Failed to clone repository: %v\n", err)
		return
	}
	fmt.Println("\n✅ Project created successfully!")
	fmt.Printf("Next steps:\n  cd %s\n  mulyo-go serve\n\n", projectName)
}