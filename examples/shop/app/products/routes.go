package products

import (
	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/data"
)

type module struct{}

func (module) Name() string { return "products" }

// Register adds the five routes of the products, each behind its permission:
// products:read to list and show, products:create, products:edit and
// products:delete.
func (module) Register(app *warp.App) error {
	store := data.NewStore[Product](data.For(app), "products")
	data.Resource(app, "/products", store, data.PerAction(auth.Per("products")))
	return nil
}

// Module returns the module, for app/modules.go to list.
func Module() warp.Module { return module{} }
