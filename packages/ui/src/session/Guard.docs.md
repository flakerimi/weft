---
group: session
purpose: A route guard: what it holds is shown only to a signed-in user with perm.
use: Around an island's content that only some accounts may see, such as a users page for users:read; it sends a visitor with no session to sign in and back.
avoid: For one button or action, use Button's or DataTable's perm; for what the server must refuse, guard the route with auth.Per, which Guard never replaces.
adapts:
  input: Nothing changes.
  surface: Nothing changes; the Alert and Spinner follow the surface's type scale.
  width: Nothing changes.
---
It has no stylesheet of its own: it draws `Spinner` and `Alert`.

`perm` is a permission such as `perm="users:read"`; without it, any
signed-in user passes. Who is signed in comes from `useSession()`, which asks
`GET /api/auth/me` once for the whole page, and `can(perm)` is always true for
the role `owner`.

**What it shows.** Pages are built with no session, so the server renders the
`loading` slot (a `Spinner` that says Loading), and so does the browser's
first render, so hydration always matches. Once mounted it calls
`requireSession({ perm, signIn })`:

- a visitor with no session is sent to `sign-in` (default `/login`) with
  `?next=` this page's path and query, and `loading` stays while the browser
  leaves;
- a user without `perm` sees the `denied` slot, by default an `Alert`:
  "Your account cannot see this; ask an administrator.";
- a session that could not be read (the server cannot be reached, or answers
  an error) shows why in an `Alert` with `role="alert"`, and sends no one
  away;
- otherwise the default slot, given `{ user }`, the signed-in user.

It follows the session: after `useSession().refresh()` a role that lost the
permission sees `denied`. It hides what it holds and guards nothing: the
server's routes must still check, as `auth.Per` does, and send nothing a
viewer may not read.
