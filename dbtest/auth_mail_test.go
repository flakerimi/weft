package dbtest

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/auth/authtest"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

const (
	linkGone    = "This link is not valid any more; ask for a new one."
	noSignUps   = "This app does not take sign-ups; ask for an invitation."
	tooManyMail = "Too many emails asked for; wait an hour and try again."
	linkStart   = "https://shop.example.com/password#token="
	newPassword = "a new password, for tests"
)

// withRegistration is authEnv with AUTH_REGISTRATION set, unless it is "".
func withRegistration(t *testing.T, registration string) map[string]string {
	vars := authEnv(t)
	if registration != "" {
		vars["AUTH_REGISTRATION"] = registration
	}
	return vars
}

// from sends a request from the address remote on a fresh client.
func from(c *warptest.Client, method, path string, body any, remote string) *warptest.Response {
	return c.Fork().Do(request(method, path, body, remote))
}

func tokenInfo(c *warptest.Client, token string) *warptest.Response {
	return c.Fork().Post("/api/auth/token", map[string]string{"token": token})
}

func setPassword(c *warptest.Client, token, password string) *warptest.Response {
	return c.Post("/api/auth/password", map[string]string{"token": token, "password": password})
}

func TestRegisterThenSetPassword(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	const email = "new@example.com"
	res := c.Fork().Post("/api/auth/register", map[string]string{"email": "New@Example.com"}).Want(204)
	if len(res.Body) != 0 || res.Header.Get("Set-Cookie") != "" {
		t.Fatalf("register answered %q, cookie %q", res.Body, res.Header.Get("Set-Cookie"))
	}
	msg := lastMail(t, c.App, email)
	if msg.To != email || msg.Subject != "Finish signing up to shop.example.com" {
		t.Fatalf("mailed %+v", msg)
	}
	token := mailToken(t, msg)
	var info auth.TokenInfo
	tokenInfo(c, token).Want(200).JSON(&info)
	if info != (auth.TokenInfo{Purpose: "register", Email: email}) {
		t.Fatalf("token = %+v", info)
	}
	// No user until the link is used.
	wantError(t, login(c.Fork(), email, newPassword), 401, "unauthorized", "That email and password do not match.")

	u := c.Fork()
	res = setPassword(u, token, newPassword).Want(200)
	var me auth.Me
	res.JSON(&me)
	if me.Email != email || me.Role != auth.RoleViewer || me.ID == 0 || !slices.Equal(me.Permissions, []string{"things:read"}) {
		t.Fatalf("password made %+v", me)
	}
	if !sessionCookie.MatchString(res.Header.Get("Set-Cookie")) {
		t.Fatalf("password's Set-Cookie = %q", res.Header.Get("Set-Cookie"))
	}
	u.Get("/api/auth/me").Want(200)
	login(c.Fork(), email, newPassword).Want(200)
	// The token is gone.
	wantError(t, tokenInfo(c, token), 403, "forbidden", linkGone)
	if n := rowsIn(t, c, "auth_tokens"); n != 0 {
		t.Fatalf("%d tokens kept", n)
	}
}

// rowsIn is how many rows the table has.
func rowsIn(t *testing.T, c *warptest.Client, table string) int {
	t.Helper()
	var n int
	if err := data.For(c.App).QueryRow(ctx, `SELECT count(*) FROM "`+table+`"`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// sameAnswer fails the test when a and b differ in status, body or a header
// other than Date.
func sameAnswer(t *testing.T, what string, a, b *warptest.Response) {
	t.Helper()
	if a.Status != b.Status || !bytes.Equal(a.Body, b.Body) {
		t.Fatalf("%s: %d %q and %d %q", what, a.Status, a.Body, b.Status, b.Body)
	}
	ha, hb := a.Header.Clone(), b.Header.Clone()
	ha.Del("Date")
	hb.Del("Date")
	if !maps.EqualFunc(ha, hb, slices.Equal) {
		t.Fatalf("%s: headers %v and %v", what, ha, hb)
	}
}

// Review Focus 2: register and forgot answer alike for an address with an
// account and one without.
func TestRegisterAndForgotSayNothing(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	for _, route := range []string{"/api/auth/register", "/api/auth/forgot"} {
		known := from(c, http.MethodPost, route, map[string]string{"email": ownerEmail}, "192.0.2.10:1000")
		unknown := from(c, http.MethodPost, route, map[string]string{"email": "nobody@example.com"}, "192.0.2.11:1000")
		sameAnswer(t, route, known, unknown)
		if known.Status != 204 || len(known.Body) != 0 {
			t.Fatalf("%s answered %d %q", route, known.Status, known.Body)
		}
	}
	// What was mailed: register mails both, forgot only the known address.
	var subjects []string
	for _, m := range sent(t, c.App, 3) {
		subjects = append(subjects, m.To+": "+m.Subject)
	}
	want := []string{
		"owner@example.com: You already have an account at shop.example.com",
		"nobody@example.com: Finish signing up to shop.example.com",
		"owner@example.com: Reset your password for shop.example.com",
	}
	// The mail goes after the answer, in any order.
	slices.Sort(subjects)
	slices.Sort(want)
	if !slices.Equal(subjects, want) {
		t.Fatalf("mailed %q, want %q", subjects, want)
	}
	time.Sleep(100 * time.Millisecond)
	if n := len(sent(t, c.App, 0)); n != 3 {
		t.Fatalf("%d mails, want 3: forgot mailed an address with no account", n)
	}
}

// Review I3 and m5: when mail fails, register and forgot still answer alike
// for an address with an account and one without, 204; the failure is
// logged, never answered, and leaves no token.
func TestRegisterAndForgotSayNothingWhenMailFails(t *testing.T) {
	logged := captureLogs(t)
	c := warptest.New(t, withRegistration(t, "open"), data.Module(), failMail{}, auth.Module(), things{})
	authtest.Owner(t, c)
	for _, route := range []string{"/api/auth/register", "/api/auth/forgot"} {
		known := from(c, http.MethodPost, route, map[string]string{"email": ownerEmail}, "192.0.2.10:1000")
		unknown := from(c, http.MethodPost, route, map[string]string{"email": "nobody@example.com"}, "192.0.2.11:1000")
		sameAnswer(t, route, known, unknown)
		known.Want(204)
	}
	// Three mails were asked for (register twice, forgot once), and each
	// failure is one line in the log.
	deadline := time.Now().Add(mailWait)
	for strings.Count(logged.String(), "was not sent") < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("the log has %d failures, want 3:\n%s", strings.Count(logged.String(), "was not sent"), logged)
		}
		time.Sleep(time.Millisecond)
	}
	if n := rowsIn(t, c, "auth_tokens"); n != 0 {
		t.Fatalf("%d tokens kept for mail that was not sent", n)
	}
	for _, line := range strings.Split(logged.String(), "\n") {
		if strings.Contains(line, "was not sent") && (strings.Contains(line, "#token=") || strings.Contains(line, "@")) {
			t.Fatalf("a failure names a link or an address: %s", line)
		}
	}
}

// answerTimes is the median time register or forgot takes to answer for an
// address with an account and for one without, over pairs asked in turn,
// each pair on a fresh hour of every limit. settle waits for the mail the
// pair asked for to be done.
func answerTimes(t *testing.T, c *warptest.Client, at *clock, route string, settle func()) (known, unknown time.Duration) {
	t.Helper()
	var ks, us []time.Duration
	for i := range 200 {
		at.set(at.Now().Add(time.Hour))
		emails := []string{ownerEmail, "nobody@example.com"}
		if i%2 == 1 {
			emails = []string{"nobody@example.com", ownerEmail}
		}
		for _, email := range emails {
			r := request(http.MethodPost, route, map[string]string{"email": email}, fmt.Sprintf("192.0.2.%d:1000", i%250+1))
			f := c.Fork()
			began := time.Now()
			res := f.Do(r)
			took := time.Since(began)
			if res.Status != 204 {
				t.Fatalf("%s %s: %d %s", route, email, res.Status, res.Body)
			}
			if email == ownerEmail {
				ks = append(ks, took)
			} else {
				us = append(us, took)
			}
		}
		settle()
	}
	slices.Sort(ks)
	slices.Sort(us)
	return ks[len(ks)/2], us[len(us)/2]
}

// alike fails the test when one median is more than twice the other and 50
// µs.
func alike(t *testing.T, what string, known, unknown time.Duration) {
	t.Helper()
	const slack = 50 * time.Microsecond
	if known > 2*unknown+slack || unknown > 2*known+slack {
		t.Errorf("%s: the median answer took %v for an address with an account and %v for one without", what, known, unknown)
	}
}

// Review I3: register and forgot do the same work before they answer, so
// they take alike, with mail working and with mail failing.
func TestRegisterAndForgotTakeAlike(t *testing.T) {
	at := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	c := startClocked(t, withRegistration(t, "open"), at)
	for _, route := range []string{"/api/auth/forgot", "/api/auth/register"} {
		k, u := answerTimes(t, c, at, route, func() {
			lastMail(t, c.App, ownerEmail)
			if route == "/api/auth/register" { // register mails the address without an account too
				lastMail(t, c.App, "nobody@example.com")
			}
		})
		alike(t, route, k, u)
	}

	logged := captureLogs(t)
	at = &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	f := warptest.New(t, withRegistration(t, "open"), data.Module(), failMail{}, auth.Module(), things{})
	data.For(f.App).SetClock(at.Now)
	authtest.Owner(t, f)
	for _, route := range []string{"/api/auth/forgot", "/api/auth/register"} {
		perPair := 1 // forgot mails the address with an account
		if route == "/api/auth/register" {
			perPair = 2 // register mails both
		}
		failures := strings.Count(logged.String(), "was not sent")
		k, u := answerTimes(t, f, at, route, func() {
			failures += perPair
			deadline := time.Now().Add(mailWait)
			for strings.Count(logged.String(), "was not sent") < failures {
				if time.Now().After(deadline) {
					t.Fatalf("%s: %d failures logged, want %d", route, strings.Count(logged.String(), "was not sent"), failures)
				}
				time.Sleep(100 * time.Microsecond)
			}
		})
		alike(t, route+" with mail failing", k, u)
	}
}

func TestRegisterKnownAddressMailsReset(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	c.Fork().Post("/api/auth/register", map[string]string{"email": ownerEmail}).Want(204)
	msg := lastMail(t, c.App, ownerEmail)
	if msg.Subject != "You already have an account at shop.example.com" || !strings.Contains(msg.Text, "already has an account") {
		t.Fatalf("mailed %+v", msg)
	}
	token := mailToken(t, msg)
	var info auth.TokenInfo
	tokenInfo(c, token).Want(200).JSON(&info)
	if info != (auth.TokenInfo{Purpose: "reset", Email: ownerEmail}) {
		t.Fatalf("token = %+v", info)
	}
	var me auth.Me
	setPassword(c.Fork(), token, newPassword).Want(200).JSON(&me)
	if me.ID != 1 || me.Role != auth.RoleOwner {
		t.Fatalf("the reset signed in as %+v", me)
	}
	if n := rowsIn(t, c, "users"); n != 1 {
		t.Fatalf("%d users; a reset made one", n)
	}
	login(c.Fork(), ownerEmail, newPassword).Want(200)
}

// Review Focus 1: a reset ends every session there was.
func TestForgotThenReset(t *testing.T) {
	c, _ := startAuth(t, authEnv(t)) // invite: forgot works under every setting
	authtest.Owner(t, c)
	other := c.Fork()
	copied := held(t, login(other, ownerEmail, authtest.Password).Want(200))
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": "OWNER@example.com"}).Want(204)
	msg := lastMail(t, c.App, ownerEmail)
	if msg.Subject != "Reset your password for shop.example.com" {
		t.Fatalf("mailed %+v", msg)
	}
	u := c.Fork()
	setPassword(u, mailToken(t, msg), newPassword).Want(200)
	for _, old := range []*warptest.Client{c, other} {
		wantError(t, old.Get("/api/auth/me"), 401, "unauthorized", signIn)
	}
	wantError(t, with(c, copied, http.MethodGet, "/api/auth/me"), 401, "unauthorized", signIn)
	u.Get("/api/auth/me").Want(200)
	wantError(t, login(c.Fork(), ownerEmail, authtest.Password), 401, "unauthorized", "That email and password do not match.")
	login(c.Fork(), ownerEmail, newPassword).Want(200)
}

func TestResetClearsAccountCount(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	for i := range 10 {
		from(c, http.MethodPost, "/api/auth/login", map[string]string{"email": ownerEmail, "password": "wrong"}, fmt.Sprintf("192.0.2.%d:1000", i+1)).Want(401)
	}
	wantError(t, from(c, http.MethodPost, "/api/auth/login", map[string]string{"email": ownerEmail, "password": authtest.Password}, "198.51.100.1:1000"),
		429, "rate_limited", tooManyForAccount)
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	setPassword(c.Fork(), mailToken(t, lastMail(t, c.App, ownerEmail)), newPassword).Want(200)
	from(c, http.MethodPost, "/api/auth/login", map[string]string{"email": ownerEmail, "password": newPassword}, "198.51.100.2:1000").Want(200)
}

func TestTokenOnce(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	token := mailToken(t, lastMail(t, c.App, ownerEmail))
	setPassword(c.Fork(), token, newPassword).Want(200)
	wantError(t, setPassword(c.Fork(), token, "yet another password"), 403, "forbidden", linkGone)
	wantError(t, tokenInfo(c, token), 403, "forbidden", linkGone)
	login(c.Fork(), ownerEmail, newPassword).Want(200)
	// A token no one has, and one of the wrong shape, are the same 403.
	for _, bad := range []string{newTokenText(), "short", strings.Repeat("a", 64)} {
		wantError(t, tokenInfo(c, bad), 403, "forbidden", linkGone)
		wantError(t, setPassword(c.Fork(), bad, newPassword), 403, "forbidden", linkGone)
	}
	wantField(t, tokenInfo(c, strings.Repeat("a", 65)), "token", "Use at most 64 characters.")
}

// newTokenText is a token of the right shape that no one was sent.
func newTokenText() string { return strings.Repeat("Zz", 21) + "Q" }

func TestTokenLifetimes(t *testing.T) {
	start := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	at := &clock{now: start}
	c := startClocked(t, withRegistration(t, "open"), at)
	admin, _ := authtest.As(t, c, auth.RoleAdministrator)
	ask := func(kind string, n int) string {
		email := fmt.Sprintf("%s-%d@example.com", kind, n)
		remote := fmt.Sprintf("192.0.2.%d:1000", 100+n)
		switch kind {
		case "reset":
			if _, err := auth.AddUser(ctx, c.App, email, "a password of theirs", auth.RoleViewer); err != nil {
				t.Fatal(err)
			}
			from(c, http.MethodPost, "/api/auth/forgot", map[string]string{"email": email}, remote).Want(204)
		case "register":
			from(c, http.MethodPost, "/api/auth/register", map[string]string{"email": email}, remote).Want(204)
		case "invite":
			admin.Post("/api/auth/invitations", map[string]string{"email": email, "role": "member"}).Want(201)
		}
		return mailToken(t, lastMail(t, c.App, email))
	}
	for _, tc := range []struct {
		kind         string
		inside, past time.Duration
	}{
		{"reset", 59 * time.Minute, 61 * time.Minute},
		{"register", 23 * time.Hour, 25 * time.Hour},
		{"invite", 6 * 24 * time.Hour, 8 * 24 * time.Hour},
	} {
		at.set(start)
		good, late := ask(tc.kind, 1), ask(tc.kind, 2)
		at.set(start.Add(tc.inside))
		tokenInfo(c, good).Want(200)
		tokenInfo(c, late).Want(200)
		setPassword(c.Fork(), good, newPassword).Want(200)
		at.set(start.Add(tc.past))
		wantError(t, tokenInfo(c, late), 403, "forbidden", linkGone)
		wantError(t, setPassword(c.Fork(), late, newPassword), 403, "forbidden", linkGone)
	}
}

// A token works only for its purpose.
func TestTokenPurposeIsKept(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	// A reset token, its user since removed, never makes a user.
	u, err := auth.AddUser(ctx, c.App, "leaving@example.com", "a password of theirs", auth.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": u.Email}).Want(204)
	reset := mailToken(t, lastMail(t, c.App, u.Email))
	if err := auth.RemoveUser(ctx, c.App, u.ID); err != nil {
		t.Fatal(err)
	}
	wantError(t, setPassword(c.Fork(), reset, newPassword), 403, "forbidden", linkGone)
	wantError(t, login(c.Fork(), u.Email, newPassword), 401, "unauthorized", "That email and password do not match.")

	// A register token, the address since given an account, never changes
	// its password.
	const email = "late@example.com"
	c.Fork().Post("/api/auth/register", map[string]string{"email": email}).Want(204)
	register := mailToken(t, lastMail(t, c.App, email))
	if _, err := auth.AddUser(ctx, c.App, email, "the real password", auth.RoleMember); err != nil {
		t.Fatal(err)
	}
	wantError(t, setPassword(c.Fork(), register, newPassword), 403, "forbidden", linkGone)
	login(c.Fork(), email, "the real password").Want(200)
}

func TestNewResetEndsOldOne(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	first := mailToken(t, lastMail(t, c.App, ownerEmail))
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	second := mailToken(t, lastMail(t, c.App, ownerEmail))
	if first == second {
		t.Fatal("two resets mailed one token")
	}
	wantError(t, tokenInfo(c, first), 403, "forbidden", linkGone)
	setPassword(c.Fork(), second, newPassword).Want(200)
}

// No row holds a mailed token as it is.
func TestTokensStoredAsHashes(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	c.Fork().Post("/api/auth/register", map[string]string{"email": "new@example.com"}).Want(204)
	c.Post("/api/auth/invitations", map[string]string{"email": "asked@example.com", "role": "member"}).Want(201)
	var tokens []string
	for _, m := range sent(t, c.App, 3) {
		tokens = append(tokens, mailToken(t, m))
	}
	if len(tokens) != 3 {
		t.Fatalf("%d mails", len(tokens))
	}
	for _, table := range []string{"auth_tokens", "invitations"} {
		rows, err := data.For(c.App).Query(ctx, `SELECT * FROM "`+table+`"`)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := rows.Columns()
		n := 0
		for rows.Next() {
			n++
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			for _, v := range vals {
				for _, tok := range tokens {
					if strings.Contains(fmt.Sprint(v), tok) {
						t.Errorf("%s holds a mailed token as it is: %v", table, vals)
					}
				}
			}
		}
		rows.Close()
		if n == 0 {
			t.Errorf("%s has no rows", table)
		}
	}
}

// Every link carries its token after #, which a browser never sends.
func TestLinksUseFragment(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	c.Fork().Post("/api/auth/register", map[string]string{"email": ownerEmail}).Want(204)
	c.Fork().Post("/api/auth/register", map[string]string{"email": "new@example.com"}).Want(204)
	c.Post("/api/auth/invitations", map[string]string{"email": "asked@example.com", "role": "viewer"}).Want(201)
	mails := sent(t, c.App, 4)
	if len(mails) != 4 {
		t.Fatalf("%d mails", len(mails))
	}
	for _, m := range mails {
		token := mailToken(t, m)
		link := linkStart + token
		if !strings.Contains(m.Text, link) || !strings.Contains(m.HTML, `href="`+link+`"`) {
			t.Errorf("%q does not carry %s in its text and HTML:\n%s\n%s", m.Subject, link, m.Text, m.HTML)
		}
		if strings.Contains(m.Text, "?") || strings.Contains(m.HTML, "?") {
			t.Errorf("%q has a ? in it", m.Subject)
		}
		if strings.ContainsAny(m.Subject, "\r\n") || !strings.Contains(m.Text, "ignore") {
			t.Errorf("%q: subject or text: %q", m.Subject, m.Text)
		}
	}
}

func TestInvite(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, au := authtest.As(t, c, auth.RoleAdministrator)
	const email = "asked@example.com"
	var inv auth.Invitation
	admin.Post("/api/auth/invitations", map[string]string{"email": "Asked@Example.com", "role": "member"}).Want(201).JSON(&inv)
	if inv.ID == 0 || inv.Email != email || inv.Role != "member" || inv.InvitedBy != au.ID || inv.ExpiresAt.Sub(inv.CreatedAt).Round(time.Minute) != 7*24*time.Hour {
		t.Fatalf("invite answered %+v", inv)
	}
	msg := lastMail(t, c.App, email)
	if msg.Subject != "You are invited to shop.example.com" || !strings.Contains(msg.Text, "member") {
		t.Fatalf("mailed %+v", msg)
	}
	var list data.List[auth.Invitation]
	admin.Get("/api/auth/invitations").Want(200).JSON(&list)
	if list.Total != 1 || list.Items[0] != inv {
		t.Fatalf("invitations = %+v", list)
	}
	body := string(admin.Get("/api/auth/invitations").Body)
	if strings.Contains(body, "token") {
		t.Fatalf("the list has a token: %s", body)
	}
	token := mailToken(t, msg)
	var info auth.TokenInfo
	tokenInfo(c, token).Want(200).JSON(&info)
	if info != (auth.TokenInfo{Purpose: "invite", Email: email, Role: "member"}) {
		t.Fatalf("token = %+v", info)
	}
	var me auth.Me
	m := c.Fork()
	setPassword(m, token, newPassword).Want(200).JSON(&me)
	if me.Email != email || me.Role != auth.RoleMember {
		t.Fatalf("accepting made %+v", me)
	}
	m.Get("/api/auth/me").Want(200)
	admin.Get("/api/auth/invitations").Want(200).JSON(&list)
	if list.Total != 0 {
		t.Fatalf("the invitation stays: %+v", list)
	}
	wantError(t, setPassword(c.Fork(), token, newPassword), 403, "forbidden", linkGone)
}

func TestInviteLimits(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, _ := authtest.As(t, c, auth.RoleAdministrator)
	member, _ := authtest.As(t, c, auth.RoleMember)
	wantError(t, admin.Post("/api/auth/invitations", map[string]string{"email": "boss@example.com", "role": "owner"}), 403, "forbidden", "Your account cannot give the role owner.")
	wantField(t, admin.Post("/api/auth/invitations", map[string]string{"email": "OWNER@example.com", "role": "member"}), "email", "This address already has an account.")
	wantField(t, admin.Post("/api/auth/invitations", map[string]string{"email": "x@example.com", "role": "editor"}), "role", "Choose one of: owner, administrator, member, viewer.")
	wantError(t, member.Post("/api/auth/invitations", map[string]string{"email": "x@example.com", "role": "viewer"}), 403, "forbidden", "Your account cannot do that.")
	wantError(t, member.Get("/api/auth/invitations"), 403, "forbidden", "Your account cannot do that.")
	wantError(t, c.Fork().Post("/api/auth/invitations", map[string]string{"email": "x@example.com", "role": "viewer"}), 401, "unauthorized", signIn)
	// An administrator may invite an administrator, and the owner an owner.
	admin.Post("/api/auth/invitations", map[string]string{"email": "peer@example.com", "role": "administrator"}).Want(201)
	c.Post("/api/auth/invitations", map[string]string{"email": "boss@example.com", "role": "owner"}).Want(201)
	// A second invitation for an address replaces the first.
	admin.Post("/api/auth/invitations", map[string]string{"email": "peer@example.com", "role": "viewer"}).Want(201)
	var list data.List[auth.Invitation]
	c.Get("/api/auth/invitations?q=peer").Want(200).JSON(&list)
	if list.Total != 1 || list.Items[0].Role != "viewer" {
		t.Fatalf("invitations for peer = %+v", list)
	}
	// 50 an hour per inviter: the admin has sent 2.
	for i := range 48 {
		admin.Post("/api/auth/invitations", map[string]string{"email": fmt.Sprintf("p%d@example.com", i), "role": "viewer"}).Want(201)
	}
	wantError(t, admin.Post("/api/auth/invitations", map[string]string{"email": "one-more@example.com", "role": "viewer"}), 429, "rate_limited", "Too many invitations; wait an hour and try again.")
	c.Post("/api/auth/invitations", map[string]string{"email": "one-more@example.com", "role": "viewer"}).Want(201)
}

func TestInviterDemotedVoidsInvitation(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, au := authtest.As(t, c, auth.RoleAdministrator)
	admin.Post("/api/auth/invitations", map[string]string{"email": "peer@example.com", "role": "administrator"}).Want(201)
	peer := mailToken(t, lastMail(t, c.App, "peer@example.com"))
	admin.Post("/api/auth/invitations", map[string]string{"email": "low@example.com", "role": "member"}).Want(201)
	low := mailToken(t, lastMail(t, c.App, "low@example.com"))
	tokenInfo(c, peer).Want(200)

	if _, err := data.For(c.App).Exec(ctx, `UPDATE "users" SET "role" = 'member' WHERE "id" = ?`, au.ID); err != nil {
		t.Fatal(err)
	}
	wantError(t, tokenInfo(c, peer), 403, "forbidden", linkGone)
	wantError(t, setPassword(c.Fork(), peer, newPassword), 403, "forbidden", linkGone)
	tokenInfo(c, low).Want(200) // a member may still have invited a member

	// An inviter removed takes the invitations they sent.
	if err := auth.RemoveUser(ctx, c.App, au.ID); err != nil {
		t.Fatal(err)
	}
	wantError(t, tokenInfo(c, low), 403, "forbidden", linkGone)
	if n := rowsIn(t, c, "invitations"); n != 0 {
		t.Fatalf("%d invitations of a removed user are kept", n)
	}
}

func TestRegistrationSettings(t *testing.T) {
	for _, tc := range []struct {
		setting          string
		register, invite int
		inviteMessage    string
	}{
		{"open", 204, 201, ""},
		{"invite", 403, 201, ""},
		{"", 403, 201, ""},
		{"closed", 403, 403, "This app takes no invitations; set AUTH_REGISTRATION=invite in .env."},
	} {
		c, _ := startAuth(t, withRegistration(t, tc.setting))
		authtest.Owner(t, c)
		want := tc.setting
		if want == "" {
			want = "invite"
		}
		var s auth.Settings
		c.Fork().Get("/api/auth/settings").Want(200).JSON(&s)
		if s != (auth.Settings{Registration: want}) {
			t.Errorf("AUTH_REGISTRATION=%q: settings = %+v", tc.setting, s)
		}
		res := c.Fork().Post("/api/auth/register", map[string]string{"email": "new@example.com"})
		if res.Status != tc.register {
			t.Errorf("AUTH_REGISTRATION=%q: register = %d %s", tc.setting, res.Status, res.Body)
		} else if tc.register == 403 {
			wantError(t, res, 403, "forbidden", noSignUps)
		}
		res = c.Post("/api/auth/invitations", map[string]string{"email": "asked@example.com", "role": "member"})
		if res.Status != tc.invite {
			t.Errorf("AUTH_REGISTRATION=%q: invite = %d %s", tc.setting, res.Status, res.Body)
		} else if tc.inviteMessage != "" {
			wantError(t, res, 403, "forbidden", tc.inviteMessage)
		}
		// forgot works under every setting.
		c.Fork().Post("/api/auth/forgot", map[string]string{"email": ownerEmail}).Want(204)
	}
	// Before the owner is set up, no one may sign up, so the setup link stays the owner's.
	c, _ := startAuth(t, withRegistration(t, "open"))
	wantError(t, c.Fork().Post("/api/auth/register", map[string]string{"email": "early@example.com"}), 403, "forbidden", noSignUps)
	authtest.Owner(t, c)
}

func TestMailLimits(t *testing.T) {
	c, _ := startAuth(t, withRegistration(t, "open"))
	authtest.Owner(t, c)
	// 3 an hour per address mailed, from any client.
	for i := range 3 {
		from(c, http.MethodPost, "/api/auth/forgot", map[string]string{"email": ownerEmail}, fmt.Sprintf("192.0.2.%d:1000", i+1)).Want(204)
	}
	wantError(t, from(c, http.MethodPost, "/api/auth/forgot", map[string]string{"email": ownerEmail}, "192.0.2.9:1000"), 429, "rate_limited", tooManyMail)
	wantError(t, from(c, http.MethodPost, "/api/auth/register", map[string]string{"email": "Owner@example.com"}, "192.0.2.9:1000"), 429, "rate_limited", tooManyMail)
	// 10 an hour per client, to any address.
	const client = "198.51.100.7:1000"
	for i := range 10 {
		route := "/api/auth/forgot"
		if i%2 == 1 {
			route = "/api/auth/register"
		}
		from(c, http.MethodPost, route, map[string]string{"email": fmt.Sprintf("a%d@example.com", i)}, client).Want(204)
	}
	wantError(t, from(c, http.MethodPost, "/api/auth/forgot", map[string]string{"email": "a10@example.com"}, client), 429, "rate_limited", tooManyMail)
	// Another client, to that address, is not refused.
	from(c, http.MethodPost, "/api/auth/forgot", map[string]string{"email": "a10@example.com"}, "198.51.100.8:1000").Want(204)
}

// 30 a minute per client for token and password, which take a mailed token.
func TestTokenRoutesLimited(t *testing.T) {
	at := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	c := startClocked(t, authEnv(t), at)
	const client = "198.51.100.9:1000"
	for i := range 30 {
		route := "/api/auth/token"
		if i%2 == 1 {
			route = "/api/auth/password"
		}
		body := map[string]string{"token": newTokenText()}
		if route == "/api/auth/password" {
			body["password"] = newPassword
		}
		wantError(t, from(c, http.MethodPost, route, body, client), 403, "forbidden", linkGone)
	}
	wantError(t, from(c, http.MethodPost, "/api/auth/token", map[string]string{"token": newTokenText()}, client), 429, "rate_limited", "Too many tries; wait a minute and try again.")
	at.set(at.Now().Add(time.Minute))
	wantError(t, from(c, http.MethodPost, "/api/auth/token", map[string]string{"token": newTokenText()}, client), 403, "forbidden", linkGone)
}

func TestInvitationsListAndRemove(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	for _, e := range []string{"ann@example.com", "bob@example.com", "cy@example.org"} {
		c.Post("/api/auth/invitations", map[string]string{"email": e, "role": "viewer"}).Want(201)
	}
	var list data.List[auth.Invitation]
	c.Get("/api/auth/invitations?per_page=2").Want(200).JSON(&list)
	if list.Total != 3 || len(list.Items) != 2 || list.Items[0].Email != "cy@example.org" {
		t.Fatalf("newest first, paged: %+v", list)
	}
	c.Get("/api/auth/invitations?q=example.com").Want(200).JSON(&list)
	if list.Total != 2 {
		t.Fatalf("q=example.com: %+v", list)
	}
	viewer, _ := authtest.As(t, c, auth.RoleViewer)
	wantError(t, viewer.Delete("/api/auth/invitations/1"), 403, "forbidden", "Your account cannot do that.")
	token := mailToken(t, lastMail(t, c.App, "ann@example.com"))
	c.Delete("/api/auth/invitations/1").Want(204)
	wantError(t, c.Delete("/api/auth/invitations/1"), 404, "not_found", "No invitation has id 1.")
	wantError(t, tokenInfo(c, token), 403, "forbidden", linkGone)
}

func TestMailSettingsMistakes(t *testing.T) {
	const prefix = `warp: module "auth" could not start: `
	for _, tc := range []struct{ key, value, want string }{
		{"AUTH_REGISTRATION", "maybe", `AUTH_REGISTRATION is "maybe"; set it to open, invite or closed in .env.`},
		{"AUTH_REGISTRATION", "Open", `AUTH_REGISTRATION is "Open"; set it to open, invite or closed in .env.`},
		{"API_KEYS", "yes", `API_KEYS is "yes"; set it to on or off in .env.`},
	} {
		vars := authEnv(t)
		vars[tc.key] = tc.value
		app := warp.New(warp.ConfigFrom(vars))
		app.Use(data.Module(), mailModule(), auth.Module())
		err := app.Start(context.Background())
		if err == nil {
			app.Stop(context.Background())
			t.Errorf("%s=%q started", tc.key, tc.value)
			continue
		}
		if err.Error() != prefix+tc.want {
			t.Errorf("%s=%q: %v\nwant %s%s", tc.key, tc.value, err, prefix, tc.want)
		}
	}
	for _, v := range []string{"on", "off", ""} {
		vars := authEnv(t)
		vars["API_KEYS"] = v
		c, _ := startAuth(t, vars)
		var s auth.Settings
		c.Get("/api/auth/settings").Want(200).JSON(&s)
		if s.APIKeys != (v == "on") {
			t.Errorf("API_KEYS=%q: settings = %+v", v, s)
		}
	}
}

// The plan's TestAuthNeedsMail: without mail.Module(), the sentence of Calls.
func TestAuthNeedsMail(t *testing.T) {
	t.Skip("waits on Task B1: until warp/mail is merged, the login looks for its mail module when it first sends, not in Register; see warp/auth/postman.go.")
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), auth.Module())
	want := `warp: module "auth" needs the "mail" module; list mail.Module() before it in app/modules.go.`
	if err := app.Err(); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Err = %v, want %q", err, want)
	}
}

// Review I1: a reset link dies with its user, and never acts on a new
// account made later with the same address.
func TestResetLinkDiesWithItsUser(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	const email = "jane@example.com"
	old, err := auth.AddUser(ctx, c.App, email, "jane's old password", auth.RoleAdministrator)
	if err != nil {
		t.Fatal(err)
	}
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": email}).Want(204)
	token := mailToken(t, lastMail(t, c.App, email))
	if err := auth.RemoveUser(ctx, c.App, old.ID); err != nil {
		t.Fatal(err)
	}
	if n := rowsIn(t, c, "auth_tokens"); n != 0 {
		t.Fatalf("%d reset tokens outlive their user", n)
	}
	if _, err := auth.AddUser(ctx, c.App, email, "the new jane's password", auth.RoleViewer); err != nil {
		t.Fatal(err)
	}
	wantError(t, tokenInfo(c, token), 403, "forbidden", linkGone)
	wantError(t, setPassword(c.Fork(), token, newPassword), 403, "forbidden", linkGone)
	login(c.Fork(), email, "the new jane's password").Want(200)

	// Through the route as well: remove_user takes the user's reset links.
	_, u := authtest.As(t, c, auth.RoleMember)
	c.Fork().Post("/api/auth/forgot", map[string]string{"email": u.Email}).Want(204)
	token = mailToken(t, lastMail(t, c.App, u.Email))
	c.Delete(fmt.Sprintf("/api/auth/users/%d", u.ID)).Want(204)
	if _, err := auth.AddUser(ctx, c.App, u.Email, "another password", auth.RoleViewer); err != nil {
		t.Fatal(err)
	}
	wantError(t, setPassword(c.Fork(), token, newPassword), 403, "forbidden", linkGone)
}

// Review m2: removing an invitation follows the rank rule, as every other
// users route does.
func TestRemoveInvitationByRank(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	admin, _ := authtest.As(t, c, auth.RoleAdministrator)
	var boss, peer auth.Invitation
	c.Post("/api/auth/invitations", map[string]string{"email": "boss@example.com", "role": "owner"}).Want(201).JSON(&boss)
	c.Post("/api/auth/invitations", map[string]string{"email": "peer@example.com", "role": "administrator"}).Want(201).JSON(&peer)
	wantError(t, admin.Delete(fmt.Sprintf("/api/auth/invitations/%d", boss.ID)), 403, "forbidden", "Your account cannot remove an invitation for a higher role.")
	admin.Delete(fmt.Sprintf("/api/auth/invitations/%d", peer.ID)).Want(204)
	c.Delete(fmt.Sprintf("/api/auth/invitations/%d", boss.ID)).Want(204)
}
