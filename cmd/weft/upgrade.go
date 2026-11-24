package main

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

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
	if newerVersion(cfg.Version, weft.Version) {
		fmt.Fprintf(stderr, "weft upgrade: this app is on Weft %s, newer than this weft %s; run weft update to get %s or later.\n", cfg.Version, weft.Version, cfg.Version)
		return 1
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

// newerVersion says whether version a is newer than b, by semantic
// versioning's precedence, for versions shaped as scripts/version.sh accepts
// (1.2.3, 1.2.3-alpha.1). A development build's version (one ending -dev) is
// older than any other. A version of another shape is never newer, nor older.
func newerVersion(a, b string) bool {
	if !versionShape.MatchString(a) || !versionShape.MatchString(b) {
		return false
	}
	if devBuild(a) != devBuild(b) {
		return devBuild(b)
	}
	return compareVersions(a, b) > 0
}

func devBuild(v string) bool { return strings.HasSuffix(v, "-dev") }

// compareVersions returns -1, 0 or 1 as a is older than, as old as, or newer
// than b, and 0 when either is not a version.
func compareVersions(a, b string) int {
	ma, mb := versionShape.FindStringSubmatch(a), versionShape.FindStringSubmatch(b)
	if ma == nil || mb == nil {
		return 0
	}
	for i := 1; i <= 3; i++ {
		if c := compareNumbers(ma[i], mb[i]); c != 0 {
			return c
		}
	}
	pa, pb := ma[4], mb[4]
	switch {
	case pa == pb:
		return 0
	case pa == "":
		return 1 // a release is newer than its pre-releases
	case pb == "":
		return -1
	}
	fa, fb := strings.Split(pa, "."), strings.Split(pb, ".")
	for i := 0; i < len(fa) && i < len(fb); i++ {
		na, nb := numeric.MatchString(fa[i]), numeric.MatchString(fb[i])
		var c int
		switch {
		case na && nb:
			c = compareNumbers(fa[i], fb[i])
		case na:
			c = -1 // a number is older than a word
		case nb:
			c = 1
		default:
			c = strings.Compare(fa[i], fb[i])
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(fa), len(fb))
}

var (
	versionShape = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-z0-9.]+))?$`)
	numeric      = regexp.MustCompile(`^\d+$`)
)

// compareNumbers compares two strings of digits by their value, however long.
func compareNumbers(a, b string) int {
	a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
	if c := cmp.Compare(len(a), len(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
