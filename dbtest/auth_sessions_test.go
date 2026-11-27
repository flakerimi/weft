package dbtest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/auth/authtest"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

var cookieValue = regexp.MustCompile(`^session=([^;]+);`)

// held is the session cookie an answer set, as a copy someone kept.
func held(t *testing.T, res *warptest.Response) *http.Cookie {
	t.Helper()
	m := cookieValue.FindStringSubmatch(res.Header.Get("Set-Cookie"))
	if m == nil {
		t.Fatalf("the answer set no session cookie: %q", res.Header.Values("Set-Cookie"))
	}
	return &http.Cookie{Name: "session", Value: m[1]}
}

// with sends a request carrying only the cookie ck, from a fresh client.
func with(c *warptest.Client, ck *http.Cookie, method, path string) *warptest.Response {
	r := request(method, path, nil, "")
	r.AddCookie(ck)
	return c.Fork().Do(r)
}

// signInFrom signs c in as email from the address remote with the agent.
func signInFrom(t *testing.T, c *warptest.Client, email, password, remote, agent string) *warptest.Response {
	t.Helper()
	r := request(http.MethodPost, "/api/auth/login", map[string]string{"email": email, "password": password}, remote)
	r.Header.Set("User-Agent", agent)
	return c.Do(r).Want(200)
}

func sessionsOf(t *testing.T, c *warptest.Client) []auth.SessionInfo {
	t.Helper()
	var list []auth.SessionInfo
	c.Get("/api/auth/sessions").Want(200).JSON(&list)
	return list
}

func rows(t *testing.T, c *warptest.Client, where string, args ...any) int {
	t.Helper()
	var n int
	if err := data.For(c.App).QueryRow(ctx, `SELECT count(*) FROM "sessions" `+where, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

const signIn = "Sign in to do that."

// Review Focus 1: a cookie copied before logout is no session after.
func TestLogoutEndsTheSession(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	u := c.Fork()
	copied := held(t, login(u, ownerEmail, authtest.Password).Want(200))
	with(c, copied, http.MethodGet, "/api/auth/me").Want(200)
	res := u.Post("/api/auth/logout", nil).Want(204)
	if got := res.Header.Get("Set-Cookie"); got != "session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax" {
		t.Fatalf("logout's Set-Cookie = %q", got)
	}
	wantError(t, with(c, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, with(c, copied, http.MethodGet, "/api/things"), 401, "unauthorized", signIn)
	wantError(t, with(c, copied, http.MethodPost, "/api/auth/logout"), 401, "unauthorized", signIn)
	// The owner's other session lives.
	c.Get("/api/auth/me").Want(200)
}

func TestLogoutEverywhere(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	clients := []*warptest.Client{c, c.Fork(), c.Fork()}
	var copies []*http.Cookie
	for _, u := range clients[1:] {
		copies = append(copies, held(t, login(u, ownerEmail, authtest.Password).Want(200)))
	}
	other, _ := authtest.As(t, c, auth.RoleMember)

	wantError(t, c.Fork().Post("/api/auth/logout/everywhere", nil), 401, "unauthorized", signIn)
	res := clients[1].Post("/api/auth/logout/everywhere", nil).Want(204)
	if got := res.Header.Get("Set-Cookie"); got != "session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax" {
		t.Fatalf("logout everywhere's Set-Cookie = %q", got)
	}
	for _, u := range clients {
		wantError(t, u.Get("/api/auth/me"), 401, "unauthorized", signIn)
	}
	for _, ck := range copies {
		wantError(t, with(c, ck, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	}
	other.Get("/api/auth/me").Want(200)
	if n := rows(t, c, `WHERE "user_id" = 1`); n != 0 {
		t.Fatalf("%d sessions of the owner are kept", n)
	}
}

func TestListSessions(t *testing.T) {
	at := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	c := startClocked(t, authEnv(t), at) // session 1, from warptest's address
	phone, laptop := c.Fork(), c.Fork()
	at.now = at.now.Add(time.Hour)
	signInFrom(t, phone, ownerEmail, authtest.Password, "192.0.2.7:4000", "Phone/1.0")
	at.now = at.now.Add(time.Hour)
	agent := strings.Repeat("L", 199) + "é" + "tail"
	signInFrom(t, laptop, ownerEmail, authtest.Password, "[2001:db8::5]:4000", agent)
	// Another user's sessions are not listed.
	authtest.As(t, c, auth.RoleViewer)

	list := sessionsOf(t, phone)
	if len(list) != 3 {
		t.Fatalf("%d sessions listed, want 3: %+v", len(list), list)
	}
	if list[0].Address != "2001:db8::5" || list[0].Agent != strings.Repeat("L", 199) ||
		list[1].Address != "192.0.2.7" || list[1].Agent != "Phone/1.0" || list[2].Address != "192.0.2.1" {
		t.Fatalf("the sessions are, newest first, %+v", list)
	}
	for i, s := range list {
		if s.Current != (i == 1) {
			t.Errorf("session %d: current is %v", s.ID, s.Current)
		}
		if !s.ExpiresAt.Equal(s.CreatedAt.Add(30 * 24 * time.Hour)) {
			t.Errorf("session %d: made %v, ends %v; want 30 days apart", s.ID, s.CreatedAt, s.ExpiresAt)
		}
		if s.SeenAt.IsZero() || s.SeenAt.Before(s.CreatedAt) {
			t.Errorf("session %d: seen %v", s.ID, s.SeenAt)
		}
	}
	if !list[0].CreatedAt.Equal(at.now) {
		t.Errorf("the newest was made at %v, want %v", list[0].CreatedAt, at.now)
	}
	body := string(phone.Get("/api/auth/sessions").Want(200).Body)
	for _, hidden := range []string{"token", "user_id"} {
		if strings.Contains(body, hidden) {
			t.Errorf("the list has %q: %s", hidden, body)
		}
	}
	wantError(t, c.Fork().Get("/api/auth/sessions"), 401, "unauthorized", signIn)
}

func TestEndOneSession(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c) // session 1
	u := c.Fork()
	copied := held(t, login(u, ownerEmail, authtest.Password).Want(200)) // session 2

	res := c.Delete("/api/auth/sessions/2").Want(204)
	if got := res.Header.Get("Set-Cookie"); got != "" {
		t.Fatalf("ending another session cleared this one's cookie: %q", got)
	}
	wantError(t, with(c, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, u.Get("/api/auth/me"), 401, "unauthorized", signIn)
	c.Get("/api/auth/me").Want(200)
	wantError(t, c.Delete("/api/auth/sessions/2"), 404, "not_found", "No session of yours has id 2.")

	// Ending the current one clears the cookie too.
	res = c.Delete("/api/auth/sessions/1").Want(204)
	if got := res.Header.Get("Set-Cookie"); got != "session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax" {
		t.Fatalf("ending the current session set %q", got)
	}
	wantError(t, c.Get("/api/auth/me"), 401, "unauthorized", signIn)
}

// Review Focus 1: a session another user owns is 404, as one no one has,
// and lives.
func TestEndAnotherUsersSessionIs404(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c) // session 1
	member, _ := authtest.As(t, c, auth.RoleMember)
	wantError(t, member.Delete("/api/auth/sessions/1"), 404, "not_found", "No session of yours has id 1.")
	wantError(t, member.Delete("/api/auth/sessions/7"), 404, "not_found", "No session of yours has id 7.")
	c.Get("/api/auth/me").Want(200)
	wantError(t, c.Fork().Delete("/api/auth/sessions/1"), 401, "unauthorized", signIn)
}

// A user has at most 50 sessions: a sign-in past that ends the one seen
// longest ago.
func TestFiftySessions(t *testing.T) {
	at := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	c := startClocked(t, authEnv(t), at) // session 1, seen at 9:00
	// 49 more, each seen a minute later than the last, and one seen first of all.
	stamp := func(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000000Z") }
	insert := func(seen time.Time) {
		_, err := data.For(c.App).Exec(ctx, `INSERT INTO "sessions" ("created_at", "updated_at", "user_id", "token_hash", "expires_at", "seen_at", "address", "agent") `+
			`VALUES (?, ?, 1, ?, ?, ?, '192.0.2.9', 'Old')`, stamp(seen), stamp(seen), fmt.Sprintf("%064x", seen.UnixNano()), stamp(at.now.Add(24*time.Hour)), stamp(seen))
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(at.now.Add(-time.Hour)) // session 2, seen longest ago
	for i := range 48 {
		insert(at.now.Add(time.Duration(i+1) * time.Minute))
	}
	if n := rows(t, c, `WHERE "user_id" = 1`); n != 50 {
		t.Fatalf("%d sessions, want 50", n)
	}
	at.now = at.now.Add(time.Hour)
	u := c.Fork()
	login(u, ownerEmail, authtest.Password).Want(200)
	if n := rows(t, c, `WHERE "user_id" = 1`); n != 50 {
		t.Fatalf("%d sessions after the 51st sign-in, want 50", n)
	}
	if n := rows(t, c, `WHERE "id" = 2`); n != 0 {
		t.Fatal("the session seen longest ago is kept")
	}
	c.Get("/api/auth/me").Want(200)
	u.Get("/api/auth/me").Want(200)
	if list := sessionsOf(t, u); len(list) != 50 || !list[0].Current {
		t.Fatalf("the list has %d, the newest current: %v", len(list), len(list) > 0 && list[0].Current)
	}
}

func seenAt(t *testing.T, c *warptest.Client, id int) time.Time {
	t.Helper()
	var text string
	if err := data.For(c.App).QueryRow(ctx, `SELECT "seen_at" FROM "sessions" WHERE "id" = ?`, id).Scan(&text); err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

// seen_at is written at most once every 5 minutes, so a request is not a
// write.
func TestSeenAtWrittenEveryFiveMinutes(t *testing.T) {
	start := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	at := &clock{now: start}
	c := startClocked(t, authEnv(t), at)
	if got := seenAt(t, c, 1); !got.Equal(start) {
		t.Fatalf("seen at %v after sign-in, want %v", got, start)
	}
	for _, step := range []struct {
		after time.Duration
		want  time.Duration
	}{
		{time.Minute, 0},
		{2 * time.Minute, 0},
		{5*time.Minute - time.Nanosecond, 0},
		{5 * time.Minute, 5 * time.Minute},
		{6 * time.Minute, 5 * time.Minute},
		{9 * time.Minute, 5 * time.Minute},
		{10 * time.Minute, 10 * time.Minute},
	} {
		at.now = start.Add(step.after)
		c.Get("/api/things").Want(200)
		if got := seenAt(t, c, 1); !got.Equal(start.Add(step.want)) {
			t.Fatalf("after %v: seen at %v, want %v", step.after, got, start.Add(step.want))
		}
	}
}

// A session lasts AUTH_SESSION_DAYS from sign-in, however much it is used,
// and an expired one is removed at the next sign-in and at start.
func TestExpiredSessionIsOutAndRemoved(t *testing.T) {
	vars := authEnv(t)
	vars["AUTH_SESSION_DAYS"] = "1"
	start := time.Now().Add(-72 * time.Hour).UTC().Truncate(time.Second)
	at := &clock{now: start}
	c := startClocked(t, vars, at) // session 1
	for _, d := range []time.Duration{time.Hour, 12 * time.Hour, 24*time.Hour - time.Second} {
		at.now = start.Add(d)
		c.Get("/api/auth/me").Want(200)
	}
	at.now = start.Add(24 * time.Hour)
	wantError(t, c.Get("/api/auth/me"), 401, "unauthorized", signIn)
	if n := rows(t, c, ""); n != 1 {
		t.Fatalf("%d sessions before a sign-in, want the expired one", n)
	}
	// A sign-in removes it.
	u := c.Fork()
	login(u, ownerEmail, authtest.Password).Want(200) // session 2, ends at start+48h
	if n := rows(t, c, ""); n != 1 || rows(t, c, `WHERE "id" = 1`) != 0 {
		t.Fatalf("%d sessions after a sign-in, want the new one alone", n)
	}
	// A start, by the real clock two days on, removes it too.
	if err := c.App.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	again, _ := startAuth(t, vars)
	if n := rows(t, again, ""); n != 0 {
		t.Fatalf("%d sessions after a start, want none", n)
	}
}

func TestRemovedUserSessionsGone(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	member, u := authtest.As(t, c, auth.RoleMember)
	other := c.Fork()
	copied := held(t, login(other, u.Email, authtest.Password).Want(200))
	if n := rows(t, c, `WHERE "user_id" = ?`, u.ID); n != 2 {
		t.Fatalf("%d sessions of the member, want 2", n)
	}
	if err := auth.RemoveUser(ctx, c.App, u.ID); err != nil {
		t.Fatal(err)
	}
	if n := rows(t, c, `WHERE "user_id" = ?`, u.ID); n != 0 {
		t.Fatalf("%d sessions of a removed user are kept", n)
	}
	wantError(t, member.Get("/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, with(c, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	c.Get("/api/auth/me").Want(200)
}

// The slice's cookie, <id>.<created_at ns>.<expiry>.<mac>, signed with the
// right secret, is not read: everyone signs in once after upgrading.
func TestSliceCookieIsNotRead(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	owner := authtest.Owner(t, c)
	signed := "1." + strconv.FormatInt(owner.CreatedAt.UnixNano(), 10) + "." + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	slice := &http.Cookie{Name: "session", Value: signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))}
	wantError(t, with(c, slice, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	wantError(t, with(c, slice, http.MethodGet, "/api/things"), 401, "unauthorized", signIn)
}

// Changing AUTH_SECRET signs everyone out; changing it back lets the
// sessions in again, since they are still in the table.
func TestCookieUnderSecretChange(t *testing.T) {
	vars := authEnv(t)
	c, _ := startAuth(t, vars)
	authtest.Owner(t, c)
	copied := held(t, login(c.Fork(), ownerEmail, authtest.Password).Want(200))
	if err := c.App.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	changed := make(map[string]string, len(vars))
	for k, v := range vars {
		changed[k] = v
	}
	changed["AUTH_SECRET"] = "another secret of at least thirty-two bytes"
	other, _ := startAuth(t, changed)
	wantError(t, with(other, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	if err := other.App.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	back, _ := startAuth(t, vars)
	with(back, copied, http.MethodGet, "/api/auth/me").Want(200)
}

// A cookie signed with the secret, naming a session that exists, with a
// token that is not the session's, is no session: the row's hash decides.
func TestTokenIsComparedToTheRow(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c) // session 1
	signed := "1." + base64.RawURLEncoding.EncodeToString([]byte("thirty-two bytes, not the token!"))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	forged := &http.Cookie{Name: "session", Value: signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))}
	wantError(t, with(c, forged, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	c.Get("/api/auth/me").Want(200)
}
