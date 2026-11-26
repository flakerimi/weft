package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A session is the cookie named session: <user id>.<expiry unix>.<mac>, the
// mac HMAC-SHA256 over <user id>.<expiry unix>, keyed by AUTH_SECRET, in
// base64url without padding. Nothing about it is kept on the server.
const (
	cookieName = "session"
	maxSession = 256 // bytes; a longer cookie is not read
)

// sessionValue is the cookie value of a session of the user id that ends at
// expires.
func sessionValue(secret []byte, id int64, expires time.Time) string {
	signed := strconv.FormatInt(id, 10) + "." + strconv.FormatInt(expires.Unix(), 10)
	return signed + "." + base64.RawURLEncoding.EncodeToString(sign(secret, signed))
}

func sign(secret []byte, signed string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signed))
	return mac.Sum(nil)
}

// sessionID returns the user id of the session value at now. A value over 256
// bytes, one without exactly three parts, an id or expiry that is not a
// whole number, is refused before anything is computed; then a mac that does
// not match, or an expiry at or before now.
func sessionID(secret []byte, value string, now time.Time) (int64, bool) {
	if len(value) > maxSession {
		return 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 || !digits(parts[0]) || !digits(parts[1]) {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id < 1 {
		return 0, false
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	mac, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(mac, sign(secret, parts[0]+"."+parts[1])) {
		return 0, false
	}
	if now.Unix() >= expiry {
		return 0, false
	}
	return id, true
}

// cookie is the session cookie with value, lasting maxAge seconds; a
// negative maxAge removes it. It is Secure when the request came over HTTPS.
func cookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}
