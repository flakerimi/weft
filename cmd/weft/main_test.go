package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func runCmd(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := runCmd("version")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	re := regexp.MustCompile(`^weft 0\.0\.0-dev \((darwin|linux|windows)/(amd64|arm64)\)\n$`)
	if !re.MatchString(out) {
		t.Fatalf("stdout %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errs := runCmd("nope")
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	want := "weft: unknown command \"nope\". Run \"weft help\".\n"
	if errs != want {
		t.Fatalf("stderr %q, want %q", errs, want)
	}
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{{}, {"help"}} {
		code, out, _ := runCmd(args...)
		if code != 0 || !strings.Contains(out, "version") {
			t.Fatalf("args %v: exit %d, stdout %q", args, code, out)
		}
	}
}

// -h and --help ask for a command's usage: it is printed on stdout, with
// exit 0, not reported as a wrong flag.
func TestCommandHelp(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, c := range []struct{ command, usage string }{
		{"create", "Usage is weft create <name> [--module <path>] [--theme neutral|lisa|weft|synthwave|editor|sandstorm] [--no-install].\n"},
		{"build", "Usage is weft build [-o <file>].\n"},
		{"dev", "Usage is weft dev [--port n].\n"},
		{"gen", "Usage is weft gen.\n"},
		{"routes", "Usage is weft routes.\n"},
		{"upgrade", "Usage is weft upgrade [--no-install].\n"},
		{"catalog", "Usage is weft catalog [<Name>...] [--examples] [--json], or weft catalog --find <word> [<word>...].\n"},
	} {
		for _, flag := range []string{"-h", "--help"} {
			code, out, errs := runCmd(c.command, flag)
			if code != 0 || out != c.usage || errs != "" {
				t.Errorf("weft %s %s: exit %d, stdout %q, stderr %q, want only %q", c.command, flag, code, out, errs, c.usage)
			}
		}
	}
}
