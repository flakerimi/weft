package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	testSecret = []byte("a secret of thirty-two bytes, ok")
	testNow    = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	testMade   = time.Date(2026, 11, 20, 9, 30, 15, 123456789, time.UTC)
)

func TestSessionRoundTrip(t *testing.T) {
	expires := testNow.Add(30 * 24 * time.Hour)
	v := sessionValue(testSecret, 7, testMade, expires)
	parts := strings.Split(v, ".")
	made := strconv.FormatInt(testMade.UnixNano(), 10)
	if len(parts) != 4 || parts[0] != "7" || parts[1] != made || parts[2] != strconv.FormatInt(expires.Unix(), 10) {
		t.Fatalf("value = %q, want 7.<created_at ns>.<expiry>.<mac>", v)
	}
	mac := hmac.New(sha256.New, testSecret)
	mac.Write([]byte(parts[0] + "." + parts[1] + "." + parts[2]))
	if parts[3] != base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("mac = %q, want HMAC-SHA256 over the first three parts in base64url", parts[3])
	}
	s, ok := readSession(testSecret, v, testNow)
	if !ok || s.id != 7 || s.made != testMade.UnixNano() {
		t.Fatalf("readSession = %+v, %v; want 7 made at %d", s, ok, testMade.UnixNano())
	}
	if s, ok := readSession(testSecret, v, expires.Add(-time.Second)); !ok || s.id != 7 {
		t.Fatalf("a second before expiry: %+v, %v", s, ok)
	}
}

func TestSessionRefusesTamperedAndExpired(t *testing.T) {
	expires := testNow.Add(time.Hour)
	v := sessionValue(testSecret, 7, testMade, expires)
	parts := strings.Split(v, ".")
	made, exp, mac := parts[1], parts[2], parts[3]
	signed := func(fields ...string) string {
		m := hmac.New(sha256.New, testSecret)
		m.Write([]byte(strings.Join(fields, ".")))
		return strings.Join(fields, ".") + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	}
	cases := map[string]string{
		"another id":                "8." + made + "." + exp + "." + mac,
		"another making":            "7." + strconv.FormatInt(testMade.UnixNano()+1, 10) + "." + exp + "." + mac,
		"a later expiry":            "7." + made + "." + strconv.FormatInt(expires.Unix()+86400, 10) + "." + mac,
		"another mac":               "7." + made + "." + exp + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		"a cut mac":                 "7." + made + "." + exp + "." + mac[:len(mac)-2],
		"a mac that is not b64":     "7." + made + "." + exp + ".!!!",
		"no mac":                    "7." + made + "." + exp + ".",
		"three parts, signed":       signed("7", exp),
		"three parts":               "7." + exp + "." + mac,
		"five parts":                v + ".x",
		"empty":                     "",
		"a signed sign":             signed("+7", made, exp),
		"a signed negative id":      signed("-7", made, exp),
		"a signed zero id":          signed("0", made, exp),
		"a signed id of letters":    signed("seven", made, exp),
		"a signed empty id":         signed("", made, exp),
		"a signed negative making":  signed("7", "-"+made, exp),
		"a signed empty making":     signed("7", "", exp),
		"a signed spaced expiry":    signed("7", made, " "+exp),
		"a signed huge id":          signed("99999999999999999999", made, exp),
		"a signed huge making":      signed("7", "99999999999999999999", exp),
		"a signed expiry of text":   signed("7", made, "soon"),
		"too long":                  signed(strings.Repeat("1", 300), made, exp),
		"another secret's":          sessionValue([]byte("another secret of thirty-two b!!"), 7, testMade, expires),
		"a signed value with a dot": signed("7", made, exp+".5"),
	}
	for name, value := range cases {
		if s, ok := readSession(testSecret, value, testNow); ok {
			t.Errorf("%s (%q) is a session of %+v", name, value, s)
		}
	}
	if _, ok := readSession(testSecret, v, expires); ok {
		t.Error("a session is good at its expiry")
	}
	if _, ok := readSession(testSecret, v, expires.Add(time.Hour)); ok {
		t.Error("a session is good after its expiry")
	}
	// The limit is on the value as sent: 256 bytes pass to the signature
	// check, 257 do not.
	pad := maxSession - len(signed("7", made, exp))
	at := signed(strings.Repeat("0", pad)+"7", made, exp)
	over := signed(strings.Repeat("0", pad+1)+"7", made, exp)
	if len(at) != maxSession || len(over) != maxSession+1 {
		t.Fatalf("test values are %d and %d bytes, want %d and %d", len(at), len(over), maxSession, maxSession+1)
	}
	if s, ok := readSession(testSecret, at, testNow); !ok || s.id != 7 {
		t.Errorf("a signed value of exactly %d bytes: %+v, %v; want 7, true", maxSession, s, ok)
	}
	if _, ok := readSession(testSecret, over, testNow); ok {
		t.Errorf("a signed value of %d bytes is read", maxSession+1)
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
