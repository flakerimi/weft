// Package app lists the modules of the app.
package app

import (
	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
	"github.com/flakerimi/weft/warp/middleware"

	"shop/app/products"
)

// Modules is the one list of the app's modules, in the order they start:
// middleware first, so every line the others log is in its format; then the
// database, since the login and the products keep their records in it, and
// mail, which sends the login's links.
func Modules() []warp.Module {
	return []warp.Module{
		middleware.Module(),
		data.Module(),
		mail.Module(),
		auth.Module(),
		products.Module(),
	}
}
