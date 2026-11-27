package mail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
)

// syncBuffer is a buffer the logger and a test may use at once.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLog sends the standard logger, and so slog's default, to a buffer
// until the test ends. A test that calls it does not run in parallel.
func captureLog(t *testing.T) *syncBuffer {
	t.Helper()
	b := &syncBuffer{}
	out, flags := log.Writer(), log.Flags()
	log.SetOutput(b)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(out)
		log.SetFlags(flags)
	})
	return b
}

// started is an app with the mail module, started with vars; it is stopped
// when the test ends unless the test stopped it.
func started(t *testing.T, vars map[string]string) (*warp.App, *Mailer) {
	t.Helper()
	app := warp.New(warp.ConfigFrom(vars))
	app.Use(Module())
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { app.Stop(context.Background()) })
	return app, For(app)
}

// cfgOf is the configuration an app made from vars has, defaults filled in.
func cfgOf(vars map[string]string) warp.Config { return warp.New(warp.ConfigFrom(vars)).Config() }

func TestSettings(t *testing.T) {
	smtpOK := map[string]string{"MAIL": "smtp", "MAIL_HOST": "smtp.example.com", "MAIL_FROM": "Shop <noreply@shop.example.com>"}
	with := func(base map[string]string, kv ...string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for i := 0; i < len(kv); i += 2 {
			if kv[i+1] == "" {
				delete(m, kv[i])
			} else {
				m[kv[i]] = kv[i+1]
			}
		}
		return m
	}
	for _, c := range []struct {
		name string
		vars map[string]string
		want []string
	}{
		{"mail unknown", map[string]string{"MAIL": "x"}, []string{`MAIL is "x"; set it to log, which prints mail, or smtp, which sends it.`}},
		{"no host", with(smtpOK, "MAIL_HOST", ""), []string{"MAIL_HOST is not set; set it to your mail server, such as smtp.example.com."}},
		{"port zero", with(smtpOK, "MAIL_PORT", "0"), []string{`MAIL_PORT is "0"; set it to a number from 1 to 65535, such as 587.`}},
		{"port words", with(smtpOK, "MAIL_PORT", "smtp"), []string{`MAIL_PORT is "smtp"; set it to a number from 1 to 65535, such as 587.`}},
		{"port high", with(smtpOK, "MAIL_PORT", "65536"), []string{`MAIL_PORT is "65536"; set it to a number from 1 to 65535, such as 587.`}},
		{"user alone", with(smtpOK, "MAIL_USER", "u"), []string{"MAIL_USER is set and MAIL_PASSWORD is not; set both, or neither for a server that takes mail without signing in."}},
		{"password alone", with(smtpOK, "MAIL_PASSWORD", "p"), []string{"MAIL_PASSWORD is set and MAIL_USER is not; set both, or neither for a server that takes mail without signing in."}},
		{"no from", with(smtpOK, "MAIL_FROM", ""), []string{"MAIL_FROM is not set; set it to the address mail comes from, such as Shop <noreply@shop.example.com>."}},
		{"bad from", with(smtpOK, "MAIL_FROM", "shop"), []string{`MAIL_FROM is "shop"; set it to the address mail comes from, such as Shop <noreply@shop.example.com>.`}},
		{"bad from under log", map[string]string{"MAIL_FROM": "a, b"}, []string{`MAIL_FROM is "a, b"; set it to the address mail comes from, such as Shop <noreply@shop.example.com>.`}},
		{"several", map[string]string{"MAIL": "smtp", "MAIL_PORT": "x", "MAIL_USER": "u"}, []string{
			"MAIL_HOST is not set; set it to your mail server, such as smtp.example.com.",
			`MAIL_PORT is "x"; set it to a number from 1 to 65535, such as 587.`,
			"MAIL_USER is set and MAIL_PASSWORD is not; set both, or neither for a server that takes mail without signing in.",
			"MAIL_FROM is not set; set it to the address mail comes from, such as Shop <noreply@shop.example.com>.",
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := readSettings(cfgOf(c.vars))
			if err == nil {
				t.Fatalf("readSettings(%v) = nil, want %q", c.vars, c.want)
			}
			if got := strings.Split(err.Error(), "\n"); strings.Join(got, "\n") != strings.Join(c.want, "\n") {
				t.Errorf("readSettings(%v) =\n%s\nwant\n%s", c.vars, err, strings.Join(c.want, "\n"))
			}
		})
	}

	// Through Start, the module's sentence.
	app := warp.New(warp.ConfigFrom(map[string]string{"MAIL": "x"}))
	app.Use(Module())
	err := app.Start(context.Background())
	if want := `warp: module "mail" could not start: MAIL is "x"; set it to log, which prints mail, or smtp, which sends it.`; err == nil || err.Error() != want {
		t.Errorf("Start = %v, want %q", err, want)
	}

	// The defaults.
	s, err := readSettings(cfgOf(map[string]string{"APP_URL": "https://shop.example.com:8443"}))
	if err != nil || s.driver != "log" || s.from.Address != "noreply@shop.example.com" || s.hello != "shop.example.com" {
		t.Errorf("unset MAIL gives %+v, %v; want log from noreply@shop.example.com", s, err)
	}
	s, err = readSettings(cfgOf(smtpOK))
	if err != nil || s.driver != "smtp" || s.port != "587" || s.from.Name != "Shop" {
		t.Errorf("smtp with no port gives %+v, %v; want port 587", s, err)
	}
	if _, err := readSettings(cfgOf(with(smtpOK, "MAIL_USER", "u", "MAIL_PASSWORD", "p", "MAIL_PORT", "465"))); err != nil {
		t.Errorf("a whole smtp setting is refused: %v", err)
	}
}

func TestLogLineAtStart(t *testing.T) {
	logs := captureLog(t)
	started(t, nil)
	if want := "mail is printed, not sent; set MAIL=smtp in .env to send it.\n"; !strings.Contains(logs.String(), want) {
		t.Errorf("the log has %q, want %q", logs, want)
	}
}

func TestSendChecks(t *testing.T) {
	captureLog(t)
	ok := Message{To: "ann@example.com", Subject: "Hi", Text: "Hello"}

	before := warp.New(warp.ConfigFrom(nil))
	before.Use(Module())
	if err := For(before).Send(context.Background(), ok); err == nil || err.Error() != "mail: Send was called before the app started; send from a handler, or after Start." {
		t.Errorf("Send before Start = %v", err)
	}

	_, m := started(t, nil)
	with := func(change func(*Message)) Message { msg := ok; change(&msg); return msg }
	for _, c := range []struct {
		msg  Message
		want string
	}{
		{with(func(m *Message) { m.To = "" }), `mail: the message's To is ""; give it one email address.`},
		{with(func(m *Message) { m.To = "ann" }), `mail: the message's To is "ann"; give it one email address.`},
		{with(func(m *Message) { m.To = "a@example.com, b@example.com" }), `mail: the message's To is "a@example.com, b@example.com"; give it one email address.`},
		{with(func(m *Message) { m.To = "a@example.com\r\nBcc: b@example.com" }), `mail: the message's To is "a@example.com\r\nBcc: b@example.com"; give it one email address.`},
		{with(func(m *Message) { m.Subject = "" }), "mail: the message has no Subject; give it one."},
		{with(func(m *Message) { m.Subject = "Hi\r\nBcc: b@example.com" }), "mail: the message's Subject has a line break; keep it on one line."},
		{with(func(m *Message) { m.Subject = "Hi\nthere" }), "mail: the message's Subject has a line break; keep it on one line."},
		{with(func(m *Message) { m.Subject = strings.Repeat("s", 201) }), "mail: the message's Subject is 201 bytes; keep it to 200."},
		{with(func(m *Message) { m.Text = "" }), "mail: the message has no Text; give it a plain-text body, which every message needs."},
		{with(func(m *Message) { m.Text = strings.Repeat("t", 256<<10+1) }), "mail: the message's Text is 262145 bytes; keep it to 256 KiB."},
		{with(func(m *Message) { m.HTML = strings.Repeat("h", 256<<10+1) }), "mail: the message's HTML is 262145 bytes; keep it to 256 KiB."},
	} {
		if err := m.Send(context.Background(), c.msg); err == nil || err.Error() != c.want {
			t.Errorf("Send(%.80q) = %v, want %q", fmt.Sprint(c.msg), err, c.want)
		}
	}
	if len(m.Kept()) != 0 {
		t.Errorf("a refused message was kept: %v", m.Kept())
	}
	// At the limits, a message goes.
	at := Message{To: "Ann <ann@example.com>", Subject: strings.Repeat("s", 200), Text: strings.Repeat("t", 256<<10), HTML: strings.Repeat("h", 256<<10)}
	if err := m.Send(context.Background(), at); err != nil {
		t.Errorf("a message at the limits is refused: %v", err)
	}
}

func TestSendAfterStop(t *testing.T) {
	captureLog(t)
	app, m := started(t, nil)
	app.Stop(context.Background())
	err := m.Send(context.Background(), Message{To: "a@example.com", Subject: "Hi", Text: "x"})
	if want := "mail: Send was called after the app stopped; send while it runs."; err == nil || err.Error() != want {
		t.Errorf("Send after Stop = %v, want %q", err, want)
	}
}

func TestLogDriverPrintsAndKeeps(t *testing.T) {
	logs := captureLog(t)
	_, m := started(t, nil)
	for i := 1; i <= 21; i++ {
		msg := Message{To: fmt.Sprintf("a%d@example.com", i), Subject: fmt.Sprintf("Subject %d", i), Text: "line one\nline two"}
		if err := m.Send(context.Background(), msg); err != nil {
			t.Fatal(err)
		}
	}
	kept := m.Kept()
	if len(kept) != 20 || kept[0].To != "a2@example.com" || kept[19].To != "a21@example.com" {
		t.Fatalf("Kept = %d messages, %v first and %v last; want the last 20, oldest first", len(kept), kept[0].To, kept[len(kept)-1].To)
	}
	kept[0].To = "changed"
	if m.Kept()[0].To != "a2@example.com" {
		t.Error("Kept answers the mailer's own slice")
	}
	if want := "mail to a1@example.com: Subject 1\n  line one\n  line two\n"; !strings.Contains(logs.String(), want) {
		t.Errorf("the log has\n%s\nwant\n%s", logs, want)
	}
}

func TestQueueFull(t *testing.T) {
	logs := captureLog(t)
	srv := newFakeSMTP(t, func(s *fakeSMTP) { s.stall = true })
	app, m := started(t, srv.vars())
	msg := Message{To: "a@example.com", Subject: "Hi", Text: "x"}
	for i := range 100 {
		if err := m.Send(context.Background(), msg); err != nil {
			t.Fatalf("message %d: %v", i+1, err)
		}
	}
	err := m.Send(context.Background(), msg)
	var we *warp.Error
	if !errors.As(err, &we) || we.Status != http.StatusServiceUnavailable || we.Code != "unavailable" || we.Message != "The server is busy sending mail; try again in a moment." {
		t.Fatalf("the 101st message gives %#v, want the 503", err)
	}
	if len(m.Kept()) != 0 {
		t.Errorf("the smtp driver keeps messages: %v", m.Kept())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	app.Stop(ctx)
	if want := "mail: 100 messages were not sent, since the app stopped."; !strings.Contains(logs.String(), want) {
		t.Errorf("the log has %q, want %q", logs, want)
	}
}

func TestStopDrains(t *testing.T) {
	logs := captureLog(t)
	srv := newFakeSMTP(t)
	app, m := started(t, srv.vars())
	for i := range 5 {
		if err := m.Send(context.Background(), Message{To: fmt.Sprintf("a%d@example.com", i), Subject: "Hi", Text: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(srv.received()); n != 5 {
		t.Errorf("Stop let %d of 5 messages go, want all", n)
	}
	if strings.Contains(logs.String(), "not sent") {
		t.Errorf("the log says some were not sent: %s", logs)
	}

	logs = captureLog(t)
	stalled := newFakeSMTP(t, func(s *fakeSMTP) { s.stall = true })
	app, m = started(t, stalled.vars())
	for range 3 {
		if err := m.Send(context.Background(), Message{To: "a@example.com", Subject: "Hi", Text: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	begun := time.Now()
	app.Stop(ctx)
	if took := time.Since(begun); took > 2*time.Second {
		t.Errorf("Stop took %v with a server that never answers, want it to end with its context", took)
	}
	if want := "mail: 3 messages were not sent, since the app stopped."; !strings.Contains(logs.String(), want) {
		t.Errorf("the log has %q, want %q", logs, want)
	}

	logs = captureLog(t)
	app, m = started(t, stalled.vars())
	m.Send(context.Background(), Message{To: "a@example.com", Subject: "Hi", Text: "x"})
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	app.Stop(ctx)
	if want := "mail: 1 message was not sent, since the app stopped."; !strings.Contains(logs.String(), want) {
		t.Errorf("the log has %q, want %q", logs, want)
	}
}

func TestNeedsNoNetworkAtStart(t *testing.T) {
	captureLog(t)
	var mu sync.Mutex
	dials := 0
	swapDial(t, func(ctx context.Context, network, addr string) (net.Conn, error) {
		mu.Lock()
		dials++
		mu.Unlock()
		return nil, errors.New("no network in this test")
	})
	started(t, map[string]string{"MAIL": "smtp", "MAIL_HOST": "smtp.invalid", "MAIL_FROM": "noreply@example.com"})
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if dials != 0 {
		t.Errorf("Start dialled the mail server %d times, want none", dials)
	}
}

func TestForWithoutModule(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	if For(app) != nil {
		t.Error("For gives a mailer to an app without the module")
	}
	want := `warp: the app needs the "mail" module; list mail.Module() in app/modules.go.`
	if err := app.Err(); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Err = %v, want %q", err, want)
	}
}
