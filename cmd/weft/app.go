package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/flakerimi/weft"
)

// appConfig is what an app's package.json records: its name, and under "weft"
// the Weft version and the theme.
type appConfig struct {
	Name, Version, Theme string
}

// readApp reads the name and the "weft" object of dir/package.json.
func readApp(dir string) (appConfig, error) {
	file := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return appConfig{}, fmt.Errorf("cannot read %s; run the command inside a Weft app", file)
	}
	var pkg struct {
		Name string `json:"name"`
		Weft *struct {
			Version string `json:"version"`
			Theme   string `json:"theme"`
		} `json:"weft"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return appConfig{}, fmt.Errorf("%s is not valid JSON (%v); fix it and run the command again", file, err)
	}
	if pkg.Weft == nil {
		return appConfig{}, fmt.Errorf("%s has no \"weft\" key; run the command inside a Weft app", file)
	}
	if pkg.Weft.Version == "" {
		return appConfig{}, fmt.Errorf("%s has no weft.version; add \"version\": %q, this weft's version, to its \"weft\" object", file, weft.Version)
	}
	return appConfig{Name: pkg.Name, Version: pkg.Weft.Version, Theme: pkg.Weft.Theme}, nil
}

// writeAppVersion sets weft.version in dir/package.json and changes no other
// byte of the file. It writes a temporary file beside it and renames that into
// place, so the file is never half-written.
func writeAppVersion(dir, version string) error {
	file := filepath.Join(dir, "package.json")
	if real, err := filepath.EvalSymlinks(file); err == nil {
		file = real
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("cannot read %s; run the command inside a Weft app", file)
	}
	start, end, err := versionSpan(data)
	if err != nil {
		return fmt.Errorf("%s: %v; add \"weft\": {\"version\": \"...\"} to it", file, err)
	}
	quoted, _ := json.Marshal(version)
	out := append(append(append([]byte{}, data[:start]...), quoted...), data[end:]...)
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".package.json.*")
	if err != nil {
		return fmt.Errorf("cannot write beside %s (%v); make its folder writable", file, err)
	}
	_, err = tmp.Write(out)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), info.Mode().Perm())
	}
	if err == nil {
		err = os.Rename(tmp.Name(), file)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("cannot write %s (%v); make it writable", file, err)
	}
	return nil
}

// versionSpan finds the byte range of the string value of the top-level
// "weft" object's "version" key.
func versionSpan(data []byte) (int, int, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return 0, 0, errors.New("it is not a JSON object")
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return 0, 0, err
		}
		if key != "weft" {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return 0, 0, err
			}
			continue
		}
		if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
			return 0, 0, errors.New("\"weft\" is not an object")
		}
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				return 0, 0, err
			}
			before := int(dec.InputOffset())
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return 0, 0, err
			}
			if k != "version" {
				continue
			}
			if len(raw) == 0 || raw[0] != '"' {
				return 0, 0, errors.New("\"weft.version\" is not a string")
			}
			end := int(dec.InputOffset())
			start := end - len(raw)
			if start < before {
				return 0, 0, errors.New("cannot locate \"weft.version\"")
			}
			return start, end, nil
		}
		return 0, 0, errors.New("\"weft\" has no \"version\"")
	}
	return 0, 0, errors.New("it has no \"weft\" key")
}

// findApp returns the nearest folder at or above start whose package.json has
// a "weft" object, or whose package.json does not parse: that one may be the
// app with a typo in it, so readApp's sentence, which names the file and says
// to fix it, is what the command prints. A package.json that parses and has
// no "weft" object is passed over, as a folder above may be the app.
func findApp(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
			if !json.Valid(data) {
				return dir, nil
			}
			var pkg struct {
				Weft json.RawMessage `json:"weft"`
			}
			if json.Unmarshal(data, &pkg) == nil && bytes.HasPrefix(bytes.TrimSpace(pkg.Weft), []byte("{")) {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no package.json with a \"weft\" key in %s or above it; run the command inside a Weft app, or make one with weft create", start)
		}
		dir = parent
	}
}

var versionRe = regexp.MustCompile(`\d+(\.\d+)?`)

// toolVersion runs `name --version` and returns its major and minor numbers.
func toolVersion(name string) (major, minor int, err error) {
	out, err := exec.Command(name, "--version").Output()
	if err != nil {
		return 0, 0, err
	}
	m := versionRe.FindString(string(out))
	if m == "" {
		return 0, 0, fmt.Errorf("no version in %q", strings.TrimSpace(string(out)))
	}
	parts := strings.SplitN(m, ".", 2)
	major, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor, nil
}

// prerequisites checks that go, node 22.18 or later and pnpm 10 or later are
// on PATH.
func prerequisites() error {
	if _, err := exec.LookPath("go"); err != nil {
		return errors.New("go is not on PATH; install Go 1.26 or later from https://go.dev/dl and run the command again")
	}
	if _, err := exec.LookPath("node"); err != nil {
		return errors.New("node is not on PATH; install Node 22.18 or later and run the command again")
	}
	major, minor, err := toolVersion("node")
	if err != nil || major < 22 || (major == 22 && minor < 18) {
		return errors.New("node must be 22.18 or later; install a newer Node and run the command again")
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		return errors.New("pnpm is not on PATH; install pnpm 10 or later and run the command again")
	}
	if major, _, err := toolVersion("pnpm"); err != nil || major < 10 {
		return errors.New("pnpm must be 10 or later; install a newer pnpm and run the command again")
	}
	return nil
}

// checkedApp reads the app and refuses one recorded at another Weft version,
// pointing an older one at weft upgrade.
func checkedApp(appDir string) (appConfig, error) {
	cfg, err := readApp(appDir)
	if err != nil {
		return cfg, err
	}
	if cfg.Version != weft.Version && newerVersion(cfg.Version, weft.Version) {
		// weft upgrade never moves an app backwards, so it is not offered.
		return cfg, fmt.Errorf("this app is on Weft %s and this weft is %s; run weft update --version %s to match it", cfg.Version, weft.Version, cfg.Version)
	}
	if cfg.Version != weft.Version {
		return cfg, fmt.Errorf("this app is on Weft %s and this weft is %s; run weft upgrade to move the app, or weft update --version %s to match it", cfg.Version, weft.Version, cfg.Version)
	}
	return cfg, nil
}

// unpackCmd makes sure .weft/ is there and current, as ensure does before it
// installs, so that go and an editor can resolve the Weft Go module in a fresh
// clone before any weft dev or weft build has run pnpm install.
func unpackCmd(args []string, stdout, stderr io.Writer) int {
	_, code, ok := appCommandWith("unpack", args, stdout, stderr, func(appDir string) error {
		cfg, err := checkedApp(appDir)
		if err != nil {
			return err
		}
		did, err := refreshWeft(appDir, cfg.Version)
		switch {
		case err != nil:
			return err
		case did:
			fmt.Fprintf(stdout, "Unpacked .weft (Weft %s)\n", cfg.Version)
		default:
			fmt.Fprintf(stdout, ".weft is current (Weft %s)\n", cfg.Version)
		}
		return nil
	})
	if !ok {
		return code
	}
	return 0
}

// ensure is the gate every command that runs an app passes. It refuses an app
// recorded at another Weft version (pointing an older one at weft upgrade),
// checks the prerequisites, unpacks the bundle into .weft/ when that holds
// another version or nothing (a fresh clone), or, for a development weft,
// another bundle (refreshWeft), runs pnpm install when it
// unpacked or node_modules/ is missing, and writes the components'
// declaration file when it installed or the file is missing.
// Its errors carry no "weft <command>: " prefix; the command adds it.
func ensure(appDir string, stdout, stderr io.Writer) (appConfig, error) {
	cfg, err := checkedApp(appDir)
	if err != nil {
		return cfg, err
	}
	if err := prerequisites(); err != nil {
		return cfg, err
	}
	install, err := refreshWeft(appDir, cfg.Version)
	if err != nil {
		return cfg, err
	}
	if info, err := os.Stat(filepath.Join(appDir, "node_modules")); err != nil || !info.IsDir() {
		install = true
	}
	if install {
		if err := pnpmInstall(appDir, stdout, stderr); err != nil {
			return cfg, err
		}
	}
	if _, err := os.Stat(filepath.Join(appDir, componentTypes)); install || err != nil {
		if err := writeTypes(appDir, stdout, stderr); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// componentTypes is the declaration file that tells TypeScript what each
// auto-imported Weft component is; the app's web/tsconfig.json includes it.
var componentTypes = filepath.Join("web", ".weft-types", "components.d.ts")

// writeTypes writes the app's componentTypes with @weft/vite's weft-types, as
// a build or dev run would, so pnpm typecheck checks the components' props
// before either has run.
func writeTypes(appDir string, stdout, stderr io.Writer) error {
	if err := runIn(appDir, stdout, stderr, "pnpm", "exec", "weft-types", "--root", "web"); err != nil {
		return fmt.Errorf("pnpm exec weft-types did not write %s; fix what its output above says and run the command again", filepath.Join(appDir, componentTypes))
	}
	return nil
}

// pnpmInstall runs pnpm install in appDir, passing its output through.
func pnpmInstall(appDir string, stdout, stderr io.Writer) error {
	if err := runIn(appDir, stdout, stderr, "pnpm", "install"); err != nil {
		return fmt.Errorf("pnpm install did not finish in %s; fix what its output above says and run the command again", appDir)
	}
	return nil
}
