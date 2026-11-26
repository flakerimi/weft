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

// Route describes one registered route. Module is the name of the module
// whose Register added it, empty for a route added outside one. Name is the
// one Name gave, or one made from the method and path. Path has /api in
// front; In and Out are nil for a route registered with Handle. Permission
// is the one Permission recorded.
type Route struct {
	Module, Name, Method, Path string
	In, Out                    reflect.Type
	Status                     int
	Permission                 string
}

// RouteOption changes one route.
type RouteOption func(*routeConfig)

type routeConfig struct {
	status     int
	statusSet  bool // Status was given
	name       string
	nameSet    bool // Name was given
	permission string
	wraps      []func(http.Handler) http.Handler
}

// Name names the route within its module, in place of the name made from its
// method and path. Two routes of one module with one name are a mistake,
// reported by Err; routes added outside a module are not checked.
func Name(name string) RouteOption {
	return func(c *routeConfig) { c.name, c.nameSet = name, true }
}

// Permission records the permission the route needs, for Routes to report.
// It does not guard the route: the module that enforces permissions supplies
// an option that does, built with Options.
func Permission(permission string) RouteOption {
	return func(c *routeConfig) { c.permission = permission }
}

// Options makes one route option out of several, applied in the order given,
// so a package can write an option that, say, records a permission and
// guards the route with With.
func Options(opts ...RouteOption) RouteOption {
	return func(c *routeConfig) {
		for _, opt := range opts {
			opt(c)
		}
	}
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
	xo  *http.CrossOriginProtection // shared by every request

	mu         sync.Mutex
	modules    []Module
	names      map[string]bool
	current    string               // the module whose Register is running
	routeNames map[string]routeName // a module's name and a route's name to the route
	routes     []Route
	plans      []*plan // each route's binding plan, beside routes; nil for Handle
	wraps      []func(http.Handler) http.Handler
	provided   map[reflect.Type]any
	badConfig  []error // the configuration's problems New found, beside cfg.Err
	mistakes   []error // the registration mistakes
	started    bool
	running    []Module // the modules Start started, which Stop stops
}

// New returns an app configured by cfg, with no modules. A mistake in cfg is
// reported by Err, and by Run.
func New(cfg Config) *App {
	xo, bad := cfg.xo, []string(nil)
	if xo == nil || !slices.Equal(cfg.TrustedOrigins, cfg.xoFor) {
		xo, bad = newOriginProtection(cfg.TrustedOrigins) // a Config made by hand, or changed since
	}
	a := &App{
		cfg:        cfg.withDefaults(),
		mux:        http.NewServeMux(),
		xo:         xo,
		names:      map[string]bool{},
		routeNames: map[string]routeName{},
		provided:   map[reflect.Type]any{},
	}
	for _, o := range bad {
		a.badConfig = append(a.badConfig, fmt.Errorf("warp: TrustedOrigins has %q; write each origin like https://admin.example.com.", o))
	}
	return a
}

// Config returns the app's configuration, with its defaults filled in.
func (a *App) Config() Config { return a.cfg }

// Use registers each module at once, in order. A module's Register may call
// Use itself; while the inner module registers, its routes are its own. A
// mistake is reported by Err.
func (a *App) Use(modules ...Module) {
	for _, m := range modules {
		name := m.Name()
		a.mu.Lock()
		if a.started {
			a.mu.Unlock()
			a.fail("Use was called after the app started; list every module before Serve or Start.")
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
		if err := a.register(m); err != nil {
			a.fail("module %q could not register: %w", name, err)
		}
	}
}

// routeName is the route a module's route name was given to, and whether it
// was given with Name.
type routeName struct {
	route string
	named bool
}

// Module returns the name of the module whose Register is running, or "" when
// none is: the module a route added now belongs to. A package that registers
// routes for a module, as data.Resource does, reads it to name them.
func (a *App) Module() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.current
}

// register runs m's Register with m as the current module.
func (a *App) register(m Module) error {
	a.mu.Lock()
	outer := a.current
	a.current = m.Name()
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.current = outer
		a.mu.Unlock()
	}()
	return m.Register(a)
}

// Provide stores value as the one value of its Go type T, for a later module
// to get with Need. A module calls it from its Register. A second value of
// one type, or a Provide after the app started, is a mistake, reported by
// Err.
func Provide[T any](app *App, value T) {
	t := reflect.TypeFor[T]()
	app.mu.Lock()
	_, taken := app.provided[t]
	started := app.started
	if !started && !taken {
		app.provided[t] = value
	}
	app.mu.Unlock()
	switch {
	case started:
		app.fail("Provide was called after the app started; provide every value in a module's Register.")
	case taken:
		app.fail("two modules provide a %s; list only one of them.", t)
	}
}

// Need returns the value of type T that the module named module provided.
// When there is none it returns the zero T and records a mistake, reported
// by Err, that says to list that module first; after the app started it is
// also logged. A module wraps it in a typed function of its own, such as
// data.For(app).
func Need[T any](app *App, module string) T {
	t := reflect.TypeFor[T]()
	app.mu.Lock()
	v, ok := app.provided[t]
	current := app.current
	app.mu.Unlock()
	if !ok {
		if current != "" {
			app.fail("module %q needs the %q module; list %s.Module() before it in app/modules.go.", current, module, module)
		} else {
			app.fail("the app needs the %q module; list %s.Module() in app/modules.go.", module, module)
		}
		var zero T
		return zero
	}
	value, _ := v.(T) // a nil interface value is stored as nil
	return value
}

// Handle registers h at pattern, which is a method, a space and a path
// without /api in front ("GET /export.csv"), or a path alone for every
// method. A mistake is reported by Err.
func (a *App) Handle(pattern string, h http.Handler, opts ...RouteOption) {
	method, path := "", pattern
	if i := strings.IndexAny(pattern, " \t"); i >= 0 {
		method, path = pattern[:i], strings.TrimLeft(pattern[i:], " \t")
	}
	a.route(method, path, nil, nil, nil, 0, func(int) http.Handler { return h }, opts, nil)
}

// route registers the handler mk makes at /api plus path, for method or, when
// method is empty, for every method. Handle and the typed routes register
// through it; p is a typed route's binding plan, kept for Describe, and nil
// for Handle. mk is given the status the route answers with: status, unless
// a Status option changes it, and 204 when out is None. mistakes are the
// ones the caller found in the route's input; each is reported, after the
// route's own, and any mistake leaves the route unregistered. A route
// registered after the app started reports that alone.
func (a *App) route(method, path string, in, out reflect.Type, p *plan, status int, mk func(status int) http.Handler, opts []RouteOption, mistakes []string) {
	name := strings.TrimSpace(method + " " + path)
	a.mu.Lock()
	started := a.started
	a.mu.Unlock()
	if started {
		a.fail("route %q was registered after the app started; register every route before Serve or Start.", name)
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
	if rc.nameSet && rc.name == "" {
		a.fail(`Name("") is empty; give the route a name.`)
		ok = false
	}
	for _, m := range mistakes {
		a.fail("%s: %s", name, m)
		ok = false
	}
	if !ok {
		return
	}
	if !rc.nameSet {
		rc.name = defaultName(method, path)
	}
	// Names are unique within a module. The same pattern twice is left to the
	// mux, whose reason says more.
	a.mu.Lock()
	module := a.current
	key := module + "\x00" + rc.name
	other, clash := a.routeNames[key]
	a.mu.Unlock()
	if module != "" && clash && other.route != name {
		if other.named && rc.nameSet {
			a.fail("module %q has two routes named %q (%s and %s); give them different names.", module, rc.name, other.route, name)
		} else {
			a.fail("module %q has two routes named %q (%s and %s); give one a warp.Name.", module, rc.name, other.route, name)
		}
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
	if module != "" {
		a.routeNames[key] = routeName{name, rc.nameSet}
	}
	a.routes = append(a.routes, Route{
		Module: module, Name: rc.name, Method: method, Path: "/api" + path,
		In: in, Out: out, Status: rc.status, Permission: rc.permission,
	})
	a.plans = append(a.plans, p)
	a.mu.Unlock()
}

// defaultName is the name of a route given no Name: the static segments of
// its path joined by _, in lower case with - turned to _, and the method and
// _ in front unless the method is GET or empty. Wildcards are left out; a path
// of wildcards alone is index under GET, and the method alone otherwise.
func defaultName(method, path string) string {
	var parts []string
	for seg := range strings.SplitSeq(path, "/") {
		if seg == "" || strings.HasPrefix(seg, "{") {
			continue
		}
		parts = append(parts, strings.ReplaceAll(strings.ToLower(seg), "-", "_"))
	}
	if method != "" && method != http.MethodGet {
		parts = append([]string{strings.ToLower(method)}, parts...)
	}
	if len(parts) == 0 {
		return "index"
	}
	return strings.Join(parts, "_")
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
		a.fail("Wrap was called after the app started; wrap before Serve or Start.")
		return
	}
	defer a.mu.Unlock()
	a.wraps = append(a.wraps, mw)
}

// Mistake records a registration mistake found by a package outside the core,
// such as a module that checks its own types: one sentence that names the
// thing and says what to do. It is reported with the others by Err, Start and
// Serve, with "warp: " in front. A mistake recorded after the app started is
// also logged, since nothing reads Err then.
func (a *App) Mistake(format string, args ...any) { a.fail(format, args...) }

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
	all := append([]error{a.cfg.Err()}, a.badConfig...)
	return errors.Join(append(all, a.mistakes...)...)
}

// registrationErr returns the registration mistakes alone, joined, leaving
// out the configuration's problems; nil when there is none.
func (a *App) registrationErr() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return errors.Join(a.mistakes...)
}

// Routes returns every route, in the order registered.
func (a *App) Routes() []Route {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.routes)
}

// Handler returns the API alone, at /api/..., inside every Wrap given so far.
// A state-changing request from another site is a 403 before any Wrap runs.
// A path no route has is a 404 and a method a route does not take is a 405,
// both in the error shape, and so is the redirect of a path that is not clean
// (/api/a//b, a {rest...} route asked without its tail) to the clean one,
// with its status and Location. A panic is logged and answered as a 500.
// Every answer carries Cache-Control: no-store, so no cache keeps what a
// session sees, unless the handler sets a Cache-Control of its own.
func (a *App) Handler() http.Handler {
	a.mu.Lock()
	wraps := slices.Clone(a.wraps)
	a.mu.Unlock()
	var h http.Handler = http.HandlerFunc(a.dispatch)
	for i := len(wraps) - 1; i >= 0; i-- {
		h = wraps[i](h)
	}
	return noStore(recoverPanics(refuseCrossSite(a.xo, h)))
}

// noStore sets Cache-Control: no-store on the answer before h runs, which h
// may replace.
func noStore(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", "no-store")
		}
		h.ServeHTTP(w, r)
	})
}

// dispatch serves a request through the mux, answering its own 404 and 405
// in the error shape.
func (a *App) dispatch(w http.ResponseWriter, r *http.Request) {
	h, pattern := a.mux.Handler(r)
	if pattern != "" && !muxRedirect(h) {
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
	case http.StatusMovedPermanently, http.StatusTemporaryRedirect, http.StatusPermanentRedirect, http.StatusFound:
		// The mux's redirect to the clean path, with its body in the error
		// shape.
		to := p.header.Get("Location")
		w.Header().Set("Location", to)
		WriteJSON(w, p.status, errorBody{&Error{Status: p.status, Code: "moved", Message: fmt.Sprintf("This is at %s; ask there.", to)}})
	default:
		h.ServeHTTP(w, r)
	}
}

// redirectType is the type of the handler http.ServeMux returns to send a
// path that is not clean, or a {rest...} or folder route asked without its
// last slash, to the one it serves: http.RedirectHandler's.
var redirectType = reflect.TypeOf(http.RedirectHandler("/", http.StatusMovedPermanently))

// muxRedirect reports whether h is that redirect, which runs no route, so
// dispatch may run it on a probe and answer it in the error shape.
func muxRedirect(h http.Handler) bool { return reflect.TypeOf(h) == redirectType }

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
