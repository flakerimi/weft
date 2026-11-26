//go:build goexperiment.jsonv2

package warp

// pathHasIndexes says whether encoding/json writes a list's index in a type
// error's field path: under GOEXPERIMENT=jsonv2 it does ("lines.1.qty").
const pathHasIndexes = true
