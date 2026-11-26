package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

func TestCount(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	counted := func(ctx context.Context) int {
		t.Helper()
		n, err := s.Count(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := counted(ctx); n != 0 {
		t.Fatalf("Count of an empty table = %d, want 0", n)
	}
	for _, name := range []string{"Lamp", "Desk", "Chair"} {
		if err := s.Create(ctx, &Product{Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	if n := counted(ctx); n != 3 {
		t.Fatalf("Count after three creates = %d, want 3", n)
	}
	if err := s.Delete(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if n := counted(ctx); n != 2 {
		t.Fatalf("Count after a delete = %d, want 2", n)
	}

	failed := errors.New("the payment was declined")
	err := data.For(c.App).Tx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Shelf"}); err != nil {
			return err
		}
		if n := counted(ctx); n != 3 {
			t.Errorf("Count inside the transaction = %d, want 3 (its own row seen)", n)
		}
		return failed
	})
	if !errors.Is(err, failed) {
		t.Fatalf("Tx = %v, want the function's error", err)
	}
	if n := counted(ctx); n != 2 {
		t.Fatalf("Count after the rollback = %d, want 2", n)
	}
}

func TestCountBeforeStart(t *testing.T) {
	app := warp.New(warp.ConfigFrom(env(dbFile(t))))
	var s *data.Store[Product]
	app.Use(data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[Product](db, "products") }})
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	_, want := s.Get(ctx, 1)
	n, err := s.Count(ctx)
	if want == nil || err == nil || err.Error() != want.Error() || n != 0 {
		t.Fatalf("Count before start = %d, %v; want 0 and the error Get gives, %v", n, err, want)
	}
}
