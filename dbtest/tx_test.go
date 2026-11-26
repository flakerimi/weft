package dbtest

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
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

// A Tx whose request ends, at any moment, never loses a database in memory,
// whose one connection is the whole database: the transaction is rolled
// back by Tx itself, in order, and the connection is kept.
func TestCancelledTxKeepsTheDatabase(t *testing.T) {
	var s *data.Store[Product]
	c := warptest.New(t, map[string]string{"DB_DRIVER": "sqlite", "DB_URL": ":memory:"},
		data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[Product](db, "products") }})
	db := data.For(c.App)
	if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
		t.Fatal(err)
	}
	want := 1 // the products there should be
	check := func(when string, i int) {
		t.Helper()
		if n, err := s.Count(ctx); err != nil || n != want {
			t.Fatalf("%s, run %d: %d products, %v; want %d", when, i, n, err, want)
		}
		conn, err := db.SQL().Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
			t.Fatalf("%s, run %d: the pooled connection is inside a transaction: %v", when, i, err)
		}
		conn.ExecContext(ctx, "ROLLBACK")
	}
	goroutines := runtime.NumGoroutine()

	// Ended before the function runs.
	ended, cancel := context.WithCancel(ctx)
	cancel()
	if err := db.Tx(ended, func(context.Context) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("Tx on an ended context = %v", err)
	}
	check("ended before", 0)

	// Ended during a statement: the statement is interrupted.
	during, cancel := context.WithCancel(ctx)
	time.AfterFunc(50*time.Millisecond, cancel)
	err := db.Tx(during, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Desk"}); err != nil {
			return err
		}
		var n int
		return db.QueryRow(ctx, `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c) SELECT count(*) FROM c`).Scan(&n)
	})
	if err == nil {
		t.Fatal("a statement that never ends returned")
	}
	check("ended during a statement", 0)

	// Ended between the last statement and the commit, 2000 times: where
	// database/sql once rolled back on a goroutine of its own while the
	// commit failed.
	for i := range 2000 {
		between, cancel := context.WithCancel(ctx)
		if err := db.Tx(between, func(ctx context.Context) error {
			if err := s.Create(ctx, &Product{Name: "Desk"}); err != nil {
				return err
			}
			cancel()
			return nil
		}); !errors.Is(err, context.Canceled) {
			t.Fatalf("run %d: Tx ended before its commit = %v", i, err)
		}
		check("ended before the commit", i)
	}

	// Ended at about the moment of the commit, 2000 times.
	for i := range 2000 {
		racing, cancel := context.WithCancel(ctx)
		err := db.Tx(racing, func(ctx context.Context) error {
			if err := s.Create(ctx, &Product{Name: fmt.Sprintf("Chair %d", i)}); err != nil {
				return err
			}
			go cancel()
			return nil
		})
		cancel()
		if err == nil {
			want++ // the commit came first
		} else if !errors.Is(err, context.Canceled) {
			t.Fatalf("run %d: Tx = %v", i, err)
		}
		check("ended at the commit", i)
	}
	time.Sleep(100 * time.Millisecond)
	if n := runtime.NumGoroutine(); n > goroutines+5 {
		t.Fatalf("%d goroutines left over (%d before)", n-goroutines, goroutines)
	}
}
