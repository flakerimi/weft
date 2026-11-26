package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/flakerimi/weft/warp"
)

func TestNeedsData(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(Module())
	want := `warp: module "auth" needs the "data" module; list data.Module() before it in app/modules.go.`
	if err := app.Err(); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Err = %v, want %q", err, want)
	}
}

func TestPer(t *testing.T) {
	per := Per("products")
	for action, want := range map[string]string{
		"list": "products:read", "show": "products:read", "create": "products:create",
		"update": "products:edit", "remove": "products:delete", "export": "products:export",
	} {
		app := warp.New(warp.ConfigFrom(nil))
		warp.Get(app, "/x", func(_ context.Context, _ warp.None) (warp.None, error) { return warp.None{}, nil }, per(action)...)
		if r := app.Routes(); len(r) != 1 || r[0].Permission != want {
			t.Errorf("Per(products)(%q) records %+v, want %q", action, r, want)
		}
	}
}
