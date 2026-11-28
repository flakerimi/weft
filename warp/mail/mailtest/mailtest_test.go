package mailtest

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/mail"
)

// fakeT records a failure instead of ending the test.
type fakeT struct {
	testing.TB
	failed string
}

func (f *fakeT) Helper() {}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.failed = fmt.Sprintf(format, args...)
	panic(f)
}

// failure runs fn with a fakeT and returns the failure it reported.
func failure(t *testing.T, fn func(tb testing.TB)) (failed string) {
	f := &fakeT{TB: t}
	defer func() {
		if v := recover(); v != nil && v != f {
			panic(v)
		}
		failed = f.failed
	}()
	fn(f)
	return ""
}

// mailApp is a started app whose mail is printed to nowhere and kept, with
// a wait of d.
func mailApp(t *testing.T, d time.Duration) (*warp.App, *mail.Mailer) {
	out := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(out) })
	was := wait
	wait = d
	t.Cleanup(func() { wait = was })

	app := warp.New(warp.ConfigFrom(nil))
	app.Use(mail.Module())
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Stop(context.Background()) })
	return app, mail.For(app)
}

// sendLater sends msg through m after d, in the background.
func sendLater(t *testing.T, m *mail.Mailer, d time.Duration, msg mail.Message) {
	sent := make(chan error, 1)
	go func() {
		time.Sleep(d)
		sent <- m.Send(context.Background(), msg)
	}()
	t.Cleanup(func() {
		if err := <-sent; err != nil {
			t.Error(err)
		}
	})
}

func TestWaitIsFiveSeconds(t *testing.T) {
	if wait != 5*time.Second {
		t.Errorf("Last and Wait wait %v, want 5s", wait)
	}
}

func TestLastWaits(t *testing.T) {
	app, m := mailApp(t, 2*time.Second)
	sendLater(t, m, 100*time.Millisecond, mail.Message{To: "ann@example.com", Subject: "Late", Text: "#token=late"})
	if got := Last(t, app, "ann@example.com"); got.Subject != "Late" {
		t.Errorf("Last = %+v, want the message sent after the call", got)
	}
}

func TestLastReturnsAMessageOnce(t *testing.T) {
	app, m := mailApp(t, 300*time.Millisecond)
	if err := m.Send(context.Background(), mail.Message{To: "ann@example.com", Subject: "First", Text: "#token=first"}); err != nil {
		t.Fatal(err)
	}
	if got := Last(t, app, "ann@example.com"); got.Subject != "First" {
		t.Fatalf("Last = %+v, want First", got)
	}
	begun := time.Now()
	f := failure(t, func(tb testing.TB) { Last(tb, app, "ANN@example.com") })
	if want := "mailtest: no new message to ANN@example.com came within 300ms; check that the app sends one, and that MAIL is log or unset."; f != want {
		t.Errorf("Last with no new message fails with\n%q\nwant\n%q", f, want)
	}
	if took := time.Since(begun); took < 300*time.Millisecond {
		t.Errorf("Last failed after %v, before its wait of 300ms", took)
	}
	// The same message again is a new one.
	sendLater(t, m, 50*time.Millisecond, mail.Message{To: "ann@example.com", Subject: "First", Text: "#token=first"})
	if got := Last(t, app, "ann@example.com"); got.Subject != "First" {
		t.Errorf("Last = %+v, want First again", got)
	}
}

func TestLastNoneComes(t *testing.T) {
	app, _ := mailApp(t, 200*time.Millisecond)
	begun := time.Now()
	f := failure(t, func(tb testing.TB) { Last(tb, app, "carol@example.com") })
	if want := "mailtest: no new message to carol@example.com came within 200ms; check that the app sends one, and that MAIL is log or unset."; f != want {
		t.Errorf("Last with no message fails with\n%q\nwant\n%q", f, want)
	}
	if took := time.Since(begun); took < 200*time.Millisecond {
		t.Errorf("Last failed after %v, before its wait of 200ms", took)
	}
}

func TestWait(t *testing.T) {
	app, m := mailApp(t, 2*time.Second)
	for i, d := range []time.Duration{20, 60, 100} {
		sendLater(t, m, d*time.Millisecond, mail.Message{To: fmt.Sprintf("a%d@example.com", i), Subject: "Hi", Text: "hello"})
	}
	Wait(t, app, 3)
	if n := len(m.Kept()); n != 3 {
		t.Errorf("after Wait(3) the app kept %d messages", n)
	}

	wait = 200 * time.Millisecond
	begun := time.Now()
	f := failure(t, func(tb testing.TB) { Wait(tb, app, 5) })
	if want := "mailtest: the app printed 3 messages within 200ms, not 5; check that it sends them, and that MAIL is log or unset."; f != want {
		t.Errorf("Wait for more than come fails with\n%q\nwant\n%q", f, want)
	}
	if took := time.Since(begun); took < 200*time.Millisecond {
		t.Errorf("Wait failed after %v, before its wait of 200ms", took)
	}
	bare := warp.New(warp.ConfigFrom(nil))
	if f := failure(t, func(tb testing.TB) { Wait(tb, bare, 1) }); !strings.Contains(f, "has no mail module") {
		t.Errorf("Wait without the module fails with %q", f)
	}
}

func TestLastAndToken(t *testing.T) {
	app, m := mailApp(t, 100*time.Millisecond)
	for _, msg := range []mail.Message{
		{To: "ann@example.com", Subject: "First", Text: "Open http://localhost:8080/password#token=first-_1 now."},
		{To: "bob@example.com", Subject: "Bob's", Text: "#token=bob"},
		{To: "Ann <ANN@example.com>", Subject: "Second", Text: "Open http://localhost:8080/password#token=Second_2"},
	} {
		if err := m.Send(context.Background(), msg); err != nil {
			t.Fatal(err)
		}
	}
	got := Last(t, app, "ann@example.com")
	if got.Subject != "Second" || Token(t, got) != "Second_2" {
		t.Errorf("Last = %+v with token %q, want the second to Ann", got, Token(t, got))
	}
	if tok := Token(t, Last(t, app, "bob@example.com")); tok != "bob" {
		t.Errorf("Token = %q", tok)
	}
	if tok := Token(t, mail.Message{Text: "#token=first-_1 now."}); tok != "first-_1" {
		t.Errorf("Token stops at %q, want first-_1", tok)
	}

	if f := failure(t, func(tb testing.TB) { Last(tb, app, "carol@example.com") }); !strings.Contains(f, "no new message to") {
		t.Errorf("Last with no message fails with %q", f)
	}
	if f := failure(t, func(tb testing.TB) { Token(tb, mail.Message{Text: "no link"}) }); !strings.Contains(f, "has no #token=") {
		t.Errorf("Token with no token fails with %q", f)
	}
	if f := failure(t, func(tb testing.TB) { Token(tb, mail.Message{Text: "#token= "}) }); !strings.Contains(f, "has no #token=") {
		t.Errorf("Token with an empty token fails with %q", f)
	}
	bare := warp.New(warp.ConfigFrom(nil))
	if f := failure(t, func(tb testing.TB) { Last(tb, bare, "ann@example.com") }); !strings.Contains(f, "has no mail module") {
		t.Errorf("Last without the module fails with %q", f)
	}
}
