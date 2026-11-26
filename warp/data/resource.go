package data

import (
	"context"
	"reflect"
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

// resourcePath is a path of static segments of letters, digits, -, _ and .,
// such as /products or /shop/order-items.
var resourcePath = regexp.MustCompile(`^(/[A-Za-z0-9._-]+)+$`)

// goodPath reports whether path is one Resource takes, leaving /api aside:
// static segments, none of them . or .. alone.
func goodPath(path string) bool {
	return resourcePath.MatchString(path) &&
		!slices.ContainsFunc(strings.Split(path, "/"), func(s string) bool { return s == "." || s == ".." })
}

// selfChecker is a model with a check across its fields.
type selfChecker interface {
	Validate(ctx context.Context) error
}

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
//     fields the body sent, checking their rules alone. When the model has a
//     method Validate(ctx context.Context) error, it is called on the stored
//     record with the sent fields laid over it, in the transaction that then
//     writes them, and its error is answered as a handler's is. With no field
//     sent it answers the record as it is, unchecked.
//   - remove: DELETE path/{id} answers 204, or 404.
//
// A PATCH or DELETE of an id no record has is a 404 only when the request is
// otherwise well formed: the body is read and checked before the record is
// looked for, so a PATCH with a body it refuses gets that 400 or 422 first.
//
// A value another record has in a unique field is a 422 on that field. Call
// it from a module's Register; a mistake is reported by the app.
func Resource[T any](app *warp.App, path string, store *Store[T], opts ...ResourceOption) {
	// /api alone is refused too, since its routes would be /api/{id}.
	if rest, ok := strings.CutPrefix(path, "/api"); ok && goodPath(rest) {
		app.Mistake("data.Resource was given the path %q; write it without /api, as %s.", path, rest)
		return
	}
	if !goodPath(path) || path == "/api" {
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
	checks := reflect.PointerTo(reflect.TypeFor[T]()).Implements(reflect.TypeFor[selfChecker]())
	warp.Patch(app, one, func(ctx context.Context, in patchInput[T]) (T, error) {
		sent := warp.SentKeys(ctx)
		if len(sent) == 0 || !checks {
			return store.Update(ctx, in.ID, &in.Body, sent)
		}
		var out T
		if _, _, err := store.ready(ctx); err != nil { // before Tx, which needs the database
			return out, err
		}
		err := store.db.Tx(ctx, func(ctx context.Context) error {
			merged, err := store.Get(ctx, in.ID)
			if err != nil {
				return err
			}
			store.t.copyFields(reflect.ValueOf(&merged).Elem(), reflect.ValueOf(&in.Body).Elem(), sent)
			if err := any(&merged).(selfChecker).Validate(ctx); err != nil {
				return err
			}
			out, err = store.Update(ctx, in.ID, &merged, sent)
			return err
		})
		return out, err
	}, options("update")...)
	warp.Delete(app, one, func(ctx context.Context, in ID) (warp.None, error) {
		return warp.None{}, store.Delete(ctx, in.ID)
	}, options("remove")...)
}

// copyFields sets the fields of dst named by their json names to src's. A
// name with no column is left for Store.Update to refuse.
func (t *table) copyFields(dst, src reflect.Value, names []string) {
	for _, name := range names {
		if c := t.col(name); c != nil && !c.auto {
			dst.FieldByIndex(c.index).Set(src.FieldByIndex(c.index))
		}
	}
}
