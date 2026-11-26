// Package dbtest holds the tests of warp/data, and later of warp/auth, that
// need a real database. It is a module of its own so that the driver is its
// dependency, never warp's.
package dbtest

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"

	_ "modernc.org/sqlite"
)

type Product struct {
	data.Record
	Name   string  `json:"name" warp:"required,unique,max=120"`
	Price  float64 `json:"price" warp:"min=0"`
	Status string  `json:"status" warp:"enum=draft|live"`
}

// stores is a module that makes its stores in Register, as an app's module
// does.
type stores struct {
	name     string
	register func(db *data.DB)
}

func (m stores) Name() string { return m.name }

func (m stores) Register(app *warp.App) error {
	m.register(data.For(app))
	return nil
}

// dbFile is a database file in a folder of the test's own.
func dbFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "data", "app.db")
}

func env(file string) map[string]string {
	return map[string]string{"DB_DRIVER": "sqlite", "DB_URL": "file:" + file}
}

// shop starts an app over the database file with a store of T on the table
// products, and returns the client and the store.
func shop[T any](t *testing.T, file string) (*warptest.Client, *data.Store[T]) {
	t.Helper()
	var s *data.Store[T]
	c := warptest.New(t, env(file), data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[T](db, "products") }})
	return c, s
}

// clock is a clock a test moves by hand.
type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }
