package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// devVersion is the version of a weft built from the repository. It names no
// release, so .weft/VERSION also records a hash of the bundle it unpacked.
const devVersion = "0.0.0-dev"

// bundleHash is the SHA-256, in hex, over the names and bytes of every file in
// the bundle's weft/ folder, in the order fs.WalkDir visits them.
func bundleHash(bundle fs.FS) (string, error) {
	src, err := fs.Sub(bundle, "weft")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	err = fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(p)))
		h.Write(n[:])
		io.WriteString(h, p)
		binary.BigEndian.PutUint64(n[:], uint64(len(data)))
		h.Write(n[:])
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashBundle is bundleHash; a test counts its calls.
var hashBundle = bundleHash

// refreshWeft unpacks the bundle into appDir/.weft when .weft/VERSION holds
// another version or there is none, and says whether it did. For devVersion
// it also unpacks when the hash VERSION records is not the bundle's, so a
// change to the framework reaches the app; a release is the version it says.
// A development weft with no bundle of its own leaves a .weft that is there.
//
// The bundle is hashed once. Another weft command unpacking in the same app
// is waited for, and what it unpacked is not unpacked again.
func refreshWeft(appDir, version string) (bool, error) {
	have, haveHash := unpackedRecord(appDir)
	need := have != version
	if !need && version != devVersion {
		return false, nil
	}
	bundle, err := openBundle()
	if err != nil {
		if need {
			return false, err
		}
		return false, nil
	}
	hash := ""
	if version == devVersion {
		if hash, err = hashBundle(bundle); err != nil {
			return false, unreadableBundle(err)
		}
		if !need && hash == haveHash {
			return false, nil
		}
	}
	unlock, err := lockWeft(appDir)
	if err != nil {
		return false, err
	}
	defer unlock()
	if have, haveHash := unpackedRecord(appDir); have == version && (version != devVersion || haveHash == hash) {
		return false, nil // another weft command unpacked it while this one waited
	}
	return true, unpackHashed(bundle, appDir, version, hash)
}

// unreadableBundle is the sentence for a bundle a file of which cannot be
// read, naming the folder WEFT_BUNDLE names, or the bundle weft carries.
func unreadableBundle(err error) error {
	if dir := os.Getenv("WEFT_BUNDLE"); dir != "" {
		return fmt.Errorf("cannot read the bundle in %s (%v); make its files readable, or point WEFT_BUNDLE at a folder made by scripts/bundle.sh", dir, err)
	}
	return fmt.Errorf("cannot read the bundle this weft carries (%v); install weft again from https://weft.base.al/install", err)
}

// lockSays is where lockWeft says that it waits, under which command's name;
// run points it at the command's standard error.
var lockSays = struct {
	command string
	to      io.Writer
}{"", io.Discard}

// lockWait is how long a weft command waits for another one unpacking in
// the same app, and how old a lock may be before it is taken as left behind:
// longer than any unpack takes.
var lockWait = 2 * time.Minute

// lockWeft makes appDir/.weft.lock, holding this process's id, and returns
// the function that removes it. A lock left by a weft command that did not
// finish is taken over at once: one naming a process that is gone, one with
// no process a second after it was made, or one older than lockWait. While
// a running command holds it, lockWeft waits, up to lockWait, and says so
// once.
func lockWeft(appDir string) (unlock func(), err error) {
	lock := filepath.Join(appDir, ".weft.lock")
	began := time.Now()
	said := false
	for {
		f, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(lock) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("cannot make %s (%v); make the app folder writable", lock, err)
		}
		if held, ok := leftBehind(lock); ok {
			// Removed only if it still holds what was read, so a lock another
			// command has just made in its place stays.
			if now, err := os.ReadFile(lock); err == nil && bytes.Equal(now, held) {
				os.Remove(lock)
			}
			continue
		}
		if time.Since(began) > lockWait {
			return nil, fmt.Errorf("another weft command has been unpacking .weft in %s for %s (it holds %s); let it finish, and run the command again", appDir, lockWait, lock)
		}
		if !said {
			fmt.Fprintf(lockSays.to, "weft %s: another weft command is unpacking in this app; waiting for it.\n", lockSays.command)
			said = true
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// leftBehind reads the lock and says whether the command that made it is
// gone: the process it names is not running, it names none a second after it
// was made (its maker stopped before writing its id), or it is older than
// lockWait.
func leftBehind(lock string) (held []byte, gone bool) {
	fi, err := os.Stat(lock)
	if err != nil {
		return nil, false // removed meanwhile: try again
	}
	held, err = os.ReadFile(lock)
	if err != nil {
		return nil, false
	}
	age := time.Since(fi.ModTime())
	if age > lockWait {
		return held, true
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(held)))
	if err != nil || pid <= 0 {
		return held, age > time.Second
	}
	return held, !running(pid)
}

// unpack replaces appDir/.weft with the bundle's weft/ folder and records
// version in .weft/VERSION, and under devVersion the bundle's hash on a second
// line. It leaves a .weft that has no VERSION file alone,
// since this command did not write it. It writes the new tree into a folder of
// its own (.weft-new-*), moves the old .weft into another (.weft-old-*), and
// renames the new tree into place; it removes only those two folders, on
// success and on failure, and puts the old .weft back when the last rename
// fails. Beside .weft it makes only those and .weft.lock, which it holds
// while it works and then removes; nothing else is touched, whatever its
// name.
//
// Another weft command unpacking in the same app is waited for.
func unpack(bundle fs.FS, appDir, version string) error {
	unlock, err := lockWeft(appDir)
	if err != nil {
		return err
	}
	defer unlock()
	return unpackHashed(bundle, appDir, version, "")
}

// unpackHashed is unpack, under the app's lock, with the bundle's hash when
// the caller has it already ("" to take it here, under devVersion).
func unpackHashed(bundle fs.FS, appDir, version, hash string) (err error) {
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
		var pe *fs.PathError
		if errors.As(err, &pe) && !strings.HasPrefix(pe.Path, tmp) {
			return unreadableBundle(err) // the bundle's file, named within it
		}
		return err
	}
	// go:embed leaves out a folder that holds a go.mod, so the bundle keeps
	// the Go package's as go.mod.bundle.
	goMod := filepath.Join(tmp, "go", "go.mod")
	if err := os.Rename(goMod+".bundle", goMod); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	record := version + "\n"
	if version == devVersion {
		if hash == "" {
			if hash, err = hashBundle(bundle); err != nil {
				return unreadableBundle(err)
			}
		}
		record += hash + "\n"
	}
	if err := os.WriteFile(filepath.Join(tmp, "VERSION"), []byte(record), 0o644); err != nil {
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
	version, _ := unpackedRecord(appDir)
	return version
}

// unpackedRecord returns the version, the first line of appDir/.weft/VERSION,
// and the bundle hash, its second line, or "" for each that is not there.
func unpackedRecord(appDir string) (version, hash string) {
	b, err := os.ReadFile(filepath.Join(appDir, ".weft", "VERSION"))
	if err != nil {
		return "", ""
	}
	version, rest, _ := strings.Cut(strings.TrimSpace(string(b)), "\n")
	return strings.TrimSpace(version), strings.TrimSpace(rest)
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
