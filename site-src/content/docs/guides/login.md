---
title: Login
description: The auth module - the owner's setup link, sessions kept in a table, four roles with permissions, API keys, useSession, perm and Guard in Vue, the limits on guessing, and what it does not protect against.
order: 26
---

On this page you learn how an app signs people in with the `auth` module, `github.com/flakerimi/weft/warp/auth`: the owner is set up once, everyone signs in with an email and a password, and each sign-in is a session kept in a table and named by a signed cookie, which guarded routes check on every request. People other than the owner come by invitation or sign-up, which [Accounts](accounts.md) covers. What the login does not do is at the [end of this page](#what-it-does-not-protect-against).

## Setting it up

An app `weft create` makes has all of this: `auth.Module()`, a fresh `AUTH_SECRET` in `.env`, and the pages below. The login keeps its users through [Data](data.md) and mails its links through [Mail](mail.md), so `app/modules.go` lists `data.Module(), mail.Module(), auth.Module()` in that order; without mail the app says `warp: module "auth" needs the "mail" module; list mail.Module() before it in app/modules.go.` It reads these keys from `.env` when the app starts:

| Key | What it is | Default |
| --- | --- | --- |
| `AUTH_SECRET` | The key that signs the session cookie: 32 bytes or more, random | none: the app does not start |
| `AUTH_SESSION_DAYS` | How many days a sign-in lasts, 1 to 365 | `30` |
| `AUTH_OWNER_EMAIL` | The owner's address | none: the app does not start |
| `AUTH_REGISTRATION` | Who may ask for an account: `invite`, `open` or `closed` ([Accounts](accounts.md#who-may-join)) | `invite` |
| `API_KEYS` | `on` lets other programs sign in with a key | `off` |

A secret that is too short stops the app with `warp: module "auth" could not start: AUTH_SECRET is 5 bytes; set it to 32 or more random ones in .env, for example from openssl rand -base64 48.` Changing the secret ends every session.

## The owner

While no user exists, the app logs a one-time link at each start, `Set up the owner at http://localhost:8080/setup#token=…`. It begins with `APP_URL` (see [Configuration](warp.md#configuration)); under `weft dev` that is the page server's address. The token is random, kept only in memory, made again at each start while no user exists, and useless once one does. It is after `#`, so it reaches no proxy's log and no `Referer`. The `/setup` page sends it with a password, 8 to 200 characters, to `setup`, which makes the user `AUTH_OWNER_EMAIL` with the role `owner` and signs them in. A wrong token is a 403, `This setup link is not valid; restart the app to get a new one.`, and the link once a user exists is a 403 that says `The owner is set up already; sign in at /login.`

## Signing in and sessions

| Name | Route | Answers |
| --- | --- | --- |
| `login` | `POST /api/auth/login`, with `{"email": …, "password": …}` | 200 and `Me`, with the session cookie; 401 `That email and password do not match.` for a wrong email or a wrong password alike |
| `logout` | `POST /api/auth/logout` | 204: this session ends, and the cookie is cleared |
| `logout_everywhere` | `POST /api/auth/logout/everywhere` | 204: every session of the user ends |
| `me` | `GET /api/auth/me` | 200 and `Me`, or 401 `Sign in to do that.` |
| `sessions` | `GET /api/auth/sessions` | the user's sessions, newest first, `current` on this one |
| `end_session` | `DELETE /api/auth/sessions/{id}` | 204; another user's is 404 `No session of yours has id 7.` |

`Me` is the user and what their role may do: `{"id":1,"created_at":…,"updated_at":…,"email":"owner@example.com","name":"","role":"owner","permissions":["products:create",…]}`. The password's hash is kept beside the user in the `users` table, and never answered. The tables `users`, `sessions`, `auth_tokens`, `invitations` and `api_keys`, and the type name `User`, are the login's: an app's own store on one of them is told to take another table, and its own `User` another name.

A session is a row of `sessions`: the address and browser it signed in from, when, when it was last seen (written at most every 5 minutes), and the SHA-256 of a random token. The cookie is `<id>.<token>.<mac>`, the mac an HMAC-SHA256 with `AUTH_SECRET`, so a forged cookie costs no database read. Over plain HTTP it is `session=…; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax`; over HTTPS it is `__Host-session` and `Secure`. Behind a proxy that ends TLS, set `TRUST_PROXY=on` ([Behind a proxy](warp.md#behind-a-proxy)). A session lasts `AUTH_SESSION_DAYS` from sign-in, however it is used; a user has at most 50, and the 51st sign-in ends the one seen longest ago. Expired sessions are removed at start and at each sign-in.

Passwords are hashed with PBKDF2-HMAC-SHA256 from Go's standard library, 600,000 iterations and a random salt; an unknown email is checked against a stand-in hash, so both wrong answers take as long.

## Roles and permissions

There are four roles, each with what the ones below it have: `viewer` has `*:read`; `member` also `*:create`, `*:edit` and `*:delete`; `administrator` also `users:read`, `users:invite`, `users:edit` and `users:delete`; `owner` has every permission. A `*` never covers `users`, so a role that edits products cannot edit accounts. `auth.Grant(role, permissions...)` adds to a role and every role above it, as `auth.Module(auth.Grant(auth.RoleMember, "reports:export"))`.

`auth.Can("products:edit")` is a route option that records the permission on the route and checks it: a request with no session is 401 `Sign in to do that.`, and one whose role lacks the permission 403 `Your account cannot do that.` The user is read on every guarded request, so a removed user or a changed role applies to the next one; the handler gets it with `auth.UserOf(ctx)`. `auth.Allows(app, role, permission)` says what the guard would, and `me`'s `permissions` comes from the same function. A route guarded so in an app without `auth.Module()` is a registration mistake.

```go
warp.Get(app, "/goods/cheapest", m.cheapest, warp.Name("cheapest"), auth.Can("goods:read"))
```

`auth.Per("products")` gives [`data.Resource`](data.md#the-five-routes) a permission per action: `products:read` to list and show, `products:create`, `products:edit` and `products:delete`. The shop's whole products module is that line:

```go
data.Resource(app, "/products", store, data.PerAction(auth.Per("products")))
```

For tools and tests, `auth.AddUser(ctx, app, email, password, role)` makes a user and `auth.RemoveUser(ctx, app, id)` removes one, with their sessions, keys and the invitations they sent; both work once the app has started.

## API keys

With `API_KEYS=on`, a user makes keys for other programs on `/account`, or with `create_key` (`POST /api/auth/keys`, `{"name": …, "role": …, "days": …}`), which answers the whole key, `wk_<id>_<secret>`, once; only its SHA-256 is kept. A program sends it as `Authorization: Bearer wk_…`, and the key acts as its user with the lower of its role and the user's role now. It reaches `me` and the routes guarded by a permission, but none that manages accounts or keys: with a key, `invite`, `remove_invitation`, `update_user`, `remove_user`, `end_user_sessions` and the key routes answer 403 `An API key cannot manage accounts or keys; sign in to do that.` So a stolen key cannot make keys, give its thief an account, read sessions or change a password; it still reads what its role reads, so give a key the least role it needs. A key lasts 1 to 365 days (90 by default) and has a name of up to 60 characters; a user has at most 20. `keys` lists the user's own, and `remove_key` ends one at once. Without `API_KEYS=on`, a key is refused with 401 `This app does not take API keys.`

## Limits

Each login try is counted before anything is looked up or hashed: 5 a minute for one email from one address, 30 a minute from one address across every email, and 10 failed tries in 10 minutes for one email across every address. Past the first two the answer is 429 `Too many tries; wait a minute and try again.`; past the third, `Too many tries for this account; wait 10 minutes, or reset the password.` So ten wrong tries in ten minutes, from anywhere, keep an account from signing in with its password for that time; a reset link from `/forgot` still signs the person in, and clears the count. A success does not clear it, so someone who knows the password cannot clear it for a guesser elsewhere. Every email is counted, with an account or without, so the answer says nothing. The address is [`ClientKey`](warp.md#behind-a-proxy): an IPv4 address, or the /64 an IPv6 one is in.

At most as many hashes run at once as Go has processors (`GOMAXPROCS`). A login that finds none free within 3 seconds is answered 503 `The server is busy signing people in; try again in a moment.`, and costs no try.

## In the browser

Pages are built with no session, so who is signed in is asked in the browser. `@weft/ui` has it:

- **`useSession()`** asks `GET /api/auth/me` once per page load, however many islands call it, and gives `session` (`{ user, loading, problem }`), `ready`, `can(permission)`, `signOut()` and `refresh()`. A 401 is no user; a server out of reach is a `problem`, and sends no one away.
- **`perm`** on `Button`, and on `DataTable` as a resource name (`perm="products"`), draws the button, or New, Edit and Delete, only for a user with the permission. It hides; the route still decides.
- **`<Guard perm="users:read">`** shows what it holds to a user with the permission, `denied` to one without, and sends a visitor with no session to `/login?next=` this page. `nextPath(next, origin)` is where the login form goes after, only on this site.
- **`sessionEnded()`** is for a call answered 401: it asks again and sends the visitor to sign in. The template's `formProblem(e, sessionEnded)` calls it. A hidden tab learns of a sign-out elsewhere when it is shown again; a window that stays visible beside it learns at its next request.

`weft create` writes `/login`, `/setup`, `/register`, `/forgot`, `/password`, `/account` and `/users`, each an island over the [generated client](types-from-go.md), and `web/session-check.ts`, which fails `pnpm typecheck` if the generated `Me` stops fitting `@weft/ui`'s `SessionUser`. They are the reference: copy them as they are. A `Button` with `perm` in a page, outside an island, never shows, since only an island runs in the browser.

## Testing

`authtest.Owner(t, c)`, from `github.com/flakerimi/weft/warp/auth/authtest`, sets up the owner of a [warptest](warp.md#testing-a-module) app with the password `authtest.Password`, and leaves the client signed in. `authtest.As(t, c, auth.RoleViewer)` makes a user of that role and returns a fork of the client signed in as them. The mail a test's app sends is read with [mailtest](mail.md#testing).

## Everything it exports

```go
package auth // github.com/flakerimi/weft/warp/auth

const RoleOwner, RoleAdministrator, RoleMember, RoleViewer = "owner", "administrator", "member", "viewer"
func Module(opts ...Option) warp.Module
func Grant(role string, permissions ...string) Option
func Can(permission string) warp.RouteOption
func Per(resource string) func(action string) []warp.RouteOption
func Allows(app *warp.App, role, permission string) bool
func UserOf(ctx context.Context) (User, bool)
func AddUser(ctx context.Context, app *warp.App, email, password, role string) (User, error)
func RemoveUser(ctx context.Context, app *warp.App, id int64) error
func SetupToken(app *warp.App) string // "" once a user exists

package authtest // github.com/flakerimi/weft/warp/auth/authtest

const Password = "the owner's password, for tests"
func Owner(t testing.TB, c *warptest.Client) auth.User
func As(t testing.TB, c *warptest.Client, role string) (*warptest.Client, auth.User)
```

The types the routes take and answer (`User`, `Me`, `LoginInput`, `SessionInfo`, `APIKey` and the rest) are in the generated client, as `weft gen` writes them.

## What it does not protect against

- **Stolen cookies, until the session ends.** A copied cookie works until its session is ended (signing out, `logout_everywhere`, `end_session`, a password change or reset, the user removed) or expires.
- **Whoever holds `AUTH_SECRET` and a session's token.** The secret alone signs no one in, since the table keeps only a token's hash; with a copied cookie, a session works until it ends. The secret has no rotation: changing it ends every session.
- **Limits are per process and per address.** The counts are in memory: several app processes each keep their own, a restart forgets them, and past 10,000 keys per limiter the oldest are dropped. Many addresses (a botnet) are limited together only by the per-account count, and a client holding a /48 holds many /64 keys.
- **Anyone can keep an account from signing in with its password**, with ten wrong tries in ten minutes from anywhere, or with 5 a minute from the owner's own address. A reset link still works. Behind a proxy with `TRUST_PROXY` off, every request has the proxy's address.
- **Timing the mail.** `register` and `forgot` answer at once and mail later, the same for an address with an account and one without; but a mailing for an address with an account holds one of the 32 mailing slots a little longer (a lookup, a transaction and a send, against a lookup alone). Someone who fills the other 31 and times when the last frees could in theory tell the two apart; at 3 tries an hour per address, through the network, it is impractical.
- **Spending the CPU.** A flood can fill the hashing slots; logins then wait, the owner's included, and get the 503 after 3 seconds.
- **Lying proxy headers.** With `TRUST_PROXY=on`, the address and the HTTPS check trust `X-Forwarded-For` and `X-Forwarded-Proto`, which are honest only behind a proxy that sets them.
- **Plain HTTP.** Without HTTPS the cookie is sent in the clear and is the plain `session`, which a sibling subdomain can plant. `__Host-session`, under HTTPS, closes that.
- **`TRUSTED_ORIGINS`.** Every origin listed there is fully trusted for every API request.
- **A leaked setup link.** Until the owner is set up, anyone with the logged link can become the owner; the app's log must be private, as it is for the [mail](mail.md) the `log` driver prints.
- **Missing account features.** There is no second factor, no lockout notice, no breached-password check, and no change of email. Any 8-character password is accepted.
