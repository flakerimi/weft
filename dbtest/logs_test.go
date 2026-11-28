package dbtest

import (
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/mail"
	"github.com/flakerimi/weft/warp/mail/mailtest"
)

// The tests list mail.Module(), whose log driver keeps what the login sends,
// and read it with mailtest. failMail is the one mail module of their own.

// failMail is a mail module whose Mailer is never started, so every Send is
// refused, as a mail module that cannot take a message refuses it.
type failMail struct{}

func (failMail) Name() string { return "mail" }

func (failMail) Register(app *warp.App) error {
	warp.Provide(app, &mail.Mailer{})
	return nil
}

// kept waits until the app's mail module has printed n messages in all and
// returns the ones it keeps (the last 20), oldest first.
func kept(t testing.TB, app *warp.App, n int) []mail.Message {
	t.Helper()
	mailtest.Wait(t, app, n)
	return mail.For(app).Kept()
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

// mailWait is how long a test waits for a failure the login logs after it
// has answered, as mailtest waits for a message.
const mailWait = 5 * time.Second
