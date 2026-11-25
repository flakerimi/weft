package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/flakerimi/weft/warp/data"
)

func count(t *testing.T, db *data.DB) int {
	t.Helper()
	var n int
	if err := db.SQL().QueryRow(`SELECT count(*) FROM "products"`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTxCommitsAndRollsBack(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	db := data.For(c.App)

	err := db.Tx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
			return err
		}
		// Inside, the transaction's own write is seen.
		if _, err := s.Get(ctx, 1); err != nil {
			return err
		}
		return s.Create(ctx, &Product{Name: "Desk"})
	})
	if err != nil || count(t, db) != 2 {
		t.Fatalf("Tx = %v, %d rows", err, count(t, db))
	}

	failed := errors.New("the payment was declined")
	err = db.Tx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Chair"}); err != nil {
			return err
		}
		if _, err := s.Update(ctx, 1, &Product{Name: "Lamp 2"}, []string{"name"}); err != nil {
			return err
		}
		if err := s.Delete(ctx, 2); err != nil {
			return err
		}
		return failed
	})
	if !errors.Is(err, failed) {
		t.Fatalf("Tx = %v, want the function's error", err)
	}
	if n := count(t, db); n != 2 {
		t.Fatalf("%d rows after a rollback, want 2", n)
	}
	if lamp, _ := s.Get(ctx, 1); lamp.Name != "Lamp" {
		t.Fatalf("the update was kept: %+v", lamp)
	}

	// A unique violation inside rolls back the rest.
	err = db.Tx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Stool"}); err != nil {
			return err
		}
		return s.Create(ctx, &Product{Name: "Lamp"})
	})
	if err == nil || count(t, db) != 2 {
		t.Fatalf("Tx = %v, %d rows", err, count(t, db))
	}
}

func TestTxNested(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	db := data.For(c.App)

	inner := errors.New("inner failed")
	err := db.Tx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
			return err
		}
		// The inner Tx joins: it neither commits nor rolls back by itself.
		if err := db.Tx(ctx, func(ctx context.Context) error {
			if err := s.Create(ctx, &Product{Name: "Desk"}); err != nil {
				return err
			}
			return inner
		}); !errors.Is(err, inner) {
			t.Errorf("inner Tx = %v", err)
		}
		return db.Tx(ctx, func(ctx context.Context) error { return s.Create(ctx, &Product{Name: "Chair"}) })
	})
	if err != nil || count(t, db) != 3 {
		t.Fatalf("Tx = %v, %d rows; the outer commit keeps all three", err, count(t, db))
	}

	err = db.Tx(ctx, func(ctx context.Context) error {
		if err := db.Tx(ctx, func(ctx context.Context) error { return s.Create(ctx, &Product{Name: "Stool"}) }); err != nil {
			return err
		}
		return inner
	})
	if !errors.Is(err, inner) || count(t, db) != 3 {
		t.Fatalf("Tx = %v, %d rows; the outer rollback undoes the inner write", err, count(t, db))
	}
}

func TestTxPanicRollsBack(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	db := data.For(c.App)
	func() {
		defer func() {
			if v := recover(); v != "boom" {
				t.Fatalf("recovered %v, want the panic again", v)
			}
		}()
		db.Tx(ctx, func(ctx context.Context) error {
			if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
				return err
			}
			panic("boom")
		})
	}()
	if n := count(t, db); n != 0 {
		t.Fatalf("%d rows after a panic, want 0", n)
	}
	// The connection went back to the pool usable.
	if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
		t.Fatal(err)
	}
}

// A transaction of one app is not used by another app's store.
func TestTxOfAnotherDB(t *testing.T) {
	c1, s1 := shop[Product](t, dbFile(t))
	_, s2 := shop[Product](t, dbFile(t))
	db1 := data.For(c1.App)
	err := db1.Tx(ctx, func(ctx context.Context) error {
		if err := s1.Create(ctx, &Product{Name: "Lamp"}); err != nil {
			return err
		}
		if err := s2.Create(ctx, &Product{Name: "Lamp"}); err != nil {
			return err
		}
		return errors.New("roll back")
	})
	if err == nil {
		t.Fatal("Tx = nil")
	}
	if _, err := s2.Get(ctx, 1); err != nil {
		t.Fatalf("the other database's write was rolled back with this one: %v", err)
	}
	if count(t, db1) != 0 {
		t.Fatal("the transaction was not rolled back")
	}
}
