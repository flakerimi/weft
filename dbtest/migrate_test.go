package dbtest

import (
	"reflect"
	"strings"
	"testing"

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
