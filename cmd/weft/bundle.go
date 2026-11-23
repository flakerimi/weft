package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// bundleFS holds what scripts/bundle.sh staged into cmd/weft/bundle at release
// time; in git it holds only README.md.
//
//go:embed all:bundle
var bundleFS embed.FS

const errNoBundle = "this weft was built without its packages; install a release from https://weft.base.al/install, or set WEFT_BUNDLE to a folder made by scripts/bundle.sh"

// openBundle returns the bundle: the folder $WEFT_BUNDLE names, else the
// embedded one when it has its packages.
func openBundle() (fs.FS, error) {
	if dir := os.Getenv("WEFT_BUNDLE"); dir != "" {
		return os.DirFS(dir), nil
	}
	sub, err := fs.Sub(bundleFS, "bundle")
	if err != nil {
		return nil, errors.New(errNoBundle)
	}
	if info, err := fs.Stat(sub, "weft/packages"); err != nil || !info.IsDir() {
		return nil, errors.New(errNoBundle)
	}
	return sub, nil
}

// unpack replaces appDir/.weft with the bundle's weft/ folder and records
// version in .weft/VERSION. It writes .weft.new first and renames it, and it
// leaves a .weft that has no VERSION file alone, since this command did not
// write it.
func unpack(bundle fs.FS, appDir, version string) error {
	src, err := fs.Sub(bundle, "weft")
	if err != nil {
		return err
	}
	dst := filepath.Join(appDir, ".weft")
	if _, err := os.Lstat(dst); err == nil {
		if _, err := os.Stat(filepath.Join(dst, "VERSION")); err != nil {
			return fmt.Errorf("%s has no VERSION file, so weft did not write it; move it away and run the command again", dst)
		}
	}
	tmp := dst + ".new"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := copyFS(src, tmp); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	old := dst + ".old"
	if _, err := os.Lstat(dst); err == nil {
		if err := os.RemoveAll(old); err != nil {
			return err
		}
		if err := os.Rename(dst, old); err != nil {
			os.RemoveAll(tmp)
			return err
		}
	}
	if err := os.Rename(tmp, dst); err != nil {
		if _, statErr := os.Lstat(old); statErr == nil {
			os.Rename(old, dst)
		}
		os.RemoveAll(tmp)
		return err
	}
	return os.RemoveAll(old)
}

// unpacked returns the version recorded in appDir/.weft/VERSION, or "".
func unpacked(appDir string) string {
	b, err := os.ReadFile(filepath.Join(appDir, ".weft", "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// copyFS writes every file of src under dir, keeping executable bits.
func copyFS(src fs.FS, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path.Clean(p)))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if info, err := d.Info(); err == nil && info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
}
