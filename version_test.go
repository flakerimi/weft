package weft

import (
	"runtime/debug"
	"testing"
)

// An app that requires the module from the proxy gets its release as
// Version; the module itself, a replaced module or one with no release
// version leaves the development version.
func TestModuleVersion(t *testing.T) {
	dep := func(path, version string, replace *debug.Module) *debug.Module {
		return &debug.Module{Path: path, Version: version, Replace: replace}
	}
	for _, c := range []struct {
		name string
		bi   *debug.BuildInfo
		want string
	}{
		{"a release", &debug.BuildInfo{Deps: []*debug.Module{dep("example.com/x", "v1.0.0", nil), dep(modulePath, "v0.0.12", nil)}}, "0.0.12"},
		{"a pre-release", &debug.BuildInfo{Deps: []*debug.Module{dep(modulePath, "v0.1.0-alpha.1", nil)}}, "0.1.0-alpha.1"},
		{"replaced", &debug.BuildInfo{Deps: []*debug.Module{dep(modulePath, "v0.0.0", dep("./.weft/go", "", nil))}}, ""},
		{"the main module", &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v0.0.12"}}, ""},
		{"no version", &debug.BuildInfo{Deps: []*debug.Module{dep(modulePath, "(devel)", nil)}}, ""},
		{"another module", &debug.BuildInfo{Deps: []*debug.Module{dep(modulePath+"/x", "v0.0.12", nil)}}, ""},
		{"no build information", nil, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := moduleVersion(c.bi); got != c.want {
				t.Errorf("moduleVersion = %q, want %q", got, c.want)
			}
		})
	}
}

// This test binary is the module itself, so Version stays the development
// version (no -X sets it here).
func TestVersionInTheModule(t *testing.T) {
	if Version != "0.0.0-dev" {
		t.Errorf("Version = %q, want 0.0.0-dev", Version)
	}
}
