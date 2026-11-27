package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// releaseTree writes a fake release under dir/releases/<version> for the host platform.
func releaseTree(t *testing.T, dir, version string, badSum bool) string {
	t.Helper()
	name := archiveName(runtime.GOOS, runtime.GOARCH)
	var buf bytes.Buffer
	body := []byte("#!/bin/sh\necho fake weft\n")
	if runtime.GOOS == "windows" {
		buf.Write(zipOf(t, "weft.exe", body))
	} else {
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "weft", Mode: 0o755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		tw.Write(body)
		tw.Close()
		gz.Close()
	}
	rel := filepath.Join(dir, "releases", version)
	os.MkdirAll(rel, 0o755)
	os.WriteFile(filepath.Join(rel, name), buf.Bytes(), 0o644)
	sum := sha256.Sum256(buf.Bytes())
	hex := fmt.Sprintf("%x", sum)
	if badSum {
		hex = strings.Repeat("0", 64)
	}
	os.WriteFile(filepath.Join(rel, "SHA256SUMS"), []byte(hex+"  "+name+"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "releases", "latest"), []byte(version+"\n"), 0o644)
	return name
}

// zipOf is a zip holding body as name, as scripts/release.sh zips weft.exe.
func zipOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(body)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func runInstaller(t *testing.T, base, bin string, extra ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command("sh", "../../site/install")
	cmd.Env = append(os.Environ(), "WEFT_BASE="+base, "WEFT_BIN_DIR="+bin)
	cmd.Env = append(cmd.Env, extra...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, out.String(), errb.String()
}

func skipWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer is POSIX sh")
	}
}

func TestInstallLatest(t *testing.T) {
	skipWindows(t)
	dir, bin := t.TempDir(), t.TempDir()
	releaseTree(t, dir, "0.0.1", false)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	code, out, errs := runInstaller(t, srv.URL, bin)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	target := filepath.Join(bin, "weft")
	if !strings.Contains(out, "Installed weft 0.0.1 to "+target+"\n") {
		t.Fatalf("stdout %q", out)
	}
	fi, err := os.Stat(target)
	if err != nil || fi.Mode()&0o111 == 0 {
		t.Fatalf("binary missing or not executable: %v", err)
	}
}

func TestInstallBadChecksum(t *testing.T) {
	skipWindows(t)
	dir, bin := t.TempDir(), t.TempDir()
	name := releaseTree(t, dir, "0.0.1", true)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	code, _, errs := runInstaller(t, srv.URL, bin)
	if code != 1 || errs != "weft install: checksum mismatch for "+name+"\n" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if _, err := os.Stat(filepath.Join(bin, "weft")); !os.IsNotExist(err) {
		t.Fatalf("binary should not exist")
	}
}

func TestInstallUnsupportedPlatform(t *testing.T) {
	skipWindows(t)
	dir, bin := t.TempDir(), t.TempDir()
	releaseTree(t, dir, "0.0.1", false)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	code, _, errs := runInstaller(t, srv.URL, bin, "WEFT_TEST_UNAME_S=Plan9", "WEFT_TEST_UNAME_M=x86_64")
	if code != 1 || errs != "weft install: no build for plan9/amd64\n" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
}

func TestInstallKeepsOldBinaryOnFailure(t *testing.T) {
	skipWindows(t)
	dir, bin := t.TempDir(), t.TempDir()
	releaseTree(t, dir, "0.0.1", true)
	target := filepath.Join(bin, "weft")
	os.WriteFile(target, []byte("old"), 0o755)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()
	if code, _, _ := runInstaller(t, srv.URL, bin); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if b, _ := os.ReadFile(target); string(b) != "old" {
		t.Fatalf("old binary replaced: %q", b)
	}
}

func TestInstallUnreachable(t *testing.T) {
	skipWindows(t)
	code, _, errs := runInstaller(t, "http://127.0.0.1:1", t.TempDir())
	if code != 1 || !strings.HasPrefix(errs, "weft install: ") || strings.Count(errs, "\n") != 1 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
}

// releaseTreeFor writes a fake release under dir/releases/<version> for goos/goarch.
func releaseTreeFor(t *testing.T, dir, version, goos, goarch string, badSum bool) string {
	t.Helper()
	name := archiveName(goos, goarch)
	var buf bytes.Buffer
	body := []byte("#!/bin/sh\necho fake weft\n")
	if goos == "windows" {
		buf.Write(zipOf(t, "weft.exe", body))
	} else {
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "weft", Mode: 0o755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		tw.Write(body)
		tw.Close()
		gz.Close()
	}
	rel := filepath.Join(dir, "releases", version)
	os.MkdirAll(rel, 0o755)
	os.WriteFile(filepath.Join(rel, name), buf.Bytes(), 0o644)
	sum := sha256.Sum256(buf.Bytes())
	hex := fmt.Sprintf("%x", sum)
	if badSum {
		hex = strings.Repeat("0", 64)
	}
	os.WriteFile(filepath.Join(rel, "SHA256SUMS"), []byte(hex+"  "+name+"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "releases", "latest"), []byte(version+"\n"), 0o644)
	return name
}

// The PowerShell installer, site/install.ps1, maps the Windows architecture
// (AMD64, x86_64, x64 -> amd64; ARM64, aarch64 -> arm64), refuses any other
// with "weft install: no build for windows/<arch>", reads SHA256SUMS, checks
// the archive's SHA-256, extracts weft.exe into $HOME\.local\bin (or
// $env:WEFT_BIN_DIR) via a temporary file so an old weft.exe is kept on
// failure, and updates the user's persistent Path with
// [Environment]::SetEnvironmentVariable(..., 'User') unless
// $env:WEFT_NO_MODIFY_PATH is 1.
func TestInstallPowerShellScript(t *testing.T) {
	b, err := os.ReadFile("../../site/install.ps1")
	if err != nil {
		t.Fatalf("site/install.ps1: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "\r") {
		t.Error("site/install.ps1 has CR line endings; keep LF in git")
	}
	for _, want := range []string{
		"irm https://weft.base.al/install.ps1 | iex",
		"WEFT_BASE",
		"WEFT_VERSION",
		"WEFT_BIN_DIR",
		"WEFT_TEST_ARCH",
		"WEFT_NO_MODIFY_PATH",
		"function Get-WeftArch",
		"function Get-WeftChecksum",
		"weft install: $msg",
		"no build for windows/",
		"checksum mismatch for ",
		`.local\bin`,
		"SetEnvironmentVariable('Path',",
		"'User'",
		"Installed weft ",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("site/install.ps1 does not contain %q", want)
		}
	}
	if strings.Contains(s, "'Machine'") || strings.Contains(s, `"Machine"`) {
		t.Error("site/install.ps1 touches the Machine Path; it must only touch 'User'")
	}
}

func findPowerShell() string {
	for _, name := range []string{"pwsh", "powershell"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func runPowerShellInstaller(t *testing.T, ps, base, bin string, extra ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", "../../site/install.ps1")
	cmd.Env = append(os.Environ(), "WEFT_BASE="+base, "WEFT_BIN_DIR="+bin, "WEFT_NO_MODIFY_PATH=1")
	cmd.Env = append(cmd.Env, extra...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, strings.ReplaceAll(out.String(), "\r\n", "\n"), strings.ReplaceAll(errb.String(), "\r\n", "\n")
}

func TestInstallPowerShell(t *testing.T) {
	ps := findPowerShell()
	if ps == "" {
		if runtime.GOOS == "windows" {
			t.Fatal("neither pwsh nor powershell is on PATH on Windows")
		}
		t.Skip("neither pwsh nor powershell is on PATH")
	}

	t.Run("latest", func(t *testing.T) {
		dir, bin := t.TempDir(), t.TempDir()
		releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", false)
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		code, out, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=AMD64")
		if code != 0 {
			t.Fatalf("exit %d, stderr %q", code, errs)
		}
		target := filepath.Join(bin, "weft.exe")
		if !strings.Contains(out, "Installed weft 0.0.1 to "+target+"\n") || !strings.Contains(out, "Add "+bin+" to your PATH to run weft.\n") {
			t.Fatalf("stdout %q, want Installed weft 0.0.1 to %s and PATH hint", out, target)
		}
		if b, err := os.ReadFile(target); err != nil || !strings.Contains(string(b), "fake weft") {
			t.Fatalf("weft.exe = %q, %v", b, err)
		}
	})

	t.Run("pinned version replaces existing binary", func(t *testing.T) {
		dir, bin := t.TempDir(), t.TempDir()
		releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", false)
		releaseTreeFor(t, dir, "0.0.2", "windows", "amd64", false)
		target := filepath.Join(bin, "weft.exe")
		os.WriteFile(target, []byte("older weft"), 0o755)
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		code, out, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=AMD64", "WEFT_VERSION=0.0.1")
		if code != 0 {
			t.Fatalf("exit %d, stderr %q", code, errs)
		}
		if !strings.Contains(out, "Installed weft 0.0.1 to "+target+"\n") {
			t.Fatalf("stdout %q", out)
		}
		if b, err := os.ReadFile(target); err != nil || !strings.Contains(string(b), "fake weft") {
			t.Fatalf("weft.exe = %q, %v", b, err)
		}
		if b, err := os.ReadFile(target + ".old"); err != nil || string(b) != "older weft" {
			t.Fatalf("weft.exe.old = %q, %v; want older weft set aside", b, err)
		}
	})

	t.Run("arm64", func(t *testing.T) {
		dir, bin := t.TempDir(), t.TempDir()
		releaseTreeFor(t, dir, "0.0.2", "windows", "arm64", false)
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		code, out, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=ARM64")
		if code != 0 {
			t.Fatalf("exit %d, stderr %q", code, errs)
		}
		target := filepath.Join(bin, "weft.exe")
		if !strings.Contains(out, "Installed weft 0.0.2 to "+target+"\n") {
			t.Fatalf("stdout %q", out)
		}
	})

	t.Run("bad checksum", func(t *testing.T) {
		dir, bin := t.TempDir(), t.TempDir()
		name := releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", true)
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		code, _, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=AMD64")
		if code != 1 || errs != "weft install: checksum mismatch for "+name+"\n" {
			t.Fatalf("exit %d, stderr %q", code, errs)
		}
		if _, err := os.Stat(filepath.Join(bin, "weft.exe")); !os.IsNotExist(err) {
			t.Fatal("weft.exe should not exist")
		}
	})

	t.Run("unsupported platform", func(t *testing.T) {
		dir, bin := t.TempDir(), t.TempDir()
		releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", false)
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		code, _, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=x86")
		if code != 1 || errs != "weft install: no build for windows/x86\n" {
			t.Fatalf("exit %d, stderr %q", code, errs)
		}
	})

	t.Run("keeps old binary on failure", func(t *testing.T) {
		dir, bin := t.TempDir(), t.TempDir()
		releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", true)
		target := filepath.Join(bin, "weft.exe")
		os.WriteFile(target, []byte("old"), 0o755)
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		if code, _, _ := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=AMD64"); code != 1 {
			t.Fatalf("exit %d", code)
		}
		if b, _ := os.ReadFile(target); string(b) != "old" {
			t.Fatalf("old binary replaced: %q", b)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		code, _, errs := runPowerShellInstaller(t, ps, "http://127.0.0.1:1", t.TempDir(), "WEFT_TEST_ARCH=AMD64")
		if code != 1 || !strings.HasPrefix(errs, "weft install: ") || strings.Count(errs, "\n") != 1 {
			t.Fatalf("exit %d, stderr %q", code, errs)
		}
	})
}
