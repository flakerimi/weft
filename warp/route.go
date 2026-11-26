package warp

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"net/http"
	"reflect"
	"slices"
)

// None is an input or output with nothing in it. As In, nothing is read from
// the request; as Out, the route answers 204 with no body.
type None struct{}

var noneType = reflect.TypeFor[None]()

// Get registers fn for GET at path, without /api in front. The input is filled
// from the query and the path, checked by Validate, and given to fn; what fn
// returns is answered as JSON with 200. A mistake is reported by Err.
//
// A query field that is a map[string]string takes every key[name] of the
// query, key being its tag's name: Filter map[string]string `query:"filter"`
// takes filter[status]=live as "status": "live", a name given twice keeping
// its first value. More than 32 names, or one longer than 64 bytes, is a 400.
//
// When the input, or its body field's type, has a method
// Validate(ctx context.Context) error (on the value or the pointer), it is
// called once the tag rules pass: the input's, then the body's (not on a
// route registered with Patch, whose doc says why). When both
// return Invalid, their fields are answered together, the input's first; any
// other error is answered as fn's would be.
func Get[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodGet, path, http.StatusOK, fn, opts)
}

// Post registers fn for POST at path, as Get does, and fills the input from
// the JSON body too. It answers with 201.
//
// The body fills the input's json fields; or, when one field is tagged
// body:"json", that field, whose type is a struct, takes the whole body, its
// fields keyed as the input's own (name, not body.name), as an embedded
// struct's are. An input with a body field has no json fields of its own. A
// field with the rule readonly, at any depth, is answered and never read:
// a body that sends it is a 400, and it is never validated.
//
// A body that is not empty must be sent as application/json (or a type
// ending in +json), else the answer is 415. A cross-site form or fetch can
// send text/plain with the visitor's cookies and no preflight; taking JSON
// alone keeps another site from sending a typed route a body. It does not
// protect a route that takes no body: a POST /api/like/7 with no body is
// answered whatever its Content-Type.
func Post[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodPost, path, http.StatusCreated, fn, opts)
}

// Put registers fn for PUT at path, as Post does, and answers with 200.
func Put[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption) {
	typed(app, http.MethodPut, path, http.StatusOK, fn, opts)
}

// Patch registers fn for PATCH at path, as Post does, and answers with 200.
//
// One difference: the tag rules run only on the json fields the body sent,
// so one struct serves create and update. A field left out is not checked at
// all, required included; a field sent is checked whole, so "name": "" fails
// required. Query and path fields are checked as on every route. Validate
// called directly checks every field. The input's own Validate method still
// runs, and can ask Sent what was sent. The Validate method of the body
// field's type does not: on PATCH that value holds zero for every field the
// body left out, so a check across its fields would judge a record that does
// not exist. A handler that needs it merges what was sent over the stored
// value and calls it there, as data.Resource does.
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
	sent   []string
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

// Sent reports whether the JSON body of the request a typed handler is
// answering had the top-level key, a json name the route takes. A key sent
// in another case, as encoding/json takes it ("NAME" for name), counts as the
// route's name. It is false when ctx is not a typed handler's.
func Sent(ctx context.Context, key string) bool {
	c, ok := ctx.Value(callKey{}).(*call)
	return ok && slices.Contains(c.sent, key)
}

// SentKeys returns the top-level keys the JSON body of the request a typed
// handler is answering had, each once and in the body's order, named as Sent
// names them. It is nil when the body sent none, the route reads no body, or
// ctx is not a typed handler's.
func SentKeys(ctx context.Context) []string {
	if c, ok := ctx.Value(callKey{}).(*call); ok {
		return slices.Clone(c.sent)
	}
	return nil
}

// typed registers fn at method and path, binding its input by a plan made
// once, here. When the input type has mistakes, route reports each and does
// not register the route.
func typed[In, Out any](app *App, method, path string, status int, fn func(context.Context, In) (Out, error), opts []RouteOption) {
	in, out := reflect.TypeFor[In](), reflect.TypeFor[Out]()
	p, mistakes := planFor(in, method, path)
	limit := app.cfg.MaxBodyBytes
	app.route(method, path, in, out, p, status, func(status int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var input In
			rv := reflect.ValueOf(&input).Elem()
			c := &call{header: w.Header()}
			ctx := context.WithValue(r.Context(), callKey{}, c)
			c.r = r.WithContext(ctx)
			var err error
			if c.sent, err = p.fill(w, r, rv, limit); err != nil {
				WriteError(w, c.r, err)
				return
			}
			if err := p.check(ctx, rv, c.sent); err != nil {
				WriteError(w, c.r, err)
				return
			}
			output, err := fn(ctx, input)
			if err != nil {
				WriteError(w, c.r, err)
				return
			}
			WriteJSON(w, status, output)
		})
	}, opts, mistakes)
}

// check validates the filled input rv: its tag rules, all of them or, on
// PATCH, those of the json fields in sent; then, once they pass, its own
// Validate method and, except on PATCH, its body field's.
func (p *plan) check(ctx context.Context, rv reflect.Value, sent []string) error {
	if p.none {
		return nil
	}
	if err := validate(rv, p.partial, sent); err != nil {
		return err
	}
	var first error
	if p.checkSelf {
		if first = rv.Addr().Interface().(validator).Validate(ctx); first != nil && !isInvalid(first) {
			return first
		}
	}
	if p.checkBody == nil || p.partial { // on PATCH the body holds only what was sent
		return first
	}
	err := rv.FieldByIndex(p.checkBody).Addr().Interface().(validator).Validate(ctx)
	if first == nil || err == nil || !isInvalid(err) {
		return cmp.Or(err, first)
	}
	// Both are Invalid: one answer names the fields of both, the input's
	// message kept for a field both name.
	var a, b *Error
	errors.As(first, &a)
	errors.As(err, &b)
	fields := maps.Clone(b.Fields)
	if fields == nil {
		fields = map[string]string{}
	}
	maps.Copy(fields, a.Fields)
	return Invalid(fields)
}

// isInvalid reports whether err is a 422 from Invalid.
func isInvalid(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusUnprocessableEntity && e.Code == "validation"
}
