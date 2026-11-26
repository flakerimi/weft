package auth

import (
	"context"
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
// after one hash either way.
func (m *module) login(ctx context.Context, in LoginInput) (User, error) {
	r := warp.Request(ctx)
	row, err := m.attempt(ctx, in.Email, in.Password, m.app.ClientKey(r))
	if err != nil {
		return User{}, err
	}
	m.signIn(ctx, row)
	return row.user(), nil
}

// attempt checks one login from the client ip, as ClientKey names it (an
// IPv6 client by its /64). Before anything is looked up or hashed it takes a
// try from the address's count and from the email's, either of which, when
// spent, is the 429; when the email's is spent, the address's try stays
// taken, as it always was. A success clears the email's count. A try that
// learned nothing, since its user could not be looked up or it found no
// hashing slot in time, gives both back, to the counts it took them from.
func (m *module) attempt(ctx context.Context, email, password, ip string) (userRow, error) {
	email = normalEmail(email)
	key := limitKey(email, ip)
	byAddress := m.perAddress.take(ip)
	if byAddress == nil {
		return userRow{}, warp.TooMany(errTooMany)
	}
	byKey := m.perKey.take(key)
	if byKey == nil {
		return userRow{}, warp.TooMany(errTooMany)
	}
	giveBack := func() {
		m.perAddress.give(byAddress)
		m.perKey.give(byKey)
	}
	row, found, err := m.find(ctx, email)
	if err != nil {
		giveBack() // no hash ran
		return userRow{}, err
	}
	stored := dummyHash
	if found {
		stored = row.PasswordHash
	}
	ok, err := m.verify(ctx, stored, password)
	if err != nil {
		if isBusy(err) {
			giveBack()
		}
		return userRow{}, err
	}
	if !ok || !found {
		return userRow{}, warp.Unauthorized(errNoMatch)
	}
	m.perKey.clear(key)
	return row, nil
}

// signIn sets the session cookie of the user on the answer.
func (m *module) signIn(ctx context.Context, row userRow) {
	r := warp.Request(ctx)
	https := m.app.IsHTTPS(r)
	value := sessionValue(m.secret, row.ID, row.CreatedAt, time.Now().Add(time.Duration(m.maxAge)*time.Second))
	warp.ResponseHeader(ctx).Add("Set-Cookie", cookie(https, value, m.maxAge).String())
}

// logout clears the session cookie. It needs a session. The cookie's value
// stays good until it expires, for anyone who copied it.
func (m *module) logout(ctx context.Context, _ warp.None) (warp.None, error) {
	r := warp.Request(ctx)
	if _, err := m.signedIn(r); err != nil {
		return warp.None{}, err
	}
	warp.ResponseHeader(ctx).Add("Set-Cookie", cookie(m.app.IsHTTPS(r), "", -1).String())
	return warp.None{}, nil
}

// me answers the signed-in user, or the 401.
func (m *module) me(ctx context.Context, _ warp.None) (User, error) {
	return m.signedIn(warp.Request(ctx))
}
