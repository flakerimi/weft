package auth

import (
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/flakerimi/weft/warp"
)

// The four roles, ranked owner > administrator > member > viewer. Each has
// the permissions of the roles below it.
const (
	RoleOwner         = "owner"
	RoleAdministrator = "administrator"
	RoleMember        = "member"
	RoleViewer        = "viewer"
)

// ranked is the roles, lowest first.
var ranked = [...]string{RoleViewer, RoleMember, RoleAdministrator, RoleOwner}

// rank is role's place in ranked, or -1 for a role that is not one of the
// four.
func rank(role string) int {
	for i, r := range ranked {
		if r == role {
			return i
		}
	}
	return -1
}

// above reports whether role a ranks above role b. A role that is not one of
// the four ranks below every one that is.
func above(a, b string) bool { return rank(a) > rank(b) }

// usersResource is the resource a * never covers: accounts are granted only
// by name, so a role that edits every record cannot edit accounts.
const usersResource = "users"

// defaultGrants is what each role is given before any Grant: a viewer reads
// every resource; a member also makes, edits and deletes; an administrator
// also manages users. The owner has every permission and needs no grant.
func defaultGrants() map[string][]string {
	return map[string][]string{
		RoleViewer:        {"*:read"},
		RoleMember:        {"*:create", "*:edit", "*:delete"},
		RoleAdministrator: {"users:read", "users:invite", "users:edit", "users:delete"},
	}
}

// Option changes the auth module.
type Option func(*options)

type options struct {
	grants []grant
}

type grant struct {
	role        string
	permissions []string
}

// Grant gives role, and every role above it, the permissions: each is
// resource:action in lower case, such as reports:export, and either part may
// be *, which matches any resource but users, or any action. A mistake in a
// role or a permission is reported by the app.
func Grant(role string, permissions ...string) Option {
	return func(o *options) { o.grants = append(o.grants, grant{role, slices.Clone(permissions)}) }
}

var permissionForm = regexp.MustCompile(`^([a-z0-9_-]+|\*):([a-z0-9_-]+|\*)$`)

// roles is what each role is granted, fixed once the module is registered.
type roles struct {
	grants map[string][]string // a role's own grants, not those below it
	named  []string            // every granted permission without a *, sorted
}

// newRoles is the default grants with grants added, and a sentence, without
// warp: in front, for each mistake in them.
func newRoles(grants []grant) (*roles, []string) {
	r := &roles{grants: defaultGrants()}
	var mistakes []string
	for _, g := range grants {
		if rank(g.role) < 0 {
			mistakes = append(mistakes, `auth.Grant names the role "`+g.role+`"; use owner, administrator, member or viewer.`)
			continue
		}
		for _, p := range g.permissions {
			if !permissionForm.MatchString(p) {
				mistakes = append(mistakes, `auth.Grant has the permission "`+p+`"; write it as resource:action in lower case, such as products:export.`)
				continue
			}
			r.grants[g.role] = append(r.grants[g.role], p)
		}
	}
	for _, list := range r.grants {
		for _, p := range list {
			if !strings.Contains(p, "*") {
				r.named = append(r.named, p)
			}
		}
	}
	slices.Sort(r.named)
	r.named = slices.Compact(r.named)
	return r, mistakes
}

// allowed is the one answer to whether role has permission: the guard's, and
// what Me.Permissions lists. The owner has every permission. Any other of the
// four has what is granted to it and to the roles below it; a role that is
// not one of the four has none.
func (r *roles) allowed(role, permission string) bool {
	if role == RoleOwner {
		return true
	}
	n := rank(role)
	if n < 0 {
		return false
	}
	for _, below := range ranked[:n+1] {
		for _, g := range r.grants[below] {
			if covers(g, permission) {
				return true
			}
		}
	}
	return false
}

// covers reports whether the grant res:act matches permission: res is its
// resource, or * and its resource is not users; act is its action, or *.
func covers(grant, permission string) bool {
	res, act, _ := strings.Cut(grant, ":")
	pres, pact, ok := strings.Cut(permission, ":")
	if !ok || pres == "" || pact == "" {
		return false
	}
	return (res == pres || res == "*" && pres != usersResource) && (act == pact || act == "*")
}

// Allows reports whether role has permission in app, as the guard of
// auth.Can decides it, for tools and tests. It is false for an app without
// the auth module.
func Allows(app *warp.App, role, permission string) bool {
	m := of(app)
	return m != nil && m.roles != nil && m.roles.allowed(role, permission)
}

// known is every permission Me.Permissions may list: those the app's routes
// record and those named in a grant without a *, sorted. Routes are all
// registered before the app serves, so it is read once.
type known struct {
	once sync.Once
	list []string
}

// permissions is what role is allowed of the permissions the app knows,
// sorted, without repeats, and never nil.
func (m *module) permissions(role string) []string {
	m.known.once.Do(func() {
		list := slices.Clone(m.roles.named)
		for _, r := range m.app.Routes() {
			if r.Permission != "" {
				list = append(list, r.Permission)
			}
		}
		slices.Sort(list)
		m.known.list = slices.Compact(list)
	})
	out := []string{}
	for _, p := range m.known.list {
		if m.roles.allowed(role, p) {
			out = append(out, p)
		}
	}
	return out
}

// Me is the signed-in user and the permissions of their role, which a page
// reads to show what the user may do.
type Me struct {
	User
	Permissions []string `json:"permissions"` // sorted, never null
}

// meOf is the Me of a user.
func (m *module) meOf(u User) Me { return Me{User: u, Permissions: m.permissions(u.Role)} }
