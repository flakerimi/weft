package dbtest

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

var ctx = context.Background()

func TestCreateGetUpdateDelete(t *testing.T) {
	_, s := shop[Product](t, dbFile(t))
	p := Product{Name: "Lamp", Price: 12.5, Status: "draft"}
	p.ID, p.CreatedAt = 99, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // ignored
	if err := s.Create(ctx, &p); err != nil {
		t.Fatal(err)
	}
	if p.ID != 1 || p.CreatedAt.IsZero() || !p.UpdatedAt.Equal(p.CreatedAt) || p.CreatedAt.Year() == 2000 {
		t.Fatalf("Create set %+v", p.Record)
	}
	q := Product{Name: "Desk", Price: 80, Status: "live"}
	if err := s.Create(ctx, &q); err != nil || q.ID != 2 {
		t.Fatalf("second Create: id %d, %v", q.ID, err)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatalf("Get = %+v, %v; want %+v", got, err, p)
	}

	change := Product{Name: "Lamp 2", Price: 15}
	updated, err := s.Update(ctx, p.ID, &change, []string{"name", "price"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Lamp 2" || updated.Price != 15 || updated.Status != "draft" || updated.ID != p.ID || !updated.CreatedAt.Equal(p.CreatedAt) {
		t.Fatalf("Update = %+v", updated)
	}
	if again, _ := s.Get(ctx, p.ID); !reflect.DeepEqual(again, updated) {
		t.Fatalf("Get after Update = %+v, want %+v", again, updated)
	}

	if err := s.Delete(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, p.ID); !isNotFound(err) {
		t.Fatalf("Get after Delete = %v", err)
	}
	if err := s.Delete(ctx, p.ID); !isNotFound(err) {
		t.Fatalf("second Delete = %v, want not found", err)
	}
	if _, err := s.Update(ctx, p.ID, &change, []string{"name"}); !isNotFound(err) {
		t.Fatalf("Update of a deleted record = %v, want not found", err)
	}
	if left, err := s.Get(ctx, q.ID); err != nil || left.Name != "Desk" {
		t.Fatalf("the other record = %+v, %v", left, err)
	}
}

func isNotFound(err error) bool {
	var e *warp.Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

func TestUpdateWritesOnlyNamedFields(t *testing.T) {
	_, s := shop[Product](t, dbFile(t))
	p := Product{Name: "Lamp", Price: 12.5, Status: "live"}
	if err := s.Create(ctx, &p); err != nil {
		t.Fatal(err)
	}
	// Every field differs; only the price is named.
	change := Product{Name: "", Price: 5, Status: "draft"}
	got, err := s.Update(ctx, p.ID, &change, []string{"price"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Lamp" || got.Status != "live" || got.Price != 5 {
		t.Fatalf("Update = %+v, want only the price changed", got)
	}
	if stored, _ := s.Get(ctx, p.ID); stored.Name != "Lamp" || stored.Status != "live" || stored.Price != 5 {
		t.Fatalf("stored %+v", stored)
	}

	// No fields: nothing is written, not even updated_at.
	same, err := s.Update(ctx, p.ID, &change, nil)
	if err != nil || !reflect.DeepEqual(same, got) {
		t.Fatalf("Update with no fields = %+v, %v; want %+v", same, err, got)
	}
	if _, err := s.Update(ctx, 404, &change, nil); !isNotFound(err) {
		t.Fatalf("Update with no fields of a missing record = %v", err)
	}

	// A name the model does not have, or one the store sets, is the caller's mistake.
	for field, want := range map[string]string{
		"cost":       `data: products has no field "cost" to update`,
		"id":         `data: products has no field "id" to update, since the store sets it`,
		"created_at": `data: products has no field "created_at" to update, since the store sets it`,
		"updated_at": `data: products has no field "updated_at" to update, since the store sets it`,
	} {
		if _, err := s.Update(ctx, p.ID, &change, []string{"price", field}); err == nil || err.Error() != want {
			t.Errorf("Update %s = %v, want %q", field, err, want)
		}
	}
	if stored, _ := s.Get(ctx, p.ID); !reflect.DeepEqual(stored, got) {
		t.Fatalf("a refused Update wrote: %+v", stored)
	}
}

func TestGetMissingIsNotFound(t *testing.T) {
	_, s := shop[Product](t, dbFile(t))
	_, err := s.Get(ctx, 7)
	var e *warp.Error
	if !errors.As(err, &e) || e.Status != 404 || e.Code != "not_found" || e.Message != "No product has id 7." {
		t.Fatalf("Get = %#v", err)
	}

	type Staff struct {
		data.Record
		Name string `json:"name"`
	}
	var staff *data.Store[Staff]
	warptest.New(t, env(dbFile(t)), data.Module(), stores{"people", func(db *data.DB) { staff = data.NewStore[Staff](db, "staff") }})
	if _, err := staff.Get(ctx, 7); err == nil || err.Error() != "No staff has id 7." {
		t.Fatalf("Get = %v", err)
	}
}

func TestTimesAreSetAndRoundTrip(t *testing.T) {
	type Event struct {
		data.Record
		Name  string     `json:"name"`
		At    time.Time  `json:"at"`
		Ends  *time.Time `json:"ends"`
		Notes *string    `json:"notes"`
		Blob  []byte     `json:"blob"`
		Live  bool       `json:"live"`
		Small int8       `json:"small"`
		Big   uint64     `json:"big"`
	}
	var s *data.Store[Event]
	var db *data.DB
	warptest.New(t, env(dbFile(t)), data.Module(), stores{"events", func(d *data.DB) { db, s = d, data.NewStore[Event](d, "events") }})
	cet := time.FixedZone("CET", 3600)
	c := &clock{time.Date(2026, 11, 25, 10, 0, 0, 123456789, cet)}
	db.SetClock(c.Now)

	e := Event{Name: "launch", At: time.Date(2026, 12, 1, 9, 0, 0, 1, cet), Live: true, Small: -5, Big: 1 << 40}
	if err := s.Create(ctx, &e); err != nil {
		t.Fatal(err)
	}
	wantCreated := time.Date(2026, 11, 25, 9, 0, 0, 123456789, time.UTC)
	if e.CreatedAt != wantCreated || e.UpdatedAt != wantCreated {
		t.Fatalf("times %v %v, want %v in UTC", e.CreatedAt, e.UpdatedAt, wantCreated)
	}
	got, err := s.Get(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	e.At = e.At.UTC() // read back as UTC
	e.Blob = []byte{} // a NOT NULL blob is never NULL
	if !reflect.DeepEqual(got, e) {
		t.Fatalf("Get =\n%+v\nwant\n%+v", got, e)
	}
	if got.Ends != nil || got.Notes != nil {
		t.Fatalf("nil pointers came back as %v, %v", got.Ends, got.Notes)
	}

	// Pointers and bytes with values; updated_at moves and created_at stays.
	c.now = c.now.Add(time.Hour)
	ends, notes := time.Date(2027, 1, 1, 0, 0, 0, 0, cet), "bring cake"
	change := Event{Ends: &ends, Notes: &notes, Blob: []byte{0, 1, 2}}
	got, err = s.Update(ctx, e.ID, &change, []string{"ends", "notes", "blob"})
	if err != nil {
		t.Fatal(err)
	}
	if got.CreatedAt != wantCreated || got.UpdatedAt != wantCreated.Add(time.Hour) {
		t.Fatalf("after Update: created %v, updated %v", got.CreatedAt, got.UpdatedAt)
	}
	if got.Ends == nil || *got.Ends != ends.UTC() || got.Notes == nil || *got.Notes != notes || string(got.Blob) != "\x00\x01\x02" {
		t.Fatalf("after Update: %+v", got)
	}

	// The column holds RFC 3339 text in UTC with nanoseconds.
	var text string
	if err := db.SQL().QueryRow(`SELECT "created_at" FROM "events" WHERE "id" = ?`, e.ID).Scan(&text); err != nil {
		t.Fatal(err)
	}
	if text != "2026-11-25T09:00:00.123456789Z" {
		t.Fatalf("created_at is stored as %q", text)
	}
}

func TestUniqueViolationIsInvalid(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	a, b := Product{Name: "Lamp"}, Product{Name: "Desk"}
	if err := s.Create(ctx, &a); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, &b); err != nil {
		t.Fatal(err)
	}
	dup := Product{Name: "Lamp"}
	err := s.Create(ctx, &dup)
	var e *warp.Error
	if !errors.As(err, &e) || e.Status != 422 || !reflect.DeepEqual(e.Fields, map[string]string{"name": "This is already taken."}) {
		t.Fatalf("Create of a taken name = %#v", err)
	}
	if dup.ID != 0 {
		t.Fatalf("a refused Create set the id %d", dup.ID)
	}
	if _, err := s.Update(ctx, b.ID, &dup, []string{"name"}); !errors.As(err, &e) || e.Fields["name"] != "This is already taken." {
		t.Fatalf("Update to a taken name = %#v", err)
	}
	// Writing a record's own name again is not a violation.
	if _, err := s.Update(ctx, a.ID, &dup, []string{"name"}); err != nil {
		t.Fatal(err)
	}
	// The index is named for the table and the column.
	var name string
	if err := data.For(c.App).SQL().QueryRow(`SELECT "name" FROM "sqlite_master" WHERE "type" = 'index' AND "tbl_name" = 'products'`).Scan(&name); err != nil || name != "products_name_unique" {
		t.Fatalf("index %q, %v", name, err)
	}

	// Another constraint's error is returned as it is, to be a 500.
	type Child struct {
		data.Record
		Parent int64 `json:"parent"`
	}
	var children *data.Store[Child]
	app := warptest.New(t, env(dbFile(t)), data.Module(), stores{"children", func(db *data.DB) { children = data.NewStore[Child](db, "children") }}).App
	db := data.For(app).SQL()
	if _, err := db.Exec(`CREATE TRIGGER "no_odd" BEFORE INSERT ON "children" WHEN NEW."parent" % 2 = 1 BEGIN SELECT RAISE(ABORT, 'odd parents are refused'); END`); err != nil {
		t.Fatal(err)
	}
	err = children.Create(ctx, &Child{Parent: 1})
	if err == nil || errors.As(err, &e) || !strings.Contains(err.Error(), "odd parents are refused") {
		t.Fatalf("Create = %#v, want the driver's error", err)
	}
}

func TestStartMistakes(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "missing.db")
	cases := []struct {
		vars map[string]string
		want string
	}{
		{map[string]string{"DB_URL": "file:x.db"}, `DB_DRIVER is not set; add DB_DRIVER=sqlite and DB_URL=file:data/app.db to .env.`},
		{map[string]string{"DB_DRIVER": "oracle", "DB_URL": "file:x.db"}, `DB_DRIVER is "oracle", and this version of Warp speaks sqlite alone; set DB_DRIVER=sqlite in .env.`},
		{map[string]string{"DB_DRIVER": "sqlite", "DB_URL": "file:"}, `DB_URL is file: with no file name, which SQLite opens as a temporary database lost when the app stops; name a file, as file:data/app.db, or write :memory: for a database in memory.`},
		{map[string]string{"DB_DRIVER": "sqlite"}, `DB_URL is not set; add DB_URL=file:data/app.db to .env.`},
		{map[string]string{"DB_DRIVER": "sqlite", "DB_URL": "file:" + blocked + "?mode=ro"}, `the database at file:` + blocked + `?mode=ro cannot be opened (`},
	}
	for _, c := range cases {
		app := warp.New(warp.ConfigFrom(c.vars))
		app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
		if err := app.Err(); err != nil {
			t.Fatalf("Err = %v: the environment is checked at start, not while registering", err)
		}
		err := app.Start(ctx)
		// The core alone puts warp: in front.
		prefix := `warp: module "data" could not migrate: `
		if err == nil || !strings.HasPrefix(err.Error(), prefix+c.want) {
			t.Errorf("Start = %v, want %q", err, prefix+c.want)
		}
		if strings.HasSuffix(c.want, "(") && !strings.HasSuffix(err.Error(), "); check DB_URL in .env.") {
			t.Errorf("Start = %v, want it to end with what to check", err)
		}
		if data.For(app).SQL() != nil {
			t.Errorf("SQL() is not nil after a failed start")
		}
	}
}

// readonly is about what a request may send: server code writes such a field
// through Update. Only the three times and the id the store sets are refused.
func TestUpdateWritesOwnReadonlyField(t *testing.T) {
	type Page struct {
		data.Record
		Title string `json:"title"`
		Slug  string `json:"slug" warp:"readonly"`
	}
	_, s := shop[Page](t, dbFile(t))
	p := Page{Title: "Hello", Slug: "hello"}
	if err := s.Create(ctx, &p); err != nil {
		t.Fatal(err)
	}
	got, err := s.Update(ctx, p.ID, &Page{Title: "ignored", Slug: "hello-world"}, []string{"slug"})
	if err != nil || got.Slug != "hello-world" || got.Title != "Hello" {
		t.Fatalf("Update slug = %+v, %v", got, err)
	}
	for _, field := range []string{"id", "created_at", "updated_at", "cost"} {
		if _, err := s.Update(ctx, p.ID, &Page{Slug: "other"}, []string{"slug", field}); err == nil {
			t.Errorf("Update of %s was taken", field)
		}
	}
	if stored, _ := s.Get(ctx, p.ID); stored.Slug != "hello-world" {
		t.Fatalf("a refused Update wrote: %+v", stored)
	}
}

func TestStoreBeforeStart(t *testing.T) {
	file := dbFile(t)
	app := warp.New(warp.ConfigFrom(env(file)))
	var s *data.Store[Product]
	app.Use(data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[Product](db, "products") }})
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	want := `warp: the products store was used before the app started; use it from a handler, or after Start.`
	_, getErr := s.Get(ctx, 1)
	_, updateErr := s.Update(ctx, 1, &Product{}, []string{"name"})
	for _, err := range []error{getErr, s.Create(ctx, &Product{}), updateErr, s.Delete(ctx, 1)} {
		if err == nil || err.Error() != want {
			t.Errorf("before start: %v, want %q", err, want)
		}
	}
	if err := data.For(app).Tx(ctx, func(context.Context) error { return nil }); err == nil ||
		err.Error() != `warp: the database was used before the app started; use it from a handler, or after Start.` {
		t.Errorf("Tx before start = %v", err)
	}
	if data.For(app).SQL() != nil {
		t.Error("SQL() is not nil before start")
	}
	// Registering made nothing on disk.
	if _, err := os.Stat(filepath.Dir(file)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("registering made %s: %v", filepath.Dir(file), err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer app.Stop(ctx)
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("Start did not make the file: %v", err)
	}
	if data.For(app).SQL() == nil {
		t.Fatal("SQL() is nil after start")
	}
}

func TestTwoConnectionsSeeEachOther(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	db := data.For(c.App).SQL()

	// Every pooled connection has the settings.
	c1, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	c2, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	for _, conn := range []*sql.Conn{c1, c2} {
		var mode string
		var fk, busy int
		if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
			t.Fatalf("journal_mode %q, %v", mode, err)
		}
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
			t.Fatalf("foreign_keys %d, %v", fk, err)
		}
		if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil || busy != 5000 {
			t.Fatalf("busy_timeout %d, %v", busy, err)
		}
	}
	c1.Close()
	c2.Close()

	// A write waits for another connection's transaction, and then sees it.
	inTx, release := make(chan struct{}), make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	var txErr error
	go func() {
		defer wg.Done()
		txErr = data.For(c.App).Tx(ctx, func(ctx context.Context) error {
			if err := s.Create(ctx, &Product{Name: "first"}); err != nil {
				return err
			}
			close(inTx)
			<-release
			return nil
		})
	}()
	<-inTx
	done := make(chan error, 1)
	go func() { done <- s.Create(ctx, &Product{Name: "second"}) }()
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	if txErr != nil {
		t.Fatal(txErr)
	}
	if err := <-done; err != nil {
		t.Fatalf("the second connection's write: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM "products"`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("%d rows, %v", n, err)
	}
}

// A database another connection holds is a 503 with a sentence to try again,
// not a 500, for a store call, a Tx and the developer's own SQL alike. The
// client did not cause it, so it is logged at warn level, with the route,
// and never at error level. Each call waits out Warp's 5 s busy timeout, so
// they wait at once.
func TestBusyIsUnavailable(t *testing.T) {
	file := dbFile(t)
	var s *data.Store[Product]
	c := warptest.New(t, env(file), data.Module(), resourceModule[Product]{&s, nil})
	db := data.For(c.App)
	if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
		t.Fatal(err)
	}
	other, err := sql.Open("sqlite", "file:"+file)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	release := lockWrites(t, other)
	defer release()

	errs, warns := captureLevels(t)
	calls := map[string]func() error{
		"Create": func() error { return s.Create(ctx, &Product{Name: "Desk"}) },
		"Update": func() error { _, err := s.Update(ctx, 1, &Product{Name: "Desk"}, []string{"name"}); return err },
		"Delete": func() error { return s.Delete(ctx, 1) },
		"Tx":     func() error { return db.Tx(ctx, func(context.Context) error { return nil }) },
		"Exec":   func() error { _, err := db.Exec(ctx, `UPDATE "products" SET "price" = 1`); return err },
	}
	var wg sync.WaitGroup
	got := map[string]error{}
	var mu sync.Mutex
	for name, call := range calls {
		wg.Go(func() {
			err := call()
			mu.Lock()
			got[name] = err
			mu.Unlock()
		})
	}
	var res *warptest.Response
	wg.Go(func() { res = c.Post("/api/products", map[string]any{"name": "Chair"}) })
	// Reading is not held up: the WAL journal lets readers past a writer.
	if _, err := s.Get(ctx, 1); err != nil {
		t.Errorf("Get while the database is held = %v", err)
	}
	wg.Wait()
	for name, err := range got {
		var e *warp.Error
		if !errors.As(err, &e) || e.Status != 503 || e.Code != "unavailable" || e.Message != "The database is busy; try again in a moment." {
			t.Errorf("%s while the database is held = %v, want the 503", name, err)
		}
	}
	if e := res.Want(503).Error(); e.Code != "unavailable" || e.Message != "The database is busy; try again in a moment." {
		t.Fatalf("POST while the database is held: %+v", e)
	}
	if errs.Len() != 0 {
		t.Fatalf("the 503 was logged as an error: %s", errs)
	}
	if !strings.Contains(warns.String(), "method=POST path=/api/products") || strings.Count(warns.String(), "path=/api/products") != 1 {
		t.Fatalf("the warn log names the route %d times, want once: %s", strings.Count(warns.String(), "path=/api/products"), warns)
	}
	release()
	c.Post("/api/products", map[string]any{"name": "Chair"}).Want(201)
}

// captureLevels collects what slog writes at error level, and at warn level
// and above, for the rest of the test.
func captureLevels(t *testing.T) (errs, warns *bytes.Buffer) {
	t.Helper()
	errs, warns = &bytes.Buffer{}, &bytes.Buffer{}
	old := slog.Default()
	slog.SetDefault(slog.New(levels{
		slog.NewTextHandler(errs, &slog.HandlerOptions{Level: slog.LevelError}),
		slog.NewTextHandler(warns, &slog.HandlerOptions{Level: slog.LevelWarn}),
	}))
	t.Cleanup(func() { slog.SetDefault(old) })
	return errs, warns
}

// levels sends a record to every handler that takes its level.
type levels []slog.Handler

func (l levels) Enabled(ctx context.Context, lv slog.Level) bool {
	for _, h := range l {
		if h.Enabled(ctx, lv) {
			return true
		}
	}
	return false
}

func (l levels) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range l {
		if h.Enabled(ctx, r.Level) {
			h.Handle(ctx, r)
		}
	}
	return nil
}

func (l levels) WithAttrs([]slog.Attr) slog.Handler { return l }
func (l levels) WithGroup(string) slog.Handler      { return l }

// captureErrors collects what slog writes at error level for the rest of the
// test.
func captureErrors(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

// Warp makes the database's folder and file readable by the app's user
// alone: they hold every record and the password hashes. A folder or file
// that was there is left as it is.
func TestDatabaseFileIsPrivate(t *testing.T) {
	file := dbFile(t)
	c, s := shop[Product](t, file)
	if err := s.Create(ctx, &Product{Name: "Lamp"}); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]os.FileMode{filepath.Dir(file): 0o700, file: 0o600, file + "-wal": 0o600} {
		fi, err := os.Stat(path)
		if err != nil || fi.Mode().Perm() != want {
			t.Errorf("%s: %v, %v; want %v", filepath.Base(path), fi.Mode().Perm(), err, want)
		}
	}
	c.App.Stop(ctx)
	other := filepath.Join(t.TempDir(), "mine.db")
	os.WriteFile(other, nil, 0o640)
	_, _ = shop[Product](t, other)
	if fi, _ := os.Stat(other); fi.Mode().Perm() != 0o640 {
		t.Errorf("an existing file became %v", fi.Mode().Perm())
	}
}

// A database the app cannot write to stops the start with a sentence, where
// it would otherwise start and answer every write with a 500.
func TestReadOnlyDatabaseStopsTheStart(t *testing.T) {
	file := dbFile(t)
	c, _ := shop[Product](t, file)
	c.App.Stop(ctx)
	for _, url := range []string{"file:" + file + "?mode=ro"} {
		app := warp.New(warp.ConfigFrom(map[string]string{"DB_DRIVER": "sqlite", "DB_URL": url}))
		app.Use(data.Module(), stores{"products", func(db *data.DB) { data.NewStore[Product](db, "products") }})
		err := app.Start(ctx)
		want := `warp: module "data" could not migrate: the database at ` + url + ` cannot be written (`
		if err == nil || !strings.HasPrefix(err.Error(), want) || !strings.HasSuffix(err.Error(), "); let the app write the file and its folder, and take mode=ro out of DB_URL.") {
			t.Errorf("%s: Start = %v", url, err)
		}
	}
}
