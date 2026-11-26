package main

import (
	"context"
	"io"
	"os/signal"
	"strings"
)

func routesCmd(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), stopSignals...)
	defer stop()
	appDir, code, ok := appCommand("routes", args, stdout, stderr)
	if !ok {
		return code
	}
	d, err := buildAndDescribe(ctx, "routes", appDir, stderr)
	if err != nil {
		return genFailed(stderr, "routes", err)
	}
	io.WriteString(stdout, formatRoutes(d))
	return 0
}

// formatRoutes is the table weft routes prints: a route a line, in the
// description's order, with its method, path, module, name and permission,
// each column as wide as its widest value. A route that takes every method
// is ANY; no module and no permission are -.
func formatRoutes(d description) string {
	rows := [][]string{{"METHOD", "PATH", "MODULE", "NAME", "PERMISSION"}}
	or := func(s, none string) string {
		if s == "" {
			return none
		}
		return s
	}
	for _, r := range d.Routes {
		rows = append(rows, []string{or(r.Method, "ANY"), r.Path, or(r.Module, "-"), r.Name, or(r.Permission, "-")})
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], len([]rune(cell)))
		}
	}
	var b strings.Builder
	for _, row := range rows {
		for i, cell := range row {
			if i == len(row)-1 {
				b.WriteString(cell)
				break
			}
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", widths[i]-len([]rune(cell))+2))
		}
		b.WriteString("\n")
	}
	return b.String()
}
