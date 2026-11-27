// Command site-src serves weft.base.al: the pages Weft builds from this folder,
// embedded in the Go binary, and the installer and every release from a
// folder on disk, $WEFT_RELEASES_DIR (in the image, /data/releases), so the
// binary does not grow with each release.
//
// Build the pages first (scripts/site.sh build); scripts/publish-site.sh ships
// this binary and the releases to the Lisa server in a container.
package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/flakerimi/weft"
	"github.com/flakerimi/weft/warp"
)

//go:embed all:dist
var siteDist embed.FS

// releasesDir is where the installer and the releases are: $WEFT_RELEASES_DIR,
// or /data/releases, where the image has them.
func releasesDir() string {
	if d := os.Getenv("WEFT_RELEASES_DIR"); d != "" {
		return d
	}
	return "/data/releases"
}

// releaseFiles opens dir, the folder scripts/stage-releases.sh stages: the
// installers as install and install.ps1 and the releases under releases/. Its
// files are served at their own paths, so it may hold nothing else. It is
// opened as an os.Root, so nothing put in it later, a symlink say, leads out of it.
func releaseFiles(dir string) (fs.FS, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("the releases folder %s cannot be read (%w)", dir, err)
	}
	files := root.FS()
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		root.Close()
		return nil, fmt.Errorf("the releases folder %s cannot be read (%w)", dir, err)
	}
	for _, e := range entries {
		if name := e.Name(); name != "install" && name != "install.ps1" && name != "releases" {
			root.Close()
			return nil, fmt.Errorf("the releases folder %s holds %s; it holds only install, install.ps1 and releases/, as scripts/stage-releases.sh stages them", dir, name)
		}
	}
	for _, name := range []string{"install", "install.ps1"} {
		if fi, err := fs.Stat(files, name); errors.Is(err, fs.ErrNotExist) || (err == nil && !fi.Mode().IsRegular()) {
			root.Close()
			return nil, fmt.Errorf("the releases folder %s has no %s; stage it with scripts/stage-releases.sh, or set WEFT_RELEASES_DIR", dir, name)
		}
	}
	// The root stays open as long as the server serves from it.
	return files, nil
}

// handler serves the embedded pages, and /install, /install.ps1 and /releases/ from releases.
func handler(releases fs.FS) http.Handler {
	dist, err := fs.Sub(siteDist, "dist")
	if err != nil {
		panic(err)
	}
	return warp.Serve(dist, http.HandlerFunc(demoAPI), warp.WithFiles("/", releases))
}

// demoMe is who the gallery's examples see signed in: a demo viewer who may
// also edit products, so the perm Button example draws Edit and not Delete,
// and of the Guard example's two Guards one opens and one is denied. It is the
// JSON of packages/vite/test/fixtures/demo-me.json, which examples-hydrate
// answers too.
var demoMe = []byte(`{"id":1,"created_at":"2026-11-28T00:00:00Z","updated_at":"2026-11-28T00:00:00Z","email":"demo@example.com","name":"Demo","role":"viewer","permissions":["products:edit","products:read"]}` + "\n")

// demoAPI is the site's whole API: GET /api/auth/me answers demoMe, and every
// other request under /api is the 404 of a site with no API, as warp.Serve
// answers with none. It is a demo, not a session: the site has no accounts,
// reads and sets no cookie, and has no other route, so nothing else on it acts
// as if it had an API.
func demoAPI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/auth/me" || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		warp.WriteError(w, r, warp.NotFound("This app has no API."))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		w.Write(demoMe)
	}
}

func main() {
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	releases, err := releaseFiles(releasesDir())
	if err != nil {
		log.Fatalf("weft.base.al: %v.", err)
	}
	h := handler(releases)
	log.Printf("weft.base.al (weft %s) on http://localhost%s, releases from %s", weft.Version, addr, releasesDir())
	log.Fatal(http.ListenAndServe(addr, h))
}
