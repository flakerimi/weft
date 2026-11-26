package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	neturl "net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// dialect is what the store needs to know of one database engine. The store
// writes standard SQL and takes everything engine-specific from here, so the
// next engine adds a dialect and does not touch the store.
type dialect interface {
	// placeholder is the n-th value's placeholder, counted from 1.
	placeholder(n int) string
	// quote writes name as a quoted identifier.
	quote(name string) string
	// columnType is the column type a field of kind k is kept in.
	columnType(k kind) string
	// zero is the literal default of a NOT NULL column of kind k.
	zero(k kind) string
	// idColumn is the type and constraints of the id column.
	idColumn() string
	// timeValue is how a time is written; scanTime reads it back, in UTC.
	timeValue(t time.Time) any
	scanTime(src any) (time.Time, error)
	// columns reads the table's columns by name; none when the table does
	// not exist.
	columns(ctx context.Context, q querier, table string) (map[string]columnInfo, error)
	// index reads the index named name, wherever it is; ok is false when
	// there is none.
	index(ctx context.Context, q querier, name string) (info indexInfo, ok bool, err error)
	// madeIndexes lists the names of the indexes on table that were made by
	// CREATE INDEX, not by a constraint of the table's own.
	madeIndexes(ctx context.Context, q querier, table string) ([]string, error)
	// uniqueColumn reports the column of table whose unique index err
	// violates.
	uniqueColumn(err error, table string) (string, bool)
	// busy reports whether err says another connection holds the database.
	busy(err error) bool
	// settingsProblem reads back, on connections of pool, each setting the
	// store relies on, and returns the sentence naming DB_URL when one is not
	// what the store needs; url is the DB_URL as written, memory whether the
	// database lives in memory.
	settingsProblem(ctx context.Context, pool *sql.DB, url string, memory bool) error
	// outsideTx reports whether conn is outside any transaction, asking the
	// connection itself and leaving it as it found it; false when it is
	// inside one or cannot tell.
	outsideTx(ctx context.Context, conn *sql.Conn) bool
	// dsn is the driver's name for url with the settings every connection
	// needs, the folder to make for the file (none for memory), whether the
	// database lives in memory, where one connection must hold it, and a
	// sentence for each setting of url's that Warp's replaced.
	dsn(url string) (dsn, dir string, memory bool, notes []string)
	// contains is the condition that the column named name holds the n-th
	// value, written by containsText, ignoring case as far as the engine does.
	contains(name string, n int) string
	// containsText is the value contains takes for the text s, every wildcard
	// in s escaped so that it is taken literally.
	containsText(s string) string
}

// columnInfo is what the database says of one column.
type columnInfo struct {
	typ     string // the declared type
	notNull bool
	dflt    bool // it has a default
	pk      bool // it is part of the primary key
}

// indexInfo is what the database says of one index.
type indexInfo struct {
	table           string
	unique, partial bool
	columns         []string // in order; "" for an expression
}

// sqlite is the dialect of SQLite.
type sqlite struct{}

func (sqlite) placeholder(int) string { return "?" }

func (sqlite) quote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func (sqlite) columnType(k kind) string {
	switch k {
	case kindInteger, kindBool:
		return "INTEGER"
	case kindReal:
		return "REAL"
	case kindBlob:
		return "BLOB"
	}
	return "TEXT" // text and times
}

func (d sqlite) zero(k kind) string {
	switch k {
	case kindInteger, kindBool, kindReal:
		return "0"
	case kindBlob:
		return "X''"
	case kindTime:
		return "'" + d.timeValue(time.Time{}).(string) + "'"
	}
	return "''"
}

func (sqlite) idColumn() string { return "INTEGER PRIMARY KEY AUTOINCREMENT" }

// timeLayout is RFC 3339 with all nine digits of the nanoseconds, so that
// every time has one width and the text sorts as the times do.
const timeLayout = "2006-01-02T15:04:05.000000000Z07:00"

func (sqlite) timeValue(t time.Time) any { return t.UTC().Format(timeLayout) }

func (sqlite) scanTime(src any) (time.Time, error) {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case time.Time: // a driver that reads times itself
		return v.UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("%v is not a time", src)
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a time", s)
	}
	return t.UTC(), nil
}

func (sqlite) columns(ctx context.Context, q querier, table string) (map[string]columnInfo, error) {
	rows, err := q.QueryContext(ctx, `SELECT "name", "type", "notnull", "dflt_value" IS NOT NULL, "pk" > 0 FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]columnInfo{}
	for rows.Next() {
		var name string
		var info columnInfo
		if err := rows.Scan(&name, &info.typ, &info.notNull, &info.dflt, &info.pk); err != nil {
			return nil, err
		}
		cols[name] = info
	}
	return cols, rows.Err()
}

func (sqlite) madeIndexes(ctx context.Context, q querier, table string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT "name" FROM pragma_index_list(?) WHERE "origin" = 'c' ORDER BY "name"`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (sqlite) index(ctx context.Context, q querier, name string) (indexInfo, bool, error) {
	var info indexInfo
	err := q.QueryRowContext(ctx, `SELECT m."tbl_name", l."unique", l."partial" FROM "sqlite_master" AS m `+
		`JOIN pragma_index_list(m."tbl_name") AS l ON l."name" = m."name" WHERE m."type" = 'index' AND m."name" = ?`, name).
		Scan(&info.table, &info.unique, &info.partial)
	if errors.Is(err, sql.ErrNoRows) {
		return info, false, nil
	}
	if err != nil {
		return info, false, err
	}
	rows, err := q.QueryContext(ctx, `SELECT "name" FROM pragma_index_info(?) ORDER BY "seqno"`, name)
	if err != nil {
		return info, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var col sql.NullString // NULL for an expression
		if err := rows.Scan(&col); err != nil {
			return info, false, err
		}
		info.columns = append(info.columns, col.String)
	}
	return info, true, rows.Err()
}

// uniqueColumn reads SQLite's own words, "UNIQUE constraint failed:
// products.name", which the drivers pass on; dbtest fails if they change. An
// index on several columns lists them with ", " and is not one column.
func (sqlite) uniqueColumn(err error, table string) (string, bool) {
	_, rest, ok := strings.Cut(err.Error(), "UNIQUE constraint failed: "+table+".")
	if !ok {
		return "", false
	}
	if i := strings.Index(rest, " ("); i >= 0 { // the driver's result code
		rest = rest[:i]
	}
	if rest == "" || strings.Contains(rest, ", ") {
		return "", false
	}
	return rest, true
}

// busy reports whether err says another connection held the database past
// the busy timeout, which trying again in a moment may cure. It reads the
// driver's result code when an error in the chain has one (Code() int, as
// modernc.org/sqlite's errors do): SQLITE_BUSY, 5, in any of its extended
// forms (BUSY_RECOVERY, BUSY_SNAPSHOT, BUSY_TIMEOUT). SQLITE_LOCKED, a
// conflict within one connection, is not: trying again never cures it. With
// no code in the chain (mattn/go-sqlite3's errors have none) it reads the
// words SQLite writes for it, "database is locked", as the whole message of
// the innermost error, so that a value quoted in an error never matches.
func (sqlite) busy(err error) bool {
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		return coded.Code()&0xff == sqliteBusy
	}
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			break
		}
		err = inner
	}
	return err.Error() == "database is locked"
}

// settingsProblem asks the database rather than trusting the parse of
// DB_URL: a _pragma is SQL, which no rewriting can follow in every spelling
// (two statements, a quoted name, a comment). On two connections (one for a
// database in memory, which has one), held at once, since every connection
// runs the settings anew, it reads back:
//   - busy_timeout, which must be at least busyTimeout;
//   - journal_mode, which must be wal (memory for a database in memory);
//   - foreign_keys, which must be 1;
//   - locking_mode, which must be normal.
//
// A _pragma that names journal_mode or locking_mode is refused before that,
// whatever it sets: the drivers run it on each new connection before
// Warp's _journal_mode, so it switches the file's mode while another
// connection has it open, and the second connection then fails to open;
// that cannot be read back afterwards. SQLite compares a name without regard
// to case and cannot spell it otherwise (quotes, brackets and comments go
// around it), so a search for the name finds every spelling.
//
// The transaction lock mode cannot be read back: a connection does not say
// how its driver begins a transaction. It holds because SQL cannot set it (no
// pragma does), and dsn drops the one key the drivers read it from, _txlock,
// in every form their parser accepts (any escape of the key; the key's case
// is the drivers' to match, and they match it exactly).
func (sqlite) settingsProblem(ctx context.Context, pool *sql.DB, url string, memory bool) error {
	_, query, _ := strings.Cut(url, "?")
	for part := range strings.SplitSeq(query, "&") {
		key, value, _ := strings.Cut(part, "=")
		if unescaped, err := neturl.QueryUnescape(key); err == nil {
			key = unescaped
		}
		if unescaped, err := neturl.QueryUnescape(value); err == nil {
			value = unescaped
		}
		if key != "_pragma" {
			continue
		}
		for _, name := range []string{"journal_mode", "locking_mode"} {
			if strings.Contains(strings.ToLower(value), name) {
				return fmt.Errorf("DB_URL sets %s in _pragma=%s, and the store needs Warp's own; remove it from DB_URL.", name, value)
			}
		}
	}
	n := 2
	if memory {
		n = 1
	}
	var conns []*sql.Conn
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()
	for i := range n {
		c, err := pool.Conn(ctx)
		if err != nil {
			return fmt.Errorf("connection %d to the database cannot be opened (%v); check the settings in DB_URL.", i+1, err)
		}
		conns = append(conns, c)
		var wait, fk int
		var journal, locking string
		for _, read := range []struct {
			pragma string
			into   any
		}{{"busy_timeout", &wait}, {"journal_mode", &journal}, {"foreign_keys", &fk}, {"locking_mode", &locking}} {
			if err := c.QueryRowContext(ctx, "PRAGMA "+read.pragma).Scan(read.into); err != nil {
				return fmt.Errorf("the database's %s cannot be read (%v); check the settings in DB_URL.", read.pragma, err)
			}
		}
		journal, locking = strings.ToLower(journal), strings.ToLower(locking)
		switch {
		case wait < busyTimeout:
			return fmt.Errorf("DB_URL sets busy_timeout to %d, and the store needs at least %d; remove it from DB_URL.", wait, busyTimeout)
		case journal != "wal" && !(memory && journal == "memory"):
			return fmt.Errorf("DB_URL sets journal_mode to %s, and the store needs wal; remove it from DB_URL.", journal)
		case fk != 1:
			return fmt.Errorf("DB_URL sets foreign_keys to %d, and the store needs 1; remove it from DB_URL.", fk)
		case locking != "normal":
			return fmt.Errorf("DB_URL sets locking_mode to %s, and the store needs normal; remove it from DB_URL.", locking)
		}
	}
	return nil
}

// outsideTx asks SQLite whether the connection is in autocommit mode, outside
// any transaction: a BEGIN succeeds only then, and is rolled back at once.
// BEGIN (deferred) and ROLLBACK take no lock, so neither waits on another
// connection.
func (sqlite) outsideTx(ctx context.Context, conn *sql.Conn) bool {
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		return false // already inside one, or the connection is not well
	}
	_, err := conn.ExecContext(ctx, "ROLLBACK")
	return err == nil
}

// sqliteBusy is SQLITE_BUSY, the low byte of each of its extended codes.
const sqliteBusy = 5

// settings are set on every connection the pool opens, through the DSN, which
// the driver applies to each new connection: foreign keys on, the WAL journal
// so that readers and a writer do not block each other, and transactions
// that take the write lock when they begin, so two of them never deadlock
// upgrading a read; beside them, busyTimeout. The names are the ones both
// modernc.org/sqlite and mattn/go-sqlite3 read.
const settings = "_foreign_keys=1&_journal_mode=WAL&_txlock=immediate"

// busyTimeout is how many milliseconds a connection waits for a lock before
// "database is locked". The store relies on it: the app's own writers take
// turns by waiting it out.
const busyTimeout = 5000

// ownSettings are the keys of settings the store relies on, with the
// drivers' aliases for them, and Warp's value for each: dsn drops them from
// a DB_URL, so Warp's values are the connection's whatever the URL says.
var ownSettings = map[string]string{
	"_txlock": "immediate", "_journal_mode": "WAL", "_journal": "WAL", "_foreign_keys": "1", "_fk": "1",
}

// The forms of a DB_URL that set the busy timeout, as the drivers read them
// (modernc.org/sqlite v1.63.0, sqlite.go, applyQueryParams; mattn/go-sqlite3
// reads the first two):
//   - _busy_timeout=N, and its alias _timeout=N, which wins when both are
//     given; only the first value of a key repeated is read;
//   - _pragma=busy_timeout(N) and _pragma=busy_timeout=N, run as "PRAGMA
//     busy_timeout…" after the two keys above, so any of them wins over
//     those; every _pragma given is run, in any case, with spaces, a schema
//     before the name and the value quoted, as SQL takes them.
//
// Every key and value may be percent-escaped, a + is a space. A DB_URL may
// raise the busy timeout above busyTimeout and never lower it: dsn drops
// every one of these and writes the largest of Warp's and theirs. A value
// that is not a whole number from 0 to 2147483647 (SQLite's int; a larger
// one is read as none) counts as not given.
var busyKeys = []string{"_busy_timeout", "_timeout"}

// pragmaBusy reads a _pragma value that sets the busy timeout: ok when it
// is one, and its value as written.
func pragmaBusy(pragma string) (value string, ok bool) {
	p := strings.ToLower(strings.TrimSpace(pragma))
	if i := strings.IndexByte(p, '.'); i >= 0 && !strings.ContainsAny(p[:i], "=( ") {
		p = strings.TrimSpace(p[i+1:]) // a schema: main.busy_timeout
	}
	rest, ok := strings.CutPrefix(p, "busy_timeout")
	if !ok {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	switch {
	case strings.HasPrefix(rest, "="):
		value = rest[1:]
	case strings.HasPrefix(rest, "(") && strings.HasSuffix(rest, ")"):
		value = rest[1 : len(rest)-1]
	case rest == "":
		return "", false // a question, PRAGMA busy_timeout, which sets nothing
	default:
		value = rest // not SQL the driver can run; read as a value not given
	}
	value = strings.Trim(strings.TrimSpace(value), `'"`)
	return value, true
}

// waitOf reads a busy timeout as SQLite takes it, ok false when it is not a
// whole number SQLite keeps.
func waitOf(value string) (int, bool) {
	n, err := strconv.ParseInt(strings.TrimPrefix(value, "+"), 10, 64)
	if err != nil || n > math.MaxInt32 {
		return 0, false
	}
	return int(max(n, 0)), true
}

func (sqlite) dsn(url string) (dsn, dir string, memory bool, notes []string) {
	path, query, _ := strings.Cut(url, "?")
	var kept []string
	wait := busyTimeout
	for part := range strings.SplitSeq(query, "&") {
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		if unescaped, err := neturl.QueryUnescape(key); err == nil {
			key = unescaped
		}
		if unescaped, err := neturl.QueryUnescape(value); err == nil {
			value = unescaped
		}
		if own, ok := ownSettings[key]; ok {
			if !strings.EqualFold(value, own) {
				notes = append(notes, fmt.Sprintf("DB_URL sets %s=%s, which Warp sets itself; Warp's %s is used, since the store relies on it.", key, value, own))
			}
			continue
		}
		given, isBusy := value, slices.Contains(busyKeys, key)
		if key == "_pragma" {
			given, isBusy = pragmaBusy(value)
		}
		if isBusy {
			written := key + "=" + value
			n, ok := waitOf(given)
			switch {
			case !ok:
				notes = append(notes, fmt.Sprintf("DB_URL sets %s, which is not a busy timeout SQLite keeps; Warp's %d is used.", written, busyTimeout))
			case n < busyTimeout:
				notes = append(notes, fmt.Sprintf("DB_URL sets %s, below Warp's %d; %d is used, since the app's own writers wait on it.", written, busyTimeout, busyTimeout))
			}
			if ok {
				wait = max(wait, n)
			}
			continue
		}
		kept = append(kept, part)
	}
	kept = append(kept, "_busy_timeout="+strconv.Itoa(wait), settings)
	dsn = path + "?" + strings.Join(kept, "&")
	values := "&" + query + "&"
	if path == ":memory:" || path == "file::memory:" || path == "file:" || path == "" || strings.Contains(values, "&mode=memory&") {
		return dsn, "", true, notes
	}
	if rest, ok := strings.CutPrefix(path, "file:"); ok { // a URI, percent-escaped
		path = strings.TrimPrefix(rest, "//") // file:///srv/app.db has no host
		if unescaped, err := neturl.PathUnescape(path); err == nil {
			path = unescaped
		}
	}
	return dsn, filepath.Dir(path), false, notes
}

// contains uses LIKE, which in SQLite ignores the case of ASCII letters alone,
// with \ as its escape character.
func (d sqlite) contains(name string, n int) string {
	return d.quote(name) + " LIKE " + d.placeholder(n) + ` ESCAPE '\'`
}

// likeEscapes escapes the escape character and LIKE's two wildcards.
var likeEscapes = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (sqlite) containsText(s string) string { return "%" + likeEscapes.Replace(s) + "%" }
