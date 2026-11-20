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
