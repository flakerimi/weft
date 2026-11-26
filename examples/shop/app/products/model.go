// Package products is the shop's one resource: products, kept in the
// products table and served under /api/products to the signed-in owner.
package products

import "github.com/flakerimi/weft/warp/data"

// Product is one thing the shop sells. Its id and two times come from
// data.Record and are set by the store.
type Product struct {
	data.Record
	// Name is unique, sorted on and searched.
	Name string `json:"name" warp:"required,max=120,unique" data:"sort,search"`
	// Price is in the shop's currency, 0 or more, and sorted on.
	Price float64 `json:"price" warp:"min=0" data:"sort"`
	// Sale is the price while it is on sale, 0 or more, or none (null).
	Sale *float64 `json:"sale" warp:"min=0"`
	// Status is draft or live, and filtered on.
	Status string `json:"status" warp:"required,enum=draft|live" data:"filter"`
}
