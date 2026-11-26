package auth

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// Logins are limited per email and address: after maxTries failures within
// a window that begins at the first, the rest of the window is refused. The
// count is kept in memory, by one process.
const (
	maxTries    = 5
	window      = time.Minute
	maxKeys     = 10_000 // keys remembered; the oldest is dropped past this
	maxKeyEmail = 200    // bytes of the email a key keeps
)

// limitKey is the limiter's key for email, lower-cased, trimmed and cut to
// 200 bytes, and the client's address.
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
	start time.Time // the window's first failure
	n     int
}

// limiter counts failed logins. The list holds the keys oldest window first,
// so the oldest is the one dropped.
type limiter struct {
	now func() time.Time
	max int

	mu    sync.Mutex
	keys  map[string]*list.Element // of *tries
	order *list.List
}

func newLimiter(now func() time.Time, max int) *limiter {
	return &limiter{now: now, max: max, keys: map[string]*list.Element{}, order: list.New()}
}

// current returns the count of key in the window now is in, nil when none.
func (l *limiter) current(key string, now time.Time) *list.Element {
	e := l.keys[key]
	if e == nil || !now.Before(e.Value.(*tries).start.Add(window)) {
		return nil
	}
	return e
}

// blocked reports whether key has failed maxTries times in its window.
func (l *limiter) blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.current(key, l.now())
	return e != nil && e.Value.(*tries).n >= maxTries
}

// fail counts one failure of key.
func (l *limiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if e := l.current(key, now); e != nil {
		e.Value.(*tries).n++
		return
	}
	if e := l.keys[key]; e != nil { // its window ended: a new one begins
		l.order.Remove(e)
		delete(l.keys, key)
	}
	for l.order.Len() >= l.max {
		oldest := l.order.Front()
		l.order.Remove(oldest)
		delete(l.keys, oldest.Value.(*tries).key)
	}
	l.keys[key] = l.order.PushBack(&tries{key: key, start: now, n: 1})
}

// clear forgets key's failures, after a success.
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
