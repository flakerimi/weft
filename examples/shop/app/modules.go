// Package app lists the modules of the app.
package app

import (
	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"

	"shop/app/products"
)

// Modules is the one list of the app's modules, in the order they start: the
// database first, since the login and the products keep their records in it.
func Modules() []warp.Module {
	return []warp.Module{
		data.Module(),
		mail.Module(),
		auth.Module(),
		products.Module(),
	}
}
