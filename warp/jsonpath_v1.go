//go:build !goexperiment.jsonv2

package warp

// pathHasIndexes says whether encoding/json writes a list's index in a type
// error's field path: v1 does not ("lines.qty").
const pathHasIndexes = false
