//go:build unix

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The released binary carries the packages and the template: with no
// WEFT_BUNDLE it makes an app and builds it. release.sh runs the -short
// tests, so this one does not run itself.
func TestReleasedBinaryCreatesAndBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a release and installs an app")
	}
	if err := prerequisites(); err != nil {
		t.Skipf("prerequisites: %v", err)
	}
	status := func() string {
		t.Helper()
		out, err := exec.Command("git", "-C", "../..", "status", "--porcelain").Output()
		if err != nil {
			t.Fatalf("git status: %v", err)
		}
		return string(out)
	}
	before := status()

	releases := t.TempDir()
	rel := exec.Command("sh", "../../scripts/release.sh", "0.0.0-test", releases)
	if out, err := rel.CombinedOutput(); err != nil {
		t.Fatalf("release.sh: %v\n%s", err, out)
	}

	// The folder go:embed reads is as git has it again, and nothing else changed.
	entries, err := os.ReadDir("bundle")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "README.md" {
		t.Errorf("cmd/weft/bundle holds %v, want only README.md", entries)
	}
	if after := status(); after != before {
		t.Errorf("git status changed from %q to %q", before, after)
	}

	bin := ""
	for _, platform := range []string{"darwin_arm64", "darwin_amd64", "linux_arm64", "linux_amd64"} {
		archive, err := os.ReadFile(filepath.Join(releases, "0.0.0-test", "weft_"+platform+".tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		binary, err := extractBinary(archive)
		if err != nil {
			t.Fatalf("%s: %v", platform, err)
		}
		t.Logf("weft for %s: %d B", platform, len(binary))
		if platform != runtime.GOOS+"_"+runtime.GOARCH {
			continue
		}
		if len(binary) >= 15<<20 {
			t.Errorf("weft for %s is %d B, not under 15 MB", platform, len(binary))
		}
		bin = filepath.Join(t.TempDir(), "weft")
		if err := os.WriteFile(bin, binary, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if bin == "" {
		t.Fatalf("no release for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "WEFT_BUNDLE=") {
			env = append(env, kv)
		}
	}
	weftIn := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir, cmd.Env = dir, env
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("weft %s: %v\n%s", strings.Join(args, " "), err, out.String())
		}
		return out.String()
	}
	work := t.TempDir()
	if out := weftIn(work, "version"); !strings.HasPrefix(out, "weft 0.0.0-test (") {
		t.Errorf("weft version = %q", out)
	}
	weftIn(work, "create", "demo")
	app := filepath.Join(work, "demo")
	weftIn(app, "build")
	if got := unpacked(app); got != "0.0.0-test" {
		t.Errorf("demo/.weft/VERSION = %q, want 0.0.0-test", got)
	}
	cfg, err := readApp(app)
	if err != nil || cfg.Version != "0.0.0-test" {
		t.Errorf("readApp = %+v, %v", cfg, err)
	}
	assertServes(t, filepath.Join(app, "demo"))
}
