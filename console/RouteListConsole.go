package console

import (
	"fmt"
	"strings"

	"github.com/mulyo-go/framework/http/dispatcher"
)

type RouteListConsole struct{}

func init() {
	Register(&RouteListConsole{})
}

func (c *RouteListConsole) Name() string {
	return "route:list"
}

func (c *RouteListConsole) Aliases() []string {
	return []string{"routes", "routes:list"}
}

func (c *RouteListConsole) Usage() string {
	return ""
}

func (c *RouteListConsole) Description() string {
	return "display all registered routes & controllers"
}

func (c *RouteListConsole) Run(args []string) error {
	if AppBootstrapper != nil {
		AppBootstrapper()
	}

	controllers := dispatcher.ListControllers()
	if len(controllers) == 0 {
		fmt.Println("Tidak ada controller yang terdaftar.")
		return nil
	}

	fmt.Printf("\n=== Daftar Route & Controller Mulyo Go (%d Controllers) ===\n", len(controllers))
	fmt.Printf("%-20s | %-35s | %-35s\n", "MODULE / PLUGIN", "CONTROLLER", "ACTION / METHOD")
	fmt.Println(strings.Repeat("-", 96))

	for _, ctrl := range controllers {
		for _, m := range ctrl.Methods {
			fmt.Printf("%-20s | %-35s | %-35s\n", ctrl.Module, ctrl.Controller, m)
		}
	}
	fmt.Println()
	return nil
}
