package dbtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

// Visit is a model with a time field of its own, for DeleteBefore.
type Visit struct {
	data.Record
	Who  string    `json:"who"`
	At   time.Time `json:"at"`
	Note string    `json:"note"`
}

func visits(t *testing.T) (*data.DB, *data.Store[Visit]) {
	t.Helper()
	var s *data.Store[Visit]
	c := warptest.New(t, env(dbFile(t)), data.Module(), stores{"visits", func(db *data.DB) { s = data.NewStore[Visit](db, "visits") }})
	return data.For(c.App), s
}

func TestDeleteByRemovesOnlyMatches(t *testing.T) {
	_, s := visits(t)
	for _, who := range []string{"ana", "ben", "ana", "cy", "ana"} {
		if err := s.Create(ctx, &Visit{Who: who}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.DeleteBy(ctx, "who", "ana")
	if err != nil || n != 3 {
		t.Fatalf("DeleteBy(who, ana) = %d, %v; want 3", n, err)
	}
	left, err := s.List(ctx, data.Query{Sort: "id"})
	if err != nil || left.Total != 2 || left.Items[0].Who != "ben" || left.Items[1].Who != "cy" {
		t.Fatalf("left = %+v, %v; want ben and cy", left, err)
	}
	// A value is a value, never SQL; and none matching is 0, not an error.
	if n, err := s.DeleteBy(ctx, "who", `ben" OR 1=1 --`); err != nil || n != 0 {
		t.Fatalf("DeleteBy with SQL in the value = %d, %v; want 0", n, err)
	}
	// A pointer is what it points to.
	cy := "cy"
	if n, err := s.DeleteBy(ctx, "who", &cy); err != nil || n != 1 {
		t.Fatalf("DeleteBy(who, &cy) = %d, %v; want 1", n, err)
	}
}

func TestDeleteBeforeRemovesOlder(t *testing.T) {
	_, s := visits(t)
	at := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	for _, v := range []Visit{
		{Who: "old", At: at.Add(-time.Hour)},
		{Who: "just", At: at.Add(-time.Nanosecond)},
		{Who: "at", At: at},
		{Who: "new", At: at.Add(time.Hour)},
	} {
		if err := s.Create(ctx, &v); err != nil {
			t.Fatal(err)
		}
	}
	// A time in another zone is the same instant.
	n, err := s.DeleteBefore(ctx, "at", at.In(time.FixedZone("CET", 3600)))
	if err != nil || n != 2 {
		t.Fatalf("DeleteBefore(at) = %d, %v; want 2", n, err)
	}
	left, err := s.List(ctx, data.Query{Sort: "id"})
	if err != nil || left.Total != 2 || left.Items[0].Who != "at" || left.Items[1].Who != "new" {
		t.Fatalf("left = %+v, %v; want at and new", left, err)
	}
}

func TestDeleteByUnknownField(t *testing.T) {
	_, s := visits(t)
	if _, err := s.DeleteBy(ctx, "usr_id", 1); err == nil || err.Error() != `data: visits has no field "usr_id" to delete by` {
		t.Fatalf("DeleteBy(usr_id) = %v", err)
	}
	if _, err := s.DeleteBefore(ctx, "when", time.Now()); err == nil || err.Error() != `data: visits has no field "when" to delete by` {
		t.Fatalf("DeleteBefore(when) = %v", err)
	}
	// A value the column does not hold is refused as First refuses it.
	if _, err := s.DeleteBy(ctx, "who", 7); err == nil {
		t.Fatal("DeleteBy(who, 7) was taken")
	}
}

func TestDeleteBeforeNotATime(t *testing.T) {
	_, s := visits(t)
	want := `data: visits cannot delete by "note" before a time, since it is not a time; name a time field.`
	if _, err := s.DeleteBefore(ctx, "note", time.Now()); err == nil || err.Error() != want {
		t.Fatalf("DeleteBefore(note) = %v, want %q", err, want)
	}
}

func TestDeleteByInTx(t *testing.T) {
	db, s := visits(t)
	at := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	for _, who := range []string{"ana", "ben"} {
		if err := s.Create(ctx, &Visit{Who: who, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	failed := errors.New("rolled back")
	err := db.Tx(ctx, func(ctx context.Context) error {
		if n, err := s.DeleteBy(ctx, "who", "ana"); err != nil || n != 1 {
			t.Errorf("DeleteBy inside = %d, %v", n, err)
		}
		if n, err := s.DeleteBefore(ctx, "at", at.Add(time.Second)); err != nil || n != 1 {
			t.Errorf("DeleteBefore inside = %d, %v; want 1, since ana is gone in the transaction", n, err)
		}
		return failed
	})
	if !errors.Is(err, failed) {
		t.Fatal(err)
	}
	if left, err := s.List(ctx, data.Query{Sort: "id"}); err != nil || left.Total != 2 {
		t.Fatalf("after the rollback = %+v, %v; want both", left, err)
	}
}
