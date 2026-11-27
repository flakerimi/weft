package warp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"syscall"
	"time"
)

// stopLimit is how long the server has to finish the requests in flight, and
// the modules to stop, once the app is told to stop.
const stopLimit = 10 * time.Second

// Serve starts the app, as Start does, then listens on the configured host
// and port, logs listening on http://localhost:<port>, and runs the app until
// SIGINT or SIGTERM, as Run does. dist is the page build, or nil to serve the
// API alone. Nothing listens while the modules migrate and start; a port that
// is taken stops the modules it started.
//
// A dist that is only the empty page build marker weft create writes into
// web/dist (the pages not built) is refused with a sentence, before anything
// starts.
//
// Under weft dev (WEFT_DEV_PARENT=1) it logs nothing, and it also stops when
// standard input closes, which is when weft dev is gone.
//
// With WARP_DESCRIBE=1 in the environment, as weft gen runs it, it writes the
// app's Describe to standard output as indented JSON and returns nil, without
// migrating, starting or listening. It then returns the registration
// mistakes, and Describe's, and leaves out the configuration's problems: a
// description needs routes and types, not a working .env.
func (a *App) Serve(dist fs.FS, opts ...ServeOption) error {
	if os.Getenv("WARP_DESCRIBE") == "1" {
		return a.describeTo(describeOut)
	}
	if err := a.Err(); err != nil {
		return err
	}
	if dist != nil && unbuilt(dist) {
		return errUnbuilt
	}
	ctx, stop := notifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Once the first signal ends ctx, the signals are no longer taken, so a
	// second Ctrl-C during the drain ends the process the default way.
	context.AfterFunc(ctx, stop)
	if os.Getenv("WEFT_DEV_PARENT") == "1" {
		var cancel context.CancelFunc
		ctx, cancel = untilEOF(ctx, os.Stdin)
		defer cancel()
	}
	// The app starts before it listens, so nothing answers on the port while
	// the modules migrate, and a start that fails never says it listens.
	if err := a.Start(ctx); err != nil {
		// Stopped while the modules started, such as while one opened the
		// database: asked to stop, it is a clean exit, not a failed start;
		// but a module Start undid that could not stop is still said. Start
		// joins its cause first, then the undo's errors.
		if ctx.Err() != nil {
			if j, ok := err.(interface{ Unwrap() []error }); ok && len(j.Unwrap()) > 1 {
				return errors.Join(j.Unwrap()[1:]...)
			}
			return nil
		}
		return err
	}
	addr := net.JoinHostPort(a.cfg.Host, a.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if addrInUse(err) {
			err = fmt.Errorf("warp: port %s is taken; stop what uses it or set PORT in .env.", a.cfg.Port)
		} else {
			err = fmt.Errorf("warp: cannot listen on %s (%w); set HOST and PORT in .env.", addr, err)
		}
		return errors.Join(append([]error{err}, a.stop(context.WithoutCancel(ctx))...)...)
	}
	if os.Getenv("WEFT_DEV_PARENT") != "1" {
		log.Printf("listening on http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)
	}
	return a.serve(ctx, ln, dist, opts...)
}

// addrInUse says whether err is a listen on an address in use: EADDRINUSE,
// or on Windows WSAEADDRINUSE (10048), which syscall does not take for it.
func addrInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE) || runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(10048))
}

// notifyContext is signal.NotifyContext; a test puts another in its place.
var notifyContext = signal.NotifyContext

// describeOut is where Serve writes the description; a test puts another in
// its place.
var describeOut io.Writer = os.Stdout

// exit ends the process; a test puts another in its place.
var exit = os.Exit

// describeTo writes the description to w as indented JSON, unless the app has
// a registration mistake or Describe one.
func (a *App) describeTo(w io.Writer) error {
	if err := a.registrationErr(); err != nil {
		return err
	}
	d, err := a.Describe()
	if err != nil {
		return err
	}
	// Encoded whole before it is written, so a failure prints nothing; < and
	// > are left as they are, for Record<string, T> to be read as written.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return fmt.Errorf("warp: the description cannot be written as JSON (%w); report this as a bug in warp.", err)
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("warp: the description cannot be written (%w); check that standard output is open.", err)
	}
	return nil
}

// newServer is the server Run serves h with. A client gets 10 seconds to
// send a request's headers.
func newServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
}

// untilEOF returns a context that also ends when r reaches its end.
func untilEOF(parent context.Context, r io.Reader) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		io.Copy(io.Discard, r)
		cancel()
	}()
	return ctx, cancel
}

// Run runs the app on ln until ctx ends: it starts the app, as Start does,
// then serves dist and the API (the API alone when dist is nil). When ctx
// ends it lets the requests in flight finish, for up to 10 seconds, and
// stops the app, as Stop does. Run closes ln, and returns nil after a clean
// stop.
//
// With WARP_DESCRIBE=1 in the environment it closes ln and does what Serve
// does then: it writes the description and returns, starting nothing.
func (a *App) Run(ctx context.Context, ln net.Listener, dist fs.FS, opts ...ServeOption) error {
	if os.Getenv("WARP_DESCRIBE") == "1" {
		if ln != nil {
			ln.Close()
		}
		return a.describeTo(describeOut)
	}
	if ln == nil {
		return errors.New("warp: Run needs a listener; call Serve, or pass one from net.Listen.")
	}
	if err := a.Start(ctx); err != nil {
		ln.Close()
		return err
	}
	return a.serve(ctx, ln, dist, opts...)
}

// serve is Run once the app has started.
func (a *App) serve(ctx context.Context, ln net.Listener, dist fs.FS, opts ...ServeOption) error {
	h := a.Handler()
	if dist != nil {
		h = serveBuild(dist, h, opts...)
	}

	srv := newServer(h)
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	var errs []error
	select {
	case err := <-served:
		errs = append(errs, fmt.Errorf("warp: the server stopped serving (%w); start the app again.", err))
		errs = append(errs, drain(srv)...)
	case <-ctx.Done():
		errs = append(errs, drain(srv)...)
		<-served
	}
	return errors.Join(append(errs, a.stop(context.WithoutCancel(ctx))...)...)
}

// Start starts the app without listening: it migrates every Migrator, then
// starts every Starter, in the order the modules were listed. When a module
// fails to migrate or to start, every module that already migrated or
// started is stopped, last first, so that what a Migrate opened (data's
// database) is closed. It returns the registration mistakes first, as Err
// does, and may be called once; Run calls it. A test that serves Handler
// calls Start and Stop around it.
//
// With WARP_DESCRIBE=1 in the environment it starts nothing: it writes the
// description, as Serve does then, and ends the process, with status 0, or
// 1 after printing a registration mistake.
func (a *App) Start(ctx context.Context) error {
	// A describing run starts nothing, whatever the app does next: an app
	// that starts its App itself before it serves would otherwise migrate
	// the database in DB_URL, so the description is written here and the
	// process ends, as the package-level Serve ends it.
	if os.Getenv("WARP_DESCRIBE") == "1" {
		if err := a.describeTo(describeOut); err != nil {
			fmt.Fprintln(describeErr, err)
			exit(1)
		}
		exit(0)
	}
	if err := a.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return errors.New("warp: the app was already started.")
	}
	a.started = true
	modules := slices.Clone(a.modules)
	a.mu.Unlock()

	// done[i] is whether modules[i] migrated or started, and so is stopped
	// when a later one fails.
	done := make([]bool, len(modules))
	failed := func(err error) error {
		var undo []Module
		for i, m := range modules {
			if done[i] {
				undo = append(undo, m)
			}
		}
		return errors.Join(append([]error{err}, stopAll(context.WithoutCancel(ctx), undo)...)...)
	}
	for i, m := range modules {
		if mg, ok := m.(Migrator); ok {
			if err := mg.Migrate(ctx); err != nil {
				return failed(fmt.Errorf("warp: module %q could not migrate: %w", m.Name(), err))
			}
			done[i] = true
		}
	}
	for i, m := range modules {
		if s, ok := m.(Starter); ok {
			if err := s.Start(ctx); err != nil {
				return failed(fmt.Errorf("warp: module %q could not start: %w", m.Name(), err))
			}
			done[i] = true
		}
	}
	a.mu.Lock()
	a.running = modules
	a.mu.Unlock()
	return nil
}

// Stop stops every Stopper that Start started, last first, giving them up
// to 10 seconds, and returns what failed. It is for an app started with
// Start; Run stops its own. After a Start that failed, or a second time, it
// does nothing. A Stop called while Start is still running is not supported:
// it may find nothing started yet and return, leaving Start's modules
// running.
func (a *App) Stop(ctx context.Context) error {
	return errors.Join(a.stop(ctx)...)
}

func (a *App) stop(ctx context.Context) []error {
	a.mu.Lock()
	modules := a.running
	a.running = nil
	a.mu.Unlock()
	return stopAll(ctx, modules)
}

// drain shuts srv down, letting the requests in flight finish for up to 10
// seconds before it cuts them off.
func drain(srv *http.Server) []error {
	ctx, cancel := context.WithTimeout(context.Background(), stopLimit)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		srv.Close()
		return []error{errors.New("warp: requests were still running 10 s after the app was told to stop, so they were cut off; make long handlers watch their request's context.")}
	}
	return nil
}

// stopAll stops every Stopper in modules, last first, within 10 seconds of
// ctx, and returns what failed. A Stop that has not returned when the time
// is up is left running, and said so; each module after it still gets a
// second to stop, with the context that has ended.
func stopAll(ctx context.Context, modules []Module) []error {
	ctx, cancel := context.WithTimeout(ctx, stopLimit)
	defer cancel()
	deadline, _ := ctx.Deadline()
	var errs []error
	for _, m := range slices.Backward(modules) {
		s, ok := m.(Stopper)
		if !ok {
			continue
		}
		done := make(chan error, 1)
		go func() { done <- s.Stop(ctx) }()
		limit := time.NewTimer(max(time.Until(deadline), time.Second))
		select {
		case err := <-done:
			if err != nil {
				errs = append(errs, fmt.Errorf("warp: module %q could not stop: %w", m.Name(), err))
			}
		case <-limit.C:
			errs = append(errs, fmt.Errorf("warp: module %q did not stop before the app's time to stop ran out, so the app went on without it; make its Stop return when its context ends.", m.Name()))
		}
		limit.Stop()
	}
	return errs
}
