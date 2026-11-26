package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/flakerimi/weft/warp"
)

// The guard's answers.
const (
	errNoSession = "Sign in to do that."
	errCannot    = "Your account cannot do that."
)

type moduleKey struct{}

type userKey struct{}

// carry puts the module on the context of every API request, for Can's guard
// to find, since a route option is made before any app exists.
func (m *module) carry(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), moduleKey{}, m)))
	})
}

// Can is the route option that records permission on the route and guards
// it: a request without a session is answered 401, and one whose user's role
// lacks the permission 403. The user is read from the users table on every
// request, so a removed user is out at once; UserOf gives it to the handler.
// The role owner has every permission, and no other role has any yet.
func Can(permission string) warp.RouteOption {
	return warp.Options(warp.Permission(permission), warp.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m, _ := r.Context().Value(moduleKey{}).(*module)
			if m == nil {
				warp.WriteError(w, r, errors.New("auth: a route guarded by auth.Can was asked with no auth module; list auth.Module() in app/modules.go."))
				return
			}
			u, err := m.signedIn(r)
			if err != nil {
				warp.WriteError(w, r, err)
				return
			}
			if !allowed(u.Role, permission) {
				warp.WriteError(w, r, warp.Forbidden(errCannot))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
		})
	}))
}

// allowed reports whether role has permission: the owner has every one.
func allowed(role, permission string) bool { return role == "owner" }

// UserOf returns the signed-in user of a route guarded by Can, read when the
// request came in.
func UserOf(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}

// Per gives a resource a permission per action, for data.Resource: list and
// show need <resource>:read, create <resource>:create, update
// <resource>:edit and remove <resource>:delete; any other action
// <resource>:<action>.
func Per(resource string) func(action string) []warp.RouteOption {
	return func(action string) []warp.RouteOption {
		verb := action
		switch action {
		case "list", "show":
			verb = "read"
		case "update":
			verb = "edit"
		case "remove":
			verb = "delete"
		}
		return []warp.RouteOption{Can(resource + ":" + verb)}
	}
}

// signedIn returns the user of r's session cookie, or the 401 when there is
// no cookie, it is not good, or its user is gone.
func (m *module) signedIn(r *http.Request) (User, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return User{}, warp.Unauthorized(errNoSession)
	}
	id, ok := sessionID(m.secret, c.Value, time.Now())
	if !ok {
		return User{}, warp.Unauthorized(errNoSession)
	}
	row, err := m.users.Get(r.Context(), id)
	var e *warp.Error
	if errors.As(err, &e) && e.Status == http.StatusNotFound {
		return User{}, warp.Unauthorized(errNoSession)
	}
	if err != nil {
		return User{}, err
	}
	return row.user(), nil
}
