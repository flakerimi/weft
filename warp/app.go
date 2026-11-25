package warp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
)

// Module is one part of an app: it registers its routes on the app it is
// given. Modules are listed in the app's modules.go, never registered from an
// init function.
type Module interface {
	Name() string
	Register(app *App) error
}

// Migrator is a module that brings its storage up to date before the app
// starts.
type Migrator interface {
	Migrate(ctx context.Context) error
}

// Starter is a module with work to start before the app serves.
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper is a module with work to stop after the app has stopped serving.
type Stopper interface {
	Stop(ctx context.Context) error
}

// Route describes one registered route. Path has /api in front; In and Out
// are nil for a route registered with Handle.
type Route struct {
	Method, Path string
	In, Out      reflect.Type
	Status       int
}

// RouteOption changes one route.
type RouteOption func(*routeConfig)

type routeConfig struct {
	status    int
	statusSet bool // Status was given
	wraps     []func(http.Handler) http.Handler
}

// With puts mw around this route alone, inside every Wrap. The first With
// given is the outermost.
func With(mw func(http.Handler) http.Handler) RouteOption {
	return func(c *routeConfig) { c.wraps = append(c.wraps, mw) }
}

// Status sets the status a typed route answers with. A route registered with
// Handle answers as its handler does; Routes reports the status given. A code
// outside 200 to 599 is a mistake, reported by Err: a 1xx is not an answer.
func Status(code int) RouteOption {
	return func(c *routeConfig) { c.status, c.statusSet = code, true }
}

// App is a Weft app: its configuration, its modules and its API, which it
// serves under /api.
type App struct {
	cfg Config
	mux *http.ServeMux

	mu       sync.Mutex
	modules  []Module
	names    map[string]bool
	routes   []Route
	wraps    []func(http.Handler) http.Handler
	mistakes []error
	started  bool
}

// New returns an app configured by cfg, with no modules. A mistake in cfg is
// reported by Err, and by Run.
func New(cfg Config) *App {
	return &App{cfg: cfg.withDefaults(), mux: http.NewServeMux(), names: map[string]bool{}}
}

// Config returns the app's configuration, with its defaults filled in.
func (a *App) Config() Config { return a.cfg }

// Use registers each module at once, in order. A mistake is reported by Err.
func (a *App) Use(modules ...Module) {
	for _, m := range modules {
		name := m.Name()
		a.mu.Lock()
		if a.started {
			a.mu.Unlock()
			a.fail("Use was called after the app started serving; list every module before Serve.")
			return
		}
		if a.names[name] {
			a.mu.Unlock()
			a.fail("two modules are named %q; give one another name.", name)
			continue
		}
		a.names[name] = true
		a.modules = append(a.modules, m)
		a.mu.Unlock()
		if err := m.Register(a); err != nil {
			a.fail("module %q could not register: %w", name, err)
		}
	}
}

// Handle registers h at pattern, which is a method, a space and a path
// without /api in front ("GET /export.csv"), or a path alone for every
// method. A mistake is reported by Err.
func (a *App) Handle(pattern string, h http.Handler, opts ...RouteOption) {
	method, path := "", pattern
	if i := strings.IndexAny(pattern, " \t"); i >= 0 {
		method, path = pattern[:i], strings.TrimLeft(pattern[i:], " \t")
	}
	a.route(method, path, nil, nil, 0, func(int) http.Handler { return h }, opts, nil)
}

// route registers the handler mk makes at /api plus path, for method or, when
// method is empty, for every method. Handle and the typed routes register
// through it. mk is given the status the route answers with: status, unless a
// Status option changes it, and 204 when out is None. mistakes are the ones
// the caller found in the route's input; each is reported, after the route's
// own, and any mistake leaves the route unregistered. A route registered
// after the app started reports that alone.
func (a *App) route(method, path string, in, out reflect.Type, status int, mk func(status int) http.Handler, opts []RouteOption, mistakes []string) {
	name := strings.TrimSpace(method + " " + path)
	a.mu.Lock()
	started := a.started
	a.mu.Unlock()
	if started {
		a.fail("route %q was registered after the app started serving; register every route before Serve.", name)
		return
	}
	ok := true
	switch {
	case !strings.HasPrefix(path, "/"):
		a.fail("route %q has a path that does not start with /; write it as %q.", name, strings.TrimSpace(method+" /"+path))
		ok = false
	case strings.HasPrefix(path, "/api/"):
		a.fail("route %q already gets /api in front; write it as %q.", path, strings.TrimPrefix(path, "/api"))
		ok = false
	}
	rc := routeConfig{status: status}
	for _, opt := range opts {
		opt(&rc)
	}
	if rc.statusSet && (rc.status < 200 || rc.status > 599) {
		a.fail("%s: Status(%d) is not a status a route can answer with; use one from 200 to 599.", name, rc.status)
		ok = false
	}
	for _, m := range mistakes {
		a.fail("%s: %s", name, m)
		ok = false
	}
	if !ok {
		return
	}
	if out == noneType {
		rc.status = http.StatusNoContent
	}
	h := mk(rc.status)
	for i := len(rc.wraps) - 1; i >= 0; i-- {
		h = rc.wraps[i](h)
	}
	pattern := strings.TrimSpace(method + " /api" + path)
	if err := register(a.mux, pattern, h); err != nil {
		a.fail("route %q cannot be registered: %v", name, err)
		return
	}
	a.mu.Lock()
	a.routes = append(a.routes, Route{Method: method, Path: "/api" + path, In: in, Out: out, Status: rc.status})
	a.mu.Unlock()
}

// registeredAt is where the mux says a pattern was registered, which is
// always this file, so it is left out of the mux's reason.
var registeredAt = regexp.MustCompile(` \(registered at [^)]*\)`)

// register adds pattern to mux, turning the mux's panic into an error of one
// line.
func register(mux *http.ServeMux, pattern string, h http.Handler) (err error) {
	defer func() {
		if v := recover(); v != nil {
			reason := registeredAt.ReplaceAllString(fmt.Sprint(v), "")
			err = errors.New(strings.Join(strings.Fields(reason), " "))
		}
	}()
	mux.Handle(pattern, h)
	return nil
}

// Wrap puts mw around every API request. The first Wrap given is the
// outermost. A Wrap after the app started is a mistake, reported by Err.
func (a *App) Wrap(mw func(http.Handler) http.Handler) {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		a.fail("Wrap was called after the app started serving; wrap before Serve.")
		return
	}
	defer a.mu.Unlock()
	a.wraps = append(a.wraps, mw)
}

// fail records a registration mistake, with warp: in front. One recorded
// after the app started is logged too, since Run has checked Err by then.
func (a *App) fail(format string, args ...any) {
	err := fmt.Errorf("warp: "+format, args...)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mistakes = append(a.mistakes, err)
	if a.started {
		slog.Error(err.Error())
	}
}

// Err returns every mistake so far, the configuration's first, joined; nil
// when there is none.
func (a *App) Err() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return errors.Join(append([]error{a.cfg.Err()}, a.mistakes...)...)
}

// Routes returns every route, in the order registered.
func (a *App) Routes() []Route {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.routes)
}

// Handler returns the API alone, at /api/..., inside every Wrap given so far.
// A path no route has is a 404 and a method a route does not take is a 405,
// both in the error shape; a panic is logged and answered as a 500.
func (a *App) Handler() http.Handler {
	a.mu.Lock()
	wraps := slices.Clone(a.wraps)
	a.mu.Unlock()
	var h http.Handler = http.HandlerFunc(a.dispatch)
	for i := len(wraps) - 1; i >= 0; i-- {
		h = wraps[i](h)
	}
	return recoverPanics(h)
}

// dispatch serves a request through the mux, answering its own 404 and 405
// in the error shape.
func (a *App) dispatch(w http.ResponseWriter, r *http.Request) {
	h, pattern := a.mux.Handler(r)
	if pattern != "" {
		a.mux.ServeHTTP(w, r)
		return
	}
	p := &probe{header: http.Header{}}
	h.ServeHTTP(p, r)
	switch p.status {
	case http.StatusMethodNotAllowed:
		allow := p.header.Get("Allow")
		w.Header().Set("Allow", allow)
		WriteError(w, r, &Error{
			Status:  http.StatusMethodNotAllowed,
			Code:    "method_not_allowed",
			Message: fmt.Sprintf("%s does not take %s; use %s.", r.URL.Path, r.Method, allowedList(allow)),
		})
	case http.StatusNotFound:
		WriteError(w, r, NotFound(fmt.Sprintf("Nothing is at %s.", r.URL.Path)))
	default:
		h.ServeHTTP(w, r) // a redirect to the clean path
	}
}

// allowedList writes the mux's Allow header as words: "GET, HEAD, POST" is
// "GET or POST". HEAD and OPTIONS are left out when another method is there.
func allowedList(allow string) string {
	var all, named []string
	for m := range strings.SplitSeq(allow, ",") {
		if m = strings.TrimSpace(m); m == "" {
			continue
		}
		all = append(all, m)
		if m != http.MethodHead && m != http.MethodOptions {
			named = append(named, m)
		}
	}
	if len(named) == 0 {
		named = all
	}
	if len(named) < 2 {
		return strings.Join(named, "")
	}
	return strings.Join(named[:len(named)-1], ", ") + " or " + named[len(named)-1]
}

// probe is a ResponseWriter that keeps only the status and the header, to
// read what the mux would answer.
type probe struct {
	header http.Header
	status int
}

func (p *probe) Header() http.Header { return p.header }

func (p *probe) WriteHeader(code int) {
	if p.status == 0 {
		p.status = code
	}
}

func (p *probe) Write(b []byte) (int, error) {
	p.WriteHeader(http.StatusOK)
	return len(b), nil
}

// recoverPanics answers a panic in h as a 500 and logs it with its stack.
func recoverPanics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			// After the handler wrote, the 500 goes after what it sent, and the body is broken.
			WriteError(w, r, fmt.Errorf("panic: %v\n%s", v, debug.Stack()))
		}()
		h.ServeHTTP(w, r)
	})
}
