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
// while still inside one is seen at its next use. By default a failed COMMIT
// leaves the connection outside the transaction, as modernc.org/sqlite does
// (it rolls back itself); commitLeavesOpen leaves it inside, as SQLite does
// after a busy COMMIT without the driver's help.

type fakeState struct {
	mu               sync.Mutex
	pingBusy         int   // how many pings answer busy, -1 for every one
	commitErr        error // what COMMIT answers
	commitLeavesOpen bool  // a failed COMMIT leaves the transaction open
	rollbackFails    bool  // a ROLLBACK statement fails
	rollbacks        int   // ROLLBACK statements that ended a transaction BeginTx began
	opened           int   // connections the driver opened
	begun            int
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
	st := fakeStates[name]
	st.mu.Lock()
	st.opened++
	st.mu.Unlock()
	return &fakeConn{st: st}, nil
}

type fakeConn struct {
	st    *fakeState
	inTx  bool
	rawTx bool // the transaction is one a BEGIN statement began
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
	c.st.mu.Lock()
	defer c.st.mu.Unlock()
	switch query {
	case "BEGIN":
		if c.inTx {
			return nil, errors.New("fake: cannot start a transaction within a transaction")
		}
		c.inTx, c.rawTx = true, true
		return driver.RowsAffected(0), nil
	case "ROLLBACK":
		switch {
		case c.st.rollbackFails:
			return nil, errors.New("fake: disk I/O error")
		case !c.inTx:
			return nil, errors.New("fake: cannot rollback - no transaction is active")
		}
		if !c.rawTx {
			c.st.rollbacks++ // it ended a transaction BeginTx began
		}
		c.inTx, c.rawTx = false, false
		return driver.RowsAffected(0), nil
	}
	return nil, errors.New("fake: no statements")
}

type fakeTx struct{ c *fakeConn }

func (tx fakeTx) Commit() error {
	tx.c.st.mu.Lock()
	err, open := tx.c.st.commitErr, tx.c.st.commitLeavesOpen
	tx.c.st.mu.Unlock()
	if err == nil || !open {
		tx.c.inTx = false
	}
	return err
}

func (tx fakeTx) Rollback() error {
	tx.c.inTx = false
	return nil
}

// A COMMIT that fails is the 503 when the database was busy, and leaves
// the pool clean whatever the driver did: a connection the driver already
// took out of the transaction is kept as it is, one left inside is rolled
// back and kept, and one that cannot be rolled back is thrown away.
func TestTxAfterAFailedCommit(t *testing.T) {
	busy := codedError{5, "database is locked (5) (SQLITE_BUSY)"}
	for _, c := range []struct {
		name                 string
		leavesOpen, rbFails  bool
		rollbacks, connsOpen int
	}{
		{"the driver rolled back", false, false, 0, 1},
		{"left inside", true, false, 1, 1},
		{"left inside, and the rollback fails", true, true, 0, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			pool, st := fakeDB(t)
			db := &DB{sql: pool, dialect: sqlite{}}
			st.commitErr, st.commitLeavesOpen, st.rollbackFails = busy, c.leavesOpen, c.rbFails
			if err := db.Tx(context.Background(), func(context.Context) error { return nil }); !isUnavailable(err) {
				t.Fatalf("Tx with a busy COMMIT = %v, want the 503", err)
			}
			st.commitErr, st.rollbackFails = nil, false
			if err := db.Tx(context.Background(), func(context.Context) error { return nil }); err != nil {
				t.Fatalf("the next Tx = %v: the connection was left inside a transaction", err)
			}
			if st.rollbacks != c.rollbacks || st.opened != c.connsOpen {
				t.Fatalf("%d ROLLBACKs and %d connections opened, want %d and %d", st.rollbacks, st.opened, c.rollbacks, c.connsOpen)
			}
		})
	}
	// Another failure is a plain error, not the 503.
	pool, st := fakeDB(t)
	db := &DB{sql: pool, dialect: sqlite{}}
	st.commitErr = errors.New("disk I/O error")
	if err := db.Tx(context.Background(), func(context.Context) error { return nil }); err == nil || isUnavailable(err) {
		t.Fatalf("Tx with a failing COMMIT = %v", err)
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
