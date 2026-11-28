---
title: Accounts
description: People other than the owner - who may join, invitations, sign-up, password reset by a mailed link, and the routes that manage users.
order: 26
---

On this page you learn how people other than the owner get an account: by an invitation an owner or administrator sends, by signing up when the app allows it, and how anyone gets back in with a password reset. Each is a link the [mail](mail.md) module sends, to one page that sets a password and signs in. Signing in, roles and sessions are on the [Login](login.md) page.

## Who may join

`AUTH_REGISTRATION` in `.env` says who may ask for an account:

| Value | What it allows |
| --- | --- |
| `invite` (the default) | Invitations only: an owner or administrator invites an address with a role |
| `open` | Invitations, and anyone may ask for a sign-up link; they join as a `viewer` |
| `closed` | Neither: users come from `auth.AddUser` and the setup link |

Any other value stops the app: `AUTH_REGISTRATION is "maybe"; set it to open, invite or closed in .env.` `GET /api/auth/settings` answers `{"registration": "invite", "api_keys": false}`, which the login page reads to show a link to sign up.

**`open` suits only an app whose records every account may read.** A `viewer` has `*:read`, so a stranger who signs up can read every record of every resource guarded by `auth.Per`. Leave it at `invite` unless that is what you want; an owner can raise a viewer's role on `/users`.

## The mailed link

Every link is `<APP_URL>/password#token=<token>`: 32 random bytes in base64url, after `#` so it reaches no proxy's log or `Referer`, kept only as its SHA-256, and good once. A reset lasts 1 hour, a sign-up 24 hours, an invitation 7 days. An address has one reset or sign-up link at a time, the newest, and one invitation. `POST /api/auth/token` with `{"token": …}` says what a link is for, `{"purpose": "invite", "email": …, "role": "member"}`, and `POST /api/auth/password` with the token and a password (8 to 200 characters) sets it and signs in, answering `Me`. A link that is unknown, used, past its time, or an invitation whose inviter is gone or now below the invited role, is 403 `This link is not valid any more; ask for a new one.`

The `/password` page `weft create` writes reads the token, takes it out of the address, says what the link is for, and asks for the password.

## Invitations

| Name | Route | Needs |
| --- | --- | --- |
| `invitations` | `GET /api/auth/invitations`, paged, `q` over the email | `users:read` |
| `invite` | `POST /api/auth/invitations`, `{"email": …, "role": …}` | `users:invite` |
| `remove_invitation` | `DELETE /api/auth/invitations/{id}` | `users:invite` |

An invitation mails `You are invited to <host>` before it answers 201, so a mail server that refuses says so. The role may be no higher than the inviter's (403 `Your account cannot give the role owner.`); an address with an account is 422 on `email`, `This address already has an account.`; a new invitation to an address replaces the old one; an inviter may send 50 an hour (429 `Too many invitations; wait an hour and try again.`). Under `closed`: 403 `This app takes no invitations; set AUTH_REGISTRATION=invite in .env.`

## Signing up and resetting

`POST /api/auth/register` (under `open` only; otherwise 403 `This app does not take sign-ups; ask for an invitation.`) and `POST /api/auth/forgot`, each with `{"email": …}`, answer 204 at once, whatever the address. After answering, `register` mails `Finish signing up to <host>` to an address with no account, and `You already have an account at <host>`, with a reset link, to one that has; `forgot` mails `Reset your password for <host>` to an address with an account, and nothing to one without. So the answer never says who has an account. A registration makes no user until its link is used, so an address is proved before an account exists, and no one signs up before the owner is set up.

A reset sets the password, ends every session of the user and deletes every API key of theirs, in one transaction: it is the way back from a stolen password, and a key the thief made would outlive it. It also clears the per-account count of failed logins ([Limits](login.md#limits)). A password change, `POST /api/auth/password/change` with `current` and `password`, is for the signed in: it ends their other sessions and keeps their API keys, which are for programs and can each be removed. A wrong `current` is 422 `This is not your current password.`, and counts as a failed login.

Mail is asked for at most 3 times an hour per address, and 10 times an hour per client (429 `Too many emails asked for; wait an hour and try again.`); `token` and `password` 30 times a minute per client. At most 32 mailings run at once, each for at most 30 seconds; past that every address gets 503 `The server is busy sending mail; try again in a moment.`

## Users

| Name | Route | Needs |
| --- | --- | --- |
| `users` | `GET /api/auth/users`, paged; `sort` on `email`, `name`, `role`, `created_at`; `q`; `filter[role]` | `users:read` |
| `update_user` | `PATCH /api/auth/users/{id}`, `{"role": …}` | `users:edit` |
| `remove_user` | `DELETE /api/auth/users/{id}` | `users:delete` |
| `end_user_sessions` | `DELETE /api/auth/users/{id}/sessions` | `users:edit` |
| `update_me` | `PATCH /api/auth/me`, `{"name": …}` | a session |

An actor acts only on a user whose role is not above theirs, and gives no role above theirs (403 `Your account cannot change a user with a higher role.`). No one changes their own role or removes themselves (409 `You cannot change your own role or remove yourself; ask another owner or administrator.`), so an owner always remains. A role change applies to the user's next request. Removing a user removes their sessions, API keys and the invitations they sent, in one transaction.

## The pages

`weft create` writes them, each an island over the generated client:

- `/register` and `/forgot` ask for a link and then say `If that address can be used, an email is on its way; open the link in it within a day.` (within the hour for a reset);
- `/password` sets the password a link is for, and signs in;
- `/account`, for whoever is signed in: their name, a new password, their sessions with End and Sign out everywhere, and their API keys when `API_KEYS=on`;
- `/users`, in a `Guard` for `users:read`: a `DataTable` of users with `perm="users"`, whose dialog changes a role, and the invitations, for `users:invite`.
