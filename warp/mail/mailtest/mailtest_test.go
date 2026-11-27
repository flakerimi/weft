package mailtest

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"

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
	f.failed = format
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

func TestLastAndToken(t *testing.T) {
	out := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(out) })

	app := warp.New(warp.ConfigFrom(nil))
	app.Use(mail.Module())
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Stop(context.Background()) })
	m := mail.For(app)
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
	if tok := Token(t, Last(t, app, "ann@example.com")); tok != "Second_2" {
		t.Errorf("Token = %q", tok)
	}
	if tok := Token(t, mail.Message{Text: "#token=first-_1 now."}); tok != "first-_1" {
		t.Errorf("Token stops at %q, want first-_1", tok)
	}

	if f := failure(t, func(tb testing.TB) { Last(tb, app, "carol@example.com") }); !strings.Contains(f, "kept no message to") {
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
