// Package mailtest reads the mail a test's app sent through the mail
// module's log driver, which keeps the last 20 messages: MAIL unset or log.
package mailtest

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/flakerimi/weft/warp"
	wmail "github.com/flakerimi/weft/warp/mail"
)

// Last returns the newest message to to that the app's log driver kept. It
// fails the test when the app has no mail module or kept no message to to.
// Addresses are compared without their names and in any case.
func Last(t testing.TB, app *warp.App, to string) wmail.Message {
	t.Helper()
	m := wmail.For(app)
	if m == nil {
		t.Fatalf("mailtest: the app has no mail module; list mail.Module() in its modules.")
	}
	want := address(to)
	kept := m.Kept()
	for i := len(kept) - 1; i >= 0; i-- {
		if address(kept[i].To) == want {
			return kept[i]
		}
	}
	t.Fatalf("mailtest: the app kept no message to %s among its last %d; check that it sends one, and that MAIL is log or unset.", to, len(kept))
	return wmail.Message{}
}

// address is the lower-cased address of an email address as To holds it.
func address(s string) string {
	if a, err := mail.ParseAddress(s); err == nil {
		s = a.Address
	}
	return strings.ToLower(strings.TrimSpace(s))
}

// Token returns the token a mailed link carries: what follows "#token=" in
// msg.Text, up to the first character a base64url token does not have. It
// fails the test when the text has none.
func Token(t testing.TB, msg wmail.Message) string {
	t.Helper()
	_, rest, ok := strings.Cut(msg.Text, "#token=")
	end := strings.IndexFunc(rest, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	})
	if end >= 0 {
		rest = rest[:end]
	}
	if !ok || rest == "" {
		t.Fatalf("mailtest: the message to %s (%q) has no #token= in its text; check that it carries the link.", msg.To, msg.Subject)
	}
	return rest
}
