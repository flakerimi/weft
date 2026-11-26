package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"
)

// A small database/sql driver for what SQLite cannot be made to do on cue:
// answer a ping or a COMMIT with SQLITE_BUSY. A connection that is inside a
// transaction refuses to begin another, so a connection put back in the pool
// while still inside one is seen at its next use.

type fakeState struct {
	mu        sync.Mutex
	pingBusy  int   // how many pings answer busy, -1 for every one
	commitErr error // what COMMIT answers; the transaction stays open after it, as SQLite's does
	rollbacks int   // ROLLBACKs run as statements
	begun     int
}

var (
	fakeOnce   sync.Once
	fakeMu     sync.Mutex
	fakeStates = map[string]*fakeState{}
)

// fakeDB opens a pool of the fake driver, its state named by the test.
func fakeDB(t *testing.T) (*sql.DB, *fakeState) {
	t.Helper()
	fakeOnce.Do(func() { sql.Register("warpfake", fakeDriver{}) })
	st := &fakeState{}
	fakeMu.Lock()
	fakeStates[t.Name()] = st
	fakeMu.Unlock()
	pool, err := sql.Open("warpfake", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1) // one connection, so the next use gets the same one
	t.Cleanup(func() { pool.Close() })
	return pool, st
}

type fakeDriver struct{}

func (fakeDriver) Open(name string) (driver.Conn, error) {
	fakeMu.Lock()
	defer fakeMu.Unlock()
	return &fakeConn{st: fakeStates[name]}, nil
}

type fakeConn struct {
	st   *fakeState
	inTx bool
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("fake: no statements")
}
func (c *fakeConn) Close() error { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *fakeConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if c.inTx {
		return nil, errors.New("fake: cannot start a transaction within a transaction")
	}
	c.inTx = true
	c.st.mu.Lock()
	c.st.begun++
	c.st.mu.Unlock()
	return fakeTx{c}, nil
}

func (c *fakeConn) Ping(context.Context) error {
	c.st.mu.Lock()
	defer c.st.mu.Unlock()
	if c.st.pingBusy != 0 {
		if c.st.pingBusy > 0 {
			c.st.pingBusy--
		}
		return codedError{5, "database is locked (5) (SQLITE_BUSY)"}
	}
	return nil
}

func (c *fakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if query == "ROLLBACK" {
		c.inTx = false
		c.st.mu.Lock()
		c.st.rollbacks++
		c.st.mu.Unlock()
		return driver.RowsAffected(0), nil
	}
	return nil, errors.New("fake: no statements")
}

type fakeTx struct{ c *fakeConn }

func (tx fakeTx) Commit() error {
	tx.c.st.mu.Lock()
	err := tx.c.st.commitErr
	tx.c.st.mu.Unlock()
	if err == nil {
		tx.c.inTx = false
	}
	return err
}

func (tx fakeTx) Rollback() error {
	tx.c.inTx = false
	return nil
}

// A COMMIT the database refuses as busy is the 503, and the transaction is
// rolled back on its connection, so the pool's next use begins cleanly.
func TestTxRollsBackAFailedCommit(t *testing.T) {
	pool, st := fakeDB(t)
	db := &DB{sql: pool, dialect: sqlite{}}
	st.commitErr = codedError{5, "database is locked (5) (SQLITE_BUSY)"}
	if err := db.Tx(context.Background(), func(context.Context) error { return nil }); !isUnavailable(err) {
		t.Fatalf("Tx with a busy COMMIT = %v, want the 503", err)
	}
	if st.rollbacks != 1 {
		t.Fatalf("%d ROLLBACKs after the failed COMMIT, want 1", st.rollbacks)
	}
	st.commitErr = nil
	if err := db.Tx(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("the next Tx = %v: the connection was left inside a transaction", err)
	}
	// Another failure is a plain error, not the 503, and rolled back too.
	st.commitErr = errors.New("disk I/O error")
	if err := db.Tx(context.Background(), func(context.Context) error { return nil }); err == nil || isUnavailable(err) {
		t.Fatalf("Tx with a failing COMMIT = %v", err)
	}
	st.commitErr = nil
	if err := db.Tx(context.Background(), func(context.Context) error { return nil }); err != nil || st.rollbacks != 2 {
		t.Fatalf("after a failing COMMIT: Tx = %v, %d ROLLBACKs", err, st.rollbacks)
	}
}

// A context a Tx gave fn is no use once the Tx returned: a call on it says
// so, rather than database/sql's own words.
func TestTxContextAfterTx(t *testing.T) {
	pool, _ := fakeDB(t)
	db := &DB{sql: pool, dialect: sqlite{}}
	var kept context.Context
	if err := db.Tx(context.Background(), func(ctx context.Context) error { kept = ctx; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(kept, "UPDATE x SET y = 1"); !errors.Is(err, errTxDone) {
		t.Fatalf("Exec on a finished Tx's context = %v", err)
	}
	if err := db.Tx(kept, func(context.Context) error { return nil }); !errors.Is(err, errTxDone) {
		t.Fatalf("Tx on a finished Tx's context = %v", err)
	}
	if errTxDone.Error() != "data: the transaction this context carries has ended; use Tx's context only inside its fn." {
		t.Fatalf("sentence %q", errTxDone)
	}
}

// A database another process holds while the app opens it is tried again
// until pingFor has passed since the first try, and not after; one released
// while it is tried is opened. (A busy timeout of Warp's is never lowered,
// so SQLite itself cannot be made to answer busy at once for this.)
func TestPingRetriesUntilItsDeadline(t *testing.T) {
	defer func(old time.Duration) { pingFor = old }(pingFor)
	pingFor = 300 * time.Millisecond

	pool, st := fakeDB(t)
	st.pingBusy = -1
	began := time.Now()
	err := ping(context.Background(), pool, sqlite{})
	took := time.Since(began)
	if !(sqlite{}).busy(err) {
		t.Fatalf("ping = %v, want the busy error", err)
	}
	// Loose above, for a loaded machine.
	if took < pingFor || took > pingFor+3*time.Second {
		t.Fatalf("ping took %v, want about %v", took, pingFor)
	}

	t.Run("released", func(t *testing.T) {
		pool, st := fakeDB(t)
		st.pingBusy = 2
		if err := ping(context.Background(), pool, sqlite{}); err != nil {
			t.Fatalf("ping = %v, want it to wait for the lock", err)
		}
	})
}
