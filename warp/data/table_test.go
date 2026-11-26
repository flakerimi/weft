package data

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
)

// module is a module named name whose Register runs register.
type module struct {
	name     string
	register func(app *warp.App)
}

func (m module) Name() string { return m.name }

func (m module) Register(app *warp.App) error {
	m.register(app)
	return nil
}

// stores registers data.Module and a module that makes stores with register,
// and returns the app's mistakes as lines.
func stores(t *testing.T, register func(db *DB)) []string {
	t.Helper()
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(Module(), module{"shop", func(app *warp.App) { register(For(app)) }})
	if err := app.Err(); err != nil {
		return strings.Split(err.Error(), "\n")
	}
	return nil
}

type Dims struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type gadget struct {
	Note   string `json:"note"`
	hidden int
}

type Product struct {
	Record
	Name    string     `json:"name" warp:"required,unique"`
	Price   float64    `json:"price" warp:"min=0"`
	Stock   int        `json:"stock"`
	Big     uint32     `json:"big"`
	Live    bool       `json:"live"`
	Ends    *time.Time `json:"ends"`
	Photo   []byte     `json:"photo"`
	Code    *string    `json:"code" warp:"unique"`
	Secret  string     `json:"-"`
	private string
	Dims
	gadget
}

func TestColumnsOf(t *testing.T) {
	tb, mistakes := tableOf(reflect.TypeFor[Product](), "products")
	if len(mistakes) > 0 {
		t.Fatal(mistakes)
	}
	type col struct {
		Name, Field string
		Kind        kind
		Null        bool
		Unique      bool
		Auto        bool
	}
	var got []col
	for _, c := range tb.cols {
		got = append(got, col{c.name, c.field, c.kind, c.null, c.unique, c.auto})
	}
	want := []col{
		{"id", "Product.ID", kindInteger, false, false, true},
		{"created_at", "Product.CreatedAt", kindTime, false, false, true},
		{"updated_at", "Product.UpdatedAt", kindTime, false, false, true},
		{"name", "Product.Name", kindText, false, true, false},
		{"price", "Product.Price", kindReal, false, false, false},
		{"stock", "Product.Stock", kindInteger, false, false, false},
		{"big", "Product.Big", kindInteger, false, false, false},
		{"live", "Product.Live", kindBool, false, false, false},
		{"ends", "Product.Ends", kindTime, true, false, false},
		{"photo", "Product.Photo", kindBlob, false, false, false},
		{"code", "Product.Code", kindText, true, true, false},
		{"width", "Product.Width", kindReal, false, false, false},
		{"height", "Product.Height", kindReal, false, false, false},
		{"note", "Product.Note", kindText, false, false, false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("columns\n%+v\nwant\n%+v", got, want)
	}
	if tb.singular != "product" || tb.model != "Product" {
		t.Fatalf("singular %q, model %q", tb.singular, tb.model)
	}
	// The record's own fields are found wherever Record is embedded.
	type Late struct {
		Name string `json:"name"`
		Record
	}
	late, mistakes := tableOf(reflect.TypeFor[Late](), "lates")
	if len(mistakes) > 0 || late.cols[1].name != "id" || late.col("id").index[0] != 1 {
		t.Fatalf("Record not first: %+v %v", late.cols, mistakes)
	}
}

func TestSingular(t *testing.T) {
	for name, want := range map[string]string{"products": "product", "staff": "staff", "s": "s", "order_items": "order_item"} {
		if got := singular(name); got != want {
			t.Errorf("singular(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestStoreNeedsRecord(t *testing.T) {
	type Plain struct {
		Name string `json:"name"`
	}
	type Pointer struct {
		*Record
		Name string `json:"name"`
	}
	got := stores(t, func(db *DB) {
		NewStore[Plain](db, "products")
		NewStore[Pointer](db, "pointers")
		NewStore[int](db, "numbers")
	})
	want := []string{
		`warp: products: Plain does not embed data.Record; embed it, since a store needs an id and two times.`,
		`warp: pointers: Pointer does not embed data.Record; embed it, since a store needs an id and two times.`,
		`warp: numbers: int does not embed data.Record; embed it, since a store needs an id and two times.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestUnsupportedField(t *testing.T) {
	type Product struct {
		Record
		Tags  []string          `json:"tags"`
		Attrs map[string]string `json:"attrs"`
		Size  Dims              `json:"size"`
		Twice **string          `json:"twice"`
		Skip  []string          `json:"-"`
	}
	got := stores(t, func(db *DB) { NewStore[Product](db, "products") })
	want := []string{
		`warp: products: Product.Tags is a []string, which a store cannot keep; use text, a number, a bool, a time or bytes, or tag it json:"-".`,
		`warp: products: Product.Attrs is a map[string]string, which a store cannot keep; use text, a number, a bool, a time or bytes, or tag it json:"-".`,
		`warp: products: Product.Size is a data.Dims, which a store cannot keep; use text, a number, a bool, a time or bytes, or tag it json:"-".`,
		`warp: products: Product.Twice is a **string, which a store cannot keep; use text, a number, a bool, a time or bytes, or tag it json:"-".`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestStoreMistakes(t *testing.T) {
	type Untagged struct {
		Record
		Name string
		Note string `json:",omitempty"`
	}
	type Clash struct {
		Record
		Key string `json:"id"`
	}
	type Extra struct {
		B string `json:"a"`
	}
	type Twice struct {
		Record
		A string `json:"a"`
		Extra
	}
	type Pointed struct {
		Record
		*Dims
	}
	type Fine struct {
		Record
	}
	got := stores(t, func(db *DB) {
		NewStore[Untagged](db, "untagged")
		NewStore[Clash](db, "clashes")
		NewStore[Twice](db, "twice")
		NewStore[Pointed](db, "pointed")
		NewStore[Fine](db, "Order Items")
		NewStore[Fine](db, "9lives")
		NewStore[Fine](db, "")
		NewStore[Fine](db, "fines")
		NewStore[Fine](db, "fines")
	})
	want := []string{
		`warp: untagged: Untagged.Name has no json tag; add one, since the column is named after it.`,
		`warp: untagged: Untagged.Note has a json tag with no name; name it, since the column is named after it.`,
		`warp: clashes: Clash.Key has the json name "id", which data.Record keeps; rename it.`,
		`warp: twice: Twice.A and Twice.B both have the json name "a"; rename one.`,
		`warp: pointed: Pointed embeds *Dims, a pointer; embed Dims by value.`,
		`warp: a store's table is named "Order Items"; use lower-case letters, digits and _, such as order_items.`,
		`warp: a store's table is named "9lives"; use lower-case letters, digits and _, such as t_9lives.`,
		`warp: a store's table is named ""; use lower-case letters, digits and _, such as items.`,
		`warp: two stores keep the table fines; make one and share it.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// Index names join the table and the column with _, which either may hold,
// so two stores can ask for one name; the second is a mistake, never an
// index silently left unmade.
func TestIndexNameClash(t *testing.T) {
	type A_B struct {
		Record
		C string `json:"c" warp:"unique"`
	}
	type A struct {
		Record
		BC string `json:"b_c" warp:"unique"`
	}
	type Plain struct {
		Record
		BC string `json:"b_c"`
	}
	got := stores(t, func(db *DB) {
		NewStore[A_B](db, "a_b")
		NewStore[A](db, "a")
		NewStore[Plain](db, "a_x") // not unique: no index, no clash
	})
	want := []string{
		`warp: a: the unique index for A.BC would be named a_b_c_unique, which the a_b store's A_B.C already uses; rename one field or table.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// Tables and indexes share one set of names in SQLite, so a table named
	// like another store's unique index is a mistake too, either way round.
	got = stores(t, func(db *DB) {
		NewStore[A_B](db, "a_b")
		NewStore[Plain](db, "a_b_c_unique")
	})
	want = []string{`warp: a_b_c_unique: the table would be named like the unique index for A_B.C of the a_b store; rename the table.`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	got = stores(t, func(db *DB) {
		NewStore[Plain](db, "a_b_c_unique")
		NewStore[A_B](db, "a_b")
	})
	want = []string{`warp: a_b: the unique index for A_B.C would be named a_b_c_unique, which is the a_b_c_unique store's table; rename one field or table.`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// codedError is an error with a result code, as modernc.org/sqlite's are.
type codedError struct {
	code int
	text string
}

func (e codedError) Error() string { return e.text }
func (e codedError) Code() int     { return e.code }

// busy reads the driver's code when it has one: SQLITE_BUSY in any of its
// extended forms, and not SQLITE_LOCKED; without a code, only the whole
// message "database is locked" of the innermost error.
func TestBusy(t *testing.T) {
	var d sqlite
	for _, c := range []struct {
		err  error
		want bool
	}{
		{codedError{5, "database is locked (5) (SQLITE_BUSY)"}, true},
		{codedError{261, "database is locked (261) (SQLITE_BUSY_RECOVERY)"}, true},
		{codedError{517, "database is locked (517) (SQLITE_BUSY_SNAPSHOT)"}, true},
		{codedError{773, "database is locked (773) (SQLITE_BUSY_TIMEOUT)"}, true},
		{fmt.Errorf("data: cannot commit: %w", codedError{5, "database is locked"}), true},
		{codedError{6, "database table is locked (6) (SQLITE_LOCKED)"}, false},
		{codedError{262, "database table is locked (262) (SQLITE_LOCKED_SHAREDCACHE)"}, false},
		{codedError{1, "SQL logic error: database is locked"}, false},
		{errString("database is locked"), true},                          // a driver with no codes
		{fmt.Errorf("query: %w", errString("database is locked")), true}, // the innermost error's whole message
		{errString(`sql: Scan error on column index 0, name "name": converting driver.Value type string ("database is locked") to a int: invalid syntax`), false},
		{errString("database table is locked"), false},
		{errString("unable to open database file (14)"), false},
		{errString("constraint failed: UNIQUE constraint failed: products.name (2067)"), false},
	} {
		if got := d.busy(c.err); got != c.want {
			t.Errorf("busy(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

// Without data.Module, NewStore is given nil, and the app reports the module
// it needs; the store does not panic.
func TestStoreWithoutModule(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	var s *Store[Product]
	app.Use(module{"shop", func(app *warp.App) { s = NewStore[Product](For(app), "products") }})
	want := `warp: module "shop" needs the "data" module; list data.Module() before it in app/modules.go.`
	if err := app.Err(); err == nil || err.Error() != want {
		t.Fatalf("Err = %v, want %q", err, want)
	}
	if _, err := s.Get(context.Background(), 1); err == nil {
		t.Fatal("Get on a store without a database worked")
	}
}

// Registering reads nothing: an empty or wrong environment is reported at
// start, not as a registration mistake.
func TestRegisterDoesNoWork(t *testing.T) {
	app := warp.New(warp.ConfigFrom(map[string]string{"DB_DRIVER": "oracle"}))
	app.Use(Module(), module{"shop", func(app *warp.App) { NewStore[Product](For(app), "products") }})
	if err := app.Err(); err != nil {
		t.Fatalf("Err = %v, want nil", err)
	}
	if For(app).SQL() != nil {
		t.Fatal("SQL() is not nil before start")
	}
}

// The driver is the app's import, so here there is none.
func TestStartWithoutDriver(t *testing.T) {
	cases := map[string]string{
		"sqlite": `no database driver is named "sqlite"; import one in main.go, such as _ "modernc.org/sqlite".`,
		"":       `DB_DRIVER is not set; add DB_DRIVER=sqlite and DB_URL=file:data/app.db to .env.`,
		"oracle": `DB_DRIVER is "oracle", and this version of Warp speaks sqlite alone; set DB_DRIVER=sqlite in .env.`,
	}
	for driver, want := range cases {
		app := warp.New(warp.ConfigFrom(map[string]string{"DB_DRIVER": driver, "DB_URL": "file:" + t.TempDir() + "/app.db"}))
		app.Use(Module())
		err := app.Start(context.Background())
		// The core puts warp: and the module in front.
		if want = `warp: module "data" could not migrate: ` + want; err == nil || err.Error() != want {
			t.Errorf("DB_DRIVER=%q: Start = %v, want %q", driver, err, want)
		}
	}
}

func TestCreateTableSQL(t *testing.T) {
	type Gadget struct {
		Record
		Name  string   `json:"name" warp:"unique"`
		Price float64  `json:"price"`
		Note  *string  `json:"note"`
		Live  bool     `json:"live"`
		Data  []byte   `json:"data"`
		Odd   string   `json:"odd\"name"`
		Count *int64   `json:"count"`
		Skip  []string `json:"-"`
	}
	tb, mistakes := tableOf(reflect.TypeFor[Gadget](), "gadgets")
	if len(mistakes) > 0 {
		t.Fatal(mistakes)
	}
	var d sqlite
	zero := `'0001-01-01T00:00:00.000000000Z'`
	checks := []struct{ got, want string }{
		{tb.createSQL(d), `CREATE TABLE "gadgets" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, ` +
			`"created_at" TEXT NOT NULL DEFAULT ` + zero + `, "updated_at" TEXT NOT NULL DEFAULT ` + zero + `, ` +
			`"name" TEXT NOT NULL DEFAULT '', "price" REAL NOT NULL DEFAULT 0, "note" TEXT, ` +
			`"live" INTEGER NOT NULL DEFAULT 0, "data" BLOB NOT NULL DEFAULT X'', "odd""name" TEXT NOT NULL DEFAULT '', "count" INTEGER)`},
		{tb.addColumnSQL(d, tb.col("price")), `ALTER TABLE "gadgets" ADD COLUMN "price" REAL NOT NULL DEFAULT 0`},
		{tb.uniqueIndexSQL(d, tb.col("name")), `CREATE UNIQUE INDEX IF NOT EXISTS "gadgets_name_unique" ON "gadgets" ("name")`},
		{tb.selectSQL(d), `SELECT "id", "created_at", "updated_at", "name", "price", "note", "live", "data", "odd""name", "count" FROM "gadgets" WHERE "id" = ?`},
		{tb.insertSQL(d), `INSERT INTO "gadgets" ("created_at", "updated_at", "name", "price", "note", "live", "data", "odd""name", "count") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING "id"`},
		{tb.updateSQL(d, []*column{tb.col("price"), tb.col("name")}), `UPDATE "gadgets" SET "price" = ?, "name" = ?, "updated_at" = ? WHERE "id" = ? ` +
			`RETURNING "id", "created_at", "updated_at", "name", "price", "note", "live", "data", "odd""name", "count"`},
		{tb.deleteSQL(d), `DELETE FROM "gadgets" WHERE "id" = ?`},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("got\n%s\nwant\n%s", c.got, c.want)
		}
	}
}

func TestTimeText(t *testing.T) {
	var d sqlite
	at := time.Date(2026, 11, 25, 10, 30, 0, 5, time.FixedZone("CET", 3600))
	text := d.timeValue(at)
	if text != "2026-11-25T09:30:00.000000005Z" {
		t.Fatalf("timeValue = %v", text)
	}
	back, err := d.scanTime(text)
	if err != nil || !back.Equal(at) || back.Location() != time.UTC {
		t.Fatalf("scanTime = %v, %v", back, err)
	}
	// Fixed width, so the text sorts as the times do.
	if a, b := d.timeValue(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).(string), d.timeValue(time.Date(2026, 1, 1, 0, 0, 0, 500, time.UTC)).(string); a >= b {
		t.Fatalf("%s sorts after %s", a, b)
	}
	if _, err := d.scanTime("yesterday"); err == nil {
		t.Fatal("scanTime took yesterday")
	}
}

func TestUniqueColumn(t *testing.T) {
	var d sqlite
	cases := []struct {
		err, table, col string
		ok              bool
	}{
		{"constraint failed: UNIQUE constraint failed: products.name (2067)", "products", "name", true},
		{"UNIQUE constraint failed: products.odd.name", "products", "odd.name", true},
		{"constraint failed: UNIQUE constraint failed: other.name (2067)", "products", "", false},
		{"constraint failed: NOT NULL constraint failed: products.name (1299)", "products", "", false},
		{"constraint failed: UNIQUE constraint failed: products.a, products.b (2067)", "products", "", false},
	}
	for _, c := range cases {
		col, ok := d.uniqueColumn(errString(c.err), c.table)
		if col != c.col || ok != c.ok {
			t.Errorf("uniqueColumn(%q) = %q, %v", c.err, col, ok)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// A URI's path is the file's, but for a drive on Windows, where SQLite reads
// /C:/srv/app.db as C:/srv/app.db.
func TestURIFile(t *testing.T) {
	for _, c := range []struct {
		path    string
		windows bool
		want    string
	}{
		{"/C:/srv/app.db", true, "C:/srv/app.db"},
		{"/c:/app.db", true, "c:/app.db"},
		{"/C:", true, "C:"},
		{"/srv/app.db", true, "/srv/app.db"},
		{"/C:x/app.db", true, "/C:x/app.db"},
		{"/1:/app.db", true, "/1:/app.db"},
		{"data/app.db", true, "data/app.db"},
		{"/C:/srv/app.db", false, "/C:/srv/app.db"},
	} {
		if got := uriFile(c.path, c.windows); got != c.want {
			t.Errorf("uriFile(%q, %v) = %q, want %q", c.path, c.windows, got, c.want)
		}
	}
}

func TestDSN(t *testing.T) {
	var d sqlite
	settings := "_busy_timeout=5000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate"
	drive := "/C:/srv"
	if runtime.GOOS == "windows" {
		drive = "C:/srv"
	}
	cases := []struct {
		url, dsn, dir string
		memory        bool
		notes         int
	}{
		{"file:data/app.db", "file:data/app.db?" + settings, "data", false, 0},
		{"file:data/app.db?cache=shared", "file:data/app.db?cache=shared&" + settings, "data", false, 0},
		{"file:///srv/app.db", "file:///srv/app.db?" + settings, "/srv", false, 0},
		// SQLite reads /C: in a URI's path as the drive on Windows alone.
		{"file:///C:/srv/app.db", "file:///C:/srv/app.db?" + settings, drive, false, 0},
		{"app.db", "app.db?" + settings, ".", false, 0},
		{":memory:", ":memory:?" + settings, "", true, 0},
		{"file::memory:?cache=shared", "file::memory:?cache=shared&" + settings, "", true, 0},
		{"file:x.db?mode=memory", "file:x.db?mode=memory&" + settings, "", true, 0},
		// Warp's own settings replace the URL's for the keys the store relies
		// on, aliases and escaped keys included, each named in a note.
		{"file:app.db?_txlock=deferred&cache=shared", "file:app.db?cache=shared&" + settings, ".", false, 1},
		{"file:app.db?_journal=DELETE&_journal_mode=OFF&_fk=0&_foreign_keys=no&%5Ftxlock=exclusive", "file:app.db?" + settings, ".", false, 5},
		{"file:app.db?_txlock=immediate", "file:app.db?" + settings, ".", false, 0},
		// A busy timeout may be raised, never lowered.
		{"file:app.db?_busy_timeout=0", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_timeout=100&_busy_timeout=4999", "file:app.db?" + settings, ".", false, 2},
		{"file:app.db?_busy_timeout=9000", "file:app.db?_busy_timeout=9000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate", ".", false, 0},
		{"file:app.db?_timeout=8000&_busy_timeout=6000", "file:app.db?_busy_timeout=8000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate", ".", false, 0},
		// Not a busy timeout SQLite keeps: as if not given, with a note.
		{"file:app.db?_busy_timeout=soon", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_busy_timeout=99999999999", "file:app.db?" + settings, ".", false, 1},
		// Every spelling of the _pragma, which the driver runs after the keys.
		{"file:app.db?_pragma=busy_timeout(0)", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_pragma=busy_timeout%3D0", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_pragma=busy_timeout=0", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_pragma=BUSY_TIMEOUT(0)", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_pragma=%20busy_timeout(1)", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_pragma=+main.busy_timeout+=+'2'", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?%5Fpragma=busy_timeout(0)", "file:app.db?" + settings, ".", false, 1},
		{"file:app.db?_busy_timeout=9000&_pragma=busy_timeout(0)", "file:app.db?_busy_timeout=9000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate", ".", false, 1},
		{"file:app.db?_pragma=busy_timeout(9000)&_pragma=busy_timeout(0)", "file:app.db?_busy_timeout=9000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate", ".", false, 1},
		{"file:app.db?_pragma=busy_timeout(0)&_pragma=busy_timeout(9000)", "file:app.db?_busy_timeout=9000&_foreign_keys=1&_journal_mode=WAL&_txlock=immediate", ".", false, 1},
		{"file:app.db?_pragma=busy_timeout", "file:app.db?_pragma=busy_timeout&" + settings, ".", false, 0}, // a question, which sets nothing
		{"file:app.db?_pragma=cache_size(-2000)", "file:app.db?_pragma=cache_size(-2000)&" + settings, ".", false, 0},
	}
	for _, c := range cases {
		dsn, file, memory, notes := d.dsn(c.url)
		dir := ""
		if file != "" {
			dir = filepath.Dir(file)
		}
		// The folder is the system's: \srv on Windows for /srv.
		if dsn != c.dsn || dir != filepath.FromSlash(c.dir) || memory != c.memory || len(notes) != c.notes {
			t.Errorf("dsn(%q) = %q, %q, %v, %q; want %q, %q, %v and %d notes", c.url, dsn, dir, memory, notes, c.dsn, filepath.FromSlash(c.dir), c.memory, c.notes)
		}
	}
	_, _, _, notes := d.dsn("file:app.db?_busy_timeout=0&_txlock=deferred&_timeout=99999999999&_pragma=busy_timeout(1)")
	if want := []string{
		"DB_URL sets _busy_timeout=0, below Warp's 5000; 5000 is used, since the app's own writers wait on it.",
		"DB_URL sets _txlock=deferred, which Warp sets itself; Warp's immediate is used, since the store relies on it.",
		"DB_URL sets _timeout=99999999999, which is not a busy timeout SQLite keeps; Warp's 5000 is used.",
		"DB_URL sets _pragma=busy_timeout(1), below Warp's 5000; 5000 is used, since the app's own writers wait on it.",
	}; !reflect.DeepEqual(notes, want) {
		t.Errorf("notes %q, want %q", notes, want)
	}
}
