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
	"sync/atomic"
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
	// Past the minute, and past the account's ten minutes, which the guesses
	// spent half of.
	clock.set(testNow.Add(accountWindow))
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

// A give returns one try to the count it was taken from, forgets a key with
// none left, and gives nothing to a later count of the same key.
func TestLimitGive(t *testing.T) {
	clock := &testClock{now: testNow}
	l := newLimiter(clock.Now, maxKeys, 2, window)
	a := l.take("k")
	b := l.take("k")
	if a != b || l.take("k") != nil {
		t.Fatal("a third try passed a limit of 2, or two tries of one window were counted apart")
	}
	l.give(a)
	if l.take("k") == nil || l.take("k") != nil {
		t.Fatal("a given-back try was not one try")
	}
	l.give(a)
	l.give(a)
	l.give(a) // more than were taken: nothing below none
	if l.size() != 0 {
		t.Fatalf("%d keys after every try was given back", l.size())
	}
	l.give(nil)
	// A try of an ended window, given back in the next, is not the next's.
	old := l.take("k")
	clock.set(testNow.Add(window))
	next := l.take("k")
	l.give(old)
	if triesOf(l, "k") != 1 || next == old {
		t.Fatalf("a try of the last window was given to this one: count %d", triesOf(l, "k"))
	}
}

// fill takes every hashing slot and returns a function that frees them.
func fill(m *module) func() {
	for range cap(m.slots) {
		m.slots <- struct{}{}
	}
	return func() {
		for range cap(m.slots) {
			<-m.slots
		}
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("the condition was not reached in 5 s")
		}
		time.Sleep(time.Millisecond)
	}
}

// A try given back is given back to the window that counted it, never to
// a later one: guesses that wait past their window's end and then give up
// leave the next window's count alone, so no window runs more hashes for
// one email than the limit.
func TestGiveBackStaysInItsWindow(t *testing.T) {
	const email, ip = "owner@example.com", "192.0.2.1"
	key := limitKey(email, ip)
	clock := &testClock{now: testNow}
	m := fastModule(t, clock)
	m.slotWait = time.Hour // only a cancel ends the wait
	free := fill(m)

	// Window 1: five guesses that wait for a slot.
	ctx1, cancel1 := context.WithCancel(context.Background())
	var w1 sync.WaitGroup
	for range maxTries {
		w1.Go(func() { m.attempt(ctx1, email, "wrong", ip) })
	}
	waitFor(t, func() bool { return triesOf(m.perKey, key) == maxTries })

	// Window 2 begins; five guesses take its tries and wait.
	clock.set(testNow.Add(window + time.Second))
	var w2 sync.WaitGroup
	var hashed atomic.Int64
	guess := func() {
		w2.Go(func() {
			if _, err := m.attempt(context.Background(), email, "wrong", ip); status(err) == 401 {
				hashed.Add(1)
			}
		})
	}
	for range maxTries {
		guess()
	}
	waitFor(t, func() bool { return triesOf(m.perKey, key) == maxTries && triesOf(m.perAddress, ip) == maxTries })
	time.Sleep(20 * time.Millisecond) // window 2's guesses are waiting for a slot

	// Window 1's guesses give up: window 2's count is not theirs.
	cancel1()
	w1.Wait()
	if n := triesOf(m.perKey, key); n != maxTries {
		t.Fatalf("after window 1's give-backs window 2's count is %d, want %d", n, maxTries)
	}
	// So five more in window 2 are refused, and no more than five hash.
	for range maxTries {
		guess()
	}
	time.Sleep(20 * time.Millisecond)
	free()
	w2.Wait()
	if n := hashed.Load(); n > maxTries {
		t.Fatalf("window 2 ran %d hashes for one email, more than the limit of %d", n, maxTries)
	}
}

// A try whose user could not be looked up (a busy database) ran no hash and
// learned nothing, so it is given back on both counts too.
func TestFailedLookupCostsNoTry(t *testing.T) {
	const email, ip = "owner@example.com", "192.0.2.1"
	m := fastModule(t, &testClock{now: testNow})
	m.find = func(context.Context, string) (userRow, bool, error) {
		return userRow{}, false, warp.Unavailable("The database is busy; try again in a moment.")
	}
	for range maxTries + 2 {
		if _, err := m.attempt(context.Background(), email, "wrong", ip); status(err) != 503 {
			t.Fatalf("a lookup on a busy database: %v, want its 503", err)
		}
	}
	if n, a := triesOf(m.perKey, limitKey(email, ip)), triesOf(m.perAddress, ip); n != 0 || a != 0 {
		t.Fatalf("the counts are %d and %d, want 0 and 0", n, a)
	}
}

// When the address has tries left and the email has none, the address's
// try stays spent, as it was before tries could be given back: a refused
// email does not loosen the per-address limit.
func TestRefusedEmailSpendsTheAddress(t *testing.T) {
	const ip = "192.0.2.1"
	m := fastModule(t, &testClock{now: testNow})
	for range maxTries {
		m.attempt(context.Background(), "owner@example.com", "wrong", ip)
	}
	if _, err := m.attempt(context.Background(), "owner@example.com", "wrong", ip); status(err) != 429 {
		t.Fatalf("a sixth guess: %v, want 429", err)
	}
	if a := triesOf(m.perAddress, ip); a != maxTries+1 {
		t.Fatalf("the address's count is %d, want %d", a, maxTries+1)
	}
}

// The account's count is taken after the address's and before the email and
// address's: a spent account gives back the address's try, and a spent email
// and address gives back the account's, so only tries that were tried count
// against an account.
func TestAccountCountOrder(t *testing.T) {
	const email = "owner@example.com"
	clock := &testClock{now: testNow}
	m := fastModule(t, clock)
	for i := range maxAccountTries {
		ip := fmt.Sprintf("192.0.2.%d", i+1)
		if _, err := m.attempt(context.Background(), email, "wrong", ip); status(err) != 401 {
			t.Fatalf("guess %d: %v, want 401", i+1, err)
		}
	}
	const fresh = "198.51.100.1"
	_, err := m.attempt(context.Background(), email, "right", fresh)
	var e *warp.Error
	if !errors.As(err, &e) || e.Status != 429 || e.Code != "rate_limited" || e.Message != "Too many tries for this account; wait 10 minutes, or reset the password." {
		t.Fatalf("the right password from a fresh address: %v, want the account's 429", err)
	}
	if a, k := triesOf(m.perAddress, fresh), triesOf(m.perKey, limitKey(email, fresh)); a != 0 || k != 0 {
		t.Fatalf("a spent account left the address's count at %d and the key's at %d, want 0 and 0", a, k)
	}
	// Another email is not this account.
	if _, err := m.attempt(context.Background(), "other@example.com", "right", fresh); err != nil {
		t.Fatalf("another email: %v", err)
	}
	// A success does not clear the account's count; its window ends it.
	clock.set(testNow.Add(accountWindow))
	if _, err := m.attempt(context.Background(), email, "right", fresh); err != nil {
		t.Fatalf("after ten minutes: %v", err)
	}
	if n := triesOf(m.perAccount, accountKey(email)); n != 1 {
		t.Fatalf("a success left the account's count at %d, want 1", n)
	}

	// A spent email and address gives back the account's try.
	m = fastModule(t, &testClock{now: testNow})
	for range maxTries {
		m.attempt(context.Background(), email, "wrong", "192.0.2.1")
	}
	if _, err := m.attempt(context.Background(), email, "wrong", "192.0.2.1"); status(err) != 429 || err.Error() != errTooMany {
		t.Fatalf("a sixth guess from one address: %v, want %q", err, errTooMany)
	}
	if n := triesOf(m.perAccount, accountKey(email)); n != maxTries {
		t.Fatalf("the account's count is %d after a refused try, want %d", n, maxTries)
	}
}

// A login that finds no hashing slot counts nothing on the account either.
func TestAccountCountGivenBackWhenBusy(t *testing.T) {
	const email, ip = "owner@example.com", "192.0.2.1"
	m := fastModule(t, &testClock{now: testNow})
	m.slotWait = 10 * time.Millisecond
	free := fill(m)
	defer free()
	for range maxAccountTries + 2 {
		if _, err := m.attempt(context.Background(), email, "right", ip); status(err) != 503 {
			t.Fatalf("with no slot: %v, want 503", err)
		}
	}
	if n := triesOf(m.perAccount, accountKey(email)); n != 0 {
		t.Fatalf("the account's count is %d, want 0", n)
	}
}
