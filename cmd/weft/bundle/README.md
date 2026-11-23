# bundle

A release stages the bundle here with `scripts/bundle.sh cmd/weft/bundle`
before it builds `weft`, so `go:embed` carries it into the binary. In git this
folder holds only this file; everything else in it is ignored.
