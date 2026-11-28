package auth

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
)

// rolesOf is the roles of a module given opts, failing the test on a mistake.
func rolesOf(t *testing.T, opts ...Option) *roles {
	t.Helper()
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	r, mistakes := newRoles(o.grants)
	if len(mistakes) > 0 {
		t.Fatal(mistakes)
	}
	return r
}

func TestAllowedTable(t *testing.T) {
	r := rolesOf(t)
	perms := []string{"products:read", "products:create", "products:edit", "products:delete",
		"users:read", "users:invite", "users:edit", "users:delete", "reports:export"}
	want := map[string][]bool{
		RoleViewer:        {true, false, false, false, false, false, false, false, false},
		RoleMember:        {true, true, true, true, false, false, false, false, false},
		RoleAdministrator: {true, true, true, true, true, true, true, true, false},
		RoleOwner:         {true, true, true, true, true, true, true, true, true},
		"editor":          {false, false, false, false, false, false, false, false, false},
		"":                {false, false, false, false, false, false, false, false, false},
		"Owner":           {false, false, false, false, false, false, false, false, false},
	}
	for role, row := range want {
		for i, p := range perms {
			if got := r.allowed(role, p); got != row[i] {
				t.Errorf("allowed(%q, %q) = %v, want %v", role, p, got, row[i])
			}
		}
	}
	// A permission that is not resource:action is the owner's alone.
	for _, p := range []string{"products", "", ":", "products:"} {
		if r.allowed(RoleViewer, p) || r.allowed(RoleAdministrator, p) {
			t.Errorf("allowed(…, %q) is true below the owner", p)
		}
		if !r.allowed(RoleOwner, p) {
			t.Errorf("allowed(owner, %q) is false", p)
		}
	}
}

// Review Focus 3: a * resource never covers users, however it is granted.
func TestWildcardNeverCoversUsers(t *testing.T) {
	r := rolesOf(t, Grant(RoleMember, "*:*"), Grant(RoleViewer, "*:edit"))
	for _, p := range []string{"users:read", "users:edit", "users:invite", "users:delete", "users:export"} {
		if r.allowed(RoleMember, p) {
			t.Errorf("a member with *:* has %s", p)
		}
		if r.allowed(RoleViewer, p) {
			t.Errorf("a viewer with *:edit has %s", p)
		}
	}
	if !r.allowed(RoleMember, "reports:export") || !r.allowed(RoleViewer, "products:edit") {
		t.Error("the wildcards cover nothing else either")
	}
	// Granted by name, users is granted.
	r = rolesOf(t, Grant(RoleMember, "users:read"))
	if !r.allowed(RoleMember, "users:read") || r.allowed(RoleViewer, "users:read") || r.allowed(RoleMember, "users:edit") {
		t.Error("users:read granted by name to a member")
	}
}

func TestGrantIncludesRolesAbove(t *testing.T) {
	r := rolesOf(t, Grant(RoleViewer, "reports:export"), Grant(RoleAdministrator, "audit:read", "audit:*"))
	for _, role := range []string{RoleViewer, RoleMember, RoleAdministrator, RoleOwner} {
		if !r.allowed(role, "reports:export") {
			t.Errorf("%s lacks reports:export, granted to viewer", role)
		}
	}
	for role, want := range map[string]bool{RoleViewer: false, RoleMember: false, RoleAdministrator: true, RoleOwner: true} {
		if r.allowed(role, "audit:purge") != want {
			t.Errorf("allowed(%s, audit:purge) = %v, want %v", role, !want, want)
		}
	}
}

func TestGrantMistakes(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), mail.Module(), Module(
		Grant(RoleMember, "products:export", "Products"),
		Grant("editor", "products:export"),
		Grant(RoleViewer, "products", "*:*:*", "a b:read", "*:read"),
	))
	err := app.Err()
	if err == nil {
		t.Fatal("Err = nil")
	}
	for _, want := range []string{
		`warp: auth.Grant has the permission "Products"; write it as resource:action in lower case, such as products:export.`,
		`warp: auth.Grant names the role "editor"; use owner, administrator, member or viewer.`,
		`warp: auth.Grant has the permission "products"; write it as resource:action in lower case, such as products:export.`,
		`warp: auth.Grant has the permission "*:*:*"; write it as resource:action in lower case, such as products:export.`,
		`warp: auth.Grant has the permission "a b:read"; write it as resource:action in lower case, such as products:export.`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Err = %v\nwant it to have %s", err, want)
		}
	}
	if n := strings.Count(err.Error(), "auth.Grant"); n != 5 {
		t.Errorf("Err has %d mistakes of Grant, want 5: %v", n, err)
	}
}

// productRoutes is a module with the routes of a products resource, guarded
// per action, and one route of its own guarded by reports:export.
type productRoutes struct{}

func (productRoutes) Name() string { return "products" }

func (productRoutes) Register(app *warp.App) error {
	ok := func(context.Context, warp.None) (warp.None, error) { return warp.None{}, nil }
	per := Per("products")
	for _, a := range []struct{ action, path string }{
		{"list", "/products"}, {"show", "/products/{id}"}, {"create", "/products/new"}, {"update", "/products/{id}/edit"}, {"remove", "/products/{id}/remove"},
	} {
		warp.Get(app, a.path, ok, append(per(a.action), warp.Name(a.action))...)
	}
	warp.Get(app, "/reports", ok, Can("reports:export"))
	warp.Get(app, "/open", ok) // no permission
	return nil
}

func TestPermissionsOfMe(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), mail.Module(), Module(Grant(RoleMember, "reports:export", "invoices:*", "archive:read")), productRoutes{})
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	m := of(app)
	want := map[string][]string{
		RoleViewer: {"archive:read", "products:read"}, // *:read covers a named grant too
		RoleMember: {"archive:read", "products:create", "products:delete", "products:edit", "products:read", "reports:export"},
		RoleAdministrator: {"archive:read", "products:create", "products:delete", "products:edit", "products:read", "reports:export",
			"users:delete", "users:edit", "users:invite", "users:read"},
		RoleOwner: {"archive:read", "products:create", "products:delete", "products:edit", "products:read", "reports:export",
			"users:delete", "users:edit", "users:invite", "users:read"},
		"editor": {},
	}
	for role, w := range want {
		got := m.permissions(role)
		if got == nil || !slices.Equal(got, w) {
			t.Errorf("permissions(%q) = %#v\nwant %#v", role, got, w)
		}
		// One function decides: every permission listed is allowed, and
		// Allows says the same.
		for _, p := range got {
			if !Allows(app, role, p) {
				t.Errorf("%s lists %s, which Allows refuses", role, p)
			}
		}
	}
	if Allows(warp.New(warp.ConfigFrom(nil)), RoleOwner, "products:read") {
		t.Error("Allows is true for an app without the auth module")
	}
}
