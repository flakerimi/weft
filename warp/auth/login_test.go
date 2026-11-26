package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
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
	if _, err := m.verify(ctx, dummyHash, "x"); status(err) != http.StatusServiceUnavailable {
		t.Fatalf("verify with every slot taken = %v, want the 503", err)
	}
	if _, err := m.hash(ctx, "a password"); status(err) != http.StatusServiceUnavailable {
		t.Fatalf("hash with every slot taken = %v, want the 503", err)
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

// triesOf is how many tries l has counted for key in its window.
func triesOf(l *limiter, key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e := l.keys[key]; e != nil {
		return e.Value.(*tries).n
	}
	return 0
}

// A login that waits too long for a hashing slot, or whose request ends
// first, gives its try back on both counts, since no hash ran and nothing
// was learned, and is answered 503 without an error in the log.
func TestBusySigningInCostsNoTry(t *testing.T) {
	const email, ip = "owner@example.com", "192.0.2.1"
	m := fastModule(t, &testClock{now: testNow})
	m.slotWait = 30 * time.Millisecond
	key := limitKey(email, ip)
	if _, err := m.attempt(context.Background(), email, "wrong", ip); status(err) != 401 {
		t.Fatalf("a first guess: %v", err)
	}
	for range cap(m.slots) {
		m.slots <- struct{}{}
	}
	hashes := m.hashes.Load()
	busy := func(what string, err error) {
		t.Helper()
		var e *warp.Error
		if !errors.As(err, &e) || e.Status != 503 || e.Code != "unavailable" || e.Message != "The server is busy signing people in; try again in a moment." {
			t.Fatalf("%s: %v, want the 503", what, err)
		}
		if n, a := triesOf(m.perKey, key), triesOf(m.perAddress, ip); n != 1 || a != 1 {
			t.Fatalf("%s: the counts are %d and %d, want 1 and 1, as before", what, n, a)
		}
		if m.hashes.Load() != hashes {
			t.Fatalf("%s: a hash ran", what)
		}
	}
	began := time.Now()
	_, err := m.attempt(context.Background(), email, "right", ip)
	busy("past the wait", err)
	if took := time.Since(began); took < m.slotWait {
		t.Fatalf("gave up after %v, before the %v wait", took, m.slotWait)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = m.attempt(ctx, email, "right", ip)
	busy("the request ended", err)

	// Over HTTP, the answer is the 503 and the log has no error.
	var logs strings.Builder
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError})))
	defer slog.SetDefault(old)
	m.app = warp.New(warp.Config{})
	warp.Post(m.app, "/auth/login", m.login, warp.Status(http.StatusOK))
	r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"owner@example.com","password":"right"}`))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = ip + ":1000"
	w := httptest.NewRecorder()
	m.app.Handler().ServeHTTP(w, r)
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"code":"unavailable"`) || logs.Len() != 0 {
		t.Fatalf("over HTTP: %d %s, log %q", w.Code, w.Body, logs.String())
	}
	if n := triesOf(m.perKey, key); n != 1 {
		t.Fatalf("over HTTP the count is %d, want 1", n)
	}
	// With a slot free again, a login is tried as usual.
	<-m.slots
	if _, err := m.attempt(context.Background(), email, "right", ip); err != nil {
		t.Fatalf("with a slot free: %v", err)
	}
}

// The default wait for a slot is 3 seconds.
func TestSlotWait(t *testing.T) {
	if w := newModule(time.Now).slotWait; w != 3*time.Second {
		t.Fatalf("the wait is %v, want 3s", w)
	}
}

// A give returns one try, and forgets a key with none left.
func TestLimitGive(t *testing.T) {
	l := newLimiter((&testClock{now: testNow}).Now, maxKeys, 2)
	l.take("k")
	l.take("k")
	if l.take("k") {
		t.Fatal("a third try passed a limit of 2")
	}
	l.give("k")
	if !l.take("k") || l.take("k") {
		t.Fatal("a given-back try was not one try")
	}
	l.give("k")
	l.give("k")
	l.give("k") // more than were taken: nothing below none
	if l.size() != 0 {
		t.Fatalf("%d keys after every try was given back", l.size())
	}
	l.give("never")
	if l.size() != 0 {
		t.Fatal("giving back a key never taken remembered it")
	}
}
