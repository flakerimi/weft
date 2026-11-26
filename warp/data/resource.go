package data

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/flakerimi/weft/warp"
)

// ResourceOption changes the routes Resource registers.
type ResourceOption func(*resourceConfig)

type resourceConfig struct {
	each []warp.RouteOption
	per  []func(action string) []warp.RouteOption
}

// Each gives all five routes of a resource the options opts.
func Each(opts ...warp.RouteOption) ResourceOption {
	return func(c *resourceConfig) { c.each = append(c.each, opts...) }
}

// PerAction gives each route of a resource the options fn returns for its
// action: list, show, create, update or remove. fn is called once per action,
// when Resource registers, and its options come after every Each's. This is
// how a permission per action is attached.
func PerAction(fn func(action string) []warp.RouteOption) ResourceOption {
	return func(c *resourceConfig) {
		if fn != nil {
			c.per = append(c.per, fn)
		}
	}
}

// patchInput is the input of a resource's update: the id from the path and
// the record as the body.
type patchInput[T any] struct {
	ID   int64 `path:"id"`
	Body T     `body:"json"`
}

// resourcePath is a path of static segments, such as /products or
// /shop/order-items.
var resourcePath = regexp.MustCompile(`^(/[^/{}\s]+)+$`)

// Resource registers the five routes of the records store keeps, under path,
// which is written like /products, named list, show, create, update and
// remove whatever the options say:
//
//   - list: GET path takes a Query and answers a List[T], as Store.List does.
//   - show: GET path/{id} answers the record, or 404.
//   - create: POST path takes a record as its body and answers it made, with
//     201. Its rules and its own Validate method, when it has one, are
//     checked; id, created_at and updated_at are never read.
//   - update: PATCH path/{id} takes a record as its body and writes only the
//     fields the body sent, checking only them; with none sent it answers the
//     record as it is.
//   - remove: DELETE path/{id} answers 204, or 404.
//
// A value another record has in a unique field is a 422 on that field. Call
// it from a module's Register; a mistake is reported by the app.
func Resource[T any](app *warp.App, path string, store *Store[T], opts ...ResourceOption) {
	if !resourcePath.MatchString(path) || slices.ContainsFunc(strings.Split(path, "/"), func(s string) bool { return s == "." || s == ".." }) {
		app.Mistake("data.Resource was given the path %q; write it like /products.", path)
		return
	}
	if store == nil {
		app.Mistake("data.Resource was given a nil store for %s; make one with data.NewStore in the module's Register.", path)
		return
	}
	var cfg resourceConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	options := func(action string) []warp.RouteOption {
		out := slices.Clone(cfg.each)
		for _, fn := range cfg.per {
			out = append(out, fn(action)...)
		}
		return append(out, warp.Name(action))
	}
	one := path + "/{id}"
	warp.Get(app, path, func(ctx context.Context, q Query) (List[T], error) {
		return store.List(ctx, q)
	}, options("list")...)
	warp.Get(app, one, func(ctx context.Context, in ID) (T, error) {
		return store.Get(ctx, in.ID)
	}, options("show")...)
	warp.Post(app, path, func(ctx context.Context, v T) (T, error) {
		err := store.Create(ctx, &v)
		return v, err
	}, options("create")...)
	warp.Patch(app, one, func(ctx context.Context, in patchInput[T]) (T, error) {
		return store.Update(ctx, in.ID, &in.Body, warp.SentKeys(ctx))
	}, options("update")...)
	warp.Delete(app, one, func(ctx context.Context, in ID) (warp.None, error) {
		return warp.None{}, store.Delete(ctx, in.ID)
	}, options("remove")...)
}
