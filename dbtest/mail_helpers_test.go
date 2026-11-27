package dbtest

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"

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
	sent map[string]int // per address, every message sent to it, kept or not
	read map[string]int // per address, how many of those lastMail has returned
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
	if f.sent == nil {
		f.sent = map[string]int{}
	}
	f.sent[address(msg.To)]++
	if len(f.kept) > 20 {
		f.kept = f.kept[len(f.kept)-20:]
	}
	return nil
}

func fakeOf(t testing.TB, app *warp.App) *fakeMail {
	t.Helper()
	f, ok := warp.Need[postman](app, "mail").(*fakeMail)
	if !ok {
		t.Fatalf("the app has no mail module of the tests'; list mailModule() before auth.Module().")
	}
	return f
}

// The login sends register's and forgot's mail after it answers, so the
// readers wait for it, up to 5 seconds.
const mailWait = 5 * time.Second

// sent waits until the app's mail module has kept n messages (at most 20)
// and returns them all, oldest first.
func sent(t testing.TB, app *warp.App, n int) []letter {
	t.Helper()
	f := fakeOf(t, app)
	deadline := time.Now().Add(mailWait)
	for {
		f.mu.Lock()
		kept := append([]letter(nil), f.kept...)
		f.mu.Unlock()
		if len(kept) >= n {
			return kept
		}
		if time.Now().After(deadline) {
			t.Fatalf("the app kept %d messages in %v, want %d", len(kept), mailWait, n)
		}
		time.Sleep(time.Millisecond)
	}
}

// lastMail is the newest message to to that lastMail has not returned yet,
// as mailtest.Last finds it, waiting for one to come.
func lastMail(t testing.TB, app *warp.App, to string) letter {
	t.Helper()
	f := fakeOf(t, app)
	want := address(to)
	deadline := time.Now().Add(mailWait)
	for {
		f.mu.Lock()
		var mine []letter
		for _, m := range f.kept {
			if address(m.To) == want {
				mine = append(mine, m)
			}
		}
		if f.read == nil {
			f.read = map[string]int{}
		}
		if f.sent[want] > f.read[want] && len(mine) > 0 {
			f.read[want] = f.sent[want]
			f.mu.Unlock()
			return mine[len(mine)-1]
		}
		f.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatalf("the app sent no new message to %s in %v", to, mailWait)
		}
		time.Sleep(time.Millisecond)
	}
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

// failMail is a mail module whose every Send is refused as a full queue is.
type failMail struct{}

func (failMail) Name() string { return "mail" }

func (f failMail) Register(app *warp.App) error {
	warp.Provide[postman](app, f)
	return nil
}

func (failMail) Send(context.Context, letter) error {
	return warp.Unavailable("The server is busy sending mail; try again in a moment.")
}

// logs is what slog's default logger writes while a test runs, safe to read
// as it is written.
type logs struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (l *logs) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *logs) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func captureLogs(t *testing.T) *logs {
	l := &logs{}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(l, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return l
}
