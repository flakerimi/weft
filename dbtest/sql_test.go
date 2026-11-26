package dbtest

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

// A developer's own SQL runs in the transaction on its context, and on the
// database outside one.
func TestOwnSQLJoinsTx(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	db := data.For(c.App)
	rows := func(ctx context.Context) int {
		t.Helper()
		var n int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM "products"`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Outside a Tx: the pool.
	if _, err := db.Exec(ctx, `INSERT INTO "products" ("name") VALUES (?)`, "Lamp"); err != nil {
		t.Fatal(err)
	}
	if n := rows(ctx); n != 1 {
		t.Fatalf("%d rows, want 1", n)
	}

	// Inside: the transaction, which sees its own rows and the store's, and a
	// rollback undoes the Exec.
	failed := errors.New("rolled back")
	err := db.Tx(ctx, func(ctx context.Context) error {
		if _, err := db.Exec(ctx, `INSERT INTO "products" ("name") VALUES (?)`, "Desk"); err != nil {
			return err
		}
		if err := s.Create(ctx, &Product{Name: "Chair"}); err != nil {
			return err
		}
		if n := rows(ctx); n != 3 {
			t.Errorf("inside the Tx %d rows, want 3", n)
		}
		r, err := db.Query(ctx, `SELECT "name" FROM "products" ORDER BY "id"`)
		if err != nil {
			return err
		}
		defer r.Close()
		var names []string
		for r.Next() {
			var name string
			if err := r.Scan(&name); err != nil {
				return err
			}
			names = append(names, name)
		}
		if got := strings.Join(names, ","); got != "Lamp,Desk,Chair" {
			t.Errorf("Query inside the Tx: %s", got)
		}
		return failed
	})
	if !errors.Is(err, failed) {
		t.Fatalf("Tx = %v", err)
	}
	if n := rows(ctx); n != 1 {
		t.Fatalf("after the rollback %d rows, want 1: the Exec was not in the transaction", n)
	}
	if _, ok, _ := s.First(ctx, "name", "Desk"); ok {
		t.Fatal("the rolled-back Exec's row is there")
	}

	// SQL() is still the pool.
	if db.SQL() == nil {
		t.Fatal("SQL() is nil")
	}
}

func TestOwnSQLBeforeStart(t *testing.T) {
	app := warp.New(warp.ConfigFrom(env(dbFile(t))))
	app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
	db := data.For(app)
	const want = `warp: the database was used before the app started; use it from a handler, or after Start.`
	_, execErr := db.Exec(ctx, `SELECT 1`)
	_, queryErr := db.Query(ctx, `SELECT 1`)
	var n int
	rowErr := db.QueryRow(ctx, `SELECT 1`).Scan(&n)
	for name, err := range map[string]error{"Exec": execErr, "Query": queryErr, "QueryRow": rowErr} {
		if err == nil || err.Error() != want {
			t.Errorf("%s before start = %v, want %q", name, err, want)
		}
	}
}

// Warp's settings are the connection's whatever DB_URL says, for those the
// store relies on: transactions that take the write lock when they begin,
// the WAL journal and foreign keys. A DB_URL's busy timeout is kept.
func TestWarpSettingsWinOverTheURL(t *testing.T) {
	file := dbFile(t)
	url := "file:" + file + "?_txlock=deferred&_journal=DELETE&_journal_mode=TRUNCATE&_fk=0&_foreign_keys=off&%5Ftxlock=exclusive&_busy_timeout=0"
	var s *data.Store[Product]
	c := warptest.New(t, map[string]string{"DB_DRIVER": "sqlite", "DB_URL": url},
		data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[Product](db, "products") }})
	db := data.For(c.App)
	var mode string
	var fk int
	if err := db.QueryRow(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Errorf("journal_mode %q, %v; want wal", mode, err)
	}
	if err := db.QueryRow(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys %d, %v; want 1", fk, err)
	}
	// An immediate transaction holds the write lock from its start, so
	// another connection cannot begin one; a deferred one would hold none.
	other, err := sql.Open("sqlite", "file:"+file+"?_busy_timeout=0")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	err = db.Tx(ctx, func(ctx context.Context) error {
		conn, err := other.Conn(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()
		if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err == nil {
			conn.ExecContext(ctx, "ROLLBACK")
			t.Error("another connection began a write while the Tx was open: the Tx was not immediate")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The URL's busy timeout of 0 is kept: a write the other connection
	// blocks fails at once rather than after Warp's 5 s.
	release := lockWrites(t, other)
	defer release()
	began := time.Now()
	if err := s.Create(ctx, &Product{Name: "Lamp"}); err == nil {
		t.Fatal("Create succeeded while another connection held the write lock")
	}
	if took := time.Since(began); took > 2*time.Second {
		t.Fatalf("Create waited %v; the URL's _busy_timeout=0 was not kept", took)
	}
}

// lockWrites holds the write lock of the database pool opens until release
// is called.
func lockWrites(t *testing.T, pool *sql.DB) (release func()) {
	t.Helper()
	conn, err := pool.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			conn.ExecContext(ctx, "ROLLBACK")
			conn.Close()
		})
	}
	t.Cleanup(release)
	return release
}
