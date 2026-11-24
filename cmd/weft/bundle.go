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
// version in .weft/VERSION. It leaves a .weft that has no VERSION file alone,
// since this command did not write it. It writes the new tree into a folder of
// its own (.weft-new-*), moves the old .weft into another (.weft-old-*), and
// renames the new tree into place; it removes only those two folders, on
// success and on failure, and puts the old .weft back when the last rename
// fails. Nothing else beside .weft is touched, whatever its name.
func unpack(bundle fs.FS, appDir, version string) (err error) {
	src, err := fs.Sub(bundle, "weft")
	if err != nil {
		return err
	}
	dst := filepath.Join(appDir, ".weft")
	_, statErr := os.Lstat(dst)
	had := statErr == nil
	if had {
		if _, err := os.Stat(filepath.Join(dst, "VERSION")); err != nil {
			return fmt.Errorf("%s has no VERSION file, so weft did not write it; move it out of the app folder and run the command again", dst)
		}
	}
	tmp, err := os.MkdirTemp(appDir, ".weft-new-*")
	if err != nil {
		return err
	}
	// Until it is renamed into place, the new tree is this run's to remove.
	defer func() {
		if err != nil {
			os.RemoveAll(tmp)
		}
	}()
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	if err := copyFS(src, tmp); err != nil {
		return err
	}
	// go:embed leaves out a folder that holds a go.mod, so the bundle keeps
	// the Go package's as go.mod.bundle.
	goMod := filepath.Join(tmp, "go", "go.mod")
	if err := os.Rename(goMod+".bundle", goMod); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
		return err
	}
	if !had {
		return renameDir(tmp, dst)
	}
	// The old tree goes inside a folder this run made, so no name beside
	// .weft is taken from anyone.
	hold, err := os.MkdirTemp(appDir, ".weft-old-*")
	if err != nil {
		return err
	}
	old := filepath.Join(hold, ".weft")
	if err := renameDir(dst, old); err != nil {
		os.Remove(hold)
		return err
	}
	if err := renameDir(tmp, dst); err != nil {
		if backErr := renameDir(old, dst); backErr != nil {
			return fmt.Errorf("cannot rename %s to %s (%v), nor put the old one back from %s (%v); move %s to %s by hand", tmp, dst, err, old, backErr, old, dst)
		}
		os.Remove(hold)
		return err
	}
	return os.RemoveAll(hold)
}

// renameDir renames a folder; a test makes it fail.
var renameDir = os.Rename

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
