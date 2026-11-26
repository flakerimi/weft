package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/flakerimi/weft/warp"
)

// errSetupDone is returned inside setup's transaction when a user exists.
var errSetupDone = errors.New("a user exists")

// setup makes the owner, AUTH_OWNER_EMAIL with the role owner, and signs the
// owner in. It needs the token of the logged link and works while no user
// exists: the count and the insert run in one transaction, and the unique
// email is the last guard, so two setups at once make one owner.
func (m *module) setup(ctx context.Context, in SetupInput) (User, error) {
	if !m.tokenIs(in.Token) {
		return User{}, warp.Forbidden(errNotSetup)
	}
	hash, err := m.hash(ctx, in.Password) // before the transaction, which holds the write lock
	if err != nil {
		return User{}, err
	}
	row := userRow{Email: m.owner, Role: "owner", PasswordHash: hash}
	err = m.db.Tx(ctx, func(ctx context.Context) error {
		n, err := m.users.Count(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return errSetupDone
		}
		if err := m.users.Create(ctx, &row); err != nil {
			return err
		}
		// Signed in with the row as stored, whose created_at the session
		// names: a database that keeps fewer digits than Go rounds it. Read
		// here, in the transaction, so that a failure leaves no owner.
		row, err = m.users.Get(ctx, row.ID)
		return err
	})
	var e *warp.Error
	switch {
	case errors.Is(err, errSetupDone),
		errors.As(err, &e) && e.Status == http.StatusUnprocessableEntity: // someone else just became the owner
		m.endSetup()
		return User{}, warp.Forbidden(errNotSetup)
	case err != nil:
		return User{}, err
	}
	m.endSetup()
	m.signIn(ctx, row)
	return row.user(), nil
}
