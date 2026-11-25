package warp

import (
	"context"
	"net/http"
	"reflect"
)

// None is an input or output with nothing in it. As In, nothing is read from
// the request; as Out, the route answers 204 with no body.
type None struct{}

var noneType = reflect.TypeFor[None]()

// Get registers fn for GET at path, without /api in front. The input is filled
// from the query and the path, checked by Validate, and given to fn; what fn
// returns is answered as JSON with 200. A mistake is reported by Err.
func Get[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodGet, path, http.StatusOK, fn, opts)
}

// Post registers fn for POST at path, as Get does, and fills the input from
// the JSON body too. It answers with 201.
//
// A body that is not empty must be sent as application/json (or a type
// ending in +json), else the answer is 415. A cross-site form or fetch can
// send text/plain with the visitor's cookies and no preflight; taking JSON
// bodies alone is what keeps another site from writing to a typed route.
func Post[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodPost, path, http.StatusCreated, fn, opts)
}

// Put registers fn for PUT at path, as Post does, and answers with 200.
func Put[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodPut, path, http.StatusOK, fn, opts)
}

// Patch registers fn for PATCH at path, as Post does, and answers with 200.
func Patch[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodPatch, path, http.StatusOK, fn, opts)
}

// Delete registers fn for DELETE at path, as Get does.
func Delete[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodDelete, path, http.StatusOK, fn, opts)
}

// callKey is the context key of the request a typed handler is answering.
type callKey struct{}

type call struct {
	r      *http.Request
	header http.Header
}

// Request returns the request a typed handler is answering, or nil when ctx
// is not a typed handler's. On POST, PUT and PATCH its body has already been
// read into the input, so there is nothing left in it to read.
func Request(ctx context.Context) *http.Request {
	if c, ok := ctx.Value(callKey{}).(*call); ok {
		return c.r
	}
	return nil
}

// ResponseHeader returns the headers a typed handler's answer is sent with,
// whether it succeeds or fails, such as Set-Cookie. When ctx is not a typed
// handler's it returns a new empty header.
func ResponseHeader(ctx context.Context) http.Header {
	if c, ok := ctx.Value(callKey{}).(*call); ok {
		return c.header
	}
	return http.Header{}
}

// typed registers fn at method and path, binding its input by a plan made
// once, here. When the input type has mistakes, route reports each and does
// not register the route.
func typed[In, Out any](app *App, method, path string, status int, fn func(context.Context, In) (Out, error), opts []RouteOption) {
	in, out := reflect.TypeFor[In](), reflect.TypeFor[Out]()
	p, mistakes := planFor(in, method, path)
	limit := app.cfg.MaxBodyBytes
	app.route(method, path, in, out, status, func(status int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var input In
			if err := p.fill(w, r, reflect.ValueOf(&input).Elem(), limit); err != nil {
				WriteError(w, r, err)
				return
			}
			if !p.none {
				if err := Validate(input); err != nil {
					WriteError(w, r, err)
					return
				}
			}
			c := &call{header: w.Header()}
			ctx := context.WithValue(r.Context(), callKey{}, c)
			c.r = r.WithContext(ctx)
			output, err := fn(ctx, input)
			if err != nil {
				WriteError(w, c.r, err)
				return
			}
			WriteJSON(w, status, output)
		})
	}, opts, mistakes)
}
