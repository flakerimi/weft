// Command site-src serves weft.base.al: the pages Weft builds from this folder,
// the installer and every release, embedded in one Go binary.
//
// Build the pages first (scripts/site.sh build); scripts/publish-site.sh ships
// this binary to the Lisa server in a container.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/flakerimi/weft"
)

//go:embed all:dist
var siteDist embed.FS

func handler() http.Handler {
	dist, err := fs.Sub(siteDist, "dist")
	if err != nil {
		panic(err)
	}
	return weft.Serve(dist, nil)
}

func main() {
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	h := handler()
	log.Printf("weft.base.al (weft %s) on http://localhost%s", weft.Version, addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
