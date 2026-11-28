package products_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/auth/authtest"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail/mailtest"
	"github.com/flakerimi/weft/warp/warptest"

	"shop/app"
	"shop/app/products"

	_ "modernc.org/sqlite"
)

// shop starts the app's modules over a database file of the test's own,
// taking API keys, with the request log off.
func shop(t *testing.T) *warptest.Client {
	t.Helper()
	return warptest.New(t, map[string]string{
		"DB_DRIVER":        "sqlite",
		"DB_URL":           "file:" + filepath.Join(t.TempDir(), "data", "shop.db"),
		"AUTH_SECRET":      "a secret for tests that is long enough to sign",
		"AUTH_OWNER_EMAIL": "owner@example.com",
		"APP_URL":          "https://shop.example.com",
		"API_KEYS":         "on",
		"LOG":              "off",
	}, app.Modules()...)
}

// names lists the names of a page of products, in order.
func names(l data.List[products.Product]) []string {
	var out []string
	for _, p := range l.Items {
		out = append(out, p.Name)
	}
	return out
}

// The whole story over HTTP: the owner is set up and signs in, makes three
// products, lists them sorted, searched, filtered and paged, changes one
// field, is refused a taken name and a sent id, deletes one; a visitor with
// no session is refused every route, and so is another site.
func TestStory(t *testing.T) {
	c := shop(t)
	if owner := authtest.Owner(t, c); owner.Email != "owner@example.com" || owner.Role != "owner" {
		t.Fatalf("the owner is %+v", owner)
	}

	made := map[string]products.Product{}
	for _, p := range []map[string]any{
		{"name": "Lamp", "price": 30, "status": "live"},
		{"name": "Chair", "price": 120, "status": "draft"},
		{"name": "Desk lamp", "price": 75, "status": "live"},
	} {
		var got products.Product
		c.Post("/api/products", p).Want(http.StatusCreated).JSON(&got)
		if got.ID == 0 || got.Name != p["name"] || got.CreatedAt.IsZero() {
			t.Fatalf("POST %v answered %+v", p, got)
		}
		made[got.Name] = got
	}

	for _, tc := range []struct {
		query string
		want  []string
		total int
	}{
		{"?sort=-price", []string{"Chair", "Desk lamp", "Lamp"}, 3},
		{"?q=lamp&sort=name", []string{"Desk lamp", "Lamp"}, 2},
		{"?filter%5Bstatus%5D=live&sort=price", []string{"Lamp", "Desk lamp"}, 2},
		{"?filter%5Bstatus%5D=draft", []string{"Chair"}, 1},
		{"?sort=name&per_page=2", []string{"Chair", "Desk lamp"}, 3},
		{"?sort=name&per_page=2&page=2", []string{"Lamp"}, 3},
	} {
		var l data.List[products.Product]
		c.Get("/api/products" + tc.query).Want(http.StatusOK).JSON(&l)
		if got := names(l); !slices.Equal(got, tc.want) || l.Total != tc.total {
			t.Errorf("GET /api/products%s = %v of %d, want %v of %d", tc.query, got, l.Total, tc.want, tc.total)
		}
	}
	var paged data.List[products.Product]
	c.Get("/api/products?per_page=2&page=2").Want(http.StatusOK).JSON(&paged)
	if paged.Page != 2 || paged.PerPage != 2 || len(paged.Items) != 1 {
		t.Errorf("page 2 of 2 a page = %+v", paged)
	}

	// Only the price changes.
	lamp := made["Lamp"]
	var changed products.Product
	c.Patch("/api/products/"+id(lamp), map[string]any{"price": 5}).Want(http.StatusOK).JSON(&changed)
	if changed.Price != 5 || changed.Name != "Lamp" || changed.Status != "live" || !changed.CreatedAt.Equal(lamp.CreatedAt) {
		t.Errorf("PATCH price answered %+v, want Lamp, 5, live, made at %v", changed, lamp.CreatedAt)
	}
	var shown products.Product
	c.Get("/api/products/" + id(lamp)).Want(http.StatusOK).JSON(&shown)
	if shown.Price != 5 || shown.Name != "Lamp" || shown.Status != "live" {
		t.Errorf("GET after the PATCH = %+v", shown)
	}

	// A name another product has is a 422 on name, made or changed.
	if e := c.Post("/api/products", map[string]any{"name": "Lamp", "price": 1, "status": "draft"}).Want(http.StatusUnprocessableEntity).Error(); e.Fields["name"] != "This is already taken." {
		t.Errorf("a second Lamp: %+v", e)
	}
	if e := c.Patch("/api/products/"+id(made["Chair"]), map[string]any{"name": "Lamp"}).Want(http.StatusUnprocessableEntity).Error(); e.Fields["name"] != "This is already taken." {
		t.Errorf("renaming Chair to Lamp: %+v", e)
	}
	// The id is the store's: the body may not carry one, and the error says so.
	if e := c.Patch("/api/products/"+id(lamp), map[string]any{"id": 9}).Want(http.StatusBadRequest).Error(); e.Code != "bad_request" || e.Message != `The body has the field "id", which this route does not take.` {
		t.Errorf(`PATCH {"id": 9}: %+v, want bad_request naming id`, e)
	}

	chair := made["Chair"]
	c.Delete("/api/products/" + id(chair)).Want(http.StatusNoContent)
	c.Get("/api/products/" + id(chair)).Want(http.StatusNotFound)
	// An id that is no number is the same 404, which the product page shows
	// for /products/abc.
	c.Get("/api/products/NaN").Want(http.StatusNotFound)

	// No session: every route of the resource is refused.
	anon := c.Fork()
	for name, r := range map[string]*warptest.Response{
		"list":   anon.Get("/api/products"),
		"show":   anon.Get("/api/products/" + id(lamp)),
		"create": anon.Post("/api/products", map[string]any{"name": "Stool", "price": 9, "status": "live"}),
		"update": anon.Patch("/api/products/"+id(lamp), map[string]any{"price": 1}),
		"remove": anon.Delete("/api/products/" + id(lamp)),
	} {
		if e := r.Want(http.StatusUnauthorized).Error(); e.Code != "unauthorized" {
			t.Errorf("%s without a session: %+v", name, e)
		}
	}

	// Another site, with the owner's session: refused, and nothing deleted.
	r := httptest.NewRequest(http.MethodDelete, "http://warptest.local/api/products/"+id(lamp), nil)
	r.Header.Set("Origin", "https://elsewhere.example")
	if e := c.Do(r).Want(http.StatusForbidden).Error(); e.Code != "forbidden" {
		t.Errorf("a cross-site DELETE: %+v", e)
	}
	c.Get("/api/products/" + id(lamp)).Want(http.StatusOK)
}

// The story with roles: the owner invites a member and a viewer, and each
// takes the invitation through the mailed link. The member makes, changes
// and deletes products; the viewer lists them and is refused making one.
// The owner makes the member a viewer, and the member's next create is
// refused. An API key of the owner's lists the products with no cookie.
func TestRolesStory(t *testing.T) {
	c := shop(t)
	authtest.Owner(t, c)

	join := func(email, role string) *warptest.Client {
		t.Helper()
		var inv auth.Invitation
		c.Post("/api/auth/invitations", map[string]string{"email": email, "role": role}).Want(http.StatusCreated).JSON(&inv)
		msg := mailtest.Last(t, c.App, email)
		if msg.Subject != "You are invited to shop.example.com" {
			t.Fatalf("the invitation to %s is %q", email, msg.Subject)
		}
		u := c.Fork()
		var me auth.Me
		u.Post("/api/auth/password", map[string]string{"token": mailtest.Token(t, msg), "password": "a password of " + email}).Want(http.StatusOK).JSON(&me)
		if me.Email != email || me.Role != role {
			t.Fatalf("%s took the invitation as %+v", email, me)
		}
		return u
	}
	member := join("member@example.com", auth.RoleMember)
	viewer := join("viewer@example.com", auth.RoleViewer)

	var lamp, chair products.Product
	member.Post("/api/products", map[string]any{"name": "Lamp", "price": 30, "status": "live"}).Want(http.StatusCreated).JSON(&lamp)
	member.Post("/api/products", map[string]any{"name": "Chair", "price": 120, "status": "draft"}).Want(http.StatusCreated).JSON(&chair)
	member.Patch("/api/products/"+id(lamp), map[string]any{"price": 35}).Want(http.StatusOK)
	member.Delete("/api/products/" + id(chair)).Want(http.StatusNoContent)

	var l data.List[products.Product]
	viewer.Get("/api/products").Want(http.StatusOK).JSON(&l)
	if got := names(l); !slices.Equal(got, []string{"Lamp"}) {
		t.Errorf("the viewer lists %v, want [Lamp]", got)
	}
	viewer.Get("/api/products/" + id(lamp)).Want(http.StatusOK)
	if e := viewer.Post("/api/products", map[string]any{"name": "Stool", "price": 9, "status": "live"}).Want(http.StatusForbidden).Error(); e.Message != "Your account cannot do that." {
		t.Errorf("the viewer's create: %+v", e)
	}

	var users data.List[auth.User]
	c.Get("/api/auth/users?q=member").Want(http.StatusOK).JSON(&users)
	if len(users.Items) != 1 {
		t.Fatalf("the member is not listed once: %+v", users)
	}
	c.Patch("/api/auth/users/"+strconv.FormatInt(users.Items[0].ID, 10), map[string]string{"role": auth.RoleViewer}).Want(http.StatusOK)
	member.Post("/api/products", map[string]any{"name": "Stool", "price": 9, "status": "live"}).Want(http.StatusForbidden)

	var key auth.NewKey
	c.Post("/api/auth/keys", map[string]any{"name": "export", "role": auth.RoleViewer, "days": 30}).Want(http.StatusCreated).JSON(&key)
	r := httptest.NewRequest(http.MethodGet, "http://warptest.local/api/products", nil)
	r.Header.Set("Authorization", "Bearer "+key.Secret)
	var byKey data.List[products.Product]
	c.Fork().Do(r).Want(http.StatusOK).JSON(&byKey)
	if got := names(byKey); !slices.Equal(got, []string{"Lamp"}) {
		t.Errorf("the key lists %v, want [Lamp]", got)
	}
}

// id is the product's id as a path segment.
func id(p products.Product) string { return strconv.FormatInt(p.ID, 10) }
