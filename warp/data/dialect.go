package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	neturl "net/url"
	"path/filepath"
	"slices"
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
	// uniqueColumn reports the column of table whose unique index err
	// violates.
	uniqueColumn(err error, table string) (string, bool)
	// busy reports whether err says another connection holds the database.
	busy(err error) bool
	// dsn is the driver's name for url with the settings every connection
	// needs, the folder to make for the file (none for memory), and whether
	// the database lives in memory, where one connection must hold it.
	dsn(url string) (dsn, dir string, memory bool)
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
	rows, err := q.QueryContext(ctx, `SELECT "name", "type", "notnull" FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]columnInfo{}
	for rows.Next() {
		var name string
		var info columnInfo
		if err := rows.Scan(&name, &info.typ, &info.notNull); err != nil {
			return nil, err
		}
		cols[name] = info
	}
	return cols, rows.Err()
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

// busy reads SQLite's words for a database another connection holds, which
// modernc.org/sqlite writes as "database is locked (5) (SQLITE_BUSY)"; dbtest
// fails if they change.
func (sqlite) busy(err error) bool {
	text := err.Error()
	return strings.Contains(text, "database is locked") || strings.Contains(text, "database table is locked") ||
		strings.Contains(text, "SQLITE_BUSY") || strings.Contains(text, "SQLITE_LOCKED")
}

// settings are set on every connection the pool opens, through the DSN, which
// the driver applies to each new connection: foreign keys on, the WAL journal
// so that readers and a writer do not block each other, 5 s of waiting for a
// lock before "database is locked", and transactions that take the write lock
// when they begin, so two of them never deadlock upgrading a read. The names
// are the ones both modernc.org/sqlite and mattn/go-sqlite3 read.
const settings = "_busy_timeout=5000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate"

// ownSettings are the keys of settings the store relies on, with the
// drivers' aliases for them: dsn drops them from a DB_URL, so Warp's values
// are the connection's whatever the URL says. A busy timeout is only how long
// to wait, so a DB_URL's (_busy_timeout or _timeout) is kept, and wins, since
// the drivers read the first value.
var ownSettings = []string{"_txlock", "_journal_mode", "_journal", "_foreign_keys", "_fk"}

func (sqlite) dsn(url string) (string, string, bool) {
	path, query, _ := strings.Cut(url, "?")
	var kept []string
	for part := range strings.SplitSeq(query, "&") {
		key, _, _ := strings.Cut(part, "=")
		if unescaped, err := neturl.QueryUnescape(key); err == nil {
			key = unescaped
		}
		if part != "" && !slices.Contains(ownSettings, key) {
			kept = append(kept, part)
		}
	}
	dsn := path + "?" + strings.Join(append(kept, settings), "&")
	values := "&" + query + "&"
	if path == ":memory:" || path == "file::memory:" || path == "file:" || path == "" || strings.Contains(values, "&mode=memory&") {
		return dsn, "", true
	}
	if rest, ok := strings.CutPrefix(path, "file:"); ok { // a URI, percent-escaped
		path = strings.TrimPrefix(rest, "//") // file:///srv/app.db has no host
		if unescaped, err := neturl.PathUnescape(path); err == nil {
			path = unescaped
		}
	}
	return dsn, filepath.Dir(path), false
}

// contains uses LIKE, which in SQLite ignores the case of ASCII letters alone,
// with \ as its escape character.
func (d sqlite) contains(name string, n int) string {
	return d.quote(name) + " LIKE " + d.placeholder(n) + ` ESCAPE '\'`
}

// likeEscapes escapes the escape character and LIKE's two wildcards.
var likeEscapes = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (sqlite) containsText(s string) string { return "%" + likeEscapes.Replace(s) + "%" }
