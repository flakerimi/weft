package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

// RoleInput is the role update_user gives.
type RoleInput struct {
	Role string `json:"role" warp:"required,enum=owner|administrator|member|viewer"`
}

// ProfileInput is what update_me changes: the name alone.
type ProfileInput struct {
	Name string `json:"name" warp:"max=120"`
}

// ChangePasswordInput is the current password and the new one.
type ChangePasswordInput struct {
	Current  string `json:"current" warp:"required,max=200"`
	Password string `json:"password" warp:"required,min=8,max=200"`
}

// userRole is update_user's input: the user by its path and the role in the
// body.
type userRole struct {
	ID   int64     `path:"id"`
	Body RoleInput `body:"json"`
}

const (
	errSelf       = "You cannot change your own role or remove yourself; ask another owner or administrator."
	errHigher     = "Your account cannot change a user with a higher role."
	errNotCurrent = "This is not your current password."
)

// userRoutes adds the routes that manage users, and a user's own profile and
// password, under /auth.
func (m *module) userRoutes() {
	warp.Get(m.app, "/auth/users", m.listUsers, warp.Name("users"), Can("users:read"))
	warp.Patch(m.app, "/auth/users/{id}", m.updateUser, warp.Name("update_user"), Can("users:edit"))
	warp.Delete(m.app, "/auth/users/{id}", m.removeUserRoute, warp.Name("remove_user"), Can("users:delete"))
	warp.Delete(m.app, "/auth/users/{id}/sessions", m.endUserSessions, warp.Name("end_user_sessions"), Can("users:edit"))
	warp.Patch(m.app, "/auth/me", m.updateMe, warp.Name("update_me"))
	warp.Post(m.app, "/auth/password/change", m.changePassword, warp.Name("change_password"))
}

// listUsers answers the users, sorted by email, name, role or created_at,
// q's search over email and name, and filter[role].
func (m *module) listUsers(ctx context.Context, q data.Query) (data.List[User], error) {
	l, err := m.users.List(ctx, q)
	if err != nil {
		return data.List[User]{}, err
	}
	out := data.List[User]{Items: make([]User, 0, len(l.Items)), Total: l.Total, Page: l.Page, PerPage: l.PerPage}
	for _, r := range l.Items {
		out.Items = append(out.Items, r.user())
	}
	return out, nil
}

// target is the user id names, for an actor to act on: a 404 for one no one
// has, and the 403 for one whose role is above the actor's.
func (m *module) target(ctx context.Context, actor User, id int64) (userRow, error) {
	t, err := m.users.Get(ctx, id)
	if err != nil {
		return userRow{}, err
	}
	if above(t.Role, actor.Role) {
		return userRow{}, warp.Forbidden(errHigher)
	}
	return t, nil
}

// updateUser gives a user a role. No one changes their own role, and an
// actor acts only on a user not above them, giving a role not above their
// own; so an owner always remains. The user's next request has the role.
func (m *module) updateUser(ctx context.Context, in userRole) (User, error) {
	actor, _ := UserOf(ctx)
	if in.ID == actor.ID {
		return User{}, warp.Conflict(errSelf)
	}
	if !warp.Sent(ctx, "role") {
		return User{}, warp.Invalid(map[string]string{"role": "This is required."})
	}
	t, err := m.target(ctx, actor, in.ID)
	if err != nil {
		return User{}, err
	}
	if above(in.Body.Role, actor.Role) {
		return User{}, warp.Forbidden(errHigher)
	}
	t.Role = in.Body.Role
	row, err := m.users.Update(ctx, t.ID, &t, []string{"role"})
	if err != nil {
		return User{}, err
	}
	return row.user(), nil
}

// removeUserRoute removes a user, as RemoveUser does, by the rules of
// updateUser.
func (m *module) removeUserRoute(ctx context.Context, in data.ID) (warp.None, error) {
	actor, _ := UserOf(ctx)
	if in.ID == actor.ID {
		return warp.None{}, warp.Conflict(errSelf)
	}
	if _, err := m.target(ctx, actor, in.ID); err != nil {
		return warp.None{}, err
	}
	return warp.None{}, m.removeUser(ctx, in.ID)
}

// endUserSessions ends every session of a user not above the actor.
func (m *module) endUserSessions(ctx context.Context, in data.ID) (warp.None, error) {
	actor, _ := UserOf(ctx)
	if _, err := m.target(ctx, actor, in.ID); err != nil {
		return warp.None{}, err
	}
	_, err := m.sessions.DeleteBy(ctx, "user_id", in.ID)
	return warp.None{}, err
}

// updateMe changes the signed-in user's name, when the body sends it.
func (m *module) updateMe(ctx context.Context, in ProfileInput) (Me, error) {
	u, _, err := m.signedIn(warp.Request(ctx))
	if err != nil {
		return Me{}, err
	}
	if warp.Sent(ctx, "name") {
		row, err := m.users.Update(ctx, u.ID, &userRow{Name: in.Name}, []string{"name"})
		if err != nil {
			return Me{}, err
		}
		u = row.user()
	}
	return m.meOf(u), nil
}

// changePassword sets a new password once current is checked as a login is,
// through attempt's three counts, so it cannot be used to guess; every other
// session of the user ends, and this one stays. API keys stay too: they are
// for programs, and each can be removed; a reset ends them.
func (m *module) changePassword(ctx context.Context, in ChangePasswordInput) (warp.None, error) {
	r := warp.Request(ctx)
	u, current, err := m.signedIn(r)
	if err != nil {
		return warp.None{}, err
	}
	if _, err := m.attempt(ctx, u.Email, in.Current, m.app.ClientKey(r)); err != nil {
		var e *warp.Error
		if errors.As(err, &e) && e.Status == http.StatusUnauthorized {
			return warp.None{}, warp.Invalid(map[string]string{"current": errNotCurrent})
		}
		return warp.None{}, err
	}
	hash, err := m.hash(ctx, in.Password) // before the transaction, which holds the write lock
	if err != nil {
		return warp.None{}, err
	}
	return warp.None{}, m.db.Tx(ctx, func(ctx context.Context) error {
		if _, err := m.users.Update(ctx, u.ID, &userRow{PasswordHash: hash}, []string{"password_hash"}); err != nil {
			return err
		}
		mine, err := m.sessions.List(ctx, data.Query{Filter: map[string]string{"user_id": strconv.FormatInt(u.ID, 10)}, PerPage: maxSessions + 1})
		if err != nil {
			return err
		}
		for _, s := range mine.Items {
			if s.ID == current {
				continue
			}
			if err := m.sessions.Delete(ctx, s.ID); err != nil && !isNotFound(err) {
				return err
			}
		}
		return nil
	})
}
