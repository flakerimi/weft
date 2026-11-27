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
	if code != 1 || errs != "weft install: checksum mismatch for "+name+"; run the installer again\n" {
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
	if code != 1 || errs != "weft install: no build for plan9/amd64; Weft supports darwin, linux and windows on amd64 and arm64\n" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	for _, u := range []string{"MINGW64_NT-10.0-22631", "MSYS_NT-10.0-22631", "CYGWIN_NT-10.0-22631"} {
		code, _, errs := runInstaller(t, srv.URL, bin, "WEFT_TEST_UNAME_S="+u, "WEFT_TEST_UNAME_M=x86_64")
		want := "weft install: this is Windows; in PowerShell, run irm " + srv.URL + "/install.ps1 | iex\n"
		if code != 1 || errs != want {
			t.Fatalf("%s: exit %d, stderr %q, want %q", u, code, errs, want)
		}
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
	if code != 1 || errs != "weft install: could not reach http://127.0.0.1:1 to find the latest version; check your connection, or set WEFT_VERSION\n" {
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

// The PowerShell installer, site/install.ps1, is pure ASCII with LF line
// endings and no UTF-8 BOM (Windows PowerShell 5.1 reads -File without a BOM
// as ANSI and irm without a charset as ISO-8859-1). It wraps its body in
// try { & { ... } } catch { ...; if ($PSCommandPath) { exit 1 } } so that
// irm ... | iex never closes the user's PowerShell window or leaks variables
// and functions into the session. It maps the Windows architecture (OSArchitecture
// first, then PROCESSOR_ARCHITEW6432 and PROCESSOR_ARCHITECTURE), verifies
// SHA256SUMS, swaps weft.exe in via .weft.new.exe and weft.exe.old, and writes
// the user's Path in HKCU:\Environment as ExpandString without expanding existing
// %VAR% entries.
func TestInstallPowerShellScript(t *testing.T) {
	b, err := os.ReadFile("../../site/install.ps1")
	if err != nil {
		t.Fatalf("site/install.ps1: %v", err)
	}
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("site/install.ps1 has a UTF-8 BOM; keep it plain ASCII")
	}
	for i, c := range b {
		if c > 0x7F {
			t.Fatalf("site/install.ps1 has non-ASCII byte 0x%02x at offset %d; keep it plain ASCII", c, i)
		}
	}
	s := string(b)
	if strings.Contains(s, "\r") {
		t.Error("site/install.ps1 has CR line endings; keep LF in git")
	}
	for _, want := range []string{
		"irm https://weft.base.al/install.ps1 | iex",
		"try {\n\t& {",
		"if ($PSCommandPath) { exit 1 }",
		`throw "weft install: $msg"`,
		`-replace '^(?!weft install: )', 'weft install: '`,
		"SecurityProtocolType]::Tls12",
		"GetUnresolvedProviderPathFromPSPath",
		"Is64BitProcess",
		"OSArchitecture",
		"WEFT_BASE",
		"WEFT_VERSION",
		"WEFT_BIN_DIR",
		"WEFT_TEST_ARCH",
		"WEFT_NO_MODIFY_PATH",
		"WEFT_TEST_PATH_NAME",
		"function Get-WeftArch",
		"function Get-WeftChecksum",
		"no build for windows/$_; Weft supports darwin, linux and windows on amd64 and arm64",
		"checksum mismatch for $name; run the installer again",
		`.local\bin`,
		"HKCU:\\Environment",
		"DoNotExpandEnvironmentNames",
		"-Type ExpandString",
		"Installed weft ",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("site/install.ps1 does not contain %q", want)
		}
	}
	if strings.Contains(s, "\n\t$msg =") {
		t.Error("site/install.ps1 sets $msg in the outer catch, which leaks into the user's session under iex")
	}
	if strings.Contains(s, "'Machine'") || strings.Contains(s, `"Machine"`) || strings.Contains(s, "HKLM:") {
		t.Error("site/install.ps1 touches the Machine Path; it must only touch HKCU:\\Environment")
	}
	// No exit outside the $PSCommandPath guard, or irm | iex would close the user's window.
	for i, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, "exit") && trimmed != "if ($PSCommandPath) { exit 1 }" {
			t.Errorf("site/install.ps1:%d has an unguarded exit (%q); only if ($PSCommandPath) { exit 1 } may call exit", i+1, trimmed)
		}
	}
}

type psShell struct {
	name string
	path string
}

func findPowerShells() []psShell {
	var out []psShell
	for _, name := range []string{"powershell", "pwsh"} {
		if p, err := exec.LookPath(name); err == nil {
			out = append(out, psShell{name: name, path: p})
		}
	}
	return out
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

func runPowerShellCommand(t *testing.T, ps, script string, extra ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.Env = append(os.Environ(), extra...)
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
	shells := findPowerShells()
	if runtime.GOOS == "windows" {
		has51 := false
		for _, sh := range shells {
			if sh.name == "powershell" {
				has51 = true
			}
		}
		if !has51 {
			t.Fatal("powershell (Windows PowerShell 5.1) is not on PATH on Windows")
		}
	}
	if len(shells) == 0 {
		t.Skip("neither powershell nor pwsh is on PATH")
	}

	for _, sh := range shells {
		t.Run(sh.name, func(t *testing.T) {
			ps := sh.path

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

			t.Run("pinned version replaces existing binary and cleans up old", func(t *testing.T) {
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
				if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
					t.Fatalf("weft.exe.old should be removed when not in use, got err %v", err)
				}
			})

			t.Run("relative WEFT_BIN_DIR resolved to full path", func(t *testing.T) {
				dir, work := t.TempDir(), t.TempDir()
				releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", false)
				srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
				defer srv.Close()
				scriptPath, err := filepath.Abs("../../site/install.ps1")
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
				cmd.Dir = work
				cmd.Env = append(os.Environ(), "WEFT_BASE="+srv.URL, "WEFT_BIN_DIR=bin", "WEFT_NO_MODIFY_PATH=1", "WEFT_TEST_ARCH=AMD64")
				var outb, errb bytes.Buffer
				cmd.Stdout, cmd.Stderr = &outb, &errb
				if err := cmd.Run(); err != nil {
					t.Fatalf("err %v, stderr %q", err, errb.String())
				}
				out := strings.ReplaceAll(outb.String(), "\r\n", "\n")
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) < 2 {
					t.Fatalf("stdout %q, want 2 lines", out)
				}
				const prefix = "Installed weft 0.0.1 to "
				if !strings.HasPrefix(lines[0], prefix) {
					t.Fatalf("line 0 = %q, want prefix %q", lines[0], prefix)
				}
				printedExe := strings.TrimPrefix(lines[0], prefix)
				if !filepath.IsAbs(printedExe) {
					t.Fatalf("printed target %q is not an absolute path", printedExe)
				}
				wantExe := filepath.Join(work, "bin", "weft.exe")
				fi1, err1 := os.Stat(printedExe)
				fi2, err2 := os.Stat(wantExe)
				if err1 != nil || err2 != nil || !os.SameFile(fi1, fi2) {
					t.Fatalf("printed %q (%v) and %q (%v) are not the same file", printedExe, err1, wantExe, err2)
				}
				printedDir := filepath.Dir(printedExe)
				if lines[1] != "Add "+printedDir+" to your PATH to run weft." {
					t.Fatalf("line 1 = %q, want %q", lines[1], "Add "+printedDir+" to your PATH to run weft.")
				}
			})

			t.Run("arm64 and 32-bit WOW64 on Arm64", func(t *testing.T) {
				dir, bin := t.TempDir(), t.TempDir()
				releaseTreeFor(t, dir, "0.0.2", "windows", "arm64", false)
				srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
				defer srv.Close()
				for _, raw := range []string{"ARM64", "aarch64", "arm64"} {
					code, out, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH="+raw)
					if code != 0 {
						t.Fatalf("%s: exit %d, stderr %q", raw, code, errs)
					}
					target := filepath.Join(bin, "weft.exe")
					if !strings.Contains(out, "Installed weft 0.0.2 to "+target+"\n") {
						t.Fatalf("%s: stdout %q", raw, out)
					}
				}
			})

			t.Run("bad checksum", func(t *testing.T) {
				dir, bin := t.TempDir(), t.TempDir()
				name := releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", true)
				srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
				defer srv.Close()
				code, _, errs := runPowerShellInstaller(t, ps, srv.URL, bin, "WEFT_TEST_ARCH=AMD64")
				if code != 1 || errs != "weft install: checksum mismatch for "+name+"; run the installer again\n" {
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
				if code != 1 || errs != "weft install: no build for windows/x86; Weft supports darwin, linux and windows on amd64 and arm64\n" {
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
				if code != 1 || errs != "weft install: could not reach http://127.0.0.1:1 to find the latest version; check your connection, or set WEFT_VERSION\n" {
					t.Fatalf("exit %d, stderr %q", code, errs)
				}
			})

			t.Run("iex failure does not close session or leak scope", func(t *testing.T) {
				script := "Get-Content -Raw ../../site/install.ps1 | Invoke-Expression; " +
					"[Console]::Out.WriteLine(\"after eap=$ErrorActionPreference pp=$ProgressPreference msg=$msg base=$base fn=$(Test-Path Function:\\Get-WeftArch)\")"
				code, out, errs := runPowerShellCommand(t, ps, script,
					"WEFT_TEST_ARCH=x86", "WEFT_NO_MODIFY_PATH=1")
				if code != 0 {
					t.Fatalf("iex exited %d, stderr %q, stdout %q", code, errs, out)
				}
				if errs != "weft install: no build for windows/x86; Weft supports darwin, linux and windows on amd64 and arm64\n" {
					t.Fatalf("stderr %q", errs)
				}
				if out != "after eap=Continue pp=Continue msg= base= fn=False\n" {
					t.Fatalf("stdout %q, want session alive with clean scope", out)
				}
			})

			t.Run("iex success does not leak scope", func(t *testing.T) {
				dir, bin := t.TempDir(), t.TempDir()
				releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", false)
				srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
				defer srv.Close()
				script := "Get-Content -Raw ../../site/install.ps1 | Invoke-Expression; " +
					"[Console]::Out.WriteLine(\"after eap=$ErrorActionPreference pp=$ProgressPreference msg=$msg base=$base target=$target fn=$(Test-Path Function:\\Get-WeftChecksum)\")"
				code, out, errs := runPowerShellCommand(t, ps, script,
					"WEFT_BASE="+srv.URL, "WEFT_BIN_DIR="+bin, "WEFT_TEST_ARCH=AMD64", "WEFT_NO_MODIFY_PATH=1")
				if code != 0 || errs != "" {
					t.Fatalf("exit %d, stderr %q, stdout %q", code, errs, out)
				}
				if !strings.HasSuffix(out, "after eap=Continue pp=Continue msg= base= target= fn=False\n") {
					t.Fatalf("stdout %q, want clean scope after iex install", out)
				}
			})

			t.Run("registry PATH preserves unexpanded %USERPROFILE% as ExpandString", func(t *testing.T) {
				if runtime.GOOS != "windows" {
					t.Skip("HKCU:\\Environment registry test is Windows-only")
				}
				dir, bin := t.TempDir(), t.TempDir()
				releaseTreeFor(t, dir, "0.0.1", "windows", "amd64", false)
				srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
				defer srv.Close()
				varName := fmt.Sprintf("WEFT_TEST_PATH_%s_%d", sh.name, os.Getpid())
				setup := fmt.Sprintf(
					"Remove-ItemProperty -LiteralPath 'HKCU:\\Environment' -Name '%s' -ErrorAction SilentlyContinue; "+
						"Set-ItemProperty -LiteralPath 'HKCU:\\Environment' -Name '%s' -Value '%%USERPROFILE%%\\AppData\\Local\\Microsoft\\WindowsApps' -Type ExpandString",
					varName, varName,
				)
				t.Cleanup(func() {
					clean := fmt.Sprintf("Remove-ItemProperty -LiteralPath 'HKCU:\\Environment' -Name '%s' -ErrorAction SilentlyContinue", varName)
					runPowerShellCommand(t, ps, clean)
				})
				if code, _, errs := runPowerShellCommand(t, ps, setup); code != 0 {
					t.Fatalf("registry setup: exit %d, stderr %q", code, errs)
				}
				script := "Get-Content -Raw ../../site/install.ps1 | Invoke-Expression; " +
					fmt.Sprintf("$k = Get-Item -LiteralPath 'HKCU:\\Environment'; "+
						"[Console]::Out.WriteLine('kind=' + $k.GetValueKind('%s')); "+
						"[Console]::Out.WriteLine('raw=' + $k.GetValue('%s', '', 'DoNotExpandEnvironmentNames'))",
						varName, varName)
				code, out, errs := runPowerShellCommand(t, ps, script,
					"WEFT_BASE="+srv.URL, "WEFT_BIN_DIR="+bin, "WEFT_TEST_ARCH=AMD64",
					"WEFT_NO_MODIFY_PATH=0", "WEFT_TEST_PATH_NAME="+varName)
				if code != 0 || errs != "" {
					t.Fatalf("exit %d, stderr %q, stdout %q", code, errs, out)
				}
				wantAdded := "Added " + bin + " to your PATH (open a new terminal elsewhere to use it).\n"
				wantKind := "kind=ExpandString\n"
				wantRaw := "raw=%USERPROFILE%\\AppData\\Local\\Microsoft\\WindowsApps;" + bin + "\n"
				for _, want := range []string{wantAdded, wantKind, wantRaw} {
					if !strings.Contains(out, want) {
						t.Errorf("output %q lacks %q", out, want)
					}
				}
			})
		})
	}
}
