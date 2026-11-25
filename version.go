// Package weft carries the Weft release a build belongs to; an app's Go code imports github.com/flakerimi/weft/warp.
package weft

// Version is the Weft release this build belongs to. Release builds set it with
// -ldflags "-X github.com/flakerimi/weft.Version=<version>".
var Version = "0.0.0-dev"
