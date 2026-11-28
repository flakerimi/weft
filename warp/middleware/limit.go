package middleware

import (
	"container/list"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flakerimi/weft/warp"
)

// The rate limit's bounds.
const (
	maxClients = 50_000    // clients remembered; the one seen longest ago is dropped past this
	maxRate    = 1_000_000 // requests a period RATE_LIMIT may give
)

// rate is a RATE_LIMIT: n requests a period, one coming back each interval.
type rate struct {
	n        int
	interval time.Duration
}

// readRate reads RATE_LIMIT: n/s, n/m or n/h, n from 1 to 1,000,000; unset
// is 300/m, and off is nil.
func readRate(v string) (*rate, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "":
		s = "300/m"
	case "off":
		return nil, nil
	}
	bad := fmt.Errorf("RATE_LIMIT is %q; set it like 300/m (requests a minute), 20/s or 5000/h, or off.", v)
	count, unit, ok := strings.Cut(s, "/")
	periods := map[string]time.Duration{"s": time.Second, "m": time.Minute, "h": time.Hour}
	period, known := periods[unit]
	n, err := strconv.Atoi(count)
	if !ok || !known || err != nil || strconv.Itoa(n) != count || n < 1 || n > maxRate {
		return nil, bad
	}
	return &rate{n: n, interval: period / time.Duration(n)}, nil
}

// limiter keeps a bucket per client that holds n requests and gets one back
// each interval, as the generic cell rate algorithm keeps it: each client's
// due is when its bucket would be full again had it sent nothing since. The
// list holds the clients seen longest ago first, so that one is dropped
// past maxClients.
type limiter struct {
	now   func() time.Time
	rate  rate
	burst time.Duration // n intervals: how far ahead of now a due may be

	mu      sync.Mutex
	clients map[string]*list.Element // of *client
	order   *list.List
}

type client struct {
	key string
	due time.Time
}

func newLimiter(now func() time.Time, r rate) *limiter {
	return &limiter{now: now, rate: r, burst: time.Duration(r.n) * r.interval, clients: map[string]*list.Element{}, order: list.New()}
}

// take counts one request of key, and returns 0 when it may go, or how long
// until one may.
func (l *limiter) take(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e := l.clients[key]
	if e == nil {
		for l.order.Len() >= maxClients {
			oldest := l.order.Front()
			l.order.Remove(oldest)
			delete(l.clients, oldest.Value.(*client).key)
		}
		e = l.order.PushBack(&client{key: key})
		l.clients[key] = e
	} else {
		l.order.MoveToBack(e)
	}
	c := e.Value.(*client)
	due := c.due
	if due.Before(now) {
		due = now
	}
	due = due.Add(l.rate.interval)
	if wait := due.Sub(now) - l.burst; wait > 0 {
		return wait
	}
	c.due = due
	return 0
}

// size is how many clients are remembered.
func (l *limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.order.Len()
}

// limitRequests answers a client past the rate limit with a 429 and, in
// Retry-After, the whole seconds until it may send again. Before Start, and
// with RATE_LIMIT=off, it limits nothing. It is around the API alone.
func (m *module) limitRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := m.limit
		if l == nil {
			next.ServeHTTP(w, r)
			return
		}
		wait := l.take(m.app.ClientKey(r))
		if wait == 0 {
			next.ServeHTTP(w, r)
			return
		}
		seconds := max(1, (wait+time.Second-1)/time.Second)
		w.Header().Set("Retry-After", strconv.FormatInt(int64(seconds), 10))
		warp.WriteError(w, r, warp.TooMany("Too many requests; wait a moment and try again."))
	})
}
