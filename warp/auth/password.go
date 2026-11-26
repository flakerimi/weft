package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strconv"
	"strings"
)

// A password is kept as pbkdf2-sha256$<iterations>$<salt>$<key>, the salt and
// the key in base64 without padding. The prefix names the scheme and the
// count is read back from the string, so a later scheme, or a higher count,
// can sit beside this one.
const (
	scheme        = "pbkdf2-sha256"
	iterations    = 600_000
	saltBytes     = 16
	keyBytes      = 32
	maxIterations = 10_000_000 // a stored count above this is refused, never computed
	minPassword   = 8          // characters
	maxPassword   = 200
)

// dummyHash is checked when a login names an email no user has, so that the
// answer costs what a wrong password costs. It is the hash of a password no
// one is given, with the same parameters.
const dummyHash = "pbkdf2-sha256$600000$yQrLJNXLN6xr8kFGZmyx+A$Xq9wZOYrYz4CsZHcRKJ/Rxe5PdXcN1Qd7HBkhx326FM"

var b64 = base64.RawStdEncoding

// hashPassword hashes password with a new random salt.
func hashPassword(password string) (string, error) { return hashWith(password, iterations) }

// hashWith hashes password with a new random salt and n iterations; a test
// uses a low n to make many tries quickly.
func hashWith(password string, n int) (string, error) {
	salt := make([]byte, saltBytes)
	rand.Read(salt) // never fails, as crypto/rand documents
	key, err := pbkdf2.Key(sha256.New, password, salt, n, keyBytes)
	if err != nil {
		return "", err
	}
	return scheme + "$" + strconv.Itoa(n) + "$" + b64.EncodeToString(salt) + "$" + b64.EncodeToString(key), nil
}

// slot waits for one of the module's hashing slots, of which there are
// GOMAXPROCS, so that a flood of logins cannot take every core; it gives up
// when ctx ends. The caller calls the release it returns.
func (m *module) slot(ctx context.Context) (release func(), err error) {
	select {
	case m.slots <- struct{}{}:
		m.hashes.Add(1)
		return func() { <-m.slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// hash is hashPassword in one of the module's slots.
func (m *module) hash(ctx context.Context, password string) (string, error) {
	release, err := m.slot(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return hashPassword(password)
}

// verify is verifyPassword in one of the module's slots.
func (m *module) verify(ctx context.Context, stored, password string) (bool, error) {
	release, err := m.slot(ctx)
	if err != nil {
		return false, err
	}
	defer release()
	return verifyPassword(stored, password), nil
}

type parsedHash struct {
	iterations int
	salt, key  []byte
}

// parseHash reads a stored hash. Anything but this scheme, with a count from
// 1 to maxIterations, a salt of 8 to 64 bytes and a key of 32, is not one.
func parseHash(stored string) (parsedHash, bool) {
	if len(stored) > 256 {
		return parsedHash{}, false
	}
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != scheme || !digits(parts[1]) || len(parts[1]) > 8 {
		return parsedHash{}, false
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 1 || n > maxIterations {
		return parsedHash{}, false
	}
	salt, err := b64.DecodeString(parts[2])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return parsedHash{}, false
	}
	key, err := b64.DecodeString(parts[3])
	if err != nil || len(key) != keyBytes {
		return parsedHash{}, false
	}
	return parsedHash{iterations: n, salt: salt, key: key}, true
}

// verifyPassword reports whether password is the one stored was made from,
// comparing the keys in constant time. A stored string it cannot read does
// not match.
func verifyPassword(stored, password string) bool {
	h, ok := parseHash(stored)
	if !ok {
		return false
	}
	key, err := pbkdf2.Key(sha256.New, password, h.salt, h.iterations, keyBytes)
	return err == nil && subtle.ConstantTimeCompare(key, h.key) == 1
}

// digits reports whether s is one or more of 0 to 9 and nothing else.
func digits(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
