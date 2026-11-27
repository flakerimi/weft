// Package middleware is what a Warp app puts around its requests when the
// middleware module is listed: security headers on every answer.
package middleware

import (
	"github.com/flakerimi/weft/warp"
)

// Option changes the middleware module.
type Option func(*module)

// Module is the middleware module, named "middleware", which needs no other.
// Registering puts the security headers around every request the app
// answers, its pages and files as well as its API, and reads nothing.
func Module(opts ...Option) warp.Module {
	m := &module{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type module struct {
	app *warp.App
}

func (m *module) Name() string { return "middleware" }

func (m *module) Register(app *warp.App) error {
	m.app = app
	app.WrapAll(m.headers)
	return nil
}
