// Package middleware is what a Warp app puts around its requests when the
// middleware module is listed: security headers on every answer, and one log
// line per request.
//
// It is configured in .env, read when the app starts: LOG is text (the
// default), json or off, the format of the request log and, once the app
// has started, of every line slog's default and the standard logger write.
package middleware

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/flakerimi/weft/warp"
)

// Option changes the middleware module.
type Option func(*module)

// Module is the middleware module, named "middleware", which needs no other.
// Registering puts the security headers and then the request log around
// every request the app answers, its pages and files as well as its API, and
// reads nothing; its Start reads LOG. List it first in app/modules.go, so
// the lines the other modules write at their start are in its format too.
func Module(opts ...Option) warp.Module {
	m := &module{out: os.Stderr}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type module struct {
	app *warp.App
	out io.Writer // where the log goes

	// Set by Start.
	requests *slog.Logger // nil while there is no request log
	previous *previous
}

func (m *module) Name() string { return "middleware" }

func (m *module) Register(app *warp.App) error {
	m.app = app
	app.WrapAll(m.headers)
	app.WrapAll(m.logRequests)
	return nil
}

// Start reads LOG and, unless it is off, makes the request log and slog's
// default that format, on the same writer.
func (m *module) Start(ctx context.Context) error {
	format, err := readLog(m.app.Config().Get("LOG"))
	if err != nil {
		return err
	}
	if format == "off" {
		return nil
	}
	m.requests = logger(format, m.out)
	m.previous = takeDefault(m.requests)
	return nil
}

// Stop puts slog's default and the standard logger back as Start found them.
func (m *module) Stop(ctx context.Context) error {
	m.previous.restore()
	return nil
}
