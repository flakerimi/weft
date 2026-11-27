package auth

import (
	"context"
	"errors"
	"net/http"

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
// request, so a removed user is out at once and a changed role counts at
// once; UserOf gives it to the handler. What a role is allowed is Allows's
// answer. An app that lists no auth module has a registration mistake that
// says so.
func Can(permission string) warp.RouteOption {
	return warp.Options(warp.Permission(permission), warp.Requires("auth", "auth.Module()"), warp.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m, _ := r.Context().Value(moduleKey{}).(*module)
			if m == nil {
				warp.WriteError(w, r, errors.New("auth: a route guarded by auth.Can was asked with no auth module; list auth.Module() in app/modules.go."))
				return
			}
			u, _, err := m.signedIn(r)
			if err != nil {
				warp.WriteError(w, r, err)
				return
			}
			if !m.roles.allowed(u.Role, permission) {
				warp.WriteError(w, r, warp.Forbidden(errCannot))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
		})
	}))
}

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
