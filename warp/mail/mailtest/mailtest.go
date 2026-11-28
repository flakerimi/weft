// Package mailtest reads the mail a test's app sent through the mail
// module's log driver, which keeps the last 20 messages: MAIL unset or log.
// Last and Wait wait up to 5 s, since an app may send from a goroutine of its
// own after it has answered.
package mailtest

import (
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	wmail "github.com/flakerimi/weft/warp/mail"
)

// wait is how long Last and Wait wait; a test of this package shortens it.
var wait = 5 * time.Second

// returned is, for each mailer and address, the number of the message Last
// returned last, counted as Printed counts, so that it never returns one
// twice. It is a test kit's bookkeeping, kept here because a Mailer has no
// room for it, and an entry is dropped when the test that made it ends.
var returned = struct {
	sync.Mutex
	last map[seen]int
}{last: map[seen]int{}}

type seen struct {
	m  *wmail.Mailer
	to string
}

// Last returns the newest message to to that the app's log driver kept since
// the one Last returned for to before, waiting up to 5 s for one to come. It
// fails the test when the app has no mail module or none comes. Addresses are
// compared without their names and in any case. What Last has returned for an
// address is remembered until the test that first asked for that address
// ends, so a parent test's Last after a subtest's may return the message the
// subtest read.
func Last(t testing.TB, app *warp.App, to string) wmail.Message {
	t.Helper()
	m := mailer(t, app)
	k := seen{m, address(to)}
	returned.Lock()
	after, ok := returned.last[k]
	returned.Unlock()
	if !ok {
		t.Cleanup(func() {
			returned.Lock()
			delete(returned.last, k)
			returned.Unlock()
		})
	}
	for deadline := time.Now().Add(wait); ; {
		kept, first := keptFrom(m)
		for i := len(kept) - 1; i >= 0 && first+i > after; i-- {
			if address(kept[i].To) == k.to {
				returned.Lock()
				returned.last[k] = first + i
				returned.Unlock()
				return kept[i]
			}
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("mailtest: no new message to %s came within %v; check that the app sends one, and that MAIL is log or unset.", to, wait)
	return wmail.Message{}
}

// Wait waits up to 5 s until the app's log driver has printed n messages in
// all, and fails the test when it has not.
func Wait(t testing.TB, app *warp.App, n int) {
	t.Helper()
	m := mailer(t, app)
	for deadline := time.Now().Add(wait); m.Printed() < n; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("mailtest: the app printed %d messages within %v, not %d; check that it sends them, and that MAIL is log or unset.", m.Printed(), wait, n)
		}
	}
}

// mailer is the app's Mailer, or fails the test.
func mailer(t testing.TB, app *warp.App) *wmail.Mailer {
	t.Helper()
	m := wmail.For(app)
	if m == nil {
		t.Fatalf("mailtest: the app has no mail module; list mail.Module() in its modules.")
	}
	return m
}

// keptFrom returns the messages m keeps and the number, counted from 1 as
// Printed counts, of the first of them, read so that no message is printed
// between the two.
func keptFrom(m *wmail.Mailer) ([]wmail.Message, int) {
	for {
		before := m.Printed()
		kept := m.Kept()
		if m.Printed() == before {
			return kept, before - len(kept) + 1
		}
	}
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
