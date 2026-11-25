// Command weft is the Weft side of the benchmark in bench/: the landing Weft
// builds from this folder, embedded in one Go binary and served by warp.Serve,
// as site-src serves weft.base.al.
//
// Build the page first (bench/run.sh does, or pnpm --filter weft-bench build).
// ADDR sets the listen address (default 127.0.0.1:4301).
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/flakerimi/weft"
	"github.com/flakerimi/weft/warp"
)

//go:embed all:dist
var benchDist embed.FS

func handler() http.Handler {
	dist, err := fs.Sub(benchDist, "dist")
	if err != nil {
		panic(err)
	}
	return warp.Serve(dist, nil)
}

func main() {
	addr := "127.0.0.1:4301"
	if a := os.Getenv("ADDR"); a != "" {
		addr = a
	}
	h := handler()
	log.Printf("weft bench (weft %s) on http://%s", weft.Version, addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
