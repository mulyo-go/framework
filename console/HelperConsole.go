package console

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mulyo-go/framework/config"
)

var (
	AppBootstrapper func()
	AppFactory      func() http.Handler
)

func SetAppBootstrapper(fn func()) {
	AppBootstrapper = fn
}

func SetAppFactory(fn func() http.Handler) {
	AppFactory = fn
}

func PrintUnknown(name string) {
	fmt.Fprintf(os.Stderr, "Unknown command: '%s'\n", name)
	fmt.Fprintln(os.Stderr, "Try: mulyo-go help or mulyo-go -h")
}

func PrintHelp() {
	fmt.Println("Usage:")
	fmt.Println("  mulyo-go <command> [options] [arguments]")
	fmt.Println("  mulyo-go help")
	fmt.Println("  mulyo-go help <command>")
	fmt.Println("  mulyo-go -h")
	fmt.Println("")
	fmt.Println("Start server:")
	fmt.Println("  mulyo-go serve")
	fmt.Println("  mulyo-go serve [-p <port>] [--watch|--debug]")
	fmt.Println("")
	fmt.Println("Available commands:")
	for _, cmd := range SortedCommands() {
		line := "  " + cmd.Name()
		if u := strings.TrimSpace(cmd.Usage()); u != "" {
			line += " " + u
		}
		if d := strings.TrimSpace(cmd.Description()); d != "" {
			line += "  -  " + d
		}
		fmt.Println(line)
	}
}

func PrintCommandHelp(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		PrintHelp()
		return
	}

	cmd, ok := registry[name]
	if !ok {
		PrintUnknown(name)
		return
	}

	fmt.Println("Command:", cmd.Name())
	if a := cmd.Aliases(); len(a) > 0 {
		fmt.Println("Aliases:", strings.Join(a, ", "))
	}
	if d := strings.TrimSpace(cmd.Description()); d != "" {
		fmt.Println("Description:", d)
	}
	if u := strings.TrimSpace(cmd.Usage()); u != "" {
		fmt.Println("Usage:")
		fmt.Println("  mulyo-go", cmd.Name(), u)
	} else {
		fmt.Println("Usage:")
		fmt.Println("  mulyo-go", cmd.Name())
	}
}

var ServeCommandName = "serve"

type ServeConsole struct{}

func init() {
	Register(&ServeConsole{})
}

func (c *ServeConsole) Name() string {
	return ServeCommandName
}

func (c *ServeConsole) Aliases() []string {
	return []string{"server"}
}

func (c *ServeConsole) Usage() string {
	return "[-p <port>] [--watch|--debug]"
}

func (c *ServeConsole) Description() string {
	return "start the Gin HTTP web server"
}

func (c *ServeConsole) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	portFlag := fs.String("p", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}

	config.Load()

	port := strings.TrimSpace(*portFlag)
	if port == "" {
		port = config.AppPort()
	}
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	if AppBootstrapper != nil {
		AppBootstrapper()
	}

	var handler http.Handler
	if AppFactory != nil {
		handler = AppFactory()
	}

	displayURL := port
	if strings.HasPrefix(displayURL, ":") {
		displayURL = "http://localhost" + displayURL
	} else if !strings.HasPrefix(displayURL, "http://") && !strings.HasPrefix(displayURL, "https://") {
		displayURL = "http://" + displayURL
	}

	fmt.Println("==================================================")
	fmt.Printf("🚀 Application is running on %s\n", displayURL)
	fmt.Println("--------------------------------------------------")

	config.CheckConnectionsStatus()
	fmt.Println("==================================================")

	if handler == nil {
		fmt.Println("Error: App handler is not configured.")
		return nil
	}

	srv := &http.Server{Addr: port, Handler: handler}
	go func() { _ = srv.ListenAndServe() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	return nil
}
