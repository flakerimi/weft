package dbtest

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/auth/authtest"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
	"github.com/flakerimi/weft/warp/warptest"
)

// products is a module with a products resource guarded per action, and a
// route guarded by reports:export, which a Grant gives.
type products struct{}

func (products) Name() string { return "products" }

func (products) Register(app *warp.App) error {
	s := data.NewStore[Product](data.For(app), "products")
	data.Resource(app, "/products", s, data.PerAction(auth.Per("products")))
	warp.Get(app, "/reports", func(context.Context, warp.None) (warp.None, error) { return warp.None{}, nil }, auth.Can("reports:export"))
	return nil
}

// startShop starts an app with data, auth, things and products, its owner
// signed in on the client it returns.
func startShop(t *testing.T, vars map[string]string, opts ...auth.Option) *warptest.Client {
	t.Helper()
	c := warptest.New(t, vars, data.Module(), mail.Module(), auth.Module(opts...), things{}, products{})
	authtest.Owner(t, c)
	return c
}

var roles = []string{auth.RoleViewer, auth.RoleMember, auth.RoleAdministrator, auth.RoleOwner}

func TestGuardByRole(t *testing.T) {
	c := startShop(t, authEnv(t))
	c.Post("/api/products", map[string]any{"name": "Lamp", "price": 12, "status": "live"}).Want(201)
	for _, role := range roles {
		rc, u := authtest.As(t, c, role)
		if u.Role != role || !strings.HasPrefix(u.Email, role+"-") || !strings.HasSuffix(u.Email, "@example.com") {
			t.Fatalf("As(%s) made %+v", role, u)
		}
		write := 200
		if role == auth.RoleViewer {
			write = 403
		}
		created := write
		if created == 200 {
			created = 201
		}
		removed := write
		if removed == 200 {
			removed = 204
		}
		at := fmt.Sprintf("%s:", role)
		if got := rc.Get("/api/products").Status; got != 200 {
			t.Errorf("%s list = %d", at, got)
		}
		if got := rc.Get("/api/products/1").Status; got != 200 {
			t.Errorf("%s show = %d", at, got)
		}
		res := rc.Post("/api/products", map[string]any{"name": "Desk " + role, "price": 80, "status": "draft"})
		if res.Status != created {
			t.Errorf("%s create = %d, want %d", at, res.Status, created)
		}
		if got := rc.Patch("/api/products/1", map[string]any{"price": 13}).Status; got != write {
			t.Errorf("%s update = %d, want %d", at, got, write)
		}
		if created == 201 {
			var p Product
			res.JSON(&p)
			if got := rc.Delete(fmt.Sprintf("/api/products/%d", p.ID)).Status; got != removed {
				t.Errorf("%s remove = %d, want %d", at, got, removed)
			}
		} else if got := rc.Delete("/api/products/1").Status; got != removed {
			t.Errorf("%s remove = %d, want %d", at, got, removed)
		}
		if role == auth.RoleViewer {
			wantError(t, rc.Patch("/api/products/1", map[string]any{"price": 13}), 403, "forbidden", "Your account cannot do that.")
		}
	}
}

// Review Focus 3: me's permissions and the guard answer from one function.
func TestMePermissionsMatchGuard(t *testing.T) {
	c := startShop(t, authEnv(t), auth.Grant(auth.RoleMember, "reports:export"))
	c.Post("/api/products", map[string]any{"name": "Lamp", "price": 12, "status": "live"}).Want(201)
	// A route per permission, to ask the guard.
	probe := map[string]func(rc *warptest.Client) int{
		"products:read":   func(rc *warptest.Client) int { return rc.Get("/api/products").Status },
		"products:create": func(rc *warptest.Client) int { return rc.Post("/api/products", map[string]any{"name": "x"}).Status },
		"products:edit":   func(rc *warptest.Client) int { return rc.Patch("/api/products/999", map[string]any{"price": 1}).Status },
		"products:delete": func(rc *warptest.Client) int { return rc.Delete("/api/products/999").Status },
		"reports:export":  func(rc *warptest.Client) int { return rc.Get("/api/reports").Status },
		"things:read":     func(rc *warptest.Client) int { return rc.Get("/api/things").Status },
		"things:purge":    func(rc *warptest.Client) int { return rc.Post("/api/things/purge", nil).Status },
		"users:read":      func(rc *warptest.Client) int { return rc.Get("/api/auth/invitations").Status },
		"users:edit":      func(rc *warptest.Client) int { return rc.Delete("/api/auth/users/999/sessions").Status },
		"users:delete":    func(rc *warptest.Client) int { return rc.Delete("/api/auth/users/999").Status },
		"users:invite": func(rc *warptest.Client) int {
			return rc.Post("/api/auth/invitations", map[string]string{"email": "probe@example.com", "role": "viewer"}).Status
		},
	}
	for _, role := range roles {
		rc, _ := authtest.As(t, c, role)
		var me auth.Me
		rc.Get("/api/auth/me").Want(200).JSON(&me)
		if me.Role != role || !slices.IsSorted(me.Permissions) {
			t.Fatalf("%s: me = %+v", role, me)
		}
		for _, r := range c.App.Routes() {
			if r.Permission == "" {
				continue
			}
			listed := slices.Contains(me.Permissions, r.Permission)
			if listed != auth.Allows(c.App, role, r.Permission) {
				t.Errorf("%s: me lists %s is %v, Allows says %v", role, r.Permission, listed, !listed)
			}
			p := probe[r.Permission]
			if p == nil {
				t.Fatalf("no probe for %s", r.Permission)
			}
			got := p(rc)
			if listed && (got == 401 || got == 403) {
				t.Errorf("%s: me lists %s, and the guard answers %d", role, r.Permission, got)
			}
			if !listed && got != 403 {
				t.Errorf("%s: me does not list %s, and the guard answers %d, not 403", role, r.Permission, got)
			}
		}
	}
	// The owner's list holds every route's permission and the named grants.
	var me auth.Me
	c.Get("/api/auth/me").Want(200).JSON(&me)
	want := []string{"products:create", "products:delete", "products:edit", "products:read", "reports:export", "things:purge", "things:read",
		"users:delete", "users:edit", "users:invite", "users:read"}
	if !slices.Equal(me.Permissions, want) {
		t.Fatalf("the owner's permissions = %q, want %q", me.Permissions, want)
	}
	// On the wire: name, and the list as a list.
	rc, _ := authtest.As(t, c, auth.RoleViewer)
	body := string(rc.Get("/api/auth/me").Want(200).Body)
	if !strings.Contains(body, `"permissions":["products:read","things:read"]`) || !strings.Contains(body, `"name":""`) {
		t.Fatalf("a viewer's me = %s", body)
	}
}

// A role with no permission at all still answers a list, never null.
func TestMePermissionsNeverNull(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	if _, err := data.For(c.App).Exec(ctx, `UPDATE "users" SET "role" = 'retired' WHERE "id" = 1`); err != nil {
		t.Fatal(err)
	}
	body := string(c.Get("/api/auth/me").Want(200).Body)
	if !strings.Contains(body, `"permissions":[]`) {
		t.Fatalf("me of a role with none = %s", body)
	}
}

// sliceUser is the users table as the slice made it, before name.
type sliceUser struct {
	data.Record
	Email        string `json:"email" warp:"unique"`
	Role         string `json:"role"`
	PasswordHash string `json:"password_hash"`
}

func TestNameColumnAdded(t *testing.T) {
	vars := authEnv(t)
	var s *data.Store[sliceUser]
	old := warptest.New(t, vars, data.Module(), stores{"old", func(db *data.DB) { s = data.NewStore[sliceUser](db, "users") }})
	// A hash in the stored form, with few iterations; the count is read back
	// from the text.
	salt := []byte("sixteen bytes!!!")
	key, err := pbkdf2.Key(sha256.New, "an old password", salt, 1000, 32)
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.RawStdEncoding
	hash := "pbkdf2-sha256$1000$" + b64.EncodeToString(salt) + "$" + b64.EncodeToString(key)
	if err := s.Create(ctx, &sliceUser{Email: ownerEmail, Role: "owner", PasswordHash: hash}); err != nil {
		t.Fatal(err)
	}
	if err := old.App.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	c, _ := startAuth(t, vars)
	var me auth.Me
	login(c, ownerEmail, "an old password").Want(200).JSON(&me)
	if me.ID != 1 || me.Email != ownerEmail || me.Name != "" || me.Role != "owner" {
		t.Fatalf("the slice's owner signs in as %+v", me)
	}
	var name string
	if err := data.For(c.App).QueryRow(ctx, `SELECT "name" FROM "users" WHERE "id" = 1`).Scan(&name); err != nil || name != "" {
		t.Fatalf("name = %q, %v", name, err)
	}
}
