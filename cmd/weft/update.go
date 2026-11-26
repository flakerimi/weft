package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/flakerimi/weft"
)

const defaultBase = "https://weft.base.al"

// executablePath finds the binary to replace; tests point it elsewhere.
var executablePath = os.Executable

var httpClient = &http.Client{Timeout: 2 * time.Minute}

type updateError string

func (e updateError) Error() string { return string(e) }

func updateCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "only report whether a newer version exists")
	pinned := fs.String("version", "", "install this version instead of the latest")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := update(stdout, *check, *pinned); err != nil {
		fmt.Fprintf(stderr, "weft update: %s\n", err)
		return 1
	}
	return 0
}

func update(stdout io.Writer, check bool, pinned string) error {
	current := weft.Version
	if strings.HasSuffix(current, "-dev") {
		return updateError("this is a development build; install a release with: curl -fsSL " + defaultBase + "/install | sh")
	}
	base := strings.TrimRight(os.Getenv("WEFT_BASE"), "/")
	if base == "" {
		base = defaultBase
	}

	target := pinned
	if target == "" {
		latest, err := fetch(base + "/releases/latest")
		if err != nil {
			return updateError(fmt.Sprintf("could not reach %s to find the latest version.", base))
		}
		target = strings.TrimSpace(string(latest))
		if target == "" {
			return updateError(fmt.Sprintf("could not reach %s to find the latest version.", base))
		}
	}

	if target == current {
		fmt.Fprintf(stdout, "weft %s is up to date.\n", current)
		return nil
	}
	if check {
		fmt.Fprintf(stdout, "weft %s is available (you have %s). Run \"weft update\" to install it.\n", target, current)
		return nil
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH
	name := archiveName(runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("%s/releases/%s", base, target)
	archive, err := fetch(url + "/" + name)
	if err != nil {
		return updateError(fmt.Sprintf("there is no weft %s for %s at %s.", target, platform, base))
	}
	sums, err := fetch(url + "/SHA256SUMS")
	if err != nil {
		return updateError(fmt.Sprintf("could not download %s/SHA256SUMS.", url))
	}
	if !checksumMatches(sums, name, archive) {
		return updateError(fmt.Sprintf("checksum mismatch for %s.", name))
	}
	binary, err := extract(name, archive)
	if err != nil {
		return updateError(fmt.Sprintf("could not unpack %s: %v.", name, err))
	}
	if err := replaceExecutable(binary); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Updated weft %s → %s.\n", current, target)
	return nil
}

func fetch(url string) ([]byte, error) {
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, res.Status)
	}
	return io.ReadAll(res.Body)
}

func checksumMatches(sums []byte, name string, data []byte) bool {
	got := sha256.Sum256(data)
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return strings.EqualFold(fields[0], hex.EncodeToString(got[:]))
		}
	}
	return false
}

// archiveName is the name of the release archive for goos and goarch, as
// scripts/release.sh writes it: weft_<os>_<arch>.tar.gz, or .zip for Windows.
func archiveName(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("weft_%s_%s.zip", goos, goarch)
	}
	return fmt.Sprintf("weft_%s_%s.tar.gz", goos, goarch)
}

// extract returns the binary in the archive called name: weft.exe from a
// .zip, weft from a .tar.gz.
func extract(name string, archive []byte) ([]byte, error) {
	if !strings.HasSuffix(name, ".zip") {
		return extractBinary(archive)
	}
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if path.Base(f.Name) == "weft.exe" && f.Mode().IsRegular() {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, errors.New("no weft.exe inside")
}

func extractBinary(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("no weft binary inside")
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == "weft" && h.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
}

// replaceExecutable writes the new binary beside the running one and renames it
// into place, so the old binary stays whole until the new one is complete.
func replaceExecutable(binary []byte) error {
	exe, err := executablePath()
	if err != nil {
		return updateError(fmt.Sprintf("could not find the running weft: %v.", err))
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".weft.new-")
	if err != nil {
		return updateError(fmt.Sprintf("could not write to %s; run the installer instead: curl -fsSL %s/install | sh", dir, defaultBase))
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpName) // our own unfinished temp file
		}
	}()
	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		return updateError(fmt.Sprintf("could not write to %s: %v.", dir, err))
	}
	if err := tmp.Close(); err != nil {
		return updateError(fmt.Sprintf("could not write to %s: %v.", dir, err))
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return updateError(fmt.Sprintf("could not make the new weft executable: %v.", err))
	}
	if err := swapIn(tmpName, exe, runtime.GOOS == "windows"); err != nil {
		return updateError(fmt.Sprintf("could not replace %s: %v.", exe, err))
	}
	ok = true
	return nil
}

// swapIn renames tmp to exe. With aside, as on Windows, where a running
// binary may be renamed but not written over, exe is first renamed to
// exe.old, replacing the one an earlier update left, and is put back if tmp
// cannot take its place.
func swapIn(tmp, exe string, aside bool) error {
	if !aside {
		return os.Rename(tmp, exe)
	}
	old := exe + ".old"
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	return nil
}
