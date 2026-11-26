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
)

func TestSessionRoundTrip(t *testing.T) {
	expires := testNow.Add(30 * 24 * time.Hour)
	v := sessionValue(testSecret, 7, expires)
	parts := strings.Split(v, ".")
	if len(parts) != 3 || parts[0] != "7" || parts[1] != strconv.FormatInt(expires.Unix(), 10) {
		t.Fatalf("value = %q, want 7.<expiry>.<mac>", v)
	}
	mac := hmac.New(sha256.New, testSecret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if parts[2] != base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("mac = %q, want HMAC-SHA256 over %q in base64url", parts[2], parts[0]+"."+parts[1])
	}
	id, ok := sessionID(testSecret, v, testNow)
	if !ok || id != 7 {
		t.Fatalf("sessionID = %d, %v; want 7, true", id, ok)
	}
	if id, ok := sessionID(testSecret, v, expires.Add(-time.Second)); !ok || id != 7 {
		t.Fatalf("a second before expiry: %d, %v", id, ok)
	}
}

func TestSessionRefusesTamperedAndExpired(t *testing.T) {
	expires := testNow.Add(time.Hour)
	v := sessionValue(testSecret, 7, expires)
	parts := strings.Split(v, ".")
	exp, mac := parts[1], parts[2]
	signed := func(id, exp string) string {
		m := hmac.New(sha256.New, testSecret)
		m.Write([]byte(id + "." + exp))
		return id + "." + exp + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	}
	cases := map[string]string{
		"another id":              "8." + exp + "." + mac,
		"a later expiry":          "7." + strconv.FormatInt(expires.Unix()+86400, 10) + "." + mac,
		"another mac":             "7." + exp + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		"a cut mac":               "7." + exp + "." + mac[:len(mac)-2],
		"a mac that is not b64":   "7." + exp + ".!!!",
		"no mac":                  "7." + exp + ".",
		"two parts":               "7." + exp,
		"four parts":              v + ".x",
		"empty":                   "",
		"a signed sign":           signed("+7", exp),
		"a signed negative id":    signed("-7", exp),
		"a signed zero id":        signed("0", exp),
		"a signed id of letters":  signed("seven", exp),
		"a signed empty id":       signed("", exp),
		"a signed spaced expiry":  signed("7", " "+exp),
		"a signed huge id":        signed("99999999999999999999", exp),
		"a signed expiry of text": signed("7", "soon"),
		"too long":                signed(strings.Repeat("1", 300), exp),
		"another secret's": func() string {
			return sessionValue([]byte("another secret of thirty-two b!!"), 7, expires)
		}(),
	}
	for name, value := range cases {
		if id, ok := sessionID(testSecret, value, testNow); ok {
			t.Errorf("%s (%q) is a session of %d", name, value, id)
		}
	}
	if _, ok := sessionID(testSecret, v, expires); ok {
		t.Error("a session is good at its expiry")
	}
	if _, ok := sessionID(testSecret, v, expires.Add(time.Hour)); ok {
		t.Error("a session is good after its expiry")
	}
	// The limit is on the value as sent: 256 bytes pass to the signature
	// check, 257 do not.
	if long := signed("7", exp); len(long) > maxSession {
		t.Fatalf("a real value is %d bytes, over the limit", len(long))
	}
	pad := maxSession - len(signed("7", exp))
	at := signed(strings.Repeat("0", pad)+"7", exp)
	over := signed(strings.Repeat("0", pad+1)+"7", exp)
	if len(at) != maxSession || len(over) != maxSession+1 {
		t.Fatalf("test values are %d and %d bytes, want %d and %d", len(at), len(over), maxSession, maxSession+1)
	}
	if id, ok := sessionID(testSecret, at, testNow); !ok || id != 7 {
		t.Errorf("a signed value of exactly %d bytes: %d, %v; want 7, true", maxSession, id, ok)
	}
	if _, ok := sessionID(testSecret, over, testNow); ok {
		t.Errorf("a signed value of %d bytes is read", maxSession+1)
	}
}
