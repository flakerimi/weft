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
