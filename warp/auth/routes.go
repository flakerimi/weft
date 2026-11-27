package auth

import (
	"context"
	"net/http"

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
	errNoMatch = "That email and password do not match."
	errTooMany = "Too many tries; wait a minute and try again."

	errTooManyForAccount = "Too many tries for this account; wait 10 minutes, or reset the password."
	errNotSetup          = "This setup link is not valid; restart the app to get a new one."
)

// routes adds the login's routes under /auth.
func (m *module) routes() {
	warp.Post(m.app, "/auth/login", m.login, warp.Name("login"), warp.Status(http.StatusOK))
	warp.Post(m.app, "/auth/logout", m.logout, warp.Name("logout"))
	warp.Post(m.app, "/auth/logout/everywhere", m.logoutEverywhere, warp.Name("logout_everywhere"))
	warp.Get(m.app, "/auth/me", m.me, warp.Name("me"))
	warp.Post(m.app, "/auth/setup", m.setup, warp.Name("setup"))
	warp.Get(m.app, "/auth/sessions", m.listSessions, warp.Name("sessions"))
	warp.Delete(m.app, "/auth/sessions/{id}", m.endSession, warp.Name("end_session"))
}

// login signs a user in. A wrong email and a wrong password get one answer,
// after one hash either way.
func (m *module) login(ctx context.Context, in LoginInput) (Me, error) {
	r := warp.Request(ctx)
	row, err := m.attempt(ctx, in.Email, in.Password, m.app.ClientKey(r))
	if err != nil {
		return Me{}, err
	}
	if err := m.signIn(ctx, row); err != nil {
		return Me{}, err
	}
	return m.meOf(row.user()), nil
}

// attempt checks one login from the client ip, as ClientKey names it (an
// IPv6 client by its /64). Before anything is looked up or hashed it takes a
// try from the address's count, then from the account's, then from the
// email and address's, any of which, when spent, is the 429. A spent account
// gives back the address's try, and a spent email and address the
// account's, so that only a try that is tried counts against an account; a
// spent email and address keeps the address's try taken, as it always was. A
// success clears the email and address's count, never the account's, which
// only its window or a completed reset ends: otherwise someone who knows the
// password could clear the count for a guesser elsewhere. A try that learned
// nothing, since its user could not be looked up or it found no hashing slot
// in time, gives all three back, to the counts it took them from.
func (m *module) attempt(ctx context.Context, email, password, ip string) (userRow, error) {
	email = normalEmail(email)
	key := limitKey(email, ip)
	byAddress := m.perAddress.take(ip)
	if byAddress == nil {
		return userRow{}, warp.TooMany(errTooMany)
	}
	byAccount := m.perAccount.take(accountKey(email))
	if byAccount == nil {
		m.perAddress.give(byAddress)
		return userRow{}, warp.TooMany(errTooManyForAccount)
	}
	byKey := m.perKey.take(key)
	if byKey == nil {
		m.perAccount.give(byAccount)
		return userRow{}, warp.TooMany(errTooMany)
	}
	giveBack := func() {
		m.perAddress.give(byAddress)
		m.perAccount.give(byAccount)
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

// me answers the signed-in user with their permissions, or the 401. It
// takes an API key as auth.Can does, the role then the key's.
func (m *module) me(ctx context.Context, _ warp.None) (Me, error) {
	u, err := m.requester(warp.Request(ctx))
	if err != nil {
		return Me{}, err
	}
	return m.meOf(u), nil
}
