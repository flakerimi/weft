// Package data keeps an app's records in a database through database/sql:
// a module that opens the database and makes its tables, a Store for each
// model, and transactions. SQLite is the one engine for now; the app's
// main.go imports its driver, such as _ "modernc.org/sqlite".
package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/flakerimi/weft/warp"
)

// Module is the data module, named "data". Registering it does no work: it
// provides the app's *DB, which For returns. DB_DRIVER and DB_URL are read in
// its Migrate, which opens the database and brings every store's table up to
// date before any module starts; its Stop closes the database.
func Module() warp.Module { return &dataModule{} }

type dataModule struct{ db *DB }

func (m *dataModule) Name() string { return "data" }

func (m *dataModule) Register(app *warp.App) error {
	m.db = &DB{app: app, names: map[string]bool{}}
	warp.Provide(app, m.db)
	return nil
}

func (m *dataModule) Migrate(ctx context.Context) error { return m.db.open(ctx) }

func (m *dataModule) Stop(ctx context.Context) error { return m.db.close() }

// For returns the app's database, which data.Module provides. Without that
// module it returns nil, and the app reports the mistake.
func For(app *warp.App) *DB { return warp.Need[*DB](app, "data") }

// DB is an app's database and the tables its stores keep.
type DB struct {
	app *warp.App

	mu      sync.Mutex
	sql     *sql.DB
	dialect dialect
	tables  []*table
	names   map[string]bool // the tables stores keep
	started bool            // Migrate ran: no store may be made now
	clock   func() time.Time
}

// SQL returns the database for queries of the app's own. It is nil until the
// app has started.
func (db *DB) SQL() *sql.DB {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.sql
}

// SetClock replaces the clock the stores read the time from, for a test; nil
// is the real one.
func (db *DB) SetClock(now func() time.Time) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.clock = now
}

// now is the time by the clock, in UTC.
func (db *DB) now() time.Time {
	db.mu.Lock()
	clock := db.clock
	db.mu.Unlock()
	if clock == nil {
		clock = time.Now
	}
	return clock().UTC()
}

// handle returns the open database and its dialect, nil before start.
func (db *DB) handle() (*sql.DB, dialect) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.sql, db.dialect
}

// add records a store's table, or reports why it cannot be kept.
func (db *DB) add(t *table) bool {
	db.mu.Lock()
	started, taken := db.started, db.names[t.name]
	if !started && !taken {
		db.names[t.name] = true
		db.tables = append(db.tables, t)
	}
	db.mu.Unlock()
	switch {
	case started:
		db.app.Mistake("the %s store was made after the app started; make it in a module's Register.", t.name)
	case taken:
		db.app.Mistake("two stores keep the table %s; make one and share it.", t.name)
	}
	return !started && !taken
}

// open reads DB_DRIVER and DB_URL, opens the database and migrates every
// table.
func (db *DB) open(ctx context.Context) error {
	db.mu.Lock()
	db.started = true
	tables := slices.Clone(db.tables)
	db.mu.Unlock()

	cfg := db.app.Config()
	driver, url := cfg.Get("DB_DRIVER"), cfg.Get("DB_URL")
	switch {
	case driver == "":
		return errors.New("warp: DB_DRIVER is not set; add DB_DRIVER=sqlite and DB_URL=file:data/app.db to .env.")
	case driver != "sqlite":
		return fmt.Errorf("warp: DB_DRIVER is %q; this version of Warp speaks sqlite.", driver)
	case !slices.Contains(sql.Drivers(), driver):
		return fmt.Errorf("warp: no database driver is named %q; import one in main.go, such as _ \"modernc.org/sqlite\".", driver)
	case url == "":
		return errors.New("warp: DB_URL is not set; add DB_URL=file:data/app.db to .env.")
	}
	var d dialect = sqlite{}
	cannot := func(err error) error {
		return fmt.Errorf("warp: the database at %s cannot be opened (%v); check DB_URL in .env.", url, err)
	}
	dsn, dir, memory := d.dsn(url)
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return cannot(err)
		}
	}
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return cannot(err)
	}
	if memory { // each connection would have a database of its own
		conn.SetMaxOpenConns(1)
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return cannot(err)
	}
	if err := migrate(ctx, conn, d, tables); err != nil {
		conn.Close()
		return err
	}
	db.mu.Lock()
	db.sql, db.dialect = conn, d
	db.mu.Unlock()
	return nil
}

func (db *DB) close() error {
	conn, _ := db.handle()
	if conn == nil {
		return nil
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("warp: the database cannot be closed (%w); check the disk it is on.", err)
	}
	return nil
}

// txKey is where a DB keeps its transaction on a context; one key per DB, so
// two databases' transactions never mix.
type txKey struct{ db *DB }

// q is what a store call on ctx runs on: the transaction on ctx, else the
// database.
func (db *DB) q(ctx context.Context, conn *sql.DB) querier {
	if tx, ok := ctx.Value(txKey{db}).(*sql.Tx); ok {
		return tx
	}
	return conn
}

// Tx runs fn in a transaction, which it commits when fn returns nil and rolls
// back otherwise, or when fn panics, which it then panics with again. A store
// call made with fn's context runs in the transaction. A Tx inside another
// one runs fn in the outer transaction, which alone commits or rolls back.
func (db *DB) Tx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{db}).(*sql.Tx); ok {
		return fn(ctx)
	}
	conn, _ := db.handle()
	if conn == nil {
		return errors.New("warp: the database was used before the app started; use it from a handler, or after Start.")
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committing := false
	defer func() {
		if !committing { // fn failed or panicked; a panic goes on after this
			tx.Rollback()
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{db}, tx)); err != nil {
		return err
	}
	committing = true
	return tx.Commit()
}
