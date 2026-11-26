package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/flakerimi/weft"
)

const oldBinary = "old weft binary"

// updateEnv serves a release tree, pretends this build is `current`, and points
// the update at a temporary "installed" binary.
func updateEnv(t *testing.T, current string, releases ...string) (target string) {
	t.Helper()
	dir := t.TempDir()
	for _, v := range releases {
		releaseTree(t, dir, v, false)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)
	t.Setenv("WEFT_BASE", srv.URL)

	target = filepath.Join(t.TempDir(), "weft")
	os.WriteFile(target, []byte(oldBinary), 0o755)

	prevVersion, prevExe := weft.Version, executablePath
	weft.Version = current
	executablePath = func() (string, error) { return target, nil }
	t.Cleanup(func() { weft.Version, executablePath = prevVersion, prevExe })
	return target
}

func contents(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUpdateUpToDate(t *testing.T) {
	target := updateEnv(t, "0.0.1", "0.0.1")
	code, out, errs := runCmd("update")
	if code != 0 || out != "weft 0.0.1 is up to date.\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if contents(t, target) != oldBinary {
		t.Fatal("binary changed")
	}
}

func TestUpdateInstallsNewer(t *testing.T) {
	target := updateEnv(t, "0.0.1", "0.0.2")
	code, out, errs := runCmd("update")
	if code != 0 || out != "Updated weft 0.0.1 → 0.0.2.\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if !strings.Contains(contents(t, target), "fake weft") {
		t.Fatal("binary not replaced")
	}
	// Windows has no mode bits for running a file; its name says.
	if fi, _ := os.Stat(target); runtime.GOOS != "windows" && fi.Mode()&0o111 == 0 {
		t.Fatal("not executable")
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".weft*")); len(left) != 0 {
		t.Fatalf("temporary files left: %v", left)
	}
}

func TestUpdateCheckOnly(t *testing.T) {
	target := updateEnv(t, "0.0.1", "0.0.2")
	code, out, _ := runCmd("update", "--check")
	if code != 0 || out != "weft 0.0.2 is available (you have 0.0.1). Run \"weft update\" to install it.\n" {
		t.Fatalf("exit %d, stdout %q", code, out)
	}
	if contents(t, target) != oldBinary {
		t.Fatal("--check changed the binary")
	}
}

func TestUpdatePinnedVersion(t *testing.T) {
	target := updateEnv(t, "0.0.2", "0.0.1", "0.0.2")
	code, out, errs := runCmd("update", "--version", "0.0.1")
	if code != 0 || out != "Updated weft 0.0.2 → 0.0.1.\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if !strings.Contains(contents(t, target), "fake weft") {
		t.Fatal("binary not replaced")
	}
}

func TestUpdateBadChecksum(t *testing.T) {
	target := updateEnv(t, "0.0.1")
	dir := t.TempDir()
	name := releaseTree(t, dir, "0.0.2", true)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	t.Setenv("WEFT_BASE", srv.URL)

	code, _, errs := runCmd("update")
	if code != 1 || errs != "weft update: checksum mismatch for "+name+".\n" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if contents(t, target) != oldBinary {
		t.Fatal("binary replaced despite bad checksum")
	}
}

func TestUpdateUnknownVersion(t *testing.T) {
	updateEnv(t, "0.0.1", "0.0.2")
	code, _, errs := runCmd("update", "--version", "9.9.9")
	if code != 1 || !strings.HasPrefix(errs, "weft update: there is no weft 9.9.9 for ") {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
}

func TestUpdateUnreachable(t *testing.T) {
	updateEnv(t, "0.0.1")
	t.Setenv("WEFT_BASE", "http://127.0.0.1:1")
	code, _, errs := runCmd("update")
	if code != 1 || errs != "weft update: could not reach http://127.0.0.1:1 to find the latest version.\n" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
}

func TestUpdateDevBuild(t *testing.T) {
	updateEnv(t, "0.0.0-dev", "0.0.2")
	code, _, errs := runCmd("update")
	want := "weft update: this is a development build; install a release with: curl -fsSL https://weft.base.al/install | sh\n"
	if code != 1 || errs != want {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
}

func TestUpdateListedInHelp(t *testing.T) {
	_, out, _ := runCmd("help")
	if !strings.Contains(out, "update") {
		t.Fatalf("help %q", out)
	}
}

// The Windows release is a zip holding weft.exe, as scripts/release.sh makes
// it; the others are tarballs holding weft.
func TestUpdateArchiveForEachSystem(t *testing.T) {
	for goos, want := range map[string]string{"windows": "weft_windows_arm64.zip", "linux": "weft_linux_arm64.tar.gz", "darwin": "weft_darwin_arm64.tar.gz"} {
		if got := archiveName(goos, "arm64"); got != want {
			t.Errorf("archiveName(%q, arm64) = %q, want %q", goos, got, want)
		}
	}
	got, err := extract("weft_windows_amd64.zip", zipOf(t, "weft.exe", []byte("windows weft")))
	if err != nil || string(got) != "windows weft" {
		t.Errorf("extract of a zip = %q, %v; want windows weft", got, err)
	}
	if _, err := extract("weft_windows_amd64.zip", zipOf(t, "weft", []byte("no exe"))); err == nil || err.Error() != "no weft.exe inside" {
		t.Errorf("extract of a zip without weft.exe = %v, want no weft.exe inside", err)
	}
}

// On Windows the running binary is renamed to weft.old, which may be done to
// a running one, before the new one takes its name; an older weft.old is
// replaced, and a failure puts the running one back.
func TestUpdateSwapsAside(t *testing.T) {
	dir := t.TempDir()
	exe, tmp := filepath.Join(dir, "weft.exe"), filepath.Join(dir, ".weft.new-1")
	os.WriteFile(exe, []byte("running"), 0o755)
	os.WriteFile(exe+".old", []byte("older"), 0o755)
	os.WriteFile(tmp, []byte("new"), 0o755)
	if err := swapIn(tmp, exe, true); err != nil {
		t.Fatal(err)
	}
	if contents(t, exe) != "new" || contents(t, exe+".old") != "running" {
		t.Errorf("after swapIn: weft.exe %q, weft.exe.old %q; want new and running", contents(t, exe), contents(t, exe+".old"))
	}
	if err := swapIn(filepath.Join(dir, "missing"), exe, true); err == nil {
		t.Error("swapIn of a missing file succeeded")
	}
	if contents(t, exe) != "new" {
		t.Errorf("a failed swapIn left weft.exe %q, want it put back as new", contents(t, exe))
	}
}
