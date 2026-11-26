package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/flakerimi/weft/warp"
)

// LoginInput is what login takes.
type LoginInput struct {
	Email    string `json:"email" warp:"required,max=254"`
	Password string `json:"password" warp:"required,max=200"`
}

// SetupInput is what setup takes: the token from the logged link and the
// owner's password.
type SetupInput struct {
	Token    string `json:"token" warp:"required,max=64"`
	Password string `json:"password" warp:"required,min=8,max=200"`
}

const (
	errNoMatch  = "That email and password do not match."
	errTooMany  = "Too many tries; wait a minute and try again."
	errNotSetup = "This setup link is not valid; restart the app to get a new one."
)

// routes adds login, logout, me and setup under /auth.
func (m *module) routes() {
	warp.Post(m.app, "/auth/login", m.login, warp.Name("login"), warp.Status(http.StatusOK))
	warp.Post(m.app, "/auth/logout", m.logout, warp.Name("logout"))
	warp.Get(m.app, "/auth/me", m.me, warp.Name("me"))
	warp.Post(m.app, "/auth/setup", m.setup, warp.Name("setup"))
}

// login signs a user in. A wrong email and a wrong password get one answer,
// after one hash either way; after 5 failures of one email from one address
// in a minute, the rest of that minute is a 429, answered before any hash.
func (m *module) login(ctx context.Context, in LoginInput) (User, error) {
	r := warp.Request(ctx)
	email := normalEmail(in.Email)
	key := limitKey(email, m.app.ClientIP(r))
	if m.limit.blocked(key) {
		return User{}, warp.TooMany(errTooMany)
	}
	row, found, err := m.byEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	stored := dummyHash
	if found {
		stored = row.PasswordHash
	}
	if !verifyPassword(stored, in.Password) || !found {
		m.limit.fail(key)
		return User{}, warp.Unauthorized(errNoMatch)
	}
	m.limit.clear(key)
	m.signIn(ctx, row.ID)
	return row.user(), nil
}

// byEmail finds the user with the email, as stored.
func (m *module) byEmail(ctx context.Context, email string) (userRow, bool, error) {
	conn := m.db.SQL()
	if conn == nil {
		return userRow{}, false, errors.New("auth: the users table was read before the app started")
	}
	var id int64
	err := conn.QueryRowContext(ctx, `SELECT "id" FROM "users" WHERE "email" = ?`, email).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return userRow{}, false, nil
	}
	if err != nil {
		return userRow{}, false, err
	}
	row, err := m.users.Get(ctx, id)
	var e *warp.Error
	if errors.As(err, &e) && e.Status == http.StatusNotFound { // removed in between
		return userRow{}, false, nil
	}
	return row, err == nil, err
}

// signIn sets the session cookie of the user id on the answer.
func (m *module) signIn(ctx context.Context, id int64) {
	r := warp.Request(ctx)
	value := sessionValue(m.secret, id, time.Now().Add(time.Duration(m.maxAge)*time.Second))
	warp.ResponseHeader(ctx).Add("Set-Cookie", cookie(value, m.maxAge, m.app.IsHTTPS(r)).String())
}

// logout clears the session cookie. It needs a session. The cookie's value
// stays good until it expires, for anyone who copied it.
func (m *module) logout(ctx context.Context, _ warp.None) (warp.None, error) {
	r := warp.Request(ctx)
	if _, err := m.signedIn(r); err != nil {
		return warp.None{}, err
	}
	warp.ResponseHeader(ctx).Add("Set-Cookie", cookie("", -1, m.app.IsHTTPS(r)).String())
	return warp.None{}, nil
}

// me answers the signed-in user, or the 401.
func (m *module) me(ctx context.Context, _ warp.None) (User, error) {
	return m.signedIn(warp.Request(ctx))
}
