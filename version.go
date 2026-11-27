// Package weft carries the Weft release a build belongs to; an app's Go code imports github.com/flakerimi/weft/warp.
package weft

import (
	"runtime/debug"
	"strings"
)

// Version is the Weft release this build belongs to. Release builds of weft
// set it with -ldflags "-X github.com/flakerimi/weft.Version=<version>"; an
// app that requires the module at a release (vX.Y.Z, from the module proxy)
// gets that release from its build information.
var Version = "0.0.0-dev"

// modulePath is the path of this module.
const modulePath = "github.com/flakerimi/weft"

func init() {
	if Version == "0.0.0-dev" {
		bi, _ := debug.ReadBuildInfo()
		if v := moduleVersion(bi); v != "" {
			Version = v
		}
	}
}

// moduleVersion is the release of this module a program was built with, as
// its build information records it, without the v: "" when the program is
// this module, replaces it, or records no release of it.
func moduleVersion(bi *debug.BuildInfo) string {
	if bi == nil {
		return ""
	}
	for _, m := range bi.Deps {
		if m.Path == modulePath && m.Replace == nil && strings.HasPrefix(m.Version, "v") {
			return strings.TrimPrefix(m.Version, "v")
		}
	}
	return ""
}
