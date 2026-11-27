package dbtest

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/auth/authtest"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

const (
	notYourself = "You cannot change your own role or remove yourself; ask another owner or administrator."
	higherRole  = "Your account cannot change a user with a higher role."
)

func usersList(t *testing.T, c *warptest.Client, query string) data.List[auth.User] {
	t.Helper()
	var l data.List[auth.User]
	c.Get("/api/auth/users" + query).Want(200).JSON(&l)
	return l
}

func emails(l data.List[auth.User]) []string {
	var out []string
	for _, u := range l.Items {
		out = append(out, u.Email)
	}
	return out
}

func TestUsersList(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c) // owner@example.com
	for _, u := range []struct{ email, name, role string }{
		{"ann@example.com", "Ann Lee", auth.RoleMember},
		{"bob@example.org", "Bob", auth.RoleViewer},
		{"cy@example.com", "Cy Ann", auth.RoleAdministrator},
	} {
		made, err := auth.AddUser(ctx, c.App, u.email, "a password of theirs", u.role)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := data.For(c.App).Exec(ctx, `UPDATE "users" SET "name" = ? WHERE "id" = ?`, u.name, made.ID); err != nil {
			t.Fatal(err)
		}
	}
	l := usersList(t, c, "")
	if l.Total != 4 || fmt.Sprint(emails(l)) != "[cy@example.com bob@example.org ann@example.com owner@example.com]" {
		t.Fatalf("users, newest first = %v", emails(l))
	}
	if l.Items[0].Name != "Cy Ann" || l.Items[0].Role != auth.RoleAdministrator {
		t.Fatalf("the first user is %+v", l.Items[0])
	}
	for query, want := range map[string]string{
		"?sort=email":                   "[ann@example.com bob@example.org cy@example.com owner@example.com]",
		"?sort=-name":                   "[cy@example.com bob@example.org ann@example.com owner@example.com]",
		"?sort=role":                    "[cy@example.com ann@example.com owner@example.com bob@example.org]",
		"?sort=created_at&per_page=2":   "[owner@example.com ann@example.com]",
		"?q=ann":                        "[cy@example.com ann@example.com]",
		"?q=example.org":                "[bob@example.org]",
		"?filter[role]=viewer":          "[bob@example.org]",
		"?sort=email&page=2&per_page=3": "[owner@example.com]",
	} {
		if got := fmt.Sprint(emails(usersList(t, c, query))); got != want {
			t.Errorf("users%s = %s, want %s", query, got, want)
		}
	}
	body := string(c.Get("/api/auth/users").Body)
	for _, hidden := range []string{"password", "pbkdf2"} {
		if strings.Contains(body, hidden) {
			t.Fatalf("the list has %q: %s", hidden, body)
		}
	}
	// Review Focus 3: a member is 403 on every users: route.
	member, _ := authtest.As(t, c, auth.RoleMember)
	for _, r := range []*warptest.Response{
		member.Get("/api/auth/users"),
		member.Patch("/api/auth/users/2", map[string]string{"role": "viewer"}),
		member.Delete("/api/auth/users/3"),
		member.Delete("/api/auth/users/3/sessions"),
		member.Get("/api/auth/invitations"),
		member.Post("/api/auth/invitations", map[string]string{"email": "x@example.com", "role": "viewer"}),
		member.Delete("/api/auth/invitations/1"),
	} {
		wantError(t, r, 403, "forbidden", "Your account cannot do that.")
	}
	wantError(t, c.Fork().Get("/api/auth/users"), 401, "unauthorized", signIn)
}

// Review Focus 3: an administrator cannot change, remove or end the
// sessions of an owner, nor give the role owner.
func TestAdministratorCannotTouchOwner(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	owner := authtest.Owner(t, c)
	admin, _ := authtest.As(t, c, auth.RoleAdministrator)
	_, member := authtest.As(t, c, auth.RoleMember)
	path := fmt.Sprintf("/api/auth/users/%d", owner.ID)
	wantError(t, admin.Patch(path, map[string]string{"role": "member"}), 403, "forbidden", higherRole)
	wantError(t, admin.Delete(path), 403, "forbidden", higherRole)
	wantError(t, admin.Delete(path+"/sessions"), 403, "forbidden", higherRole)
	wantError(t, admin.Patch(fmt.Sprintf("/api/auth/users/%d", member.ID), map[string]string{"role": "owner"}), 403, "forbidden", higherRole)
	c.Get("/api/auth/me").Want(200)
	// Below their own role, an administrator may.
	var u auth.User
	admin.Patch(fmt.Sprintf("/api/auth/users/%d", member.ID), map[string]string{"role": "administrator"}).Want(200).JSON(&u)
	if u.ID != member.ID || u.Role != auth.RoleAdministrator {
		t.Fatalf("update_user answered %+v", u)
	}
	wantError(t, admin.Patch("/api/auth/users/99", map[string]string{"role": "viewer"}), 404, "not_found", "No user has id 99.")
	wantField(t, admin.Patch(fmt.Sprintf("/api/auth/users/%d", member.ID), map[string]string{"role": "editor"}), "role", "Choose one of: owner, administrator, member, viewer.")
	wantField(t, admin.Patch(fmt.Sprintf("/api/auth/users/%d", member.ID), map[string]string{}), "role", "This is required.")
}

func TestNoOneChangesThemselves(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	owner := authtest.Owner(t, c)
	admin, au := authtest.As(t, c, auth.RoleAdministrator)
	for _, tc := range []struct {
		c  *warptest.Client
		id int64
	}{{c, owner.ID}, {admin, au.ID}} {
		path := fmt.Sprintf("/api/auth/users/%d", tc.id)
		wantError(t, tc.c.Patch(path, map[string]string{"role": "viewer"}), 409, "conflict", notYourself)
		wantError(t, tc.c.Patch(path, map[string]string{"role": "owner"}), 409, "conflict", notYourself)
		wantError(t, tc.c.Delete(path), 409, "conflict", notYourself)
		tc.c.Get("/api/auth/me").Want(200)
	}
}

// Two owners: each can demote the other, never themselves, so one always
// remains.
func TestOwnerRemains(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	first := authtest.Owner(t, c)
	other, second := authtest.As(t, c, auth.RoleOwner)
	other.Patch(fmt.Sprintf("/api/auth/users/%d", first.ID), map[string]string{"role": "administrator"}).Want(200)
	// The first is an administrator now, and cannot touch the owner left.
	wantError(t, c.Patch(fmt.Sprintf("/api/auth/users/%d", second.ID), map[string]string{"role": "member"}), 403, "forbidden", higherRole)
	wantError(t, other.Patch(fmt.Sprintf("/api/auth/users/%d", second.ID), map[string]string{"role": "member"}), 409, "conflict", notYourself)
	wantError(t, other.Delete(fmt.Sprintf("/api/auth/users/%d", second.ID)), 409, "conflict", notYourself)
	owners := usersList(t, other, "?filter[role]=owner")
	if owners.Total != 1 || owners.Items[0].ID != second.ID {
		t.Fatalf("owners = %+v", owners)
	}
	// The owner left may remove the other.
	other.Delete(fmt.Sprintf("/api/auth/users/%d", first.ID)).Want(204)
	wantError(t, c.Get("/api/auth/me"), 401, "unauthorized", signIn)
}

func TestRemoveUserTakesEverything(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, au := authtest.As(t, c, auth.RoleAdministrator)
	copied := held(t, login(c.Fork(), au.Email, authtest.Password).Want(200))
	admin.Post("/api/auth/invitations", map[string]string{"email": "asked@example.com", "role": "member"}).Want(201)
	token := mailToken(t, lastMail(t, c.App, "asked@example.com"))
	c.Delete(fmt.Sprintf("/api/auth/users/%d", au.ID)).Want(204)
	wantError(t, admin.Get("/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, with(c, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	for table, want := range map[string]int{"sessions": 1, "invitations": 0, "users": 1} {
		if n := rowsIn(t, c, table); n != want {
			t.Errorf("%s has %d rows, want %d", table, n, want)
		}
	}
	wantError(t, tokenInfo(c, token), 403, "forbidden", linkGone)
	wantError(t, c.Delete(fmt.Sprintf("/api/auth/users/%d", au.ID)), 404, "not_found", fmt.Sprintf("No user has id %d.", au.ID))
}

func TestEndUserSessions(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, _ := authtest.As(t, c, auth.RoleAdministrator)
	member, mu := authtest.As(t, c, auth.RoleMember)
	second := c.Fork()
	login(second, mu.Email, authtest.Password).Want(200)
	admin.Delete(fmt.Sprintf("/api/auth/users/%d/sessions", mu.ID)).Want(204)
	wantError(t, member.Get("/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, second.Get("/api/auth/me"), 401, "unauthorized", signIn)
	admin.Get("/api/auth/me").Want(200)
	c.Get("/api/auth/me").Want(200)
	login(c.Fork(), mu.Email, authtest.Password).Want(200) // the account stays
}

// A role changed is the role on the user's next request.
func TestRoleChangeAtOnce(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, au := authtest.As(t, c, auth.RoleAdministrator)
	admin.Get("/api/auth/users").Want(200)
	c.Patch(fmt.Sprintf("/api/auth/users/%d", au.ID), map[string]string{"role": "member"}).Want(200)
	wantError(t, admin.Get("/api/auth/users"), 403, "forbidden", "Your account cannot do that.")
	var me auth.Me
	admin.Get("/api/auth/me").Want(200).JSON(&me)
	if me.Role != auth.RoleMember || strings.Contains(fmt.Sprint(me.Permissions), "users:") {
		t.Fatalf("me after the change = %+v", me)
	}
}

func TestChangePassword(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	other := c.Fork()
	copied := held(t, login(other, ownerEmail, authtest.Password).Want(200))
	wantError(t, c.Fork().Post("/api/auth/password/change", map[string]string{"current": authtest.Password, "password": newPassword}), 401, "unauthorized", signIn)
	wantField(t, c.Post("/api/auth/password/change", map[string]string{"current": authtest.Password, "password": "short"}), "password", "Use at least 8 characters.")

	c.Post("/api/auth/password/change", map[string]string{"current": authtest.Password, "password": newPassword}).Want(204)
	c.Get("/api/auth/me").Want(200)
	wantError(t, other.Get("/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, with(c, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, login(c.Fork(), ownerEmail, authtest.Password), 401, "unauthorized", "That email and password do not match.")
	login(c.Fork(), ownerEmail, newPassword).Want(200)

	// A wrong current password counts as a failed login: the 6th from one
	// address is 429.
	for range 5 {
		wantField(t, c.Post("/api/auth/password/change", map[string]string{"current": "a wrong guess", "password": "whatever it is"}), "current", "This is not your current password.")
	}
	wantError(t, c.Post("/api/auth/password/change", map[string]string{"current": newPassword, "password": "whatever it is"}), 429, "rate_limited", "Too many tries; wait a minute and try again.")
	c.Get("/api/auth/me").Want(200)
}

func TestUpdateMe(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	var me auth.Me
	c.Patch("/api/auth/me", map[string]string{"name": "Ada Owner"}).Want(200).JSON(&me)
	if me.Name != "Ada Owner" || me.Email != ownerEmail || me.Role != auth.RoleOwner || len(me.Permissions) == 0 {
		t.Fatalf("update_me answered %+v", me)
	}
	c.Get("/api/auth/me").Want(200).JSON(&me)
	if me.Name != "Ada Owner" {
		t.Fatalf("me = %+v", me)
	}
	// Only name: nothing else is taken.
	wantError(t, c.Patch("/api/auth/me", map[string]string{"role": "viewer"}), 400, "bad_request", `The body has the field "role", which this route does not take.`)
	wantField(t, c.Patch("/api/auth/me", map[string]string{"name": strings.Repeat("n", 121)}), "name", "Use at most 120 characters.")
	c.Patch("/api/auth/me", map[string]string{}).Want(200).JSON(&me)
	if me.Name != "Ada Owner" {
		t.Fatalf("an empty PATCH changed the name: %+v", me)
	}
	c.Patch("/api/auth/me", map[string]string{"name": ""}).Want(200).JSON(&me)
	if me.Name != "" {
		t.Fatalf("name not cleared: %+v", me)
	}
	wantError(t, c.Fork().Patch("/api/auth/me", map[string]string{"name": "x"}), 401, "unauthorized", signIn)
}

// Review I2: a sign-in with the old password, under way while the password
// changes, never leaves a session behind. The sign-in starts at leads around
// one hash's time into the change, the window the review timed; whatever the
// order, the old password's session is refused or ended.
func TestSignInRacingPasswordChange(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	start := time.Now()
	login(c.Fork(), ownerEmail, authtest.Password).Want(200)
	h := time.Since(start) // about one hash
	for i := range 16 {
		lead := h/2 + time.Duration(i)*h/10
		c, _ := startAuth(t, authEnv(t))
		authtest.Owner(t, c)
		attacker := c.Fork()
		var wg sync.WaitGroup
		var changed *warptest.Response
		wg.Go(func() {
			changed = c.Post("/api/auth/password/change", map[string]string{"current": authtest.Password, "password": newPassword})
		})
		time.Sleep(lead)
		res := attacker.Do(request(http.MethodPost, "/api/auth/login", map[string]string{"email": ownerEmail, "password": authtest.Password}, "198.51.100.9:1000"))
		wg.Wait()
		changed.Want(204)
		if res.Status == 200 && attacker.Get("/api/auth/me").Status != 401 {
			t.Fatalf("lead %v: a sign-in with the old password kept its session after the change", lead)
		}
		if res.Status != 200 && res.Status != 401 {
			t.Fatalf("lead %v: the sign-in answered %d %s", lead, res.Status, res.Body)
		}
	}
}
