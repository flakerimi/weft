package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/warptest"
)

// clock is a time a test moves by hand.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// limited is a client of an app with the middleware module at RATE_LIMIT
// rate (unset when rate is ""), on c's time, with a route at /api/x.
func limited(t *testing.T, rate string, c *clock) *warptest.Client {
	vars := map[string]string{"LOG": "off"}
	if rate != "" {
		vars["RATE_LIMIT"] = rate
	}
	return warptest.New(t, vars, newModule(c.now), xModule{})
}

// xModule is a module with a route at /api/x.
type xModule struct{}

func (xModule) Name() string { return "x" }
func (xModule) Register(app *warp.App) error {
	app.Handle("GET /x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "x") }))
	return nil
}

// from sends GET /api/x from the address ip.
func from(cl *warptest.Client, ip string) *warptest.Response {
	r := httptest.NewRequest("GET", "http://warptest.local/api/x", nil)
	r.RemoteAddr = ip + ":1234"
	return cl.Do(r)
}

func TestRateLimit(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	cl := limited(t, "5/m", c)
	for i := range 5 {
		if res := from(cl, "192.0.2.1"); res.Status != 200 {
			t.Fatalf("request %d answered %d, want 200", i+1, res.Status)
		}
	}
	res := from(cl, "192.0.2.1")
	e := res.Error()
	if res.Status != http.StatusTooManyRequests || e.Code != "rate_limited" || e.Message != "Too many requests; wait a moment and try again." {
		t.Fatalf("the 6th request answered %d %+v, want the 429", res.Status, e)
	}
	if got := res.Header.Get("Retry-After"); got != "12" {
		t.Errorf("Retry-After is %q, want 12", got)
	}
	c.add(11 * time.Second)
	if res := from(cl, "192.0.2.1"); res.Status != 429 || res.Header.Get("Retry-After") != "1" {
		t.Errorf("11 s later the answer is %d with Retry-After %q, want 429 with 1", res.Status, res.Header.Get("Retry-After"))
	}
	c.add(time.Second)
	if res := from(cl, "192.0.2.1"); res.Status != 200 {
		t.Errorf("12 s later the answer is %d, want 200", res.Status)
	}
	if res := from(cl, "192.0.2.1"); res.Status != 429 {
		t.Errorf("a second request 12 s later is %d, want 429: one came back, not two", res.Status)
	}
	c.add(time.Hour)
	for i := range 5 {
		if res := from(cl, "192.0.2.1"); res.Status != 200 {
			t.Fatalf("after an hour, request %d answered %d: the bucket holds 5 again", i+1, res.Status)
		}
	}
	if res := from(cl, "192.0.2.1"); res.Status != 429 {
		t.Errorf("after an hour the 6th answered %d: the bucket holds no more than 5", res.Status)
	}
}

func TestRateLimitPerClient(t *testing.T) {
	c := &clock{t: time.Now()}
	cl := limited(t, "2/s", c)
	from(cl, "192.0.2.1")
	from(cl, "192.0.2.1")
	if res := from(cl, "192.0.2.1"); res.Status != 429 {
		t.Fatalf("the third request answered %d", res.Status)
	}
	if res := from(cl, "192.0.2.2"); res.Status != 200 {
		t.Errorf("another client answered %d, want 200", res.Status)
	}
	// One client of IPv6 is its /64.
	from(cl, "2001:db8:1:2::1")
	from(cl, "2001:db8:1:2::2")
	if res := from(cl, "2001:db8:1:2::3"); res.Status != 429 {
		t.Errorf("a third address of one /64 answered %d, want 429", res.Status)
	}
}

func TestRateLimitNotOnPages(t *testing.T) {
	c := &clock{t: time.Now()}
	app := warp.New(warp.ConfigFrom(map[string]string{"LOG": "off", "RATE_LIMIT": "1/m"}))
	app.Use(newModule(c.now), xModule{})
	url := run(t, app, site)
	for i := range 100 {
		if resp := get(t, "GET", url+"/"); resp.StatusCode != 200 {
			t.Fatalf("page request %d answered %d", i+1, resp.StatusCode)
		}
	}
	if resp := get(t, "GET", url+"/api/x"); resp.StatusCode != 200 {
		t.Errorf("the first API request answered %d", resp.StatusCode)
	}
	if resp := get(t, "GET", url+"/api/x"); resp.StatusCode != 429 {
		t.Errorf("the second API request answered %d, want 429", resp.StatusCode)
	}
	if resp := get(t, "GET", url+"/robots.txt"); resp.StatusCode != 200 {
		t.Errorf("a file after the limit answered %d", resp.StatusCode)
	}
}

func TestRateLimitOff(t *testing.T) {
	c := &clock{t: time.Now()}
	cl := limited(t, "off", c)
	for i := range 1000 {
		if res := from(cl, "192.0.2.1"); res.Status != 200 {
			t.Fatalf("request %d answered %d with RATE_LIMIT=off", i+1, res.Status)
		}
	}
}

func TestRateLimitDefault(t *testing.T) {
	c := &clock{t: time.Now()}
	cl := limited(t, "", c)
	for i := range 300 {
		if res := from(cl, "192.0.2.1"); res.Status != 200 {
			t.Fatalf("request %d answered %d, want 300 a minute", i+1, res.Status)
		}
	}
	res := from(cl, "192.0.2.1")
	if res.Status != 429 || res.Header.Get("Retry-After") != "1" {
		t.Errorf("the 301st answered %d with Retry-After %q, want 429 with 1", res.Status, res.Header.Get("Retry-After"))
	}
}

func TestRateLimitCap(t *testing.T) {
	c := &clock{t: time.Now()}
	m := newModule(c.now)
	app := warp.New(warp.ConfigFrom(map[string]string{"LOG": "off", "RATE_LIMIT": "1/h"}))
	app.Use(m, xModule{})
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer app.Stop(context.Background())
	h := app.Handler()
	send := func(ip string) int {
		r := httptest.NewRequest("GET", "http://warptest.local/api/x", nil)
		r.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	ip := func(i int) string { return fmt.Sprintf("10.%d.%d.%d", i>>16&255, i>>8&255, i&255) }
	if send(ip(0)) != 200 || send(ip(0)) != 429 {
		t.Fatal("the first client is not limited")
	}
	for i := 1; i < maxClients; i++ {
		send(ip(i))
	}
	if send(ip(0)) != 429 {
		t.Fatal("the first client is forgotten at 50,000 clients, before the cap is passed")
	}
	if n := m.limit.size(); n != maxClients {
		t.Errorf("the limiter remembers %d clients, want %d", n, maxClients)
	}
	// The 50,001st client makes the one seen longest ago go. The first was
	// just seen, so the second goes.
	send(ip(maxClients))
	if n := m.limit.size(); n != maxClients {
		t.Errorf("past the cap the limiter remembers %d clients, want %d", n, maxClients)
	}
	if send(ip(1)) != 200 {
		t.Error("the client seen longest ago is still remembered past the cap")
	}
	if send(ip(0)) != 429 {
		t.Error("a client seen lately was forgotten in its place")
	}
}

func TestRateLimitMistakes(t *testing.T) {
	for _, v := range []string{"abc", "0/m", "1000001/s", "5/d", "5", "/m", "-1/m", "+5/m", "5.5/m", "5/m/x", "5/", "05/m", " / "} {
		app := warp.New(warp.ConfigFrom(map[string]string{"LOG": "off", "RATE_LIMIT": v}))
		app.Use(Module())
		err := app.Start(context.Background())
		want := fmt.Sprintf(`warp: module "middleware" could not start: RATE_LIMIT is %q; set it like 300/m (requests a minute), 20/s or 5000/h, or off.`, v)
		if err == nil || err.Error() != want {
			t.Errorf("RATE_LIMIT=%q: Start = %v, want %q", v, err, want)
		}
	}
	app := warp.New(warp.ConfigFrom(map[string]string{"LOG": "xml", "RATE_LIMIT": "x"}))
	app.Use(Module())
	want := `warp: module "middleware" could not start: LOG is "xml"; set it to text, json or off in .env.` + "\n" +
		`RATE_LIMIT is "x"; set it like 300/m (requests a minute), 20/s or 5000/h, or off.`
	if err := app.Start(context.Background()); err == nil || err.Error() != want {
		t.Errorf("two mistakes: Start = %v, want %q", err, want)
	}
	for _, v := range []string{"1/s", "1000000/s", "20/S", "5000/h", " 300/m ", "OFF"} {
		app := warp.New(warp.ConfigFrom(map[string]string{"LOG": "off", "RATE_LIMIT": v}))
		app.Use(Module())
		if err := app.Start(context.Background()); err != nil {
			t.Errorf("RATE_LIMIT=%q is refused: %v", v, err)
		}
		app.Stop(context.Background())
	}
	if r, err := readRate("300/m"); err != nil || r.n != 300 || r.interval != 200*time.Millisecond {
		t.Errorf("300/m reads as %+v, %v; want 300, one each 200ms", r, err)
	}
	if r, err := readRate(""); err != nil || r.n != 300 || r.interval != 200*time.Millisecond {
		t.Errorf("unset reads as %+v, %v; want 300/m", r, err)
	}
	if r, err := readRate("off"); err != nil || r != nil {
		t.Errorf("off reads as %+v, %v; want none", r, err)
	}
}
