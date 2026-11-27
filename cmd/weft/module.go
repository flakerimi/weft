package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// An app gets Warp, the Go of github.com/flakerimi/weft, in one of two ways.
// An app a release made requires the module at the release's tag, and Go
// fetches it from GitHub through the module proxy, so go.sum holds its
// checksums. An app a development build made replaces the module with
// ./.weft/go, which that build's bundle fills, so working on Weft itself
// needs no tag and no network.

// localModule says whether a weft at version gives an app the module in
// .weft/go: only a development build does.
func localModule(version string) bool { return version == devVersion }

// goMod is what go mod edit -json says of a go.mod, as far as weft reads it.
type goMod struct {
	Require []modVersion
	Replace []struct{ Old, New modVersion }
}

type modVersion struct{ Path, Version string }

func (m modVersion) String() string {
	if m.Version == "" {
		return m.Path
	}
	return m.Path + " " + m.Version
}

// readGoMod reads appDir/go.mod with go mod edit -json; ok is false when
// there is none.
func readGoMod(appDir string) (mod goMod, ok bool, err error) {
	file := filepath.Join(appDir, "go.mod")
	if _, err := os.Stat(file); err != nil {
		return goMod{}, false, nil
	}
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Dir = appDir
	var errb bytes.Buffer
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		return goMod{}, false, fmt.Errorf("go mod edit cannot read %s (%s); fix it and run the command again", file, strings.TrimSpace(errb.String()))
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		return goMod{}, false, fmt.Errorf("go mod edit -json said something weft cannot read about %s (%v); run the command again", file, err)
	}
	return mod, true, nil
}

// required is the version the go.mod requires the module at, or "".
func (m goMod) required() string {
	for _, r := range m.Require {
		if r.Path == weftModule {
			return r.Version
		}
	}
	return ""
}

// replaced says whether the go.mod replaces the module (at any version).
func (m goMod) replaced() bool {
	for _, r := range m.Replace {
		if r.Old.Path == weftModule {
			return true
		}
	}
	return false
}

// intoWeftGo says whether a replace's target is the app's .weft/go folder.
func intoWeftGo(appDir string, to modVersion) bool {
	if to.Version != "" {
		return false // another module, not a folder
	}
	target := filepath.FromSlash(to.Path)
	if !filepath.IsAbs(target) {
		target = filepath.Join(appDir, target)
	}
	want, err := filepath.Abs(filepath.Join(appDir, ".weft", "go"))
	if err != nil {
		return false
	}
	got, err := filepath.Abs(target)
	return err == nil && got == want
}

// moveGoMod moves appDir/go.mod to Weft version, a release: its require of
// the module to v<version>, and without a replace of the module into .weft/go,
// which a release's bundle does not fill. A replace of the module elsewhere
// is the author's: it is kept, and named on stderr.
func moveGoMod(appDir, version string, stderr io.Writer) error {
	mod, ok, err := readGoMod(appDir)
	if err != nil || !ok {
		return err
	}
	args := []string{"mod", "edit"}
	if have := mod.required(); have != "" && have != "v"+version {
		args = append(args, "-require="+weftModule+"@v"+version)
	}
	var kept []string
	for _, r := range mod.Replace {
		if r.Old.Path != weftModule {
			continue
		}
		if !intoWeftGo(appDir, r.New) {
			kept = append(kept, r.New.String())
			continue
		}
		drop := r.Old.Path
		if r.Old.Version != "" {
			drop += "@" + r.Old.Version
		}
		args = append(args, "-dropreplace="+drop)
	}
	if len(args) > 2 {
		cmd := exec.Command("go", args...)
		cmd.Dir = appDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go mod edit cannot move %s to Weft %s (%s); fix it and run weft upgrade again", filepath.Join(appDir, "go.mod"), version, strings.TrimSpace(string(out)))
		}
	}
	for _, to := range kept {
		fmt.Fprintf(stderr, "weft upgrade: go.mod replaces %s with %s, so the app builds against that, not Weft %s from GitHub; remove that replace line from go.mod to build against the release.\n", weftModule, to, version)
	}
	return nil
}

// fetchModule makes sure an app that requires the module from the proxy has
// its checksums in go.sum, running go mod download when they are missing,
// so go build and go test work in it; an app that replaces the module, or
// does not require it, is left alone.
func fetchModule(appDir string, stdout, stderr io.Writer) error {
	mod, ok, err := readGoMod(appDir)
	if err != nil || !ok {
		return err
	}
	version := mod.required()
	if version == "" || mod.replaced() {
		return nil
	}
	sum, _ := os.ReadFile(filepath.Join(appDir, "go.sum"))
	have := func(line string) bool {
		return bytes.Contains(append([]byte("\n"), sum...), []byte("\n"+line))
	}
	if have(weftModule+" "+version+" h1:") && have(weftModule+" "+version+"/go.mod h1:") {
		return nil
	}
	cmd := exec.Command("go", "mod", "download", weftModule)
	cmd.Dir, cmd.Env = appDir, hostBuildEnv(os.Environ())
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod download %s did not finish in %s, so the app cannot build against Weft %s; fix what its output above says (Go fetches the module from the module proxy, so the network must be up) and run the command again", weftModule, appDir, strings.TrimPrefix(version, "v"))
	}
	return nil
}
