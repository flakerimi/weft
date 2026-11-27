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
// WEFT_BUNDLE it makes an app and builds it. The app requires Warp's module
// at the release, which Go fetches through the module proxy: here a proxy in
// a folder serves the module as a tag of this working tree would have it.
// release.sh runs the -short tests, so this one does not run itself.
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
		if !strings.HasPrefix(kv, "WEFT_BUNDLE=") && !strings.HasPrefix(kv, "GOFLAGS=") {
			env = append(env, kv)
		}
	}
	env = append(env, moduleProxy(t, "v0.0.0-test", repoModule(t))...)
	// The app requires the SQLite driver too, which the proxy in a folder
	// does not serve: after it, the module cache this machine has, which
	// go mod download in dbtest fills with the driver at the version the
	// template requires, serves it offline.
	for i, kv := range env {
		if strings.HasPrefix(kv, "GOPROXY=") {
			env[i] = kv + "," + fileProxyURL(filepath.ToSlash(filepath.Join(sqliteCache(t), "cache", "download")))
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
	if got, want := readFile(t, filepath.Join(app, "go.mod")), strings.Replace(templateGoMod("v0.0.0-test", false), "module myapp", "module demo", 1); got != want {
		t.Errorf("demo/go.mod = %q, want %q", got, want)
	}
	if exists(t, filepath.Join(app, ".weft", "go")) {
		t.Error("the release unpacked a Go package into demo/.weft/go")
	}
	if sum := readFile(t, filepath.Join(app, "go.sum")); !strings.Contains(sum, "github.com/flakerimi/weft v0.0.0-test h1:") {
		t.Errorf("demo/go.sum = %q, want the module's checksum", sum)
	}
	weftIn(app, "build")
	// The app was built with the module the proxy served, and Version
	// names its release.
	if out, err := exec.Command("go", "version", "-m", filepath.Join(app, "demo")).Output(); err != nil ||
		!strings.Contains(string(out), "\tdep\tgithub.com/flakerimi/weft\tv0.0.0-test\t") {
		t.Errorf("go version -m demo: %v\n%s", err, out)
	}
	if got := unpacked(app); got != "0.0.0-test" {
		t.Errorf("demo/.weft/VERSION = %q, want 0.0.0-test", got)
	}
	cfg, err := readApp(app)
	if err != nil || cfg.Version != "0.0.0-test" {
		t.Errorf("readApp = %+v, %v", cfg, err)
	}
	assertServes(t, filepath.Join(app, "demo"))
}

// sqliteCache is this machine's module cache, holding the SQLite driver and
// the modules it needs, as dbtest/go.mod requires them.
func sqliteCache(t *testing.T) string {
	t.Helper()
	dl := exec.Command("go", "mod", "download", "modernc.org/sqlite")
	dl.Dir = filepath.Join("..", "..", "dbtest")
	if out, err := dl.CombinedOutput(); err != nil {
		t.Fatalf("go mod download modernc.org/sqlite in dbtest: %v\n%s", err, out)
	}
	list := exec.Command("go", "list", "-deps", "-test", "./...")
	list.Dir = dl.Dir
	if out, err := list.CombinedOutput(); err != nil {
		t.Fatalf("go list in dbtest, which downloads what the driver needs: %v\n%s", err, out)
	}
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("go env GOMODCACHE: %v", err)
	}
	return strings.TrimSpace(string(out))
}
