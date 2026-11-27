package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// A key is wk_<id>_<secret>, the secret 43 base64url characters, at most 100
// bytes; anything else is refused before any read.
func TestKeyFormat(t *testing.T) {
	secret := newToken()
	key := keyValue(7, secret)
	if key != "wk_7_"+secret || keyPrefix(7, secret) != "wk_7_"+secret[:4] {
		t.Fatalf("key %q, prefix %q", key, keyPrefix(7, secret))
	}
	if id, s, ok := readKey(key); !ok || id != 7 || s != secret {
		t.Fatalf("readKey = %d, %q, %v", id, s, ok)
	}
	// A secret may begin with _ or -, which base64url has.
	odd := "_-" + secret[2:]
	if id, s, ok := readKey(keyValue(12, odd)); !ok || id != 12 || s != odd {
		t.Fatalf("readKey of a secret that begins with _ = %d, %q, %v", id, s, ok)
	}
	for name, bad := range map[string]string{
		"empty":            "",
		"no prefix":        "7_" + secret,
		"another prefix":   "wx_7_" + secret,
		"upper case":       "WK_7_" + secret,
		"no id":            "wk__" + secret,
		"a zero id":        "wk_0_" + secret,
		"a signed id":      "wk_+7_" + secret,
		"an id of letters": "wk_seven_" + secret,
		"a huge id":        "wk_99999999999999999999_" + secret,
		"no secret":        "wk_7_",
		"a short secret":   "wk_7_" + secret[:42],
		"a long secret":    "wk_7_" + secret + "A",
		"a bad character":  "wk_7_" + secret[:42] + "!",
		"padding":          "wk_7_" + secret[:42] + "=",
		"spaces":           "wk_7_ " + secret[:42],
		"101 bytes":        "wk_" + strings.Repeat("0", 101-4-43) + "_" + secret,
	} {
		if id, s, ok := readKey(bad); ok {
			t.Errorf("%s (%q) is the key %d, %q", name, bad, id, s)
		}
	}
	// 100 bytes, with an id padded by zeros, is read.
	at := "wk_" + strings.Repeat("0", 100-4-43-1) + "7_" + secret
	if len(at) != 100 {
		t.Fatalf("the test key is %d bytes", len(at))
	}
	if id, _, ok := readKey(at); !ok || id != 7 {
		t.Errorf("a key of 100 bytes: %d, %v", id, ok)
	}
}

// The bearer is read from Authorization, its scheme in any case; another
// scheme is no key.
func TestBearer(t *testing.T) {
	for header, want := range map[string]string{
		"Bearer wk_1_x": "wk_1_x",
		"bearer wk_1_x": "wk_1_x",
		"Bearer  x ":    "x",
		"Bearer":        "",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", header)
		if got, ok := bearer(r); !ok || got != want {
			t.Errorf("Authorization %q: %q, %v; want %q", header, got, ok, want)
		}
	}
	for _, header := range []string{"", "Basic dXNlcjpwYXNz", "Bearerwk_1_x"} {
		r := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			r.Header.Set("Authorization", header)
		}
		if got, ok := bearer(r); ok {
			t.Errorf("Authorization %q is the key %q", header, got)
		}
	}
}

func TestLowerRole(t *testing.T) {
	for _, tc := range []struct{ a, b, want string }{
		{RoleOwner, RoleMember, RoleMember},
		{RoleViewer, RoleAdministrator, RoleViewer},
		{RoleMember, RoleMember, RoleMember},
		{RoleOwner, "retired", "retired"},
	} {
		if got := lower(tc.a, tc.b); got != tc.want {
			t.Errorf("lower(%s, %s) = %s, want %s", tc.a, tc.b, got, tc.want)
		}
	}
}
