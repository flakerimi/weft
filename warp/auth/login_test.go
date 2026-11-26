package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
)

// fastModule is a module whose every email has a user with the password
// "right", hashed with one iteration so that a test can make many tries; it
// counts hashes in m.hashes.
func fastModule(t *testing.T, clock *testClock) *module {
	t.Helper()
	m := newModule(clock.Now)
	stored, err := hashWith("right", 1)
	if err != nil {
		t.Fatal(err)
	}
	m.find = func(_ context.Context, email string) (userRow, bool, error) {
		return userRow{Email: email, Role: "owner", PasswordHash: stored}, true, nil
	}
	return m
}

func status(err error) int {
	var e *warp.Error
	if errors.As(err, &e) {
		return e.Status
	}
	if err == nil {
		return http.StatusOK
	}
	return 0
}

func TestLoginLimitPerKeyIsAtomic(t *testing.T) {
	clock := &testClock{now: testNow}
	m := fastModule(t, clock)
	var wg sync.WaitGroup
	var mu sync.Mutex
	answers := map[int]int{}
	for range 20 {
		wg.Go(func() {
			_, err := m.attempt(context.Background(), "owner@example.com", "wrong", "192.0.2.1")
			mu.Lock()
			answers[status(err)]++
			mu.Unlock()
		})
	}
	wg.Wait()
	if answers[401] != maxTries || answers[429] != 20-maxTries {
		t.Fatalf("20 guesses at once answered %v, want %d of 401 and the rest 429", answers, maxTries)
	}
	if n := m.hashes.Load(); n != maxTries {
		t.Fatalf("%d hashes for 20 guesses, want %d", n, maxTries)
	}
	// The limit was reached, so the right password waits for the minute too.
	if _, err := m.attempt(context.Background(), "owner@example.com", "right", "192.0.2.1"); status(err) != 429 {
		t.Fatalf("the right password in the same minute: %v, want 429", err)
	}
	clock.set(testNow.Add(window))
	if _, err := m.attempt(context.Background(), "owner@example.com", "right", "192.0.2.1"); err != nil {
		t.Fatalf("the right password after the minute: %v", err)
	}
	// A success cleared the email's count; the address's is only spent.
	for i := range maxTries {
		if _, err := m.attempt(context.Background(), "owner@example.com", "wrong", "192.0.2.1"); status(err) != 401 {
			t.Fatalf("guess %d after a success: %v, want 401", i+1, err)
		}
	}
}

func TestLoginLimitPerAddress(t *testing.T) {
	m := fastModule(t, &testClock{now: testNow})
	for i := range maxAddressTries {
		email := fmt.Sprintf("u%d@example.com", i)
		if _, err := m.attempt(context.Background(), email, "wrong", "192.0.2.1"); status(err) != 401 {
			t.Fatalf("email %d from one address: %v, want 401", i+1, err)
		}
	}
	if n := m.hashes.Load(); n != maxAddressTries {
		t.Fatalf("%d hashes, want %d", n, maxAddressTries)
	}
	_, err := m.attempt(context.Background(), "one-more@example.com", "right", "192.0.2.1")
	if status(err) != 429 || err.Error() != errTooMany {
		t.Fatalf("email %d from one address: %v, want 429 %q", maxAddressTries+1, err, errTooMany)
	}
	if n := m.hashes.Load(); n != maxAddressTries {
		t.Fatalf("the refused try hashed: %d hashes, want %d", n, maxAddressTries)
	}
	// Another address is not refused.
	if _, err := m.attempt(context.Background(), "one-more@example.com", "right", "192.0.2.2"); err != nil {
		t.Fatalf("another address: %v", err)
	}
}

func TestHashingIsBounded(t *testing.T) {
	m := newModule(time.Now)
	if cap(m.slots) != runtime.GOMAXPROCS(0) {
		t.Fatalf("%d hashes may run at once, want GOMAXPROCS, %d", cap(m.slots), runtime.GOMAXPROCS(0))
	}
	for range cap(m.slots) {
		m.slots <- struct{}{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := m.verify(ctx, dummyHash, "x"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("verify with every slot taken = %v, want the context's error", err)
	}
	if _, err := m.hash(ctx, "a password"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("hash with every slot taken = %v, want the context's error", err)
	}
	if n := m.hashes.Load(); n != 0 {
		t.Fatalf("%d hashes ran without a slot", n)
	}
	<-m.slots
	if ok, err := m.verify(context.Background(), dummyHash, "x"); ok || err != nil {
		t.Fatalf("verify with a slot free = %v, %v", ok, err)
	}
	if len(m.slots) != cap(m.slots)-1 {
		t.Fatal("verify did not give its slot back")
	}
}
