package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// appConfig is what an app's package.json records under "weft".
type appConfig struct {
	Version, Theme string
}

// readApp reads the "weft" object of dir/package.json.
func readApp(dir string) (appConfig, error) {
	file := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return appConfig{}, fmt.Errorf("cannot read %s; run the command inside a Weft app", file)
	}
	var pkg struct {
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
	return appConfig{Version: pkg.Weft.Version, Theme: pkg.Weft.Theme}, nil
}

// writeAppVersion sets weft.version in dir/package.json and changes no other
// byte of the file.
func writeAppVersion(dir, version string) error {
	file := filepath.Join(dir, "package.json")
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
	return os.WriteFile(file, out, info.Mode().Perm())
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
// a "weft" key.
func findApp(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
			var pkg struct {
				Weft json.RawMessage `json:"weft"`
			}
			if json.Unmarshal(data, &pkg) == nil && len(pkg.Weft) > 0 {
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
