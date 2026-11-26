//go:build !windows

package main

// busy is false: outside Windows a folder is renamed or removed whoever has
// a file in it open.
var busy = func(error) bool { return false }
