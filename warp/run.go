package warp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"
)

// stopLimit is how long the server has to finish the requests in flight, and
// the modules to stop, once the app is told to stop.
const stopLimit = 10 * time.Second

// Serve listens on the configured host and port and runs the app until
// SIGINT or SIGTERM. dist is the page build, or nil to serve the API alone.
//
// Under weft dev (WEFT_DEV_PARENT=1) it logs nothing, and it also stops when
// standard input closes, which is when weft dev is gone.
func (a *App) Serve(dist fs.FS, opts ...Option) error {
	if err := a.Err(); err != nil {
		return err
	}
	addr := net.JoinHostPort(a.cfg.Host, a.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if errors.Is(err, syscall.EADDRINUSE) {
		return fmt.Errorf("warp: port %s is taken; stop what uses it or set PORT in .env.", a.cfg.Port)
	}
	if err != nil {
		return fmt.Errorf("warp: cannot listen on %s (%w); set HOST and PORT in .env.", addr, err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if os.Getenv("WEFT_DEV_PARENT") == "1" {
		var cancel context.CancelFunc
		ctx, cancel = untilEOF(ctx, os.Stdin)
		defer cancel()
	} else {
		log.Printf("listening on http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)
	}
	return a.Run(ctx, ln, dist, opts...)
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

// Run runs the app on ln until ctx ends: it migrates every Migrator and
// starts every Starter, in the order the modules were listed, then serves
// dist and the API (the API alone when dist is nil). When ctx ends it lets
// the requests in flight finish, for up to 10 seconds, and stops every
// Stopper in reverse order. When a step fails, the modules already started
// are stopped. Run closes ln, and returns nil after a clean stop.
func (a *App) Run(ctx context.Context, ln net.Listener, dist fs.FS, opts ...Option) error {
	if err := a.Err(); err != nil {
		if ln != nil {
			ln.Close()
		}
		return err
	}
	a.mu.Lock()
	a.started = true
	modules := slices.Clone(a.modules)
	a.mu.Unlock()

	h := a.Handler()
	if dist != nil {
		h = Serve(dist, h, opts...)
	}

	for _, m := range modules {
		if mg, ok := m.(Migrator); ok {
			if err := mg.Migrate(ctx); err != nil {
				ln.Close()
				return fmt.Errorf("warp: module %q could not migrate: %w", m.Name(), err)
			}
		}
	}
	for i, m := range modules {
		if s, ok := m.(Starter); ok {
			if err := s.Start(ctx); err != nil {
				ln.Close()
				err = fmt.Errorf("warp: module %q could not start: %w", m.Name(), err)
				return errors.Join(append([]error{err}, stopAll(modules[:i])...)...)
			}
		}
	}

	srv := &http.Server{Handler: h}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	var errs []error
	select {
	case err := <-served:
		errs = append(errs, fmt.Errorf("warp: the server stopped serving: %w", err))
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), stopLimit)
		err := srv.Shutdown(shutdown)
		cancel()
		if err != nil {
			srv.Close()
			errs = append(errs, errors.New("warp: requests were still running 10 s after the app was told to stop, so they were cut off; make long handlers watch their request's context."))
		}
		<-served
	}
	return errors.Join(append(errs, stopAll(modules)...)...)
}

// stopAll stops every Stopper in modules, last first, and returns what
// failed.
func stopAll(modules []Module) []error {
	ctx, cancel := context.WithTimeout(context.Background(), stopLimit)
	defer cancel()
	var errs []error
	for _, m := range slices.Backward(modules) {
		if s, ok := m.(Stopper); ok {
			if err := s.Stop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("warp: module %q could not stop: %w", m.Name(), err))
			}
		}
	}
	return errs
}
