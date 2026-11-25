// Command weft creates, runs, builds and checks Weft applications.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type command struct {
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

var commands = map[string]command{
	"build":   {"build the pages, then the app's binary (-o <file>)", buildCmd},
	"catalog": {"list the components, print one (weft catalog <Name>) or search (--find)", catalogCmd},
	"create":  {"make an app (weft create <name>)", createCmd},
	"dev":     {"run the app, restarting Go when a .go file or .env changes (--port n)", devCmd},
	"version": {"print the Weft version", versionCmd},
	"update":  {"update weft to the latest release (--check, --version X)", updateCmd},
	"upgrade": {"move the app to this weft's Weft version (--no-install)", upgradeCmd},
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

// flagsFailed answers what a command's fl.Parse returned. -h or --help asks
// for the usage, which is printed on stdout with exit 0; any other error is
// wrong usage, said in one sentence on stderr with exit 2. It returns false
// when there was no error, and the command goes on.
func flagsFailed(err error, command, usage string, stdout, stderr io.Writer) (int, bool) {
	switch {
	case err == nil:
		return 0, false
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprintln(stdout, strings.ToUpper(usage[:1])+usage[1:])
		return 0, true
	default:
		fmt.Fprintf(stderr, "weft %s: %v; %s\n", command, err, usage)
		return 2, true
	}
}
