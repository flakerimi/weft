// Command playground serves the Weft playground: every component in one page,
// embedded in a Go binary.
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/flakerimi/weft"
)

//go:embed all:web/dist
var webDist embed.FS

func handler() http.Handler {
	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		panic(err)
	}
	api := http.NewServeMux()
	api.HandleFunc("GET /api/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": weft.Version})
	})
	return weft.Serve(dist, api)
}

func main() {
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("weft playground on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler()))
}
