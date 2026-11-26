package dbtest

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

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
// the WAL journal and foreign keys. A DB_URL may raise the busy timeout and
// never lower it, since the app's own writers wait on it.
func TestWarpSettingsWinOverTheURL(t *testing.T) {
	file := dbFile(t)
	url := "file:" + file + "?_txlock=deferred&_journal=DELETE&_journal_mode=TRUNCATE&_fk=0&_foreign_keys=off&%5Ftxlock=exclusive&_busy_timeout=0"
	c := warptest.New(t, map[string]string{"DB_DRIVER": "sqlite", "DB_URL": url},
		data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
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
	// A busy timeout may be raised by the URL, never lowered, in every form
	// the driver takes: 0 is Warp's 5000, and 9000 is 9000.
	base := "file:" + file + "?"
	for url, want := range map[string]int{
		base + "_busy_timeout=0": 5000, base + "_busy_timeout=9000": 9000, base + "_timeout=0": 5000,
		base + "_pragma=busy_timeout(0)": 5000, base + "_pragma=busy_timeout%3D0": 5000, base + "_pragma=busy_timeout=0": 5000,
		base + "_pragma=BUSY_TIMEOUT(0)": 5000, base + "_pragma=%20busy_timeout(1)": 5000,
		base + "_busy_timeout=9000&_pragma=busy_timeout(0)":         9000,
		base + "_pragma=busy_timeout(0)&_pragma=busy_timeout(7000)": 7000,
		base + "_pragma=busy_timeout(7000)&_pragma=busy_timeout(0)": 7000,
		base + "_busy_timeout=99999999999":                          5000,
	} {
		c2 := warptest.New(t, map[string]string{"DB_DRIVER": "sqlite", "DB_URL": url},
			data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
		var got int
		if err := data.For(c2.App).QueryRow(ctx, "PRAGMA busy_timeout").Scan(&got); err != nil || got != want {
			t.Errorf("%s: busy_timeout %d, %v; want %d", url, got, err, want)
		}
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

// The busy check reads the driver's code: a stored value that reads
// "database is locked", and SQLITE_LOCKED, which trying again never cures,
// are errors of their own, answered as any other is, never the 503.
func TestBusyIsTheCodeNotTheWords(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	db := data.For(c.App)
	if err := s.Create(ctx, &Product{Name: "database is locked"}); err != nil {
		t.Fatal(err)
	}
	var n int
	err := db.QueryRow(ctx, `SELECT "name" FROM "products"`).Scan(&n)
	var e *warp.Error
	if err == nil || errors.As(err, &e) {
		t.Fatalf("scanning stored text into an int = %v, want the conversion error, not a warp.Error", err)
	}
	// A table dropped inside a transaction that has its rows open is locked
	// within the one connection.
	err = db.Tx(ctx, func(ctx context.Context) error {
		rows, err := db.Query(ctx, `SELECT "id" FROM "products"`)
		if err != nil {
			return err
		}
		defer rows.Close()
		rows.Next()
		_, err = db.Exec(ctx, `DROP TABLE "products"`)
		return err
	})
	if err == nil || errors.As(err, &e) || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("DROP TABLE with its rows open = %v, want SQLite's locked error, not a warp.Error", err)
	}
}

// A COMMIT that fails (here a deferred foreign key, checked at COMMIT) on a
// database in memory, which lives on its one connection, leaves the earlier
// data and the tables there and the pool usable, and the failed
// transaction's writes gone: the connection is not thrown away.
func TestFailedCommitKeepsTheDatabase(t *testing.T) {
	for _, url := range []string{":memory:", "file:" + dbFile(t)} {
		var s *data.Store[Product]
		c := warptest.New(t, map[string]string{"DB_DRIVER": "sqlite", "DB_URL": url},
			data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[Product](db, "products") }})
		db := data.For(c.App)
		db.SQL().SetMaxOpenConns(1)
		if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
			t.Fatal(err)
		}
		for _, q := range []string{
			`CREATE TABLE "parent" ("id" INTEGER PRIMARY KEY)`,
			`CREATE TABLE "child" ("pid" INTEGER REFERENCES "parent"("id") DEFERRABLE INITIALLY DEFERRED)`,
		} {
			if _, err := db.Exec(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
		err := db.Tx(ctx, func(ctx context.Context) error {
			if err := s.Create(ctx, &Product{Name: "Desk"}); err != nil {
				return err
			}
			_, err := db.Exec(ctx, `INSERT INTO "child" VALUES (999)`)
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
			t.Fatalf("%s: the COMMIT = %v, want its foreign key error", url, err)
		}
		if n, err := s.Count(ctx); err != nil || n != 1 {
			t.Fatalf("%s: after the failed COMMIT, %d products, %v; want the one from before", url, n, err)
		}
		if _, ok, _ := s.First(ctx, "name", "Desk"); ok {
			t.Fatalf("%s: the failed transaction's row is there", url)
		}
		if err := db.Tx(ctx, func(ctx context.Context) error { return s.Create(ctx, &Product{Name: "Chair"}) }); err != nil {
			t.Fatalf("%s: the next Tx = %v", url, err)
		}
	}
}
