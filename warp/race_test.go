//go:build race

package warp

// raceDetector is whether the tests run with -race, which changes how many
// allocations a call makes, so the tests that count them do not apply.
const raceDetector = true
