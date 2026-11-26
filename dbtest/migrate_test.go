package dbtest

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

// columns returns the table's columns as name and declared type.
func columns(t *testing.T, db *data.DB, table string) map[string]string {
	t.Helper()
	rows, err := db.SQL().Query(`SELECT "name", "type" FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			t.Fatal(err)
		}
		out[name] = typ
	}
	return out
}

func TestMigrateCreatesAndAddsColumns(t *testing.T) {
	file := dbFile(t)
	type V1 struct {
		data.Record
		Name string `json:"name"`
	}
	app := warp.New(warp.ConfigFrom(env(file)))
	var s1 *data.Store[V1]
	app.Use(data.Module(), stores{"products", func(db *data.DB) { s1 = data.NewStore[V1](db, "products") }})
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if got := columns(t, data.For(app), "products"); !reflect.DeepEqual(got, map[string]string{
		"id": "INTEGER", "created_at": "TEXT", "updated_at": "TEXT", "name": "TEXT",
	}) {
		t.Fatalf("columns %v", got)
	}
	for _, name := range []string{"Lamp", "Desk"} {
		if err := s1.Create(ctx, &V1{Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	// A column the code does not know is left alone.
	if _, err := data.For(app).SQL().Exec(`ALTER TABLE "products" ADD COLUMN "legacy" TEXT`); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	// The same file, with three more fields: the rows stay and take the zero values.
	type V2 struct {
		data.Record
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Code  *string `json:"code"`
		SKU   *string `json:"sku" warp:"unique"` // a pointer, so the rows there hold NULL
	}
	c, s2 := shop[V2](t, file)
	got := columns(t, data.For(c.App), "products")
	if want := map[string]string{
		"id": "INTEGER", "created_at": "TEXT", "updated_at": "TEXT", "name": "TEXT",
		"legacy": "TEXT", "price": "REAL", "code": "TEXT", "sku": "TEXT",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("columns %v, want %v", got, want)
	}
	lamp, err := s2.Get(ctx, 1)
	if err != nil || lamp.Name != "Lamp" || lamp.Price != 0 || lamp.Code != nil || lamp.SKU != nil {
		t.Fatalf("Get = %+v, %v", lamp, err)
	}
	if desk, err := s2.Get(ctx, 2); err != nil || desk.Name != "Desk" {
		t.Fatalf("Get = %+v, %v", desk, err)
	}
	sku := "C-1"
	if err := s2.Create(ctx, &V2{Name: "Chair", Price: 30, SKU: &sku}); err != nil {
		t.Fatal(err)
	}
	if err := s2.Create(ctx, &V2{Name: "Stool", SKU: &sku}); err == nil {
		t.Fatal("a second C-1 was taken")
	}
}

func TestMigrateRefusesChangedType(t *testing.T) {
	file := dbFile(t)
	type V1 struct {
		data.Record
		Name  string `json:"name"`
		Price string `json:"price"`
	}
	app := warp.New(warp.ConfigFrom(env(file)))
	var s1 *data.Store[V1]
	app.Use(data.Module(), stores{"products", func(db *data.DB) { s1 = data.NewStore[V1](db, "products") }})
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s1.Create(ctx, &V1{Name: "Lamp", Price: "cheap"}); err != nil {
		t.Fatal(err)
	}
	before := columns(t, data.For(app), "products")
	app.Stop(ctx)

	// Price becomes a number, and a new column comes with it: neither is made.
	type Product struct {
		data.Record
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Stock int     `json:"stock"`
	}
	app = warp.New(warp.ConfigFrom(env(file)))
	app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
	err := app.Start(ctx)
	want := `warp: module "data" could not migrate: table products has column price as TEXT, and Product.Price needs REAL; change the field back, or alter the column yourself, since Warp never changes or drops a column.`
	if err == nil || err.Error() != want {
		t.Fatalf("Start = %v\nwant %s", err, want)
	}

	// The table is as it was, rows and all.
	c, s := shop[V1](t, file)
	if got := columns(t, data.For(c.App), "products"); !reflect.DeepEqual(got, before) {
		t.Fatalf("columns %v, want %v", got, before)
	}
	if lamp, err := s.Get(ctx, 1); err != nil || lamp.Price != "cheap" {
		t.Fatalf("Get = %+v, %v", lamp, err)
	}
}

// Migrating is one transaction: a failure in a later table leaves an
// earlier one unmade.
func TestMigrateIsOneTransaction(t *testing.T) {
	file := dbFile(t)
	type Old struct {
		data.Record
		Price string `json:"price"`
	}
	c, _ := shop[Old](t, file)
	c.App.Stop(ctx)

	type Brand struct {
		data.Record
		Name string `json:"name"`
	}
	type New struct {
		data.Record
		Price float64 `json:"price"`
	}
	app := warp.New(warp.ConfigFrom(env(file)))
	app.Use(data.Module(), stores{"shop", func(db *data.DB) {
		data.NewStore[Brand](db, "brands")
		data.NewStore[New](db, "products")
	}})
	if err := app.Start(ctx); err == nil || !strings.Contains(err.Error(), "has column price as TEXT") {
		t.Fatalf("Start = %v", err)
	}
	c, _ = shop[Old](t, file)
	if got := columns(t, data.For(c.App), "brands"); len(got) != 0 {
		t.Fatalf("brands was made: %v", got)
	}
}

// A unique index on a column whose values repeat cannot be made, and says so.
func TestMigrateUniqueOnRepeatedValues(t *testing.T) {
	file := dbFile(t)
	type V1 struct {
		data.Record
		Name string `json:"name"`
	}
	c, s := shop[V1](t, file)
	for range 2 {
		if err := s.Create(ctx, &V1{Name: "Lamp"}); err != nil {
			t.Fatal(err)
		}
	}
	c.App.Stop(ctx)
	type V2 struct {
		data.Record
		Name string `json:"name" warp:"unique"`
	}
	app := warp.New(warp.ConfigFrom(env(file)))
	app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[V2](db, "products") }})
	err := app.Start(ctx)
	want := `warp: module "data" could not migrate: table products has the same name twice or more, so V2.Name cannot be unique; make the values differ, make the field a pointer so that rows without a value hold NULL, or remove the rule.`
	if err == nil || err.Error() != want {
		t.Fatalf("Start = %v\nwant %s", err, want)
	}
}

// A pointer field may be empty, which a NOT NULL column cannot hold; the
// reverse, a plain field over a column that may be NULL, is fine.
func TestMigrateRefusesPointerOverNotNull(t *testing.T) {
	file := dbFile(t)
	type V1 struct {
		data.Record
		Code  string  `json:"code"`
		Label *string `json:"label"`
	}
	c, _ := shop[V1](t, file)
	c.App.Stop(ctx)

	type Product struct {
		data.Record
		Code  *string `json:"code"`
		Label string  `json:"label"`
	}
	app := warp.New(warp.ConfigFrom(env(file)))
	app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
	err := app.Start(ctx)
	want := `warp: module "data" could not migrate: table products has column code as NOT NULL, and Product.Code may be empty (it is a pointer); make the field a plain value, or alter the column yourself.`
	if err == nil || err.Error() != want {
		t.Fatalf("Start = %v\nwant %s", err, want)
	}

	type V3 struct {
		data.Record
		Code  string `json:"code"`
		Label string `json:"label"`
	}
	shop[V3](t, file) // a plain field over a nullable column starts
}

// An index with the name Warp would give is used only when it is a unique
// index on exactly that column of that table; anything else stops the start,
// and Warp neither changes nor drops it.
func TestMigrateChecksIndexOfThatName(t *testing.T) {
	type V1 struct {
		data.Record
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}
	type V2 struct {
		data.Record
		Name  string  `json:"name" warp:"unique"`
		Price float64 `json:"price"`
	}
	for _, c := range []struct{ what, index string }{
		{"another column", `CREATE UNIQUE INDEX "products_name_unique" ON "products" ("price")`},
		{"not unique", `CREATE INDEX "products_name_unique" ON "products" ("name")`},
		{"two columns", `CREATE UNIQUE INDEX "products_name_unique" ON "products" ("name", "price")`},
		{"partial", `CREATE UNIQUE INDEX "products_name_unique" ON "products" ("name") WHERE "price" > 0`},
		{"another table", `CREATE UNIQUE INDEX "products_name_unique" ON "brands" ("name")`},
	} {
		t.Run(c.what, func(t *testing.T) {
			file := dbFile(t)
			cl, _ := shop[V1](t, file)
			db := data.For(cl.App).SQL()
			if _, err := db.Exec(`CREATE TABLE "brands" ("name" TEXT)`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(c.index); err != nil {
				t.Fatal(err)
			}
			cl.App.Stop(ctx)

			app := warp.New(warp.ConfigFrom(env(file)))
			app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[V2](db, "products") }})
			err := app.Start(ctx)
			want := `warp: module "data" could not migrate: table products has an index products_name_unique that does not cover column name; rename or drop that index yourself, since Warp never changes one.`
			if err == nil || err.Error() != want {
				t.Fatalf("Start = %v\nwant %s", err, want)
			}
			// The index is as it was.
			cl, _ = shop[V1](t, file)
			var sqlText string
			if err := data.For(cl.App).SQL().QueryRow(`SELECT "sql" FROM "sqlite_master" WHERE "name" = 'products_name_unique'`).Scan(&sqlText); err != nil || sqlText != c.index {
				t.Fatalf("index %q, %v", sqlText, err)
			}
		})
	}

	// Its own index, from an earlier start, is taken as it is.
	file := dbFile(t)
	cl, s := shop[V2](t, file)
	if err := s.Create(ctx, &V2{Name: "Lamp"}); err != nil {
		t.Fatal(err)
	}
	cl.App.Stop(ctx)
	_, s = shop[V2](t, file)
	if err := s.Create(ctx, &V2{Name: "Lamp"}); err == nil {
		t.Fatal("a second Lamp was taken after a restart")
	}
}

// lock holds an exclusive lock on the database file until the returned
// function is called, or the test ends.
func lock(t *testing.T, file string) (release func()) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	holder, err := sql.Open("sqlite", "file:"+file)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { holder.Close() })
	conn, err := holder.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"BEGIN EXCLUSIVE", `CREATE TABLE "held" ("x" INTEGER)`} {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			conn.ExecContext(ctx, "COMMIT")
			conn.Close()
		})
	}
	t.Cleanup(release)
	return release
}

// startLocked starts an app over url and returns what Start said and how
// long it took.
func startLocked(t *testing.T, url string) (error, time.Duration) {
	t.Helper()
	app := warp.New(warp.ConfigFrom(map[string]string{"DB_DRIVER": "sqlite", "DB_URL": url}))
	app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
	began := time.Now()
	err := app.Start(ctx)
	took := time.Since(began)
	if err == nil {
		app.Stop(ctx)
	}
	return err, took
}

// Another process holding the file while the journal mode is set is waited
// for and then named; it is not a wrong DB_URL. The opening is retried for
// about a second from the first try, and no new try starts after that.
func TestStartWhileLocked(t *testing.T) {
	locked := func(url string) string {
		return `warp: module "data" could not migrate: the database at ` + url + ` is locked by another process; stop that process, or wait and start again.`
	}

	t.Run("default busy timeout", func(t *testing.T) {
		// The first try waits the busy timeout Warp sets, _busy_timeout=5000 (5 s),
		// inside the driver; that is past the retry deadline, so there is no
		// second try. Ten tries of 5 s each would take 50 s.
		file := dbFile(t)
		lock(t, file)
		url := "file:" + file
		err, took := startLocked(t, url)
		if err == nil || err.Error() != locked(url) {
			t.Fatalf("Start = %v\nwant %s", err, locked(url))
		}
		// Loose above, for a loaded machine; ten tries would still be past it.
		if took < 4*time.Second || took > 25*time.Second {
			t.Fatalf("Start took %v, want one busy timeout of about 5 s", took)
		}
	})

	t.Run("no busy timeout", func(t *testing.T) {
		// _busy_timeout=0 in DB_URL wins over Warp's, the one setting of
		// Warp's a DB_URL may change: each try fails at once, as a fresh file being switched to
		// WAL by another process does, and the retries fill about a second.
		file := dbFile(t)
		lock(t, file)
		url := "file:" + file + "?_busy_timeout=0"
		err, took := startLocked(t, url)
		if err == nil || err.Error() != locked(url) {
			t.Fatalf("Start = %v\nwant %s", err, locked(url))
		}
		// Loose above, for a loaded machine, yet under the 5 s a busy timeout
		// of Warp's would take.
		if took < 900*time.Millisecond || took > 4500*time.Millisecond {
			t.Fatalf("Start took %v, want about the one-second retry deadline", took)
		}
	})

	t.Run("released while retrying", func(t *testing.T) {
		file := dbFile(t)
		release := lock(t, file)
		time.AfterFunc(200*time.Millisecond, release)
		if err, _ := startLocked(t, "file:"+file+"?_busy_timeout=0"); err != nil {
			t.Fatalf("Start = %v, want it to wait for the lock", err)
		}
	})
}
