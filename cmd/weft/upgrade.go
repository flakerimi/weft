package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/flakerimi/weft"
)

func upgradeCmd(args []string, stdout, stderr io.Writer) int {
	const usage = "usage is weft upgrade [--no-install]."
	fl := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	noInstall := fl.Bool("no-install", false, "do not run pnpm install")
	if err := fl.Parse(args); err != nil {
		fmt.Fprintf(stderr, "weft upgrade: %v; %s\n", err, usage)
		return 2
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "weft upgrade: %q is not a flag; %s\n", fl.Arg(0), usage)
		return 2
	}
	appDir, err := findApp(".")
	if err != nil {
		fmt.Fprintln(stderr, "weft upgrade: "+errNoApp)
		return 1
	}
	cfg, err := readApp(appDir)
	if err != nil {
		return fail(stderr, "upgrade", err)
	}
	name := cfg.Name
	if name == "" {
		name = filepath.Base(appDir)
	}
	if cfg.Version == weft.Version {
		fmt.Fprintf(stdout, "%s is already on Weft %s\n", name, weft.Version)
		return 0
	}
	if err := prerequisites(); err != nil {
		return fail(stderr, "upgrade", err)
	}
	bundle, err := openBundle()
	if err != nil {
		return fail(stderr, "upgrade", err)
	}
	if err := writeAppVersion(appDir, weft.Version); err != nil {
		return fail(stderr, "upgrade", err)
	}
	if err := unpack(bundle, appDir, weft.Version); err != nil {
		return fail(stderr, "upgrade", err)
	}
	if !*noInstall {
		if err := pnpmInstall(appDir, stdout, stderr); err != nil {
			return fail(stderr, "upgrade", err)
		}
	}
	fmt.Fprintf(stdout, "Moved %s from Weft %s to %s\n", name, cfg.Version, weft.Version)
	return 0
}
