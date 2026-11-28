package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
)

var (
	testSecret = []byte("a secret of thirty-two bytes, ok")
	testNow    = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	testMade   = time.Date(2026, 11, 20, 9, 30, 15, 123456789, time.UTC)
)

// testToken is a session token as signIn makes one: 43 base64url characters.
var testToken = base64.RawURLEncoding.EncodeToString([]byte("thirty-two bytes of a test token"))

// signedValue is id.token with the mac over it under testSecret.
func signedValue(id, token string) string {
	m := hmac.New(sha256.New, testSecret)
	m.Write([]byte(id + "." + token))
	return id + "." + token + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func TestCookieValueRoundTrip(t *testing.T) {
	if len(testToken) != 43 {
		t.Fatalf("the test token is %d characters", len(testToken))
	}
	v := sessionValue(testSecret, 7, testToken)
	if want := signedValue("7", testToken); v != want {
		t.Fatalf("value = %q, want 7.<token>.<HMAC-SHA256 over 7.<token> in base64url>, %q", v, want)
	}
	id, token, ok := readSession(testSecret, v)
	if !ok || id != 7 || token != testToken {
		t.Fatalf("readSession = %d, %q, %v; want 7 and the token", id, token, ok)
	}
	// A token is 32 random bytes, so two are never alike.
	a, b := newToken(), newToken()
	if len(a) != 43 || a == b || strings.ContainsAny(a, "+/=") {
		t.Fatalf("newToken = %q and %q", a, b)
	}
}

func TestCookieValueRefusesTamperedShortAndLong(t *testing.T) {
	v := sessionValue(testSecret, 7, testToken)
	parts := strings.Split(v, ".")
	mac := parts[2]
	other := base64.RawURLEncoding.EncodeToString([]byte("thirty-two other bytes, a token!"))
	made := strconv.FormatInt(testMade.UnixNano(), 10)
	exp := strconv.FormatInt(testNow.Add(time.Hour).Unix(), 10)
	slice := func(fields ...string) string { // the slice's form, signed right
		m := hmac.New(sha256.New, testSecret)
		m.Write([]byte(strings.Join(fields, ".")))
		return strings.Join(fields, ".") + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	}
	cases := map[string]string{
		"another id":                 "8." + testToken + "." + mac,
		"another token":              "7." + other + "." + mac,
		"another mac":                "7." + testToken + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		"a cut mac":                  "7." + testToken + "." + mac[:42],
		"a long mac":                 "7." + testToken + "." + mac + "A",
		"a mac that is not b64":      "7." + testToken + "." + strings.Repeat("!", 43),
		"no mac":                     "7." + testToken + ".",
		"a cut token, signed":        signedValue("7", testToken[:42]),
		"a long token, signed":       signedValue("7", testToken+"A"),
		"a token not b64, signed":    signedValue("7", strings.Repeat("+", 43)),
		"a padded token, signed":     signedValue("7", testToken[:42]+"="),
		"two parts":                  "7." + testToken,
		"four parts":                 v + ".x",
		"the slice's four parts":     slice("7", made, exp),
		"empty":                      "",
		"a signed sign":              signedValue("+7", testToken),
		"a signed negative id":       signedValue("-7", testToken),
		"a signed zero id":           signedValue("0", testToken),
		"a signed id of letters":     signedValue("seven", testToken),
		"a signed empty id":          signedValue("", testToken),
		"a signed huge id":           signedValue("99999999999999999999", testToken),
		"another secret's":           sessionValue([]byte("another secret of thirty-two b!!"), 7, testToken),
		"a signed value with a dot":  signedValue("7", testToken+".5"),
		"a signed value with spaces": signedValue(" 7", testToken),
	}
	for name, value := range cases {
		if id, token, ok := readSession(testSecret, value); ok {
			t.Errorf("%s (%q) is the session %d, %q", name, value, id, token)
		}
	}
	// The limit is on the value as sent: 256 bytes pass to the mac check,
	// 257 do not.
	pad := maxSession - len(signedValue("7", testToken))
	at := signedValue(strings.Repeat("0", pad)+"7", testToken)
	over := signedValue(strings.Repeat("0", pad+1)+"7", testToken)
	if len(at) != maxSession || len(over) != maxSession+1 {
		t.Fatalf("test values are %d and %d bytes, want %d and %d", len(at), len(over), maxSession, maxSession+1)
	}
	if id, _, ok := readSession(testSecret, at); !ok || id != 7 {
		t.Errorf("a signed value of exactly %d bytes: %d, %v; want 7, true", maxSession, id, ok)
	}
	if _, _, ok := readSession(testSecret, over); ok {
		t.Errorf("a signed value of %d bytes is read", maxSession+1)
	}
}

// A session's token is kept as its SHA-256, and compared in constant time.
func TestTokenHashOfSession(t *testing.T) {
	sum := sha256.Sum256([]byte(testToken))
	if got := tokenHash(testToken); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("tokenHash = %q", got)
	}
	if !hashIs(testToken, tokenHash(testToken)) || hashIs(testToken+"x", tokenHash(testToken)) || hashIs(testToken, "") {
		t.Fatal("hashIs does not compare a token to its hash")
	}
}

// An agent is kept to 200 bytes, cut where a character ends.
func TestAgentIsCut(t *testing.T) {
	if got := cutAgent(strings.Repeat("a", 300)); got != strings.Repeat("a", 200) {
		t.Fatalf("cutAgent of 300 bytes = %d bytes", len(got))
	}
	long := strings.Repeat("a", 199) + "é" // é is two bytes, across the cut
	if got := cutAgent(long); got != strings.Repeat("a", 199) {
		t.Fatalf("cutAgent split a character: %q", got[190:])
	}
	if got := cutAgent("Mozilla/5.0"); got != "Mozilla/5.0" {
		t.Fatalf("cutAgent of a short agent = %q", got)
	}
}

func TestCookieName(t *testing.T) {
	if c := cookie(true, "v", 60); c.Name != "__Host-session" || !c.Secure || c.Path != "/" || c.Domain != "" {
		t.Errorf("the HTTPS cookie is %+v, want __Host-session, Secure, Path=/, no Domain", c)
	}
	if c := cookie(false, "v", 60); c.Name != "session" || c.Secure || c.Path != "/" {
		t.Errorf("the HTTP cookie is %+v, want session, not Secure, Path=/", c)
	}
}

// A route guarded by Can in an app with no auth module is a registration
// mistake, not a 500 on every request.
func TestCanWithoutTheModule(t *testing.T) {
	app := warp.New(warp.Config{})
	warp.Get(app, "/x", func(context.Context, warp.None) (warp.None, error) { return warp.None{}, nil }, Can("x:read"))
	want := `warp: route "GET /x" needs the "auth" module; list auth.Module() in app/modules.go.`
	if err := app.Err(); err == nil || err.Error() != want {
		t.Fatalf("Err = %v, want %s", err, want)
	}
}

// A key shorter than the 32 bytes Start requires, an empty one above all,
// signs nothing that readSession takes, however Start was skipped.
func TestShortKeyReadsNoSession(t *testing.T) {
	for _, secret := range [][]byte{nil, {}, []byte("short"), testSecret[:31]} {
		if _, _, ok := readSession(secret, sessionValue(secret, 1, testToken)); ok {
			t.Errorf("a key of %d bytes read its own session", len(secret))
		}
	}
	if _, _, ok := readSession(testSecret, sessionValue(testSecret, 1, testToken)); !ok {
		t.Error("a key of 32 bytes read no session")
	}
}

// appStore is a module of an app's own whose Register runs fn.
type appStore struct{ fn func(app *warp.App) }

func (appStore) Name() string                   { return "notes" }
func (m appStore) Register(app *warp.App) error { m.fn(app); return nil }

// The login answers the type User: an app's own type of that name is told
// which is the login's and to rename its own. (dbtest holds the table's half.)
func TestTheLoginsNamesAreReserved(t *testing.T) {
	type User struct {
		Name string `json:"name"`
	}
	app := warp.New(warp.Config{})
	app.Use(data.Module(), mail.Module(), Module(), appStore{func(app *warp.App) {
		warp.Get(app, "/people", func(context.Context, warp.None) (User, error) { return User{}, nil })
	}})
	_, err := app.Describe()
	if err == nil || !strings.Contains(err.Error(), "the login's") || !strings.Contains(err.Error(), "rename yours") {
		t.Errorf("Describe = %v, want it to say which User is the login's", err)
	}
}
