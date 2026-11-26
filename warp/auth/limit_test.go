package auth

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

func TestLimit(t *testing.T) {
	clock := &testClock{now: testNow}
	l := newLimiter(clock.Now, maxKeys)
	key := limitKey("owner@example.com", "192.0.2.1")

	for i := range maxTries {
		if l.blocked(key) {
			t.Fatalf("try %d is refused, want the first %d to pass", i+1, maxTries)
		}
		l.fail(key)
		clock.now = clock.now.Add(time.Second)
	}
	if !l.blocked(key) {
		t.Fatalf("try %d passes after %d failures", maxTries+1, maxTries)
	}
	if l.blocked(limitKey("owner@example.com", "192.0.2.2")) {
		t.Error("another address is refused")
	}
	if l.blocked(limitKey("other@example.com", "192.0.2.1")) {
		t.Error("another email is refused")
	}
	// The email is compared lower-cased and trimmed.
	if limitKey("  Owner@Example.COM ", "192.0.2.1") != key {
		t.Error("the key differs by case or spaces")
	}

	// Until the minute that began with the first failure ends.
	clock.now = testNow.Add(window - time.Nanosecond)
	if !l.blocked(key) {
		t.Error("refusing ended before the minute did")
	}
	clock.now = testNow.Add(window)
	if l.blocked(key) {
		t.Error("still refused once the minute ended")
	}
	// A new minute counts afresh.
	for range maxTries - 1 {
		l.fail(key)
	}
	if l.blocked(key) {
		t.Errorf("%d failures in a new minute refuse", maxTries-1)
	}

	// A success clears the count.
	l.clear(key)
	for range maxTries - 1 {
		l.fail(key)
	}
	if l.blocked(key) {
		t.Error("the failures before a success still count")
	}
}

func TestLimitKeepsAtMostMaxKeys(t *testing.T) {
	clock := &testClock{now: testNow}
	l := newLimiter(clock.Now, maxKeys)
	block := func(key string) {
		for range maxTries {
			l.fail(key)
		}
	}
	first := limitKey("first@example.com", "192.0.2.1")
	block(first)
	for i := range maxKeys - 1 {
		clock.now = clock.now.Add(time.Microsecond)
		l.fail(limitKey(fmt.Sprintf("u%d@example.com", i), "192.0.2.1"))
	}
	if n := l.size(); n != maxKeys {
		t.Fatalf("%d keys remembered, want %d", n, maxKeys)
	}
	if !l.blocked(first) {
		t.Fatal("the oldest key was dropped before the limit was reached")
	}
	second := limitKey("u0@example.com", "192.0.2.1")
	block(second)
	l.fail(limitKey("one-more@example.com", "192.0.2.1"))
	if n := l.size(); n != maxKeys {
		t.Fatalf("%d keys remembered past the limit, want %d", n, maxKeys)
	}
	if l.blocked(first) {
		t.Error("the oldest key was kept when a new one came")
	}
	if !l.blocked(second) {
		t.Error("a newer key was dropped in place of the oldest")
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
