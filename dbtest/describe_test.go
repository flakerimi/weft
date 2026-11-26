package dbtest

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

// A describing run (WARP_DESCRIBE=1, as weft gen, routes and build run an
// app) starts nothing, however the app serves: through its Serve, through
// Run on a listener of its own, or after calling Start itself. So the
// database DB_URL names is never made or migrated, and each prints the
// description and ends with status 0.
func TestDescribingRunTouchesNoDatabase(t *testing.T) {
	if os.Getenv("DESCRIBE_HELPER") != "" {
		return
	}
	for _, how := range []string{"serve", "run", "start"} {
		dir := t.TempDir()
		db := filepath.Join(dir, "app.db")
		cmd := exec.Command(os.Args[0], "-test.run=^TestDescribeHelper$")
		cmd.Env = append(os.Environ(), "DESCRIBE_HELPER="+how, "WARP_DESCRIBE=1", "DB_DRIVER=sqlite", "DB_URL=file:"+db)
		out, err := cmd.Output()
		if err != nil {
			t.Errorf("%s: %v\n%s", how, err, out)
		}
		if !strings.Contains(string(out), `"path": "/api/products/count"`) {
			t.Errorf("%s: printed %q, want the description", how, out)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Errorf("%s: the describing run made %v beside where the database goes", how, entries)
		}
	}
}

// TestDescribeHelper is the app of TestDescribingRunTouchesNoDatabase, which
// serves the way DESCRIBE_HELPER names.
func TestDescribeHelper(t *testing.T) {
	how := os.Getenv("DESCRIBE_HELPER")
	if how == "" {
		t.Skip("run by TestDescribingRunTouchesNoDatabase")
	}
	app := warp.New(warp.FromEnv())
	app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
	warp.Get(app, "/products/count", func(context.Context, warp.None) (int, error) { return 0, nil })
	var err error
	switch how {
	case "serve":
		err = app.Serve(nil)
	case "run":
		ln, lerr := net.Listen("tcp", "127.0.0.1:0")
		if lerr != nil {
			t.Fatal(lerr)
		}
		err = app.Run(context.Background(), ln, nil)
	case "start":
		if err = app.Start(context.Background()); err == nil {
			err = app.Serve(nil)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}
