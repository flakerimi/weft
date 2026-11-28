package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

// sessionRow is a session as the sessions table keeps it: whose it is, the
// SHA-256 of its token, when it ends and was last seen, and where it was
// signed in from. No route answers it; sessions answers SessionInfo.
type sessionRow struct {
	data.Record
	UserID    int64     `json:"user_id" data:"filter"`
	TokenHash string    `json:"token_hash" warp:"unique"`
	ExpiresAt time.Time `json:"expires_at"`
	SeenAt    time.Time `json:"seen_at" data:"sort"`
	Address   string    `json:"address"`
	Agent     string    `json:"agent"`
}

// SessionInfo is one of the signed-in user's sessions, as sessions lists it.
type SessionInfo struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	SeenAt    time.Time `json:"seen_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Address   string    `json:"address"` // the client's address at sign-in, as app.ClientIP gives it
	Agent     string    `json:"agent"`   // the User-Agent at sign-in, at most 200 bytes
	Current   bool      `json:"current"` // the session of the request that asked
}

// A session lasts AUTH_SESSION_DAYS from sign-in, however much it is used. A
// user has at most maxSessions; seen_at is written at most once every
// seenEvery, so a request is not a write.
const (
	maxSessions = 50
	seenEvery   = 5 * time.Minute
)

// isNotFound reports whether err is a store's 404.
func isNotFound(err error) bool {
	var e *warp.Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

// signedIn returns the user of r's session cookie and the session's id, or
// the 401 when there is no cookie, its mac is not good, its session is not
// in the table, its token is not the session's, it has ended, or its user is
// gone. Over HTTPS only __Host-session is read, over HTTP only session. It
// writes seen_at when it is 5 minutes old or more.
func (m *module) signedIn(r *http.Request) (User, int64, error) {
	none := warp.Unauthorized(errNoSession)
	c, err := r.Cookie(cookieFor(m.app.IsHTTPS(r)))
	if err != nil {
		return User{}, 0, none
	}
	id, token, ok := readSession(m.secret, c.Value)
	if !ok {
		return User{}, 0, none
	}
	ctx := r.Context()
	row, err := m.sessions.Get(ctx, id)
	switch {
	case isNotFound(err):
		return User{}, 0, none
	case err != nil:
		return User{}, 0, err
	}
	now := m.now()
	if !hashIs(token, row.TokenHash) || !now.Before(row.ExpiresAt) {
		return User{}, 0, none
	}
	u, err := m.users.Get(ctx, row.UserID)
	switch {
	case isNotFound(err):
		return User{}, 0, none
	case err != nil:
		return User{}, 0, err
	}
	if now.Sub(row.SeenAt) >= seenEvery {
		row.SeenAt = now
		_, err := m.sessions.Update(ctx, id, &row, []string{"seen_at"})
		switch {
		case isNotFound(err): // ended while this request was read
			return User{}, 0, none
		case err != nil:
			return User{}, 0, err
		}
	}
	return u.user(), id, nil
}

// expired removes every session, mailed token, invitation and API key that
// has ended. Each ends at its expires_at, so one ending now is removed too: the
// store keeps times to the nanosecond.
func (m *module) expired(ctx context.Context, now time.Time) error {
	end := now.Add(time.Nanosecond)
	if _, err := m.sessions.DeleteBefore(ctx, "expires_at", end); err != nil {
		return err
	}
	if _, err := m.tokens.DeleteBefore(ctx, "expires_at", end); err != nil {
		return err
	}
	if _, err := m.invitations.DeleteBefore(ctx, "expires_at", end); err != nil {
		return err
	}
	_, err := m.keys.DeleteBefore(ctx, "expires_at", end)
	return err
}

// signIn makes a session of the user, as u was read when its password was
// checked, and sets its cookie on the answer. In one transaction it refuses
// with login's 401 when the user's password is not the one checked any more,
// removes every ended session, token, invitation and key, ends the user's sessions
// seen longest ago while the user has 50, and adds the new one.
func (m *module) signIn(ctx context.Context, u userRow) error {
	r := warp.Request(ctx)
	now := m.now()
	token := newToken()
	row := sessionRow{
		UserID:    u.ID,
		TokenHash: tokenHash(token),
		ExpiresAt: now.Add(time.Duration(m.maxAge) * time.Second),
		SeenAt:    now,
		Address:   m.app.ClientIP(r),
		Agent:     cutAgent(r.UserAgent()),
	}
	err := m.db.Tx(ctx, func(ctx context.Context) error {
		// The password checked may have changed, or the user gone, since it
		// was read: a sign-in under way while a password changes or is reset
		// keeps no session. In this transaction, a change after it ends the
		// session as it ends the others.
		stored, err := m.users.Get(ctx, u.ID)
		if isNotFound(err) || err == nil && stored.PasswordHash != u.PasswordHash {
			return warp.Unauthorized(errNoMatch)
		}
		if err != nil {
			return err
		}
		if err := m.expired(ctx, now); err != nil {
			return err
		}
		mine, err := m.sessions.List(ctx, data.Query{Filter: map[string]string{"user_id": strconv.FormatInt(u.ID, 10)}, Sort: "seen_at", PerPage: maxSessions + 1})
		if err != nil {
			return err
		}
		for i := 0; i <= mine.Total-maxSessions && i < len(mine.Items); i++ {
			if err := m.sessions.Delete(ctx, mine.Items[i].ID); err != nil && !isNotFound(err) {
				return err
			}
		}
		return m.sessions.Create(ctx, &row)
	})
	if err != nil {
		return err
	}
	warp.ResponseHeader(ctx).Add("Set-Cookie", cookie(m.app.IsHTTPS(r), sessionValue(m.secret, row.ID, token), m.maxAge).String())
	return nil
}

// clearCookie removes the session cookie on the answer.
func (m *module) clearCookie(ctx context.Context) {
	warp.ResponseHeader(ctx).Add("Set-Cookie", cookie(m.app.IsHTTPS(warp.Request(ctx)), "", -1).String())
}

// logout ends the request's session and clears its cookie, so a copy of the
// cookie is no session either.
func (m *module) logout(ctx context.Context, _ warp.None) (warp.None, error) {
	_, id, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return warp.None{}, err
	}
	if err := m.sessions.Delete(ctx, id); err != nil && !isNotFound(err) {
		return warp.None{}, err
	}
	m.clearCookie(ctx)
	return warp.None{}, nil
}

// logoutEverywhere ends every session of the signed-in user, this one
// included, and clears the cookie.
func (m *module) logoutEverywhere(ctx context.Context, _ warp.None) (warp.None, error) {
	u, _, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return warp.None{}, err
	}
	if _, err := m.sessions.DeleteBy(ctx, "user_id", u.ID); err != nil {
		return warp.None{}, err
	}
	m.clearCookie(ctx)
	return warp.None{}, nil
}

// listSessions answers the signed-in user's sessions, seen most recently
// first, at most 50, the request's own marked current.
func (m *module) listSessions(ctx context.Context, _ warp.None) ([]SessionInfo, error) {
	u, current, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return nil, err
	}
	mine, err := m.sessions.List(ctx, data.Query{Filter: map[string]string{"user_id": strconv.FormatInt(u.ID, 10)}, Sort: "-seen_at", PerPage: maxSessions})
	if err != nil {
		return nil, err
	}
	out := make([]SessionInfo, 0, len(mine.Items))
	for _, s := range mine.Items {
		out = append(out, SessionInfo{ID: s.ID, CreatedAt: s.CreatedAt, SeenAt: s.SeenAt, ExpiresAt: s.ExpiresAt, Address: s.Address, Agent: s.Agent, Current: s.ID == current})
	}
	return out, nil
}

// endSession ends one of the signed-in user's sessions. A session another
// user has is answered as one no one has, 404, so its id says nothing; ending
// the request's own clears the cookie too.
func (m *module) endSession(ctx context.Context, in data.ID) (warp.None, error) {
	u, current, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return warp.None{}, err
	}
	notYours := warp.NotFound(fmt.Sprintf("No session of yours has id %d.", in.ID))
	row, err := m.sessions.Get(ctx, in.ID)
	switch {
	case isNotFound(err):
		return warp.None{}, notYours
	case err != nil:
		return warp.None{}, err
	case row.UserID != u.ID:
		return warp.None{}, notYours
	}
	err = m.sessions.Delete(ctx, in.ID)
	switch {
	case isNotFound(err):
		return warp.None{}, notYours
	case err != nil:
		return warp.None{}, err
	}
	if in.ID == current {
		m.clearCookie(ctx)
	}
	return warp.None{}, nil
}
