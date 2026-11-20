package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/flakerimi/weft"
)

func versionCmd(_ []string, stdout, _ io.Writer) int {
	fmt.Fprintf(stdout, "weft %s (%s/%s)\n", weft.Version, runtime.GOOS, runtime.GOARCH)
	return 0
}
