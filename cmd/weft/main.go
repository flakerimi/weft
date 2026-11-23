// Command weft creates, runs, builds and checks Weft applications.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
)

type command struct {
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

var commands = map[string]command{
	"create":  {"make an app (weft create <name>)", createCmd},
	"version": {"print the Weft version", versionCmd},
	"update":  {"update weft to the latest release (--check, --version X)", updateCmd},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		help(stdout)
		return 0
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "weft: unknown command %q. Run \"weft help\".\n", args[0])
		return 2
	}
	return cmd.run(args[1:], stdout, stderr)
}

func help(w io.Writer) {
	fmt.Fprintln(w, "weft — one UI system for Go and Vue applications")
	fmt.Fprintln(w, "\nUsage: weft <command>\n\nCommands:")
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(w, "  %-10s %s\n", n, commands[n].summary)
	}
}
