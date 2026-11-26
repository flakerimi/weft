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

// A session is a cookie whose value is
// <user id>.<created_at in Unix nanoseconds>.<expiry unix>.<mac>, the mac
// HMAC-SHA256 over the first three parts, keyed by AUTH_SECRET, in base64url
// without padding. The user's making binds it to that user: a database
// replaced under the same secret has another user 1, made at another moment.
// Nothing about a session is kept on the server.
const (
	cookieName       = "session"        // over HTTP
	secureCookieName = "__Host-session" // over HTTPS: a browser takes it only with Secure, Path=/ and no Domain
	maxSession       = 256              // bytes; a longer cookie is not read
)

// session is what a good cookie names.
type session struct {
	id   int64
	made int64 // the user's created_at, in Unix nanoseconds
}

// sessionValue is the cookie value of a session of the user id, made at made,
// that ends at expires.
func sessionValue(secret []byte, id int64, made, expires time.Time) string {
	signed := strconv.FormatInt(id, 10) + "." + strconv.FormatInt(made.UnixNano(), 10) + "." + strconv.FormatInt(expires.Unix(), 10)
	return signed + "." + base64.RawURLEncoding.EncodeToString(sign(secret, signed))
}

func sign(secret []byte, signed string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signed))
	return mac.Sum(nil)
}

// readSession reads a session value at now. A value over 256 bytes, one
// without exactly four parts, or an id, making or expiry that is not a whole
// number, is refused before anything is computed; then a mac that does not
// match, or an expiry at or before now.
func readSession(secret []byte, value string, now time.Time) (session, bool) {
	if len(value) > maxSession {
		return session{}, false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 4 || !digits(parts[0]) || !digits(parts[1]) || !digits(parts[2]) {
		return session{}, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id < 1 {
		return session{}, false
	}
	made, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return session{}, false
	}
	expiry, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return session{}, false
	}
	mac, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil || !hmac.Equal(mac, sign(secret, parts[0]+"."+parts[1]+"."+parts[2])) {
		return session{}, false
	}
	if now.Unix() >= expiry {
		return session{}, false
	}
	return session{id: id, made: made}, true
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
