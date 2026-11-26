// Package data keeps an app's records in a database through database/sql:
// a module that opens the database and makes its tables, a Store for each
// model, and transactions. SQLite is the one engine for now; the app's
// main.go imports its driver, such as _ "modernc.org/sqlite".
package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"sync"
	"sync/atomic"
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
	m.db = &DB{app: app, names: map[string]bool{}, indexes: map[string]*indexOwner{}}
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
	names   map[string]bool        // the tables stores keep
	indexes map[string]*indexOwner // the unique indexes they need, by name
	started bool                   // Migrate ran: no store may be made now
	clock   func() time.Time
}

// SQL returns the database's pool, for queries of the app's own that need it
// whole. It is nil until the app has started. It never runs in a transaction:
// for work inside Tx, use Exec, Query and QueryRow, which do.
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
	// Index names join the table and the column with _, which either may
	// hold, so two stores can ask for one name. Tables and indexes share one
	// set of names in SQLite, so a table may not be named like an index
	// either, nor an index like a table.
	var clashes []string
	if other := db.indexes[t.name]; other != nil {
		clashes = append(clashes, fmt.Sprintf("%s: the table would be named like the unique index for %s of the %s store; rename the table.",
			t.name, other.col.field, other.table))
	}
	for _, c := range t.cols {
		if !c.unique {
			continue
		}
		name := t.indexName(c)
		if other := db.indexes[name]; other != nil {
			clashes = append(clashes, fmt.Sprintf("%s: the unique index for %s would be named %s, which the %s store's %s already uses; rename one field or table.",
				t.name, c.field, name, other.table, other.col.field))
		}
		if db.names[name] {
			clashes = append(clashes, fmt.Sprintf("%s: the unique index for %s would be named %s, which is the %s store's table; rename one field or table.",
				t.name, c.field, name, name))
		}
	}
	ok := !started && !taken && len(clashes) == 0
	if ok {
		db.names[t.name] = true
		db.tables = append(db.tables, t)
		for _, c := range t.cols {
			if c.unique {
				db.indexes[t.indexName(c)] = &indexOwner{t.name, c}
			}
		}
	}
	db.mu.Unlock()
	switch {
	case started:
		db.app.Mistake("the %s store was made after the app started; make it in a module's Register.", t.name)
	case taken:
		db.app.Mistake("two stores keep the table %s; make one and share it.", t.name)
	default:
		for _, m := range clashes {
			db.app.Mistake("%s", m)
		}
	}
	return ok
}

// indexOwner is the store whose unique index has a name.
type indexOwner struct {
	table string
	col   *column
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
		return errors.New("DB_DRIVER is not set; add DB_DRIVER=sqlite and DB_URL=file:data/app.db to .env.")
	case driver != "sqlite":
		return fmt.Errorf("DB_DRIVER is %q; this version of Warp speaks sqlite.", driver)
	case !slices.Contains(sql.Drivers(), driver):
		return fmt.Errorf("no database driver is named %q; import one in main.go, such as _ \"modernc.org/sqlite\".", driver)
	case url == "":
		return errors.New("DB_URL is not set; add DB_URL=file:data/app.db to .env.")
	}
	var d dialect = sqlite{}
	cannot := func(err error) error {
		return fmt.Errorf("the database at %s cannot be opened (%v); check DB_URL in .env.", url, err)
	}
	dsn, dir, memory, notes := d.dsn(url)
	for _, note := range notes {
		slog.Warn("data: " + note)
	}
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
	if err := ping(ctx, conn, d); err != nil {
		conn.Close()
		if d.busy(err) {
			return fmt.Errorf("the database at %s is locked by another process; stop that process, or wait and start again.", url)
		}
		return cannot(err)
	}
	if err := d.settingsProblem(ctx, conn, url, memory); err != nil {
		conn.Close()
		return err
	}
	dropped, err := migrate(ctx, conn, d, tables, url)
	if err != nil {
		conn.Close()
		return err
	}
	for _, note := range dropped {
		slog.Warn("data: " + note)
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

// pingFor is how long after the first try ping may start another, and
// pingWait the pause between tries. A test may shorten pingFor.
var pingFor = time.Second

const pingWait = 100 * time.Millisecond

// ping opens the first connection, which sets the journal mode. A fresh file
// opened by several processes at once can be busy for a moment while one of
// them switches it, beyond what the busy timeout waits for, so a busy or
// locked answer is tried again, until pingFor has passed since the first try
// or ctx ends. The bound is time, not a count: when the driver has already
// waited its busy timeout (5 s by Warp's settings) in one try, no other try
// starts, and a database another process holds is named after that one wait.
func ping(ctx context.Context, conn *sql.DB, d dialect) error {
	began := time.Now()
	for {
		err := conn.PingContext(ctx)
		if err == nil || !d.busy(err) || time.Since(began) >= pingFor {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(pingWait):
		}
		if time.Since(began) >= pingFor {
			return err
		}
	}
}

// errBeforeStart is the answer to a use of the database before it is open.
var errBeforeStart = errors.New("warp: the database was used before the app started; use it from a handler, or after Start.")

// busyMessage is the answer to a call the database refused because another
// connection held it past the busy timeout.
const busyMessage = "The database is busy; try again in a moment."

// unavailable is err, or the 503 when the dialect says another connection
// held the database past the busy timeout, or errTxDone for a call on a
// transaction that had just ended. The client did not cause that, so
// it is logged, at warn level, with the request's method and path when ctx
// is a typed handler's: once for each refusal, which is once a request
// unless the handler tries again.
func unavailable(ctx context.Context, d dialect, err error) error {
	if errors.Is(err, sql.ErrTxDone) {
		// A call on fn's context that got past the check in q just as the Tx
		// ended: the same sentence as one after it.
		return errTxDone
	}
	if err == nil || d == nil || !d.busy(err) {
		return err
	}
	if r := warp.Request(ctx); r != nil {
		slog.Warn("data: the database was busy, so the request was answered 503", "method", r.Method, "path", r.URL.Path, "error", err)
	} else {
		slog.Warn("data: the database was busy, so the call was answered 503", "error", err)
	}
	return warp.Unavailable(busyMessage)
}

// on returns what a call on ctx runs on, the transaction on ctx or the pool,
// and the dialect; or the error of a call before start, or of a call on a
// transaction's context after its Tx returned.
func (db *DB) on(ctx context.Context) (querier, dialect, error) {
	conn, d := db.handle()
	if conn == nil {
		return nil, nil, errBeforeStart
	}
	q, err := db.q(ctx, conn)
	return q, d, err
}

// Exec runs a statement of the app's own, in the transaction on ctx when
// there is one (inside Tx) and on the pool otherwise. A database held past
// the busy timeout is a warp.Unavailable, a 503.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	q, d, err := db.on(ctx)
	if err != nil {
		return nil, err
	}
	res, err := q.ExecContext(ctx, query, args...)
	return res, unavailable(ctx, d, err)
}

// Query runs a query of the app's own as Exec does, and returns its rows,
// which the caller closes. Only the query's start is turned into the 503: an
// error its rows meet later (Next, Err) is the driver's own.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	q, d, err := db.on(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, query, args...)
	return rows, unavailable(ctx, d, err)
}

// QueryRow runs a query of the app's own as Exec does, for at most one row,
// which its Scan reads. Its errors wait for Scan, as database/sql's do.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *Row {
	q, d, err := db.on(ctx)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{row: q.QueryRowContext(ctx, query, args...), d: d, ctx: ctx}
}

// Row is the result of QueryRow: a *sql.Row whose errors are the database's
// own, a busy database's 503 and the error of a call before start.
type Row struct {
	row *sql.Row
	d   dialect
	ctx context.Context
	err error
}

// Scan copies the row's columns into dest, as sql.Row's Scan does; with no
// row it returns sql.ErrNoRows.
func (r *Row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return unavailable(r.ctx, r.d, r.row.Scan(dest...))
}

// Err returns the error of the query, if any, without scanning.
func (r *Row) Err() error {
	if r.err != nil {
		return r.err
	}
	return unavailable(r.ctx, r.d, r.row.Err())
}

// txKey is where a DB keeps its transaction on a context; one key per DB, so
// two databases' transactions never mix.
type txKey struct{ db *DB }

// txState is the transaction on a context, and whether its Tx has returned.
type txState struct {
	tx   *sql.Tx
	done atomic.Bool
}

// errTxDone is the answer to a call on a transaction's context after its Tx
// returned, from a goroutine that outlived it or a context kept past it.
var errTxDone = errors.New("data: the transaction this context carries has ended; use Tx's context only inside its fn.")

// q is what a store call on ctx runs on: the transaction on ctx, else the
// database.
func (db *DB) q(ctx context.Context, conn *sql.DB) (querier, error) {
	if st, ok := ctx.Value(txKey{db}).(*txState); ok {
		if st.done.Load() {
			return nil, errTxDone
		}
		return st.tx, nil
	}
	return conn, nil
}

// Tx runs fn in a transaction, which it commits when fn returns nil and rolls
// back otherwise, or when fn panics, which it then panics with again. A store
// call, and Exec, Query and QueryRow, made with fn's context run in the
// transaction; made with it after Tx returned, they are an error. A database
// another connection holds past the busy timeout, so that the transaction
// cannot begin or commit, is a warp.Unavailable, a 503. When the commit
// fails, the transaction is rolled back on its connection before the
// connection goes back to the pool, so the pool never holds one left inside
// a transaction.
//
// A Tx inside another one joins it: it runs fn in the outer transaction and
// returns fn's error, and only the outer Tx commits or rolls back. So when an
// inner fn fails and the outer fn swallows that error and returns nil, the
// outer Tx commits, with whatever the inner fn wrote before it failed.
func (db *DB) Tx(ctx context.Context, fn func(ctx context.Context) error) error {
	if st, ok := ctx.Value(txKey{db}).(*txState); ok {
		if st.done.Load() {
			return errTxDone
		}
		return fn(ctx)
	}
	pool, d := db.handle()
	if pool == nil {
		return errBeforeStart
	}
	// A connection of its own, so that it is still at hand when the commit
	// fails: database/sql hands a failed transaction's connection back as
	// it is.
	conn, err := pool.Conn(ctx)
	if err != nil {
		if err := unavailable(ctx, d, err); isUnavailable(err) {
			return err
		}
		return fmt.Errorf("data: cannot begin a transaction: %w", err)
	}
	defer conn.Close()
	// The transaction is begun with a context that the request's ending does
	// not cancel, so that database/sql never rolls it back on a goroutine of
	// its own, behind Tx's back; Tx alone ends it, here, in order. Each
	// statement in it still runs with the request's context, so a request
	// that ends interrupts the statement running.
	bg := context.WithoutCancel(ctx)
	tx, err := conn.BeginTx(bg, nil)
	if err != nil {
		if err := unavailable(ctx, d, err); isUnavailable(err) {
			return err
		}
		return fmt.Errorf("data: cannot begin a transaction: %w", err)
	}
	st := &txState{tx: tx}
	committing := false
	defer func() {
		if !committing { // fn failed or panicked; a panic goes on after this
			st.done.Store(true)
			if tx.Rollback() != nil {
				settle(bg, conn, d)
			}
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{db}, st)); err != nil {
		return err
	}
	committing = true
	st.done.Store(true) // from here on, a call on fn's context is told the Tx ended
	if err := ctx.Err(); err != nil {
		// The request ended before the commit: what fn wrote is rolled back.
		if tx.Rollback() != nil {
			settle(bg, conn, d)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		settle(bg, conn, d)
		if err := unavailable(ctx, d, err); isUnavailable(err) {
			return err
		}
		return fmt.Errorf("data: cannot commit the transaction: %w", err)
	}
	return nil
}

// settle leaves conn outside any transaction after one ended badly: a
// driver may leave the transaction open when its COMMIT fails (SQLite does
// after a busy COMMIT; modernc.org/sqlite rolls it back itself). Asked, a
// connection outside one is kept as it is; one inside is rolled back, and
// thrown away only when it is still inside one after that, so a healthy
// connection, which for a database in memory is the whole database, is
// never lost.
func settle(ctx context.Context, conn *sql.Conn, d dialect) {
	if d.outsideTx(ctx, conn) {
		return
	}
	conn.ExecContext(ctx, "ROLLBACK")
	if !d.outsideTx(ctx, conn) {
		conn.Raw(func(any) error { return driver.ErrBadConn })
	}
}

// isUnavailable reports whether err is the busy database's 503.
func isUnavailable(err error) bool {
	var e *warp.Error
	return errors.As(err, &e) && e.Status == http.StatusServiceUnavailable && e.Message == busyMessage
}
