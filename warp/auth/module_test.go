package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
)

func TestNeedsData(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(Module())
	want := `warp: module "auth" needs the "data" module; list data.Module() before it in app/modules.go.`
	if err := app.Err(); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Err = %v, want %q", err, want)
	}
}

func TestPer(t *testing.T) {
	per := Per("products")
	for action, want := range map[string]string{
		"list": "products:read", "show": "products:read", "create": "products:create",
		"update": "products:edit", "remove": "products:delete", "export": "products:export",
	} {
		app := warp.New(warp.ConfigFrom(nil))
		warp.Get(app, "/x", func(_ context.Context, _ warp.None) (warp.None, error) { return warp.None{}, nil }, per(action)...)
		if r := app.Routes(); len(r) != 1 || r[0].Permission != want {
			t.Errorf("Per(products)(%q) records %+v, want %q", action, r, want)
		}
	}
}

// Review N4 (moved here from dbtest when the login took warp/mail's Mailer,
// which a test cannot make block): Stop waits for the mail register or
// forgot handed to later, and when its time runs out first it says so and
// ends the mailing's context.
func TestStopWaitsForLaterWork(t *testing.T) {
	start := func(t *testing.T) (*module, chan struct{}, chan error) {
		m := newModule(time.Now)
		m.base, m.cancelBase = context.WithCancel(context.Background())
		started, release, ended := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		if err := m.later(func(ctx context.Context) error {
			close(started)
			select {
			case <-release:
				ended <- nil
			case <-ctx.Done():
				ended <- ctx.Err()
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		<-started
		return m, release, ended
	}

	t.Run("waits", func(t *testing.T) {
		m, release, _ := start(t)
		done := make(chan error, 1)
		go func() { done <- m.Stop(context.Background()) }()
		select {
		case err := <-done:
			t.Fatalf("Stop returned (%v) while a mailing was under way", err)
		case <-time.After(100 * time.Millisecond):
		}
		close(release)
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Stop = %v once the mailing was done", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Stop did not return once the mailing was done")
		}
		if err := m.later(func(context.Context) error { return nil }); err == nil {
			t.Fatal("later took more work after Stop")
		}
	})

	t.Run("gives up", func(t *testing.T) {
		m, _, ended := start(t)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := m.Stop(ctx)
		want := "the mail register or forgot asked for was still being made when the app stopped, so it was not sent; ask for it again."
		if err == nil || err.Error() != want {
			t.Fatalf("Stop = %v\nwant %s", err, want)
		}
		select {
		case err := <-ended:
			if err != context.Canceled {
				t.Fatalf("the mailing ended with %v, want its context canceled", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the mailing's context was not ended")
		}
	})
}
