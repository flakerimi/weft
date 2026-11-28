package auth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/flakerimi/weft/warp/mail"
)

// mailData is what the login's mails are filled with: APP_URL's host, the
// link that carries the token after #, and for an invitation its role.
type mailData struct {
	Host string
	Link string
	Role string
}

// The four mails. Each says what its link does, how long it lasts and that
// the mail can be ignored; the HTML says the same with the link as a link.
// No mail holds a question mark, so a link in it is never read as having a
// query.
const (
	inviteSubject = `You are invited to {{.Host}}`
	inviteText    = `You are invited to {{.Host}}, as {{.Role}}.

Open this link to choose a password and sign in:

{{.Link}}

The link works once, for 7 days. If you did not expect this invitation, you can ignore this mail.
`
	inviteHTML = `<p>You are invited to {{.Host}}, as {{.Role}}.</p>
<p><a href="{{.Link}}">Choose a password and sign in</a></p>
<p>The link works once, for 7 days. If you did not expect this invitation, you can ignore this mail.</p>
`

	registerSubject = `Finish signing up to {{.Host}}`
	registerText    = `Someone, we hope you, asked to sign up to {{.Host}} with this address.

Open this link to choose a password and finish signing up:

{{.Link}}

The link works once, for 24 hours. If it was not you, you can ignore this mail; no account is made until the link is used.
`
	registerHTML = `<p>Someone, we hope you, asked to sign up to {{.Host}} with this address.</p>
<p><a href="{{.Link}}">Choose a password and finish signing up</a></p>
<p>The link works once, for 24 hours. If it was not you, you can ignore this mail; no account is made until the link is used.</p>
`

	knownSubject = `You already have an account at {{.Host}}`
	knownText    = `Someone, we hope you, asked to sign up to {{.Host}} with this address, which already has an account.

If you have forgotten its password, open this link to choose a new one and sign in:

{{.Link}}

The link works once, for 1 hour. If it was not you, you can ignore this mail; your password stays as it is.
`
	knownHTML = `<p>Someone, we hope you, asked to sign up to {{.Host}} with this address, which already has an account.</p>
<p><a href="{{.Link}}">Choose a new password and sign in</a></p>
<p>The link works once, for 1 hour. If it was not you, you can ignore this mail; your password stays as it is.</p>
`

	resetSubject = `Reset your password for {{.Host}}`
	resetText    = `Someone, we hope you, asked to reset the password of your account at {{.Host}}.

Open this link to choose a new password and sign in:

{{.Link}}

The link works once, for 1 hour. If it was not you, you can ignore this mail; your password stays as it is.
`
	resetHTML = `<p>Someone, we hope you, asked to reset the password of your account at {{.Host}}.</p>
<p><a href="{{.Link}}">Choose a new password and sign in</a></p>
<p>The link works once, for 1 hour. If it was not you, you can ignore this mail; your password stays as it is.</p>
`
)

// mails is the four templates, parsed once, in Register.
type mails struct {
	invite, register, known, reset *mail.Template
}

func parseMails() (*mails, error) {
	var m mails
	for _, t := range []struct {
		to                  **mail.Template
		name                string
		subject, text, html string
	}{
		{&m.invite, "invitation", inviteSubject, inviteText, inviteHTML},
		{&m.register, "sign-up", registerSubject, registerText, registerHTML},
		{&m.known, "known address", knownSubject, knownText, knownHTML},
		{&m.reset, "reset", resetSubject, resetText, resetHTML},
	} {
		tpl, err := mail.NewTemplate(t.subject, t.text, t.html)
		if err != nil {
			return nil, fmt.Errorf("auth: the %s mail does not parse (%v); fix its template in warp/auth/mails.go.", t.name, err)
		}
		*t.to = tpl
	}
	return &m, nil
}

// setURL keeps APP_URL, without a trailing slash, and its host, for the
// links and subjects.
func (m *module) setURL(appURL string) {
	m.url = strings.TrimSuffix(appURL, "/")
	if u, err := url.Parse(m.url); err == nil && u.Host != "" {
		m.host = u.Host
	} else {
		m.host = m.url
	}
}

// link is the page that takes a mailed token, with the token after #, which
// a browser never sends, so it reaches no access log or Referer.
func (m *module) link(token string) string { return m.url + "/password#token=" + token }
