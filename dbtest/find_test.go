package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

func TestFirst(t *testing.T) {
	c, s := shop[Product](t, dbFile(t))
	for _, p := range []Product{
		{Name: "Lamp", Price: 12, Status: "live"},
		{Name: "Desk", Price: 80, Status: "draft"},
		{Name: "Chair", Price: 40, Status: "live"},
	} {
		if err := s.Create(ctx, &p); err != nil {
			t.Fatal(err)
		}
	}

	got, ok, err := s.First(ctx, "name", "Desk")
	if err != nil || !ok || got.ID != 2 || got.Name != "Desk" || got.Price != 80 || got.CreatedAt.IsZero() {
		t.Fatalf("First(name, Desk) = %+v, %v, %v", got, ok, err)
	}
	// Several match: the lowest id.
	if got, ok, err := s.First(ctx, "status", "live"); err != nil || !ok || got.ID != 1 {
		t.Fatalf("First(status, live) = %+v, %v, %v; want id 1", got, ok, err)
	}
	if got, ok, err := s.First(ctx, "price", 40); err != nil || !ok || got.ID != 3 {
		t.Fatalf("First(price, 40) = %+v, %v, %v; want id 3", got, ok, err)
	}
	// A value is a value, never SQL.
	if got, ok, err := s.First(ctx, "name", `Desk" OR 1=1 --`); err != nil || ok {
		t.Fatalf("First with SQL in the value = %+v, %v, %v; want none", got, ok, err)
	}
	if got, ok, err := s.First(ctx, "name", "Sofa"); err != nil || ok || got.ID != 0 {
		t.Fatalf("First(name, Sofa) = %+v, %v, %v; want none", got, ok, err)
	}
	// A time is compared as the store writes it.
	if got, ok, err := s.First(ctx, "created_at", got.CreatedAt); err != nil || !ok || got.ID != 2 {
		t.Fatalf("First(created_at) = %+v, %v, %v; want id 2", got, ok, err)
	}
	if _, _, err := s.First(ctx, "cost", 1); err == nil || err.Error() != `data: products has no field "cost" to look up` {
		t.Fatalf("First(cost) = %v", err)
	}

	// Inside a transaction it sees the transaction's own rows.
	failed := errors.New("rolled back")
	err = data.For(c.App).Tx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, &Product{Name: "Shelf"}); err != nil {
			return err
		}
		if got, ok, err := s.First(ctx, "name", "Shelf"); err != nil || !ok || got.ID != 4 {
			t.Errorf("First inside the transaction = %+v, %v, %v; want id 4", got, ok, err)
		}
		return failed
	})
	if !errors.Is(err, failed) {
		t.Fatal(err)
	}
	if _, ok, err := s.First(ctx, "name", "Shelf"); err != nil || ok {
		t.Fatalf("First after the rollback = %v, %v; want none", ok, err)
	}
}

func TestFirstBeforeStart(t *testing.T) {
	app := warp.New(warp.ConfigFrom(env(dbFile(t))))
	var s *data.Store[Product]
	app.Use(data.Module(), stores{"products", func(db *data.DB) { s = data.NewStore[Product](db, "products") }})
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	_, want := s.Get(ctx, 1)
	_, ok, err := s.First(ctx, "name", "Lamp")
	if want == nil || err == nil || err.Error() != want.Error() || ok {
		t.Fatalf("First before start = %v, %v; want the error Get gives, %v", ok, err, want)
	}
}

// Status is a named text type, as a model's field may have.
type Status string

// First converts its value as a write does, and refuses one no column holds.
func TestFirstConvertsItsValue(t *testing.T) {
	_, s := shop[Product](t, dbFile(t))
	for _, p := range []Product{{Name: "Lamp", Price: 12, Status: "draft"}, {Name: "Desk", Price: 80, Status: "live"}} {
		if err := s.Create(ctx, &p); err != nil {
			t.Fatal(err)
		}
	}
	desk, _ := s.Get(ctx, 2)
	name := "Desk"
	for what, value := range map[string]any{
		"a pointer to a time": &desk.CreatedAt,
		"a named text type":   Status("live"),
		"a pointer to text":   &name,
	} {
		field := map[string]string{"a pointer to a time": "created_at", "a named text type": "status", "a pointer to text": "name"}[what]
		if got, ok, err := s.First(ctx, field, value); err != nil || !ok || got.ID != 2 {
			t.Errorf("First(%s, %s) = %+v, %v, %v; want id 2", field, what, got, ok, err)
		}
	}
	for _, c := range []struct {
		field string
		value any
		want  string
	}{
		{"name", nil, `data: products cannot look up "name" by nil, which no value equals; give it a value.`},
		{"name", (*string)(nil), `data: products cannot look up "name" by nil, which no value equals; give it a value.`},
		{"name", Product{}, `data: products cannot look up "name" by a value of type dbtest.Product; give it text.`},
		{"name", []string{"Desk"}, `data: products cannot look up "name" by a value of type []string; give it text.`},
		{"price", "80", `data: products cannot look up "price" by a value of type string; give it a number.`},
		{"created_at", "2026-11-25", `data: products cannot look up "created_at" by a value of type string; give it a time.Time.`},
	} {
		if _, ok, err := s.First(ctx, c.field, c.value); err == nil || err.Error() != c.want || ok {
			t.Errorf("First(%s, %#v) = %v, %v; want %q", c.field, c.value, ok, err, c.want)
		}
	}
}
