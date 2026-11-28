---
title: Mail
description: The mail module - printed to the log while you build, sent over SMTP once configured, from a queue in memory; templates for text and HTML; and mailtest for tests.
order: 28
---

On this page you learn how an app sends mail with the `mail` module, `github.com/flakerimi/weft/warp/mail`. The [login](login.md) mails its invitation, sign-up and reset links through it, and your own modules can too. While you build, it prints each message to the app's log; set `MAIL=smtp` and it sends them.

## Setting it up

An app `weft create` makes lists `mail.Module()` before `auth.Module()` in `app/modules.go`. It reads `.env` when the app starts, and connects to nothing then, so an app starts while its mail server is down:

| Key | What it is | Default |
| --- | --- | --- |
| `MAIL` | `log` prints each message; `smtp` sends it | `log` |
| `MAIL_HOST` | The mail server, such as `smtp.example.com` | none: required under `smtp` |
| `MAIL_PORT` | Its port, 1 to 65535 | `587` |
| `MAIL_USER`, `MAIL_PASSWORD` | Both to sign in, or neither for a server that takes mail without | none |
| `MAIL_FROM` | Who mail comes from, such as `Shop <noreply@shop.example.com>` | under `log`, `noreply@` and `APP_URL`'s host; required under `smtp` |

Each mistake is its own sentence, all reported together, such as `MAIL_HOST is not set; set it to your mail server, such as smtp.example.com.` Under `log` the app says at start `mail is printed, not sent; set MAIL=smtp in .env to send it.`

## The log driver

`MAIL=log` prints `mail to <to>: <subject>` and the text, each line indented two spaces, through Go's standard logger, so a link the login mails is in the app's log to open by hand. With the [middleware](middleware.md) module listed, the log is one line per entry, so the message's lines are written as `\n` within it; a link ends before the `\n`. It keeps the last 20 messages for `mailtest`. A printed link's token is printed as it is: the log must be as private as the mail would be.

## Sending over SMTP

`MAIL=smtp` puts each message on a queue in memory and returns, so a route that mails answers at once whatever the mail server does. One sender sends them in turn: 10 seconds to connect, 30 seconds a try, three tries, 5 and then 30 seconds apart. A 4xx answer, a connection that cannot be made or breaks off, and a try that runs out of time are tried again; a 5xx answer and a certificate that does not verify are not. A message that could not go is one error line naming its recipient and subject, never its body. The queue holds 100 messages, waiting or being sent; the 101st is refused with the 503 `The server is busy sending mail; try again in a moment.` When the app stops, the queue is sent for as long as the stop allows, and then `mail: 3 messages were not sent, since the app stopped.`

Security comes from the port. `MAIL_PORT=465` is TLS from the first byte; any other port must offer STARTTLS before anything else, the password included, is sent. TLS is at least 1.2, and the certificate must be valid for `MAIL_HOST`. A server without STARTTLS is refused: `the mail server at smtp.example.com:587 offers no STARTTLS; use port 465 or a server that does.`

**The one exception is a loopback host**, `localhost` or an address such as `127.0.0.1` or `::1`, written as such (the name is not looked up). There a relay without STARTTLS is used as it is, `MAIL_USER` and `MAIL_PASSWORD` included. A relay on the same machine is often a sidecar that adds the TLS itself, or an SSH tunnel's end, and the message never leaves the machine in the clear.

## Sending from a module

`mail.For(app)` is the app's `*mail.Mailer`; without the module, the app has the mistake that says to list it. `Send` checks a message before it is queued: one `To` address, a `Subject` on one line of at most 200 bytes, a `Text` every message needs, and `Text` and `HTML` of at most 256 KiB each. A refusal is an error saying what to fix, such as `mail: the message has no Subject; give it one.` On the wire it is `multipart/alternative`, the text and the HTML, each quoted-printable, or the text alone when `HTML` is `""`.

A `Template` makes messages of one kind from data: the subject and the text are `text/template`, the HTML `html/template`, which escapes what the data puts in it, and a key the data lacks is a mistake, not an empty value. Parse it once, in `Register`, as `warp/mail`'s own test does:

```go
tpl, err := mail.NewTemplate(
	"Reset your password for {{.App}}",
	"Hello {{.Name}},\n\nOpen {{.Link}} to reset it.",
	"<p>Hello {{.Name}},</p><p><a href=\"{{.Link}}\">Reset it</a></p>",
)
// later, in a handler:
msg, err := tpl.Message("ann@example.com", map[string]string{"App": "Shop", "Name": "Ann", "Link": link})
if err == nil {
	err = mail.For(app).Send(ctx, msg)
}
```

## Testing

`mailtest`, `github.com/flakerimi/weft/warp/mail/mailtest`, reads what a [warptest](warp.md#testing-a-module) app's log driver kept. `mailtest.Last(t, app, to)` waits up to 5 seconds for a message to `to` it has not returned before, and returns the newest; keep what it returns, since asking again for the same message waits and fails. `mailtest.Token(t, msg)` is the token after `#token=` in a message's text, and `mailtest.Wait(t, app, n)` waits until n messages were printed in all. From the shop's test:

```go
msg := mailtest.Last(t, c.App, email)
u := c.Fork()
u.Post("/api/auth/password", map[string]string{"token": mailtest.Token(t, msg), "password": "a password of " + email}).Want(http.StatusOK)
```

## Everything it exports

```go
package mail // github.com/flakerimi/weft/warp/mail

func Module() warp.Module
func For(app *warp.App) *Mailer
type Message struct{ To, Subject, Text, HTML string }
func (m *Mailer) Send(ctx context.Context, msg Message) error
func (m *Mailer) Kept() []Message // the log driver's last 20, oldest first
func (m *Mailer) Printed() int    // how many the log driver printed in all
func NewTemplate(subject, text, html string) (*Template, error)
func (t *Template) Message(to string, data any) (Message, error)

package mailtest // github.com/flakerimi/weft/warp/mail/mailtest

func Last(t testing.TB, app *warp.App, to string) mail.Message
func Wait(t testing.TB, app *warp.App, n int)
func Token(t testing.TB, msg mail.Message) string
```
