package dbtest

import (
	"context"
	"net/mail"
	"strings"
	"sync"
	"testing"

	"github.com/flakerimi/weft/warp"
)

// Until warp/mail (Task B1 of the accounts plan) is merged into this branch,
// the tests list a mail module of their own, mailModule, which keeps what
// the login sends as the log driver does, and read it with lastMail and
// mailToken, which have the shapes of mailtest.Last and mailtest.Token. With
// B1 merged, mailModule() is mail.Module(), lastMail is mailtest.Last and
// mailToken is mailtest.Token, and this file goes.

// letter and postman have the shapes of mail.Message and of the one method of
// *mail.Mailer the login uses, as warp/auth's stand-in declares them; being
// unnamed, they are the login's own types.
type letter = struct{ To, Subject, Text, HTML string }

type postman = interface {
	Send(ctx context.Context, msg letter) error
}

type fakeMail struct {
	mu   sync.Mutex
	kept []letter
}

func mailModule() warp.Module { return &fakeMail{} }

func (f *fakeMail) Name() string { return "mail" }

func (f *fakeMail) Register(app *warp.App) error {
	warp.Provide[postman](app, f)
	return nil
}

// Send keeps msg, the last 20 as the log driver keeps them.
func (f *fakeMail) Send(_ context.Context, msg letter) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kept = append(f.kept, msg)
	if len(f.kept) > 20 {
		f.kept = f.kept[len(f.kept)-20:]
	}
	return nil
}

// sent is every message the app's mail module kept, oldest first.
func sent(t testing.TB, app *warp.App) []letter {
	t.Helper()
	f, ok := warp.Need[postman](app, "mail").(*fakeMail)
	if !ok {
		t.Fatalf("the app has no mail module of the tests'; list mailModule() before auth.Module().")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]letter(nil), f.kept...)
}

// lastMail is the newest message to to, as mailtest.Last finds it.
func lastMail(t testing.TB, app *warp.App, to string) letter {
	t.Helper()
	kept := sent(t, app)
	for i := len(kept) - 1; i >= 0; i-- {
		if address(kept[i].To) == address(to) {
			return kept[i]
		}
	}
	t.Fatalf("the app kept no message to %s among its last %d", to, len(kept))
	return letter{}
}

func address(s string) string {
	if a, err := mail.ParseAddress(s); err == nil {
		s = a.Address
	}
	return strings.ToLower(strings.TrimSpace(s))
}

// mailToken is what follows #token= in msg.Text, as mailtest.Token reads it.
func mailToken(t testing.TB, msg letter) string {
	t.Helper()
	_, rest, ok := strings.Cut(msg.Text, "#token=")
	end := strings.IndexFunc(rest, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	})
	if end >= 0 {
		rest = rest[:end]
	}
	if !ok || rest == "" {
		t.Fatalf("the message to %s (%q) has no #token= in its text", msg.To, msg.Subject)
	}
	return rest
}
