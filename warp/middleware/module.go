// Package middleware is what a Warp app puts around its requests when the
// middleware module is listed: security headers on every answer, and one log
// line per request, and a rate limit per client on the API.
//
// It is configured in .env, read when the app starts: LOG is text (the
// default), json or off, the format of the request log and, once the app
// has started, of every line slog's default and the standard logger write;
// RATE_LIMIT is how many API requests a client may send, as 300/m (the
// default), 20/s or 5000/h, or off.
package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/flakerimi/weft/warp"
)

// Option changes the middleware module.
type Option func(*module)

// Module is the middleware module, named "middleware", which needs no other.
// Registering puts the security headers and then the request log around
// every request the app answers, its pages and files as well as its API, and
// the rate limit around the API alone, and reads nothing; its Start reads
// LOG and RATE_LIMIT. List it first in app/modules.go, so
// the lines the other modules write at their start are in its format too.
func Module(opts ...Option) warp.Module {
	m := newModule(time.Now)
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// newModule is a module whose rate limit reads the clock now.
func newModule(now func() time.Time) *module { return &module{out: os.Stderr, now: now} }

type module struct {
	app *warp.App
	out io.Writer        // where the log goes
	now func() time.Time // the rate limit's clock

	// Set by Start.
	requests *slog.Logger // nil while there is no request log
	previous *previous
	limit    *limiter // nil while there is no rate limit
}

func (m *module) Name() string { return "middleware" }

func (m *module) Register(app *warp.App) error {
	m.app = app
	app.WrapAll(m.headers)
	app.WrapAll(m.logRequests)
	app.Wrap(m.limitRequests)
	return nil
}

// Start reads LOG and RATE_LIMIT, a mistake in each its own sentence, and, unless LOG is off, makes the request
// log and slog's default that format, on the same writer.
func (m *module) Start(ctx context.Context) error {
	cfg := m.app.Config()
	format, errLog := readLog(cfg.Get("LOG"))
	r, errRate := readRate(cfg.Get("RATE_LIMIT"))
	if err := errors.Join(errLog, errRate); err != nil {
		return err
	}
	if r != nil {
		m.limit = newLimiter(m.now, *r)
	}
	if format != "off" {
		m.requests = logger(format, m.out)
		m.previous = takeDefault(m.requests)
	}
	return nil
}

// Stop puts slog's default and the standard logger back as Start found them.
func (m *module) Stop(ctx context.Context) error {
	m.previous.restore()
	return nil
}
