package dbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
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
	ownerEmail = "owner@example.com"
	secret     = "a secret of at least thirty-two bytes, for tests"
)

// things is a module with one route guarded by auth.Can, which answers the
// signed-in user's email.
type things struct{}

func (things) Name() string { return "things" }

func (things) Register(app *warp.App) error {
	warp.Get(app, "/things", func(ctx context.Context, _ warp.None) (map[string]string, error) {
		u, ok := auth.UserOf(ctx)
		if !ok {
			return nil, errors.New("the guard let a request through with no user")
		}
		return map[string]string{"email": u.Email, "role": u.Role}, nil
	}, auth.Can("things:read"))
	return nil
}

func authEnv(t *testing.T) map[string]string {
	return map[string]string{
		"DB_DRIVER":        "sqlite",
		"DB_URL":           "file:" + dbFile(t),
		"AUTH_SECRET":      secret,
		"AUTH_OWNER_EMAIL": "  Owner@Example.com ",
		"APP_URL":          "https://shop.example.com",
	}
}

// startAuth starts an app with data, auth and things over vars and returns
// its client and what the app logged while starting.
func startAuth(t *testing.T, vars map[string]string) (*warptest.Client, string) {
	t.Helper()
	var logged bytes.Buffer
	out, flags := log.Writer(), log.Flags()
	log.SetOutput(&logged)
	log.SetFlags(0)
	c := warptest.New(t, vars, data.Module(), auth.Module(), things{})
	log.SetOutput(out)
	log.SetFlags(flags)
	return c, logged.String()
}

var setupLine = regexp.MustCompile(`^Set up the owner at https://shop\.example\.com/setup\?token=([A-Za-z0-9_-]{43})\n$`)

// request is a same-site request from the address remote, with a JSON body
// when body is not nil.
func request(method, path string, body any, remote string) *http.Request {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	r := httptest.NewRequest(method, "http://warptest.local"+path, bytes.NewReader(b))
	r.Header.Set("Origin", "http://warptest.local")
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if remote != "" {
		r.RemoteAddr = remote
	}
	return r
}

func login(c *warptest.Client, email, password string) *warptest.Response {
	return c.Post("/api/auth/login", map[string]string{"email": email, "password": password})
}

func wantError(t *testing.T, res *warptest.Response, status int, code, message string) {
	t.Helper()
	e := res.Want(status).Error()
	if e.Code != code || e.Message != message {
		t.Fatalf("answer = %s %q, want %s %q", e.Code, e.Message, code, message)
	}
}

var sessionCookie = regexp.MustCompile(`^session=1\.(\d+)\.[A-Za-z0-9_-]{43}; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax$`)

func TestSetupThenLogin(t *testing.T) {
	c, logged := startAuth(t, authEnv(t))
	m := setupLine.FindStringSubmatch(logged)
	if m == nil {
		t.Fatalf("the app logged %q, want the setup link alone", logged)
	}
	token := m[1]
	if got := auth.SetupToken(c.App); got != token {
		t.Fatalf("SetupToken = %q, want the logged %q", got, token)
	}

	// No user yet, so no login.
	wantError(t, login(c.Fork(), ownerEmail, authtest.Password), 401, "unauthorized", "That email and password do not match.")

	const notValid = "This setup link is not valid; restart the app to get a new one."
	wantError(t, c.Post("/api/auth/setup", map[string]string{"token": token[:42] + "x", "password": authtest.Password}), 403, "forbidden", notValid)
	wantError(t, c.Post("/api/auth/setup", map[string]string{"token": "", "password": authtest.Password}), 422, "validation", "1 field needs attention.")
	wantError(t, c.Post("/api/auth/setup", map[string]string{"token": token, "password": "short"}), 422, "validation", "1 field needs attention.")

	res := c.Post("/api/auth/setup", map[string]string{"token": token, "password": authtest.Password}).Want(201)
	if bytes.Contains(res.Body, []byte("password")) || bytes.Contains(res.Body, []byte("pbkdf2")) {
		t.Fatalf("setup answered the hash: %s", res.Body)
	}
	var owner auth.User
	res.JSON(&owner)
	if owner.ID != 1 || owner.Email != ownerEmail || owner.Role != "owner" || owner.CreatedAt.IsZero() {
		t.Fatalf("setup made %+v", owner)
	}
	cookies := res.Header.Values("Set-Cookie")
	if len(cookies) != 1 {
		t.Fatalf("setup set %d cookies: %q", len(cookies), cookies)
	}
	cm := sessionCookie.FindStringSubmatch(cookies[0])
	if cm == nil {
		t.Fatalf("Set-Cookie = %q", cookies[0])
	}
	expiry, _ := strconv.ParseInt(cm[1], 10, 64)
	if d := time.Until(time.Unix(expiry, 0)); d < 30*24*time.Hour-time.Minute || d > 30*24*time.Hour+time.Minute {
		t.Fatalf("the session ends in %v, want 30 days", d)
	}

	var me auth.User
	c.Get("/api/auth/me").Want(200).JSON(&me)
	if me != owner {
		t.Fatalf("me = %+v, want %+v", me, owner)
	}

	// The token worked once.
	wantError(t, c.Post("/api/auth/setup", map[string]string{"token": token, "password": authtest.Password}), 403, "forbidden", notValid)
	if got := auth.SetupToken(c.App); got != "" {
		t.Fatalf("SetupToken after setup = %q, want empty", got)
	}

	// The owner signs in, the email compared lower-cased and trimmed.
	other := c.Fork()
	res = login(other, "  OWNER@example.COM", authtest.Password).Want(200)
	if !sessionCookie.MatchString(res.Header.Get("Set-Cookie")) {
		t.Fatalf("login's Set-Cookie = %q", res.Header.Get("Set-Cookie"))
	}
	var signedIn auth.User
	res.JSON(&signedIn)
	if signedIn != owner {
		t.Fatalf("login answered %+v, want %+v", signedIn, owner)
	}
	other.Get("/api/auth/me").Want(200)
	other.Get("/api/things").Want(200)
}

func TestSetupNotOfferedOnceAUserExists(t *testing.T) {
	vars := authEnv(t)
	c, _ := startAuth(t, vars)
	authtest.Owner(t, c)
	c.App.Stop(context.Background())

	again, logged := startAuth(t, vars)
	if logged != "" {
		t.Fatalf("a restart with an owner logged %q", logged)
	}
	if got := auth.SetupToken(again.App); got != "" {
		t.Fatalf("SetupToken = %q, want empty", got)
	}
	wantError(t, again.Post("/api/auth/setup", map[string]string{"token": "x", "password": authtest.Password}), 403, "forbidden", "This setup link is not valid; restart the app to get a new one.")
	login(again, ownerEmail, authtest.Password).Want(200)
}

// Two processes over one database, each started while no user existed, each
// with a token of its own and another owner email: once one has made its
// owner, the other's token is refused, though its email is not taken.
func TestSetupRefusedOnceAnotherMadeAUser(t *testing.T) {
	vars := authEnv(t)
	first, _ := startAuth(t, vars)
	other := maps.Clone(vars)
	other["AUTH_OWNER_EMAIL"] = "someone.else@example.com"
	second, _ := startAuth(t, other)
	token := auth.SetupToken(first.App)
	if token == "" || auth.SetupToken(second.App) == "" {
		t.Fatal("both apps should have a setup token")
	}
	authtest.Owner(t, second)
	wantError(t, first.Post("/api/auth/setup", map[string]string{"token": token, "password": authtest.Password}), 403, "forbidden", "This setup link is not valid; restart the app to get a new one.")
	wantError(t, login(first, ownerEmail, authtest.Password), 401, "unauthorized", "That email and password do not match.")
}

func TestSetupRaceMakesOneOwner(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	token := auth.SetupToken(c.App)
	const n = 8
	statuses := make(chan int, n)
	for range n {
		go func() {
			r := request(http.MethodPost, "/api/auth/setup", map[string]string{"token": token, "password": authtest.Password}, "")
			w := httptest.NewRecorder()
			c.App.Handler().ServeHTTP(w, r)
			statuses <- w.Code
		}()
	}
	made := 0
	for range n {
		switch s := <-statuses; s {
		case 201:
			made++
		case 403:
		default:
			t.Errorf("a racing setup answered %d", s)
		}
	}
	if made != 1 {
		t.Fatalf("%d setups succeeded, want 1", made)
	}
}

func TestLoginWrongIsTheSameAnswer(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)

	timed := func(email string) (*warptest.Response, time.Duration) {
		start := time.Now()
		res := login(c.Fork(), email, "not the password")
		return res, time.Since(start)
	}
	wrong, wrongTook := timed(ownerEmail)
	unknown, unknownTook := timed("nobody@example.com")
	wantError(t, wrong, 401, "unauthorized", "That email and password do not match.")
	if unknown.Status != wrong.Status || !bytes.Equal(unknown.Body, wrong.Body) {
		t.Fatalf("an unknown email answered %d %s, a wrong password %d %s", unknown.Status, unknown.Body, wrong.Status, wrong.Body)
	}
	if !maps.EqualFunc(unknown.Header, wrong.Header, func(a, b []string) bool { return strings.Join(a, "\n") == strings.Join(b, "\n") }) {
		t.Fatalf("headers differ: %v and %v", unknown.Header, wrong.Header)
	}
	if wrong.Header.Get("Set-Cookie") != "" {
		t.Fatalf("a wrong password set a cookie: %q", wrong.Header.Get("Set-Cookie"))
	}
	// Both hashed: a hash at 600,000 iterations takes tens of milliseconds.
	if wrongTook < 20*time.Millisecond || unknownTook < 20*time.Millisecond {
		t.Fatalf("a wrong password took %v and an unknown email %v; both must hash", wrongTook, unknownTook)
	}
}

func TestLoginIsLimited(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	try := func(email, password, remote string) *warptest.Response {
		return c.Fork().Do(request(http.MethodPost, "/api/auth/login", map[string]string{"email": email, "password": password}, remote))
	}
	const a, b = "192.0.2.1:1000", "192.0.2.2:1000"
	for i := range 5 {
		if res := try(ownerEmail, "wrong", a); res.Status != 401 {
			t.Fatalf("failure %d answered %d", i+1, res.Status)
		}
	}
	const tooMany = "Too many tries; wait a minute and try again."
	wantError(t, try(ownerEmail, authtest.Password, a), 429, "rate_limited", tooMany)
	wantError(t, try(" Owner@Example.com", authtest.Password, "192.0.2.1:2000"), 429, "rate_limited", tooMany)
	// Another email from that address, and that email from another, are not.
	wantError(t, try("other@example.com", "wrong", a), 401, "unauthorized", "That email and password do not match.")
	try(ownerEmail, authtest.Password, b).Want(200)

	// A blocked try costs no hashing.
	start := time.Now()
	try(ownerEmail, "wrong", a).Want(429)
	if took := time.Since(start); took > 30*time.Millisecond {
		t.Errorf("a refused try took %v, as if it hashed", took)
	}
	// A success clears the count.
	for range 4 {
		try(ownerEmail, "wrong", b).Want(401)
	}
	try(ownerEmail, authtest.Password, b).Want(200)
	for range 4 {
		try(ownerEmail, "wrong", b).Want(401)
	}
	try(ownerEmail, authtest.Password, b).Want(200)
}

func TestGuard(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	const signIn = "Sign in to do that."
	wantError(t, c.Get("/api/things"), 401, "unauthorized", signIn)
	wantError(t, c.Get("/api/auth/me"), 401, "unauthorized", signIn)
	for _, bad := range []string{"garbage", "1.2.3", "1." + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10) + ".AAAA", strings.Repeat("1", 300)} {
		r := request(http.MethodGet, "/api/things", nil, "")
		r.AddCookie(&http.Cookie{Name: "session", Value: bad})
		wantError(t, c.Do(r), 401, "unauthorized", signIn)
	}

	owner := authtest.Owner(t, c)
	var got map[string]string
	c.Get("/api/things").Want(200).JSON(&got)
	if got["email"] != owner.Email || got["role"] != "owner" {
		t.Fatalf("the guarded route saw %v", got)
	}

	clerk, err := auth.AddUser(context.Background(), c.App, " Clerk@Example.com", "clerk password", "clerk")
	if err != nil {
		t.Fatal(err)
	}
	if clerk.Email != "clerk@example.com" || clerk.Role != "clerk" || clerk.ID == 0 {
		t.Fatalf("AddUser made %+v", clerk)
	}
	cc := c.Fork()
	login(cc, "clerk@example.com", "clerk password").Want(200)
	wantError(t, cc.Get("/api/things"), 403, "forbidden", "Your account cannot do that.")
	var me auth.User
	cc.Get("/api/auth/me").Want(200).JSON(&me)
	if me != clerk {
		t.Fatalf("me = %+v, want %+v", me, clerk)
	}

	for _, r := range c.App.Routes() {
		if r.Path == "/api/things" && r.Permission != "things:read" {
			t.Fatalf("the guarded route records %q", r.Permission)
		}
	}
}

func TestAddUserMistakes(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	for _, tc := range []struct{ email, password, role, field, message string }{
		{" OWNER@example.com", "a good password", "owner", "email", "This is already taken."},
		{"not an address", "a good password", "clerk", "email", "Enter an email address."},
		{"", "a good password", "clerk", "email", "This is required."},
		{"a@example.com", "short", "clerk", "password", "Use at least 8 characters."},
		{"a@example.com", strings.Repeat("x", 201), "clerk", "password", "Use at most 200 characters."},
		{"a@example.com", "a good password", "", "role", "This is required."},
	} {
		_, err := auth.AddUser(context.Background(), c.App, tc.email, tc.password, tc.role)
		var e *warp.Error
		if !errors.As(err, &e) || e.Status != 422 || e.Fields[tc.field] != tc.message {
			t.Errorf("AddUser(%q, %q, %q) = %v, want 422 with %s: %q", tc.email, tc.password, tc.role, err, tc.field, tc.message)
		} else if len(e.Fields) != 1 {
			t.Errorf("AddUser(%q, %q, %q) names %v", tc.email, tc.password, tc.role, e.Fields)
		}
	}
	if _, err := auth.AddUser(context.Background(), c.App, "a@example.com", strings.Repeat("x", 200), "clerk"); err != nil {
		t.Errorf("a 200-character password: %v", err)
	}
}

func TestRemovedUserIsOut(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	second, err := auth.AddUser(context.Background(), c.App, "second@example.com", "second password", "owner")
	if err != nil {
		t.Fatal(err)
	}
	sc := c.Fork()
	login(sc, second.Email, "second password").Want(200)
	sc.Get("/api/things").Want(200)

	// Removing users is not auth's yet, so the test removes the row itself.
	if _, err := data.For(c.App).SQL().Exec(`DELETE FROM "users" WHERE "id" = ?`, second.ID); err != nil {
		t.Fatal(err)
	}
	wantError(t, sc.Get("/api/things"), 401, "unauthorized", "Sign in to do that.")
	wantError(t, sc.Get("/api/auth/me"), 401, "unauthorized", "Sign in to do that.")
	c.Get("/api/things").Want(200)
}

func TestLogout(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	wantError(t, c.Fork().Post("/api/auth/logout", nil), 401, "unauthorized", "Sign in to do that.")

	res := c.Post("/api/auth/logout", nil).Want(204)
	if got := res.Header.Values("Set-Cookie"); len(got) != 1 || got[0] != "session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax" {
		t.Fatalf("logout's Set-Cookie = %q", got)
	}
	wantError(t, c.Get("/api/auth/me"), 401, "unauthorized", "Sign in to do that.")
	wantError(t, c.Get("/api/things"), 401, "unauthorized", "Sign in to do that.")
}

func TestCrossSiteLogoutRefused(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	authtest.Owner(t, c)
	r := httptest.NewRequest(http.MethodPost, "http://warptest.local/api/auth/logout", nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	r.Header.Set("Origin", "https://evil.example")
	res := c.Do(r)
	wantError(t, res, 403, "forbidden", "This request came from another site.")
	if got := res.Header.Get("Set-Cookie"); got != "" {
		t.Fatalf("the refused logout set %q", got)
	}
	c.Get("/api/auth/me").Want(200)
}

func TestSecureCookieBehindProxy(t *testing.T) {
	for _, tc := range []struct {
		proxy  string
		proto  string
		secure bool
	}{
		{"on", "https", true},
		{"on", "http", false},
		{"on", "", false},
		{"", "https", false},
	} {
		vars := authEnv(t)
		vars["TRUST_PROXY"] = tc.proxy
		c, _ := startAuth(t, vars)
		authtest.Owner(t, c)
		r := request(http.MethodPost, "/api/auth/login", map[string]string{"email": ownerEmail, "password": authtest.Password}, "")
		if tc.proto != "" {
			r.Header.Set("X-Forwarded-Proto", tc.proto)
		}
		ending := "; HttpOnly; SameSite=Lax"
		if tc.secure {
			ending = "; HttpOnly; Secure; SameSite=Lax"
		}
		cookie := c.Fork().Do(r).Want(200).Header.Get("Set-Cookie")
		if !strings.HasSuffix(cookie, ending) {
			t.Errorf("TRUST_PROXY=%q, X-Forwarded-Proto %q: Set-Cookie %q, want it to end %q", tc.proxy, tc.proto, cookie, ending)
		}
		r = request(http.MethodPost, "/api/auth/logout", nil, "")
		if tc.proto != "" {
			r.Header.Set("X-Forwarded-Proto", tc.proto)
		}
		cleared := c.Do(r).Want(204).Header.Get("Set-Cookie")
		if want := "session=; Path=/; Max-Age=0" + ending; cleared != want {
			t.Errorf("TRUST_PROXY=%q, X-Forwarded-Proto %q: logout's Set-Cookie %q, want %q", tc.proxy, tc.proto, cleared, want)
		}
	}
}

func TestEnvironmentMistakes(t *testing.T) {
	// Registering reads nothing, so an app with an empty .env describes itself.
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), auth.Module())
	if err := app.Err(); err != nil {
		t.Fatalf("an empty .env: %v", err)
	}
	if _, err := app.Describe(); err != nil {
		t.Fatalf("Describe with an empty .env: %v", err)
	}

	const prefix = `warp: module "auth" could not start: `
	for _, tc := range []struct {
		key, value, want string
	}{
		{"AUTH_SECRET", "", "AUTH_SECRET is 0 bytes; set it to 32 or more random ones in .env, for example from openssl rand -base64 48."},
		{"AUTH_SECRET", "12345678", "AUTH_SECRET is 8 bytes; set it to 32 or more random ones in .env, for example from openssl rand -base64 48."},
		{"AUTH_SECRET", strings.Repeat("s", 31), "AUTH_SECRET is 31 bytes; set it to 32 or more random ones in .env, for example from openssl rand -base64 48."},
		{"AUTH_SESSION_DAYS", "0", `AUTH_SESSION_DAYS is "0"; set it to a whole number of days from 1 to 365 in .env.`},
		{"AUTH_SESSION_DAYS", "366", `AUTH_SESSION_DAYS is "366"; set it to a whole number of days from 1 to 365 in .env.`},
		{"AUTH_SESSION_DAYS", "a week", `AUTH_SESSION_DAYS is "a week"; set it to a whole number of days from 1 to 365 in .env.`},
		{"AUTH_OWNER_EMAIL", "", "AUTH_OWNER_EMAIL is not set; set it to the address of the person who will own this app."},
		{"AUTH_OWNER_EMAIL", "the owner", `AUTH_OWNER_EMAIL is "the owner"; set it to an email address, such as owner@example.com.`},
	} {
		vars := authEnv(t)
		vars[tc.key] = tc.value
		app := warp.New(warp.ConfigFrom(vars))
		app.Use(data.Module(), auth.Module())
		if err := app.Err(); err != nil {
			t.Fatalf("%s=%q is a registration mistake: %v", tc.key, tc.value, err)
		}
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

	for _, days := range []string{"1", "365"} {
		vars := authEnv(t)
		vars["AUTH_SESSION_DAYS"] = days
		c, _ := startAuth(t, vars)
		authtest.Owner(t, c)
		cookie := login(c.Fork(), ownerEmail, authtest.Password).Want(200).Header.Get("Set-Cookie")
		n, _ := strconv.Atoi(days)
		if !strings.Contains(cookie, "; Max-Age="+strconv.Itoa(n*86400)+";") {
			t.Errorf("AUTH_SESSION_DAYS=%s: Set-Cookie %q", days, cookie)
		}
	}
}

func TestOwnerHelper(t *testing.T) {
	c, _ := startAuth(t, authEnv(t))
	owner := authtest.Owner(t, c)
	if owner.ID == 0 || owner.Email != ownerEmail || owner.Role != "owner" {
		t.Fatalf("Owner = %+v", owner)
	}
	var me auth.User
	c.Get("/api/auth/me").Want(200).JSON(&me)
	if me != owner {
		t.Fatalf("the client is signed in as %+v, want %+v", me, owner)
	}
	c.Get("/api/things").Want(200)
	login(c.Fork(), ownerEmail, authtest.Password).Want(200)
}

func TestDescribeHidesTheHash(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), auth.Module())
	d, err := app.Describe()
	if err != nil {
		t.Fatal(err)
	}
	var user *warp.TypeInfo
	for i, ty := range d.Types {
		if ty.Name == "User" {
			user = &d.Types[i]
		}
	}
	if user == nil {
		t.Fatalf("Describe lists no User: %+v", d.Types)
	}
	var names []string
	for _, f := range user.Fields {
		names = append(names, f.Name)
	}
	if got := strings.Join(names, ","); got != "id,created_at,updated_at,email,role" {
		t.Fatalf("User has the fields %s", got)
	}
	all, _ := json.Marshal(d)
	for _, hidden := range []string{"password_hash", "userRow"} {
		if bytes.Contains(all, []byte(hidden)) {
			t.Errorf("the description has %q: %s", hidden, all)
		}
	}
	routes := map[string]string{}
	for _, r := range d.Routes {
		if r.Module == "auth" {
			routes[r.Name] = r.Method + " " + r.Path + " " + strconv.Itoa(r.Status)
		}
	}
	want := map[string]string{
		"login":  "POST /api/auth/login 200",
		"logout": "POST /api/auth/logout 204",
		"me":     "GET /api/auth/me 200",
		"setup":  "POST /api/auth/setup 201",
	}
	if !maps.Equal(routes, want) {
		t.Fatalf("auth's routes are %v, want %v", routes, want)
	}
}
