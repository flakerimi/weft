package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
)

// Registration, invitation and reset are one mechanism: a mailed link,
// <APP_URL>/password#token=<token>, to one page that sets a password, signs
// in and ends the token. A token is 32 random bytes in base64url, kept as
// its SHA-256 and used once. register makes no user until its link is used,
// so an address is proved before an account exists.
const (
	purposeInvite   = "invite"
	purposeRegister = "register"
	purposeReset    = "reset"

	resetLasts    = time.Hour
	registerLasts = 24 * time.Hour
	inviteLasts   = 7 * 24 * time.Hour

	registrationOpen   = "open"
	registrationInvite = "invite"
	registrationClosed = "closed"
)

// The limits on what mails and what reads a mailed token: register and
// forgot, 3 an hour per address mailed and 10 an hour per client; invite,
// 50 an hour per inviter; token and password, 30 a minute per client. Each
// keeps at most maxKeys keys, the oldest dropped.
const (
	maxMailsPerAddress = 3
	maxMailsPerClient  = 10
	maxInvites         = 50
	maxTokenTries      = 30
)

const (
	errLinkGone         = "This link is not valid any more; ask for a new one."
	errNoSignUps        = "This app does not take sign-ups; ask for an invitation."
	errNoInvitations    = "This app takes no invitations; set AUTH_REGISTRATION=invite in .env."
	errTooManyMails     = "Too many emails asked for; wait an hour and try again."
	errTooManyInvites   = "Too many invitations; wait an hour and try again."
	errHasAccount       = "This address already has an account."
	errHigherInvitation = "Your account cannot remove an invitation for a higher role."
)

// tokenRow is a mailed token of register or reset, as the table auth_tokens
// keeps it: its purpose, the address it was mailed to, for a reset the user
// it resets (0 for register), its hash and its end.
type tokenRow struct {
	data.Record
	Purpose   string    `json:"purpose"`
	Email     string    `json:"email"`
	UserID    int64     `json:"user_id"`
	TokenHash string    `json:"token_hash" warp:"unique"`
	ExpiresAt time.Time `json:"expires_at"`
}

// invitationRow is an invitation as the table invitations keeps it, with
// its token's hash. One address has at most one.
type invitationRow struct {
	data.Record
	Email     string    `json:"email" warp:"unique" data:"sort,search"`
	Role      string    `json:"role" data:"filter"`
	InvitedBy int64     `json:"invited_by" data:"filter"`
	ExpiresAt time.Time `json:"expires_at"`
	TokenHash string    `json:"token_hash" warp:"unique"`
}

func (r invitationRow) invitation() Invitation {
	return Invitation{Record: r.Record, Email: r.Email, Role: r.Role, InvitedBy: r.InvitedBy, ExpiresAt: r.ExpiresAt}
}

// Settings is how the app takes new accounts, which a page reads to know
// what to offer.
type Settings struct {
	Registration string `json:"registration" warp:"enum=open|invite|closed"`
	APIKeys      bool   `json:"api_keys"`
}

// RegisterInput is what register takes: the address to mail a sign-up link.
type RegisterInput struct {
	Email string `json:"email" warp:"required,email,max=254"`
}

// ForgotInput is what forgot takes: the address to mail a reset link.
type ForgotInput struct {
	Email string `json:"email" warp:"required,email,max=254"`
}

// TokenInput is a mailed token, from the link's fragment.
type TokenInput struct {
	Token string `json:"token" warp:"required,max=64"`
}

// TokenInfo is what a mailed token is for.
type TokenInfo struct {
	Purpose string `json:"purpose" warp:"enum=invite|register|reset"`
	Email   string `json:"email"`
	Role    string `json:"role"` // the invited role; "" for reset and register
}

// PasswordInput is a mailed token and the password to set with it.
type PasswordInput struct {
	Token    string `json:"token" warp:"required,max=64"`
	Password string `json:"password" warp:"required,min=8,max=200"`
}

// Invitation is an invitation to make an account with a role.
type Invitation struct {
	data.Record
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	InvitedBy int64     `json:"invited_by"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InviteInput is what invite takes.
type InviteInput struct {
	Email string `json:"email" warp:"required,email,max=254"`
	Role  string `json:"role" warp:"required,enum=owner|administrator|member|viewer"`
}

// mailRoutes adds settings, register, forgot, token, password and the
// invitations' routes under /auth.
func (m *module) mailRoutes() {
	warp.Get(m.app, "/auth/settings", m.settings, warp.Name("settings"))
	warp.Post(m.app, "/auth/register", m.register, warp.Name("register"))
	warp.Post(m.app, "/auth/forgot", m.forgot, warp.Name("forgot"))
	warp.Post(m.app, "/auth/token", m.tokenInfo, warp.Name("token"), warp.Status(http.StatusOK))
	warp.Post(m.app, "/auth/password", m.setPassword, warp.Name("password"), warp.Status(http.StatusOK))
	warp.Get(m.app, "/auth/invitations", m.listInvitations, warp.Name("invitations"), Can("users:read"))
	warp.Post(m.app, "/auth/invitations", m.invite, warp.Name("invite"), noKey, Can("users:invite"))
	warp.Delete(m.app, "/auth/invitations/{id}", m.removeInvitation, warp.Name("remove_invitation"), noKey, Can("users:invite"))
}

// settings answers AUTH_REGISTRATION and whether API_KEYS is on.
func (m *module) settings(context.Context, warp.None) (Settings, error) {
	return Settings{Registration: m.registration, APIKeys: m.apiKeys}, nil
}

// mailTry takes one try from the client's count and one from the address's,
// before anything is looked up. A spent address gives back the client's try,
// since nothing was mailed.
func (m *module) mailTry(ctx context.Context, email string) error {
	byClient := m.mailPerClient.take(m.app.ClientKey(warp.Request(ctx)))
	if byClient == nil {
		return warp.TooMany(errTooManyMails)
	}
	if m.mailPerAddress.take(accountKey(email)) == nil {
		m.mailPerClient.give(byClient)
		return warp.TooMany(errTooManyMails)
	}
	return nil
}

// register mails a link to finish signing up, under AUTH_REGISTRATION=open
// once the owner is set up; an address with an account is mailed a reset
// link instead. The lookup and the mail happen after the answer, so the
// answer, 204, and the work before it are the same whatever the address,
// and whatever the mail module does.
func (m *module) register(ctx context.Context, in RegisterInput) (warp.None, error) {
	if m.registration != registrationOpen {
		return warp.None{}, warp.Forbidden(errNoSignUps)
	}
	// Before the owner is set up, a sign-up would make a user and so end the
	// setup link: the app would have no owner.
	n, err := m.users.Count(ctx)
	if err != nil {
		return warp.None{}, err
	}
	if n == 0 {
		return warp.None{}, warp.Forbidden(errNoSignUps)
	}
	email := normalEmail(in.Email)
	if err := m.mailTry(ctx, email); err != nil {
		return warp.None{}, err
	}
	return warp.None{}, m.later(func(ctx context.Context) error {
		u, found, err := m.find(ctx, email)
		switch {
		case err != nil:
			return err
		case found:
			return m.mailToken(ctx, email, u.ID, purposeReset, m.mails.known)
		}
		return m.mailToken(ctx, email, 0, purposeRegister, m.mails.register)
	})
}

// forgot mails a reset link to an address with an account, and nothing to
// one without, under any AUTH_REGISTRATION. As in register, the lookup and
// the mail happen after the answer, which is the same either way.
func (m *module) forgot(ctx context.Context, in ForgotInput) (warp.None, error) {
	email := normalEmail(in.Email)
	if err := m.mailTry(ctx, email); err != nil {
		return warp.None{}, err
	}
	return warp.None{}, m.later(func(ctx context.Context) error {
		u, found, err := m.find(ctx, email)
		if err != nil || !found {
			return err
		}
		return m.mailToken(ctx, email, u.ID, purposeReset, m.mails.reset)
	})
}

// At most maxLater of register's and forgot's mailings run at once, each for
// at most laterLasts; past that, both answer the same 503 for every address.
const (
	maxLater    = 32
	laterLasts  = 30 * time.Second
	errMailBusy = "The server is busy sending mail; try again in a moment."
)

// later runs work after the request answers, on a goroutine bound to the
// app: Stop waits for it. Its error is logged, never answered; the log names
// no address and no token.
func (m *module) later(work func(ctx context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.base == nil || m.stopping {
		return warp.Unavailable(errMailBusy)
	}
	select {
	case m.laterSlots <- struct{}{}:
	default:
		return warp.Unavailable(errMailBusy)
	}
	m.pending.Add(1)
	base := m.base
	go func() {
		defer m.pending.Done()
		defer func() { <-m.laterSlots }()
		ctx, cancel := context.WithTimeout(base, laterLasts)
		defer cancel()
		if err := work(ctx); err != nil {
			slog.Error(fmt.Sprintf("auth: a mail register or forgot asked for was not sent (%v); check the mail module and the database.", err))
		}
	}()
	return nil
}

// mailToken makes a token of purpose for email (a reset for the user id),
// ending the address's earlier one, and mails its link, in one transaction,
// so a mail the mail module refuses (a full queue) leaves no token.
func (m *module) mailToken(ctx context.Context, email string, userID int64, purpose string, tpl *mail.Template) error {
	lasts := resetLasts
	if purpose == purposeRegister {
		lasts = registerLasts
	}
	token := newToken()
	row := tokenRow{Purpose: purpose, Email: email, UserID: userID, TokenHash: tokenHash(token), ExpiresAt: m.now().Add(lasts)}
	msg, err := tpl.Message(email, mailData{Host: m.host, Link: m.link(token)})
	if err != nil {
		return err
	}
	return m.db.Tx(ctx, func(ctx context.Context) error {
		if _, err := m.tokens.DeleteBy(ctx, "email", email); err != nil {
			return err
		}
		if err := m.tokens.Create(ctx, &row); err != nil {
			return err
		}
		return m.mail.Send(ctx, msg)
	})
}

// mailed is what a good mailed token is for: its purpose, its address, the
// invited role, the user a reset resets, and the id of its row in
// auth_tokens or invitations.
type mailed struct {
	purpose, email, role string
	id, userID           int64
}

// readToken returns what token is for, or the 403 for a token no one has,
// one that has ended, a reset whose user is gone, and an invitation whose
// inviter is gone or now ranks below the role invited. A token is looked up by its hash, which is all
// that is kept.
func (m *module) readToken(ctx context.Context, token string) (mailed, error) {
	gone := warp.Forbidden(errLinkGone)
	if !isToken(token) {
		return mailed{}, gone
	}
	hash, now := tokenHash(token), m.now()
	t, ok, err := m.tokens.First(ctx, "token_hash", hash)
	switch {
	case err != nil:
		return mailed{}, err
	case ok && !now.Before(t.ExpiresAt):
		return mailed{}, gone
	case ok && t.Purpose == purposeReset:
		// A reset is its user's: gone with them, never another account's.
		u, err := m.users.Get(ctx, t.UserID)
		switch {
		case isNotFound(err) || err == nil && u.Email != t.Email:
			return mailed{}, gone
		case err != nil:
			return mailed{}, err
		}
		fallthrough
	case ok:
		return mailed{purpose: t.Purpose, email: t.Email, id: t.ID, userID: t.UserID}, nil
	}
	inv, ok, err := m.invitations.First(ctx, "token_hash", hash)
	switch {
	case err != nil:
		return mailed{}, err
	case !ok || !now.Before(inv.ExpiresAt):
		return mailed{}, gone
	}
	inviter, err := m.users.Get(ctx, inv.InvitedBy)
	switch {
	case isNotFound(err):
		return mailed{}, gone
	case err != nil:
		return mailed{}, err
	case above(inv.Role, inviter.Role):
		return mailed{}, gone
	}
	return mailed{purpose: purposeInvite, email: inv.Email, role: inv.Role, id: inv.ID}, nil
}

// tokenTry takes one try from the client's count of token and password.
func (m *module) tokenTry(ctx context.Context) error {
	if m.tokenPerClient.take(m.app.ClientKey(warp.Request(ctx))) == nil {
		return warp.TooMany(errTooMany)
	}
	return nil
}

// tokenInfo answers what a mailed token is for, for the page that sets a
// password to say so.
func (m *module) tokenInfo(ctx context.Context, in TokenInput) (TokenInfo, error) {
	if err := m.tokenTry(ctx); err != nil {
		return TokenInfo{}, err
	}
	t, err := m.readToken(ctx, in.Token)
	if err != nil {
		return TokenInfo{}, err
	}
	return TokenInfo{Purpose: t.purpose, Email: t.email, Role: t.role}, nil
}

// setPassword uses a mailed token: for reset it sets the user's password and
// ends every session and API key of theirs; for register it makes a viewer; for invite
// it makes the user with the role invited. The token ends in the same
// transaction, first, so it works once. Then it signs in and answers Me.
func (m *module) setPassword(ctx context.Context, in PasswordInput) (Me, error) {
	if err := m.tokenTry(ctx); err != nil {
		return Me{}, err
	}
	if _, err := m.readToken(ctx, in.Token); err != nil {
		return Me{}, err
	}
	hash, err := m.hash(ctx, in.Password) // before the transaction, which holds the write lock
	if err != nil {
		return Me{}, err
	}
	gone := warp.Forbidden(errLinkGone)
	var row userRow
	var t mailed
	err = m.db.Tx(ctx, func(ctx context.Context) error {
		if t, err = m.readToken(ctx, in.Token); err != nil {
			return err
		}
		ended := m.tokens.Delete
		if t.purpose == purposeInvite {
			ended = m.invitations.Delete
		}
		if err := ended(ctx, t.id); err != nil {
			if isNotFound(err) { // another request used it first
				return gone
			}
			return err
		}
		u, found, err := m.find(ctx, t.email)
		if err != nil {
			return err
		}
		switch {
		case t.purpose == purposeReset && (!found || u.ID != t.userID): // the user was removed since
			return gone
		case t.purpose == purposeReset:
			u.PasswordHash = hash
			if row, err = m.users.Update(ctx, u.ID, &u, []string{"password_hash"}); err != nil {
				return err
			}
			// A reset is the way back from a stolen password: every session
			// and every API key the thief could have made ends with it.
			if _, err = m.sessions.DeleteBy(ctx, "user_id", u.ID); err != nil {
				return err
			}
			_, err = m.keys.DeleteBy(ctx, "user_id", u.ID)
			return err
		case found: // the address has an account since
			return gone
		}
		role := t.role
		if t.purpose == purposeRegister {
			role = RoleViewer
			if n, err := m.users.Count(ctx); err != nil || n == 0 { // no owner yet: the setup link is the owner's
				return cmpErr(err, gone)
			}
		}
		row = userRow{Email: t.email, Role: role, PasswordHash: hash}
		if err := m.users.Create(ctx, &row); err != nil {
			if isInvalid(err) { // made at this moment by another request
				return gone
			}
			return err
		}
		return nil
	})
	if err != nil {
		return Me{}, err
	}
	if t.purpose == purposeReset {
		m.perAccount.clear(accountKey(t.email))
	} else {
		m.endSetup()
	}
	if err := m.signIn(ctx, row); err != nil {
		return Me{}, err
	}
	return m.meOf(row.user()), nil
}

// cmpErr is err, or other when err is nil.
func cmpErr(err, other error) error {
	if err != nil {
		return err
	}
	return other
}

// isInvalid reports whether err is a 422.
func isInvalid(err error) bool {
	var e *warp.Error
	return errors.As(err, &e) && e.Status == http.StatusUnprocessableEntity
}

// invite makes an invitation of an address with a role no higher than the
// inviter's, replacing an earlier one for the address, and mails its link.
// An earlier invitation is replaced only as removing it could be: one for a
// role above the inviter's is theirs to keep.
func (m *module) invite(ctx context.Context, in InviteInput) (Invitation, error) {
	actor, _ := UserOf(ctx)
	if m.registration == registrationClosed {
		return Invitation{}, warp.Forbidden(errNoInvitations)
	}
	if above(in.Role, actor.Role) {
		return Invitation{}, warp.Forbidden(fmt.Sprintf("Your account cannot give the role %s.", in.Role))
	}
	email := normalEmail(in.Email)
	if _, found, err := m.find(ctx, email); err != nil {
		return Invitation{}, err
	} else if found {
		return Invitation{}, warp.Invalid(map[string]string{"email": errHasAccount})
	}
	higher := func(ctx context.Context) error {
		old, found, err := m.invitations.First(ctx, "email", email)
		if err == nil && found && above(old.Role, actor.Role) {
			err = warp.Forbidden(errHigherInvitation)
		}
		return err
	}
	if err := higher(ctx); err != nil {
		return Invitation{}, err
	}
	if m.invitePerUser.take(strconv.FormatInt(actor.ID, 10)) == nil {
		return Invitation{}, warp.TooMany(errTooManyInvites)
	}
	token := newToken()
	row := invitationRow{Email: email, Role: in.Role, InvitedBy: actor.ID, ExpiresAt: m.now().Add(inviteLasts), TokenHash: tokenHash(token)}
	msg, err := m.mails.invite.Message(email, mailData{Host: m.host, Link: m.link(token), Role: in.Role})
	if err != nil {
		return Invitation{}, err
	}
	err = m.db.Tx(ctx, func(ctx context.Context) error {
		if err := higher(ctx); err != nil { // again, as another inviter may have come between
			return err
		}
		if _, err := m.invitations.DeleteBy(ctx, "email", email); err != nil {
			return err
		}
		if err := m.invitations.Create(ctx, &row); err != nil {
			return err
		}
		return m.mail.Send(ctx, msg)
	})
	if err != nil {
		return Invitation{}, err
	}
	return row.invitation(), nil
}

// listInvitations answers the invitations, newest first unless q sorts them,
// q's search over the address.
func (m *module) listInvitations(ctx context.Context, q data.Query) (data.List[Invitation], error) {
	l, err := m.invitations.List(ctx, q)
	if err != nil {
		return data.List[Invitation]{}, err
	}
	out := data.List[Invitation]{Items: make([]Invitation, 0, len(l.Items)), Total: l.Total, Page: l.Page, PerPage: l.PerPage}
	for _, r := range l.Items {
		out.Items = append(out.Items, r.invitation())
	}
	return out, nil
}

// removeInvitation ends an invitation, whose link then works no more. As on
// every users route, an actor acts only on one whose role is not above
// their own.
func (m *module) removeInvitation(ctx context.Context, in data.ID) (warp.None, error) {
	actor, _ := UserOf(ctx)
	inv, err := m.invitations.Get(ctx, in.ID)
	if err != nil {
		return warp.None{}, err
	}
	if above(inv.Role, actor.Role) {
		return warp.None{}, warp.Forbidden(errHigherInvitation)
	}
	return warp.None{}, m.invitations.Delete(ctx, in.ID)
}
