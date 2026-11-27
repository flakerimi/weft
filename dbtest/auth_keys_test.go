package dbtest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/auth/authtest"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

const (
	keyNotValid = "This API key is not valid."
	noKeys      = "This app does not take API keys."
)

var wholeKey = regexp.MustCompile(`^wk_(\d+)_[A-Za-z0-9_-]{43}$`)

// keysOn is authEnv with API_KEYS=on.
func keysOn(t *testing.T) map[string]string {
	vars := authEnv(t)
	vars["API_KEYS"] = "on"
	return vars
}

// makeKey makes a key on c's session and returns it, whole.
func makeKey(t *testing.T, c *warptest.Client, name, role string, days int) auth.NewKey {
	t.Helper()
	var k auth.NewKey
	c.Post("/api/auth/keys", map[string]any{"name": name, "role": role, "days": days}).Want(201).JSON(&k)
	return k
}

// bearer sends a request with the key and no cookie.
func bearer(c *warptest.Client, key, method, path string, body any) *warptest.Response {
	r := request(method, path, body, "")
	r.Header.Set("Authorization", "Bearer "+key)
	return c.Fork().Do(r)
}

func TestKeyActsAsUser(t *testing.T) {
	c := startShop(t, keysOn(t))
	member, mu := authtest.As(t, c, auth.RoleMember)
	k := makeKey(t, member, "Importer", auth.RoleMember, 0)
	m := wholeKey.FindStringSubmatch(k.Secret)
	if m == nil || m[1] != fmt.Sprint(k.ID) || k.Name != "Importer" || k.Role != auth.RoleMember || k.SeenAt != nil {
		t.Fatalf("create_key answered %+v", k)
	}
	if want := fmt.Sprintf("wk_%d_", k.ID) + k.Secret[len(fmt.Sprintf("wk_%d_", k.ID)):][:4]; k.Prefix != want {
		t.Fatalf("prefix %q, want %q", k.Prefix, want)
	}
	if d := k.ExpiresAt.Sub(k.CreatedAt).Round(time.Hour); d != 90*24*time.Hour {
		t.Fatalf("a key of 0 days lasts %v, want 90 days", d)
	}
	var got map[string]string
	bearer(c, k.Secret, http.MethodGet, "/api/things", nil).Want(200).JSON(&got)
	if got["email"] != mu.Email || got["role"] != auth.RoleMember {
		t.Fatalf("the key acted as %v", got)
	}
	bearer(c, k.Secret, http.MethodPost, "/api/products", map[string]any{"name": "Lamp"}).Want(201)
	var me auth.Me
	bearer(c, k.Secret, http.MethodGet, "/api/auth/me", nil).Want(200).JSON(&me)
	if me.ID != mu.ID || me.Role != auth.RoleMember {
		t.Fatalf("me with the key = %+v", me)
	}
	// The list has the key, never its secret.
	var list []auth.APIKey
	member.Get("/api/auth/keys").Want(200).JSON(&list)
	if len(list) != 1 || list[0].ID != k.ID || list[0].SeenAt == nil {
		t.Fatalf("keys = %+v", list)
	}
	if body := string(member.Get("/api/auth/keys").Body); strings.Contains(body, k.Secret[len(k.Prefix):]) || strings.Contains(body, "secret") {
		t.Fatalf("the list holds the secret: %s", body)
	}
	// A key is read as a key, never as a cookie: a bad key with a good
	// cookie is still 401.
	r := request(http.MethodGet, "/api/things", nil, "")
	r.Header.Set("Authorization", "Bearer wk_1_nope")
	wantError(t, member.Do(r), 401, "unauthorized", keyNotValid)
}

// Review Focus 3: a key does no more than its user's role now.
func TestKeyRoleIsTheLower(t *testing.T) {
	c := startShop(t, keysOn(t))
	c.Post("/api/products", map[string]any{"name": "Lamp"}).Want(201)
	admin, au := authtest.As(t, c, auth.RoleAdministrator)
	wantError(t, admin.Post("/api/auth/keys", map[string]any{"name": "Boss", "role": "owner"}), 403, "forbidden", "Your account cannot give the role owner.")
	low := makeKey(t, admin, "Reader", auth.RoleViewer, 1)
	high := makeKey(t, admin, "Editor", auth.RoleMember, 1)
	// The key's role, below the user's.
	bearer(c, low.Secret, http.MethodGet, "/api/products", nil).Want(200)
	wantError(t, bearer(c, low.Secret, http.MethodPatch, "/api/products/1", map[string]any{"price": 2}), 403, "forbidden", "Your account cannot do that.")
	bearer(c, high.Secret, http.MethodPatch, "/api/products/1", map[string]any{"price": 2}).Want(200)
	// A member's key cannot manage users, though its user may.
	wantError(t, bearer(c, high.Secret, http.MethodGet, "/api/auth/users", nil), 403, "forbidden", "Your account cannot do that.")
	admin.Get("/api/auth/users").Want(200)
	// Nor invite with more than its role.
	wantError(t, bearer(c, high.Secret, http.MethodPost, "/api/auth/invitations", map[string]string{"email": "x@example.com", "role": "member"}), 403, "forbidden", "Your account cannot do that.")
	// The user demoted to viewer: the member key is a viewer's now.
	c.Patch(fmt.Sprintf("/api/auth/users/%d", au.ID), map[string]string{"role": "viewer"}).Want(200)
	wantError(t, bearer(c, high.Secret, http.MethodPatch, "/api/products/1", map[string]any{"price": 3}), 403, "forbidden", "Your account cannot do that.")
	var me auth.Me
	bearer(c, high.Secret, http.MethodGet, "/api/auth/me", nil).Want(200).JSON(&me)
	if me.Role != auth.RoleViewer {
		t.Fatalf("me with the key of a demoted user = %+v", me)
	}
}

// A stolen key cannot make keys, read sessions or change a password.
func TestKeyCannotManageKeys(t *testing.T) {
	c := startShop(t, keysOn(t))
	k := makeKey(t, c, "Owner's", auth.RoleOwner, 30)
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/auth/keys", map[string]any{"name": "More", "role": "owner"}},
		{http.MethodGet, "/api/auth/keys", nil},
		{http.MethodDelete, fmt.Sprintf("/api/auth/keys/%d", k.ID), nil},
		{http.MethodGet, "/api/auth/sessions", nil},
		{http.MethodPost, "/api/auth/logout/everywhere", nil},
		{http.MethodPost, "/api/auth/password/change", map[string]string{"current": authtest.Password, "password": newPassword}},
		{http.MethodPatch, "/api/auth/me", map[string]string{"name": "x"}},
		{http.MethodPost, "/api/auth/logout", nil},
	} {
		wantError(t, bearer(c, k.Secret, tc.method, tc.path, tc.body), 401, "unauthorized", signIn)
	}
	// What the owner's key may do: the permission-guarded routes and me.
	bearer(c, k.Secret, http.MethodGet, "/api/auth/me", nil).Want(200)
	bearer(c, k.Secret, http.MethodGet, "/api/auth/users", nil).Want(200)
	login(c.Fork(), ownerEmail, authtest.Password).Want(200)
}

// Review Focus 1: a deleted key is out at once.
func TestRemovedKeyIsOut(t *testing.T) {
	c := startShop(t, keysOn(t))
	k := makeKey(t, c, "Script", auth.RoleMember, 30)
	bearer(c, k.Secret, http.MethodGet, "/api/things", nil).Want(200)
	member, _ := authtest.As(t, c, auth.RoleMember)
	wantError(t, member.Delete(fmt.Sprintf("/api/auth/keys/%d", k.ID)), 404, "not_found", fmt.Sprintf("No API key of yours has id %d.", k.ID))
	bearer(c, k.Secret, http.MethodGet, "/api/things", nil).Want(200)
	c.Delete(fmt.Sprintf("/api/auth/keys/%d", k.ID)).Want(204)
	wantError(t, bearer(c, k.Secret, http.MethodGet, "/api/things", nil), 401, "unauthorized", keyNotValid)
	wantError(t, c.Delete(fmt.Sprintf("/api/auth/keys/%d", k.ID)), 404, "not_found", fmt.Sprintf("No API key of yours has id %d.", k.ID))
}

func TestRemovedUserKeysGone(t *testing.T) {
	c := startShop(t, keysOn(t))
	member, mu := authtest.As(t, c, auth.RoleMember)
	a := makeKey(t, member, "One", auth.RoleMember, 30)
	makeKey(t, member, "Two", auth.RoleViewer, 30)
	other, ou := authtest.As(t, c, auth.RoleMember)
	b := makeKey(t, other, "Three", auth.RoleMember, 30)
	c.Delete(fmt.Sprintf("/api/auth/users/%d", mu.ID)).Want(204)
	wantError(t, bearer(c, a.Secret, http.MethodGet, "/api/things", nil), 401, "unauthorized", keyNotValid)
	if n := rowsIn(t, c, "api_keys"); n != 1 {
		t.Fatalf("%d keys kept, want the other user's", n)
	}
	if err := auth.RemoveUser(ctx, c.App, ou.ID); err != nil {
		t.Fatal(err)
	}
	wantError(t, bearer(c, b.Secret, http.MethodGet, "/api/things", nil), 401, "unauthorized", keyNotValid)
	if n := rowsIn(t, c, "api_keys"); n != 0 {
		t.Fatalf("%d keys of removed users kept", n)
	}
}

func TestExpiredKey(t *testing.T) {
	start := time.Now().Add(-10 * 24 * time.Hour).UTC().Truncate(time.Second)
	at := &clock{now: start}
	c, _ := startAuth(t, keysOn(t))
	authtest.Owner(t, c)
	data.For(c.App).SetClock(at.Now)
	k := makeKey(t, c, "Short", auth.RoleOwner, 1)
	at.now = start.Add(24*time.Hour - time.Second)
	bearer(c, k.Secret, http.MethodGet, "/api/things", nil).Want(200)
	at.now = start.Add(24 * time.Hour)
	wantError(t, bearer(c, k.Secret, http.MethodGet, "/api/things", nil), 401, "unauthorized", keyNotValid)
	// Removed at the next sign-in.
	login(c.Fork(), ownerEmail, authtest.Password).Want(200)
	if n := rowsIn(t, c, "api_keys"); n != 0 {
		t.Fatalf("%d keys after a sign-in, want the expired one gone", n)
	}
	wantField(t, c.Post("/api/auth/keys", map[string]any{"name": "Long", "role": "owner", "days": 366}), "days", "Enter 365 or less.")
	makeKey(t, c, "Year", auth.RoleOwner, 365)
}

func TestTwentyKeys(t *testing.T) {
	c := startShop(t, keysOn(t))
	for i := range 20 {
		makeKey(t, c, fmt.Sprintf("Key %d", i), auth.RoleViewer, 1)
	}
	wantError(t, c.Post("/api/auth/keys", map[string]any{"name": "One more", "role": "viewer"}), 409, "conflict", "You have 20 API keys; remove one first.")
	var list []auth.APIKey
	c.Get("/api/auth/keys").Want(200).JSON(&list)
	if len(list) != 20 || list[0].Name != "Key 19" {
		t.Fatalf("%d keys, the first %q; want 20, newest first", len(list), list[0].Name)
	}
	// Another user has their own twenty.
	member, _ := authtest.As(t, c, auth.RoleMember)
	makeKey(t, member, "Mine", auth.RoleMember, 1)
	c.Delete(fmt.Sprintf("/api/auth/keys/%d", list[0].ID)).Want(204)
	makeKey(t, c, "Again", auth.RoleViewer, 1)
	wantField(t, c.Post("/api/auth/keys", map[string]any{"name": strings.Repeat("n", 61), "role": "viewer"}), "name", "Use at most 60 characters.")
}

func TestKeysOff(t *testing.T) {
	c := startShop(t, authEnv(t))
	wantError(t, c.Post("/api/auth/keys", map[string]any{"name": "Script", "role": "viewer"}), 403, "forbidden", "API keys are off in this app; set API_KEYS=on in .env.")
	wantError(t, bearer(c, "wk_1_"+strings.Repeat("a", 43), http.MethodGet, "/api/things", nil), 401, "unauthorized", noKeys)
	wantError(t, bearer(c, "wk_1_"+strings.Repeat("a", 43), http.MethodGet, "/api/auth/me", nil), 401, "unauthorized", noKeys)
	var list []auth.APIKey
	c.Get("/api/auth/keys").Want(200).JSON(&list)
	if list == nil || len(list) != 0 {
		t.Fatalf("keys = %#v", list)
	}
}

func TestKeySecretNotStored(t *testing.T) {
	c := startShop(t, keysOn(t))
	k := makeKey(t, c, "Script", auth.RoleOwner, 30)
	secret := k.Secret[len(k.Prefix)-4:]
	rows, err := data.For(c.App).Query(ctx, `SELECT * FROM "api_keys"`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		for i, v := range vals {
			if strings.Contains(fmt.Sprint(v), secret) {
				t.Errorf("api_keys.%s holds the secret: %v", cols[i], v)
			}
		}
	}
}

// A program sends no Origin: a POST with only the bearer header is served.
func TestKeyPassesCrossSiteCheck(t *testing.T) {
	c := startShop(t, keysOn(t))
	k := makeKey(t, c, "Script", auth.RoleMember, 30)
	r := httptest.NewRequest(http.MethodPost, "http://warptest.local/api/products", strings.NewReader(`{"name":"Desk"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+k.Secret)
	c.Fork().Do(r).Want(201)
}

func TestKeyFormatRefused(t *testing.T) {
	c := startShop(t, keysOn(t))
	k := makeKey(t, c, "Script", auth.RoleMember, 30)
	secret := k.Secret[len(fmt.Sprintf("wk_%d_", k.ID)):] // the secret itself may hold _
	for _, bad := range []string{
		"", "wk_", "wk_1_", fmt.Sprintf("wk_%d_%s", k.ID, secret[:42]), fmt.Sprintf("wk_%d_%sA", k.ID, secret),
		fmt.Sprintf("WK_%d_%s", k.ID, secret), fmt.Sprintf("wk_x_%s", secret), fmt.Sprintf("wk_%d_%s", k.ID+1, secret),
		fmt.Sprintf("wk_%d_%s!", k.ID, secret[:42]), fmt.Sprintf("wk_%d_%s", k.ID, strings.Repeat("A", 43)),
		strings.Repeat("w", 101),
	} {
		wantError(t, bearer(c, bad, http.MethodGet, "/api/things", nil), 401, "unauthorized", keyNotValid)
	}
	bearer(c, k.Secret, http.MethodGet, "/api/things", nil).Want(200)
}
