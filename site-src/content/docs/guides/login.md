---
title: Login
description: The auth module - one owner set up through a link in the log, a session in a signed cookie, routes guarded by permission, the limits on guessing, and what it does not protect against.
order: 25
---

On this page you learn how an app signs people in with the `auth` module, `github.com/flakerimi/weft/warp/auth`: the owner is set up once, signs in with an email and a password, and gets a session in a signed cookie, which guarded routes check on every request. It is the smallest login that is honest about security, and it has one role, the owner. What it does not do is at the [end of this page](#what-it-does-not-protect-against).

## Setting it up

`auth.Module()` keeps its users through [Data](data.md), so `app/modules.go` lists it after `data.Module()`; listed before, the app says `warp: module "auth" needs the "data" module; list data.Module() before it in app/modules.go.` It reads three keys from `.env` when the app starts:

| Key | What it is | Default |
| --- | --- | --- |
| `AUTH_SECRET` | The key that signs the session cookie: 32 bytes or more, random | none: the app does not start |
| `AUTH_SESSION_DAYS` | How many days a sign-in lasts, 1 to 365 | `30` |
| `AUTH_OWNER_EMAIL` | The owner's address | none: the app does not start |

A secret that is too short stops the app with `warp: module "auth" could not start: AUTH_SECRET is 5 bytes; set it to 32 or more random ones in .env, for example from openssl rand -base64 48.` Changing the secret ends every session.

## The owner

While no user exists, the app logs a one-time link at each start:

```text
Set up the owner at http://localhost:8080/setup#token=CNDYRHwnArVH_YCNVsRVYMuUx6CVM5ahbBIfHjcchKU
```

It begins with `APP_URL` (see [Configuration](warp.md#configuration)); under `weft dev` that is the page server's address. The token is random, kept only in memory, made again at each start while no user exists, and useless once one does. It is after `#`, so it reaches no proxy's log and no `Referer`. Until it is used no one can sign in. Your `/setup` page reads it from `location.hash` and sends it with the owner's password, 8 to 200 characters, to `setup`, which makes the user `AUTH_OWNER_EMAIL` with the role `owner` and signs them in. A token that is wrong, or comes once a user exists, is a 403: `This setup link is not valid; restart the app to get a new one.`

## The routes

| Name | Route | Answers |
| --- | --- | --- |
| `login` | `POST /api/auth/login`, with `{"email": …, "password": …}` | 200 and the user, with the session cookie; 401 `That email and password do not match.` for a wrong email or a wrong password alike |
| `logout` | `POST /api/auth/logout` | 204, clearing the cookie; 401 without a session |
| `me` | `GET /api/auth/me` | 200 and the user, or 401 `Sign in to do that.` |
| `setup` | `POST /api/auth/setup`, with `{"token": …, "password": …}` | 201 and the user, with the session cookie, or 403 |

The user is `{"id":1,"created_at":…,"updated_at":…,"email":"owner@example.com","role":"owner"}`. The password's hash is kept beside it in the `users` table, and never answered.

Passwords are hashed with PBKDF2-HMAC-SHA256 from Go's standard library, 600,000 iterations and a random salt; an unknown email is checked against a stand-in hash, so both wrong answers take as long. The cookie holds the user's id, the moment the user was made and an expiry, signed with HMAC-SHA256: over plain HTTP it is `session=…; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax`, and over HTTPS it is named `__Host-session` and is `Secure`. Behind a proxy that ends TLS, set `TRUST_PROXY=on` ([Behind a proxy](warp.md#behind-a-proxy)), or the cookie is the plain one.

## Guarding routes

`auth.Can("products:edit")` is a route option that records the permission on the route and checks it: a request with no session is answered 401 `Sign in to do that.`, and one whose user's role lacks the permission 403 `Your account cannot do that.` The owner has every permission, and no other role has any yet. The user is read from the `users` table on every guarded request, so a removed user is out at once; the handler gets it with `auth.UserOf(ctx)`.

```go
warp.Get(app, "/goods/cheapest", m.cheapest, warp.Name("cheapest"), auth.Can("goods:read"))
```

`auth.Per("products")` gives [`data.Resource`](data.md#the-five-routes) a permission per action: `products:read` to list and show, `products:create`, `products:edit` and `products:delete`. The shop's whole products module is that line:

```go
data.Resource(app, "/products", store, data.PerAction(auth.Per("products")))
```

For tools and tests, `auth.AddUser(ctx, app, email, password, role)` makes a user and `auth.RemoveUser(ctx, app, id)` removes one, whose sessions end at once; both work once the app has started.

## Limits

Each login try is counted before anything is looked up or hashed: 5 tries a minute for one email from one address, and 30 a minute from one address across every email. Past either, the answer is 429 `Too many tries; wait a minute and try again.`, until a minute after the first of those tries. A success clears the email's count. The address is [`ClientKey`](warp.md#behind-a-proxy): an IPv4 address, or the /64 an IPv6 one is in.

At most as many hashes run at once as Go has processors (`GOMAXPROCS`). A login that finds none free within 3 seconds is answered 503 `unavailable`, `The server is busy signing people in; try again in a moment.`, and costs no try.

## The pages

Pages are built with no session, so whatever depends on who is signed in asks in the browser, from an island. The shop's sign-in form is an island on `/login` that calls the [generated client](types-from-go.md) and checks the generated rules before it sends:

```vue
<template>
  <Form ref="form" :state="state" :schema="schema.LoginInput" @submit="signIn">
    <Stack :gap="4">
      <Alert v-if="problem" tone="danger" :title="problem" />
      <FormField label="Email" name="email">
        <Input v-model="state.email" type="email" autocomplete="username" />
      </FormField>
      <FormField label="Password" name="password">
        <Input v-model="state.password" type="password" autocomplete="current-password" />
      </FormField>
      <Button type="submit" label="Sign in" />
    </Stack>
  </Form>
</template>
```

`signIn` calls `api.auth.login(values)`. Its `web/session.ts` asks `api.auth.me()` once per page load for every island that wants the user, and sends a visitor with no session to `/login?next=…`. That code is the app's own for now: see [Coming next](../coming-next.md#login-and-permissions).

## Testing

`authtest.Owner(t, c)`, from `github.com/flakerimi/weft/warp/auth/authtest`, sets up the owner of a [warptest](warp.md#testing-a-module) app through `setup`, with the password `authtest.Password`, and leaves the client signed in.

## What it does not protect against

- **Stolen cookies.** Signing out only clears the cookie in that browser. A copied cookie stays good until it expires (up to `AUTH_SESSION_DAYS`), because nothing is kept on the server: there is no sign-out everywhere and no revoking one session.
- **Whoever holds `AUTH_SECRET`.** They can sign in as any user, if they also know the user's id and the nanosecond the user was made (both are in any cookie of that user). The secret has no rotation: changing it ends every session at once.
- **Replacing the database is safe for old cookies.** A cookie names its user's `created_at`, so a fresh database whose user 1 is someone else refuses an old cookie.
- **Role changes apply at once; password changes would not.** The row is read on each request, so a removed user or a changed role applies to the next request. A signed-in user whose password is changed (once that exists) stays signed in.
- **Limits are per process and per address.** The counts (5 per email and address a minute, 30 per address a minute) are in memory: several app processes each keep their own counts, and a restart forgets them; past 10,000 keys per limiter the oldest are dropped; many addresses (a botnet) are not limited together; a client holding a /48 holds many /64 keys.
- **Anyone sharing the owner's address can lock the owner out**, with 5 wrong tries a minute on the owner's email or 30 tries a minute on any emails. That includes every client when the app is behind a proxy and `TRUST_PROXY` is off, since every request then has the proxy's address.
- **Spending the CPU.** At most `GOMAXPROCS` hashes run at once, so a flood cannot take every core, but it can still fill the hashing slots. Logins then wait, the owner's included, and get the 503 after 3 seconds.
- **Lying proxy headers.** With `TRUST_PROXY=on`, the address and the HTTPS check trust `X-Forwarded-For` and `X-Forwarded-Proto`. They are only honest behind a proxy that sets them.
- **Plain HTTP.** Without HTTPS the cookie is sent in the clear and is the plain `session`, which a sibling subdomain can plant or shadow. The `__Host-session` cookie used under HTTPS closes that.
- **Same-site requests.** The core refuses a cross-origin state-changing request, same-site ones included. What same-site still allowed was cookie planting, and that is closed under HTTPS only.
- **`TRUSTED_ORIGINS`.** Every origin listed there is fully trusted for login, setup and every other API request.
- **A leaked setup link.** Until the owner is set up, anyone with the logged link can become the owner. The token is in the fragment, so it reaches no proxy access log or `Referer`, but the app's own log holds it and must be private.
- **Missing account features.** There is no second factor, no lockout notice, no password reset, no email verification and no breached-password check. Any 8-character password is accepted.
