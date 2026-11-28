package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

// An API key lets another program act as a user: Authorization: Bearer
// wk_<id>_<secret>, the secret 32 random bytes in base64url, kept as its
// SHA-256. It has a name, a role no higher than its maker's, and an end 1 to
// 365 days on (90 when not given). It acts with the lower of its role and
// its user's role now, so a user demoted is a key demoted. It reaches the
// routes auth.Can guards and me, never the rest of /api/auth, and none that
// manages accounts or keys: a stolen key cannot make more keys, invite,
// change or remove an account, read sessions or change a password. API_KEYS=on
// turns keys on; off, a key is refused.
const (
	keyStart       = "wk_"
	maxKeyBytes    = 100
	maxUserKeys    = 20
	defaultKeyDays = 90
)

const (
	errKeyNotValid = "This API key is not valid."
	errNoKeys      = "This app does not take API keys."
	errKeysOff     = "API keys are off in this app; set API_KEYS=on in .env."
)

// keyRow is a key as the table api_keys keeps it: whose it is, its name and
// role, the prefix that tells it apart, its secret's hash, when it was last
// used and when it ends.
type keyRow struct {
	data.Record
	UserID     int64      `json:"user_id" data:"filter"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	Prefix     string     `json:"prefix"`
	SecretHash string     `json:"secret_hash" warp:"unique"`
	SeenAt     *time.Time `json:"seen_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

// APIKey is one of a user's keys, never with its secret.
type APIKey struct {
	data.Record
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Prefix    string     `json:"prefix"` // wk_<id>_ and the secret's first 4 characters, to tell keys apart
	SeenAt    *time.Time `json:"seen_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

// KeyInput is what create_key takes.
type KeyInput struct {
	Name string `json:"name" warp:"required,max=60"`
	Role string `json:"role" warp:"required,enum=owner|administrator|member|viewer"`
	Days int    `json:"days" warp:"min=0,max=365"` // 0 is 90
}

// NewKey is a key as create_key answers it, the one time its secret is
// sent.
type NewKey struct {
	APIKey
	Secret string `json:"secret"` // the whole key, wk_<id>_<secret>
}

func (r keyRow) apiKey() APIKey {
	return APIKey{Record: r.Record, Name: r.Name, Role: r.Role, Prefix: r.Prefix, SeenAt: r.SeenAt, ExpiresAt: r.ExpiresAt}
}

// keyValue is the whole key of the row id with secret.
func keyValue(id int64, secret string) string {
	return keyStart + strconv.FormatInt(id, 10) + "_" + secret
}

// keyPrefix is what tells a key apart in a list: wk_<id>_ and the secret's
// first 4 characters.
func keyPrefix(id int64, secret string) string { return keyValue(id, secret[:4]) }

// readKey reads a key: its id and its secret. More than 100 bytes, another
// start, an id that is not a whole number above 0, or a secret that is not
// 43 base64url characters, is refused here, before any read. The id ends at
// the first _, since digits have none; the secret may hold more.
func readKey(key string) (id int64, secret string, ok bool) {
	if len(key) > maxKeyBytes {
		return 0, "", false
	}
	rest, ok := strings.CutPrefix(key, keyStart)
	if !ok {
		return 0, "", false
	}
	n, secret, ok := strings.Cut(rest, "_")
	if !ok || !digits(n) || !isToken(secret) {
		return 0, "", false
	}
	id, err := strconv.ParseInt(n, 10, 64)
	if err != nil || id < 1 {
		return 0, "", false
	}
	return id, secret, true
}

// bearer returns the credentials of r's Authorization header when its
// scheme is Bearer, in any case, and whether it is.
func bearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if len(h) < len("Bearer") || !strings.EqualFold(h[:len("Bearer")], "Bearer") || len(h) > len("Bearer") && h[len("Bearer")] != ' ' {
		return "", false
	}
	return strings.TrimSpace(h[len("Bearer"):]), true
}

// lower is the lower of two roles; one that is not of the four is lower than
// any that is.
func lower(a, b string) string {
	if rank(a) < rank(b) {
		return a
	}
	return b
}

// requester is who r comes from, for the routes auth.Can guards and me: a
// request with a bearer is read as a key and never as a cookie; any other as
// its session cookie.
func (m *module) requester(r *http.Request) (User, error) {
	if key, ok := bearer(r); ok {
		return m.keyed(r.Context(), key)
	}
	u, _, err := m.signedIn(r)
	return u, err
}

// keyed returns the user a key acts as, its role the lower of the key's and
// the user's, or the 401 for a key that is not good: keys off, a key of the
// wrong shape, one no row has, whose secret is not the row's, that has ended
// or whose user is gone. It writes seen_at when it is 5 minutes old or more.
func (m *module) keyed(ctx context.Context, key string) (User, error) {
	if !m.apiKeys {
		return User{}, warp.Unauthorized(errNoKeys)
	}
	bad := warp.Unauthorized(errKeyNotValid)
	id, secret, ok := readKey(key)
	if !ok {
		return User{}, bad
	}
	row, err := m.keys.Get(ctx, id)
	switch {
	case isNotFound(err):
		return User{}, bad
	case err != nil:
		return User{}, err
	}
	now := m.now()
	if !hashIs(secret, row.SecretHash) || !now.Before(row.ExpiresAt) {
		return User{}, bad
	}
	u, err := m.users.Get(ctx, row.UserID)
	switch {
	case isNotFound(err):
		return User{}, bad
	case err != nil:
		return User{}, err
	}
	if row.SeenAt == nil || now.Sub(*row.SeenAt) >= seenEvery {
		row.SeenAt = &now
		_, err := m.keys.Update(ctx, id, &row, []string{"seen_at"})
		switch {
		case isNotFound(err): // removed while this request was read
			return User{}, bad
		case err != nil:
			return User{}, err
		}
	}
	user := u.user()
	user.Role = lower(row.Role, u.Role)
	return user, nil
}

// keyRoutes adds keys, create_key and remove_key under /auth, which take the
// session cookie alone.
func (m *module) keyRoutes() {
	warp.Get(m.app, "/auth/keys", m.listKeys, warp.Name("keys"), noKey)
	warp.Post(m.app, "/auth/keys", m.createKey, warp.Name("create_key"), noKey)
	warp.Delete(m.app, "/auth/keys/{id}", m.removeKey, warp.Name("remove_key"), noKey)
}

// errKeyCannot is the answer of a route that manages accounts or keys to a
// request that carries an API key.
const errKeyCannot = "An API key cannot manage accounts or keys; sign in to do that."

// noKey is the route option of the routes that manage accounts or keys:
// invitations, role changes, removing a user, ending a user's sessions, and
// the keys. A request that carries an API key is answered 403 before
// anything else, whatever the key's role, so a stolen key cannot give its
// thief an account or a key of their own. Listed before Can, it is outside
// Can's guard.
var noKey = warp.With(func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := bearer(r); ok {
			warp.WriteError(w, r, warp.Forbidden(errKeyCannot))
			return
		}
		next.ServeHTTP(w, r)
	})
})

// mine is a query for the rows of the user id, at most n.
func mine(id int64, sort string, n int) data.Query {
	return data.Query{Filter: map[string]string{"user_id": strconv.FormatInt(id, 10)}, Sort: sort, PerPage: n}
}

// listKeys answers the signed-in user's keys, newest first, never a secret.
func (m *module) listKeys(ctx context.Context, _ warp.None) ([]APIKey, error) {
	u, _, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return nil, err
	}
	l, err := m.keys.List(ctx, mine(u.ID, "", maxUserKeys))
	if err != nil {
		return nil, err
	}
	out := make([]APIKey, 0, len(l.Items))
	for _, k := range l.Items {
		out = append(out, k.apiKey())
	}
	return out, nil
}

// createKey makes a key for the signed-in user, with a role no higher than
// theirs, at most 20 a user, and answers it with its secret, once.
func (m *module) createKey(ctx context.Context, in KeyInput) (NewKey, error) {
	u, _, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return NewKey{}, err
	}
	if !m.apiKeys {
		return NewKey{}, warp.Forbidden(errKeysOff)
	}
	if above(in.Role, u.Role) {
		return NewKey{}, warp.Forbidden(fmt.Sprintf("Your account cannot give the role %s.", in.Role))
	}
	days := in.Days
	if days == 0 {
		days = defaultKeyDays
	}
	secret := newToken()
	row := keyRow{UserID: u.ID, Name: in.Name, Role: in.Role, SecretHash: tokenHash(secret), ExpiresAt: m.now().Add(time.Duration(days) * 24 * time.Hour)}
	err = m.db.Tx(ctx, func(ctx context.Context) error {
		have, err := m.keys.List(ctx, mine(u.ID, "", 1))
		if err != nil {
			return err
		}
		if have.Total >= maxUserKeys {
			return warp.Conflict(fmt.Sprintf("You have %d API keys; remove one first.", maxUserKeys))
		}
		if err := m.keys.Create(ctx, &row); err != nil {
			return err
		}
		row.Prefix = keyPrefix(row.ID, secret)
		row, err = m.keys.Update(ctx, row.ID, &row, []string{"prefix"})
		return err
	})
	if err != nil {
		return NewKey{}, err
	}
	return NewKey{APIKey: row.apiKey(), Secret: keyValue(row.ID, secret)}, nil
}

// removeKey removes one of the signed-in user's keys, which is refused at
// once; another user's is answered as one no one has.
func (m *module) removeKey(ctx context.Context, in data.ID) (warp.None, error) {
	u, _, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return warp.None{}, err
	}
	notYours := warp.NotFound(fmt.Sprintf("No API key of yours has id %d.", in.ID))
	row, err := m.keys.Get(ctx, in.ID)
	switch {
	case isNotFound(err):
		return warp.None{}, notYours
	case err != nil:
		return warp.None{}, err
	case row.UserID != u.ID:
		return warp.None{}, notYours
	}
	if err := m.keys.Delete(ctx, in.ID); err != nil {
		if isNotFound(err) {
			return warp.None{}, notYours
		}
		return warp.None{}, err
	}
	return warp.None{}, nil
}
