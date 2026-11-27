package auth

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// Logins are limited three times, in memory, by one process: per email and
// address, maxTries tries in a minute that begins at the first; per address
// across every email, maxAddressTries a minute; and per account across every
// address, maxAccountTries in accountWindow, so that guesses spread over many
// addresses still stop. A try is counted when it begins, before any hashing,
// so tries sent at once cannot pass together. A success clears its email and
// address; an address's and an account's count only end with their window
// (or, for an account, a completed reset). A try that finds no hashing slot in
// time is given back on every count. The address is the app's ClientKey: an
// IPv6 client is counted by its /64. Every email is counted, with an account
// or without, so the answer says nothing of who has one.
const (
	maxTries        = 5
	maxAddressTries = 30
	window          = time.Minute
	maxAccountTries = 10
	accountWindow   = 10 * time.Minute
	maxKeys         = 10_000 // keys a limiter remembers; the oldest is dropped past this
	maxKeyEmail     = 200    // bytes of the email a key keeps
)

// accountKey is the per-account limiter's key for email: lower-cased,
// trimmed and cut to 200 bytes.
func accountKey(email string) string {
	e := normalEmail(email)
	if len(e) > maxKeyEmail {
		e = e[:maxKeyEmail]
	}
	return e
}

// limitKey is the per-email limiter's key for email, lower-cased, trimmed and
// cut to 200 bytes, and the client's address.
func limitKey(email, ip string) string { return accountKey(email) + "\x00" + ip }

// normalEmail is an email as it is stored and compared.
func normalEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

type tries struct {
	key   string
	start time.Time // the window's first try
	n     int
}

// limiter counts tries per key, maxTries in a window that begins at the
// first. The list holds the keys oldest window first, so the oldest is the
// one dropped.
type limiter struct {
	now      func() time.Time
	maxKeys  int
	maxTries int
	window   time.Duration

	mu    sync.Mutex
	keys  map[string]*list.Element // of *tries
	order *list.List
}

func newLimiter(now func() time.Time, maxKeys, maxTries int, window time.Duration) *limiter {
	return &limiter{now: now, maxKeys: maxKeys, maxTries: maxTries, window: window, keys: map[string]*list.Element{}, order: list.New()}
}

// take counts one try of key and returns the count it counted it in, or
// returns nil, counting nothing, when key already has maxTries in its
// window. It is one step under the lock, so of tries at once only maxTries
// pass.
func (l *limiter) take(key string) *tries {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if e := l.keys[key]; e != nil {
		t := e.Value.(*tries)
		if now.Before(t.start.Add(l.window)) {
			if t.n >= l.maxTries {
				return nil
			}
			t.n++
			return t
		}
		l.order.Remove(e) // its window ended: a new one begins
		delete(l.keys, key)
	}
	for l.order.Len() >= l.maxKeys {
		oldest := l.order.Front()
		l.order.Remove(oldest)
		delete(l.keys, oldest.Value.(*tries).key)
	}
	t := &tries{key: key, start: now, n: 1}
	l.keys[key] = l.order.PushBack(t)
	return t
}

// clear forgets key's tries, after a success.
func (l *limiter) clear(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e := l.keys[key]; e != nil {
		l.order.Remove(e)
		delete(l.keys, key)
	}
}

// size is how many keys are remembered.
func (l *limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.order.Len()
}

// give hands back one try to t, the count take counted it in, for a try
// that then never ran, such as a login that found no hashing slot. It does
// so only while t is still key's count: once t's window has ended, or t was
// cleared or dropped, a later count is not t, and nothing is given back to
// it. A count is never below none, and one left with none is forgotten.
func (l *limiter) give(t *tries) {
	if t == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.keys[t.key]
	if e == nil || e.Value.(*tries) != t || t.n < 1 {
		return
	}
	if t.n--; t.n == 0 {
		l.order.Remove(e)
		delete(l.keys, t.key)
	}
}
