package auth

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// Logins are limited twice, in memory, by one process: per email and address,
// maxTries tries in a window that begins at the first; and per address across
// every email, maxAddressTries. A try is counted when it begins, before any
// hashing, so tries sent at once cannot pass together. A success clears its
// email and address; an address's count only ends with its window.
const (
	maxTries        = 5
	maxAddressTries = 30
	window          = time.Minute
	maxKeys         = 10_000 // keys a limiter remembers; the oldest is dropped past this
	maxKeyEmail     = 200    // bytes of the email a key keeps
)

// limitKey is the per-email limiter's key for email, lower-cased, trimmed and
// cut to 200 bytes, and the client's address.
func limitKey(email, ip string) string {
	e := normalEmail(email)
	if len(e) > maxKeyEmail {
		e = e[:maxKeyEmail]
	}
	return e + "\x00" + ip
}

// normalEmail is an email as it is stored and compared.
func normalEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

type tries struct {
	key   string
	start time.Time // the window's first try
	n     int
}

// limiter counts tries per key. The list holds the keys oldest window first,
// so the oldest is the one dropped.
type limiter struct {
	now      func() time.Time
	maxKeys  int
	maxTries int

	mu    sync.Mutex
	keys  map[string]*list.Element // of *tries
	order *list.List
}

func newLimiter(now func() time.Time, maxKeys, maxTries int) *limiter {
	return &limiter{now: now, maxKeys: maxKeys, maxTries: maxTries, keys: map[string]*list.Element{}, order: list.New()}
}

// take counts one try of key and reports true, or reports false, counting
// nothing, when key already has maxTries in its window. It is one step under
// the lock, so of tries at once only maxTries pass.
func (l *limiter) take(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if e := l.keys[key]; e != nil {
		t := e.Value.(*tries)
		if now.Before(t.start.Add(window)) {
			if t.n >= l.maxTries {
				return false
			}
			t.n++
			return true
		}
		l.order.Remove(e) // its window ended: a new one begins
		delete(l.keys, key)
	}
	for l.order.Len() >= l.maxKeys {
		oldest := l.order.Front()
		l.order.Remove(oldest)
		delete(l.keys, oldest.Value.(*tries).key)
	}
	l.keys[key] = l.order.PushBack(&tries{key: key, start: now, n: 1})
	return true
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
