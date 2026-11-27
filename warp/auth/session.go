package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
)

// A session is a row of the sessions table and a cookie whose value is
// <session id>.<token>.<mac>: the token 32 random bytes and the mac
// HMAC-SHA256 over <session id>.<token>, keyed by AUTH_SECRET, both in
// base64url without padding. The row keeps the token's SHA-256, never the
// token. The mac keeps a forged cookie from costing a database read, and
// makes a change of AUTH_SECRET sign everyone out.
const (
	cookieName       = "session"        // over HTTP
	secureCookieName = "__Host-session" // over HTTPS: a browser takes it only with Secure, Path=/ and no Domain
	maxSession       = 256              // bytes; a longer cookie is not read
	tokenChars       = 43               // 32 bytes in base64url without padding
	maxAgent         = 200              // bytes of the User-Agent a session keeps
)

// newToken is 32 random bytes in base64url without padding, for a session or
// a mailed link.
func newToken() string {
	b := make([]byte, 32)
	rand.Read(b) // never fails, as crypto/rand documents
	return base64.RawURLEncoding.EncodeToString(b)
}

// tokenHash is the SHA-256 of a token in hex, as a table keeps it.
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// hashIs reports, in constant time, whether token's hash is hash.
func hashIs(token, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(tokenHash(token)), []byte(hash)) == 1
}

// sessionValue is the cookie value of the session id with token.
func sessionValue(secret []byte, id int64, token string) string {
	signed := strconv.FormatInt(id, 10) + "." + token
	return signed + "." + base64.RawURLEncoding.EncodeToString(sign(secret, signed))
}

func sign(secret []byte, signed string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signed))
	return mac.Sum(nil)
}

// minSecret is the shortest AUTH_SECRET, in bytes, that Start takes.
const minSecret = 32

// readSession reads a cookie value: the session id and the token, when the
// mac is good. A key shorter than minSecret, as one served without Start or
// after a Start that failed has, reads no session at all. A value over 256
// bytes, one without exactly three parts, an id that is not a whole number
// above 0, or a token or mac that is not 43 base64url characters, is refused
// before anything is computed; then a mac that does not match. Nothing is
// read from the database here.
func readSession(secret []byte, value string) (id int64, token string, ok bool) {
	if len(secret) < minSecret || len(value) > maxSession {
		return 0, "", false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 || !digits(parts[0]) || !isToken(parts[1]) || !isToken(parts[2]) {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id < 1 {
		return 0, "", false
	}
	mac, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(mac, sign(secret, parts[0]+"."+parts[1])) {
		return 0, "", false
	}
	return id, parts[1], true
}

// isToken reports whether s is 43 base64url characters, as a token and a mac
// are.
func isToken(s string) bool {
	if len(s) != tokenChars {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// cutAgent is a User-Agent cut to 200 bytes, where a character ends.
func cutAgent(agent string) string {
	if len(agent) <= maxAgent {
		return agent
	}
	return strings.ToValidUTF8(agent[:maxAgent], "") // a character the cut split is dropped
}

// cookieFor is the session cookie's name for a request over HTTPS or not.
func cookieFor(https bool) string {
	if https {
		return secureCookieName
	}
	return cookieName
}

// cookie is the session cookie with value, lasting maxAge seconds; a
// negative maxAge removes it. Over HTTPS it is __Host-session and Secure.
func cookie(https bool, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     cookieFor(https),
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   https,
	}
}
