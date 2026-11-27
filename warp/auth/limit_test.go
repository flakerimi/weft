package auth

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) set(t time.Time) {
	c.mu.Lock()
	c.now = t
	c.mu.Unlock()
}

func TestLimit(t *testing.T) {
	clock := &testClock{now: testNow}
	l := newLimiter(clock.Now, maxKeys, maxTries, window)
	key := limitKey("owner@example.com", "192.0.2.1")

	for i := range maxTries {
		if l.take(key) == nil {
			t.Fatalf("try %d is refused, want the first %d to pass", i+1, maxTries)
		}
		clock.set(clock.Now().Add(time.Second))
	}
	if l.take(key) != nil {
		t.Fatalf("try %d passes after %d tries", maxTries+1, maxTries)
	}
	if l.take(limitKey("owner@example.com", "192.0.2.2")) == nil {
		t.Error("another address is refused")
	}
	if l.take(limitKey("other@example.com", "192.0.2.1")) == nil {
		t.Error("another email is refused")
	}
	// The email is compared lower-cased and trimmed.
	if limitKey("  Owner@Example.COM ", "192.0.2.1") != key {
		t.Error("the key differs by case or spaces")
	}

	// Refused until the minute that began with the first try ends; a refused
	// try is not counted, so it does not move the end.
	clock.set(testNow.Add(window - time.Nanosecond))
	if l.take(key) != nil {
		t.Error("refusing ended before the minute did")
	}
	clock.set(testNow.Add(window))
	if l.take(key) == nil {
		t.Error("still refused once the minute ended")
	}
	// The new minute began with that try, and counts afresh.
	for range maxTries - 1 {
		l.take(key)
	}
	if l.take(key) != nil {
		t.Errorf("%d tries in a new minute pass", maxTries+1)
	}

	// A success clears the count.
	l.clear(key)
	for i := range maxTries {
		if l.take(key) == nil {
			t.Fatalf("try %d after a success is refused", i+1)
		}
	}
}

// Tries at the same moment are counted at once: of 20, 5 pass.
func TestLimitIsAtomic(t *testing.T) {
	l := newLimiter((&testClock{now: testNow}).Now, maxKeys, maxTries, window)
	key := limitKey("owner@example.com", "192.0.2.1")
	var wg sync.WaitGroup
	var mu sync.Mutex
	passed := 0
	for range 20 {
		wg.Go(func() {
			if l.take(key) != nil {
				mu.Lock()
				passed++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if passed != maxTries {
		t.Fatalf("%d of 20 tries at once passed, want %d", passed, maxTries)
	}
}

func TestLimitKeepsAtMostMaxKeys(t *testing.T) {
	clock := &testClock{now: testNow}
	l := newLimiter(clock.Now, maxKeys, maxTries, window)
	block := func(key string) {
		for range maxTries {
			l.take(key)
		}
	}
	first := limitKey("first@example.com", "192.0.2.1")
	block(first)
	for i := range maxKeys - 1 {
		clock.set(clock.Now().Add(time.Microsecond))
		l.take(limitKey(fmt.Sprintf("u%d@example.com", i), "192.0.2.1"))
	}
	if n := l.size(); n != maxKeys {
		t.Fatalf("%d keys remembered, want %d", n, maxKeys)
	}
	if l.take(first) != nil {
		t.Fatal("the oldest key was dropped before the limit was reached")
	}
	second := limitKey("u0@example.com", "192.0.2.1")
	block(second)
	l.take(limitKey("one-more@example.com", "192.0.2.1"))
	if n := l.size(); n != maxKeys {
		t.Fatalf("%d keys remembered past the limit, want %d", n, maxKeys)
	}
	// second first: taking first, now unknown, adds a key and drops another.
	if l.take(second) != nil {
		t.Error("a newer key was dropped in place of the oldest")
	}
	if l.take(first) == nil {
		t.Error("the oldest key was kept when a new one came")
	}
}

func TestLimitKeyIsBounded(t *testing.T) {
	huge := strings.Repeat("a", 1<<20) + "@example.com"
	key := limitKey(huge, "2001:db8::1")
	if len(key) > maxKeyEmail+1+len("2001:db8::1") {
		t.Fatalf("a key of a %d-byte email is %d bytes", len(huge), len(key))
	}
	if limitKey(huge+"x", "192.0.2.1") != limitKey(huge, "192.0.2.1") {
		t.Error("two emails alike in their first 200 bytes have two keys")
	}
}

// Each limiter has its own window, on one clock.
func TestLimiterWindowPerLimiter(t *testing.T) {
	clock := &testClock{now: testNow}
	short := newLimiter(clock.Now, maxKeys, 1, time.Minute)
	long := newLimiter(clock.Now, maxKeys, 1, 10*time.Minute)
	if short.take("k") == nil || long.take("k") == nil {
		t.Fatal("a first try is refused")
	}
	if short.take("k") != nil || long.take("k") != nil {
		t.Fatal("a second try passes a limit of 1")
	}
	clock.set(testNow.Add(time.Minute))
	if short.take("k") == nil {
		t.Error("the one-minute limiter still refuses after a minute")
	}
	if long.take("k") != nil {
		t.Error("the ten-minute limiter passes after a minute")
	}
	clock.set(testNow.Add(10*time.Minute - time.Nanosecond))
	if long.take("k") != nil {
		t.Error("the ten-minute limiter passes before ten minutes")
	}
	clock.set(testNow.Add(10 * time.Minute))
	if long.take("k") == nil {
		t.Error("the ten-minute limiter still refuses after ten minutes")
	}
}

func TestAccountKeyIsBounded(t *testing.T) {
	huge := strings.Repeat("a", 1<<20) + "@example.com"
	if k := accountKey(huge); len(k) != maxKeyEmail {
		t.Fatalf("the account key of a %d-byte email is %d bytes", len(huge), len(k))
	}
	if accountKey("  Owner@Example.COM ") != "owner@example.com" {
		t.Error("the account key differs by case or spaces")
	}
}
