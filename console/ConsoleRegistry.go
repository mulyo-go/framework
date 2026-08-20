package console

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type Command interface {
	Name() string
	Aliases() []string
	Usage() string
	Description() string
	Run(args []string) error
}

var registry = map[string]Command{}

func Register(cmd Command) {
	name := strings.TrimSpace(cmd.Name())
	if name == "" {
		return
	}
	registry[name] = cmd
	for _, a := range cmd.Aliases() {
		alias := strings.TrimSpace(a)
		if alias == "" {
			continue
		}
		registry[alias] = cmd
	}
}

func Execute(args []string) int {
	if len(args) == 0 {
		PrintHelp()
		return 0
	}

	cmdName := strings.TrimSpace(args[0])
	if cmdName == "" {
		PrintHelp()
		return 0
	}

	if cmdName == "help" || cmdName == "-H" || cmdName == "--help" || cmdName == "-h" {
		if len(args) >= 2 {
			PrintCommandHelp(args[1])
			return 0
		}
		PrintHelp()
		return 0
	}

	cmd, ok := registry[cmdName]
	if !ok {
		PrintUnknown(cmdName)
		return 0
	}

	if err := cmd.Run(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 0
	}
	return 0
}

func SortedCommands() []Command {
	seen := map[string]struct{}{}
	var list []Command
	for _, cmd := range registry {
		n := cmd.Name()
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		list = append(list, cmd)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}
