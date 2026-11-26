//go:build race

package auth

// raceDetector is whether the tests run with -race, which makes hashing many
// times slower, so time limits on it do not apply.
const raceDetector = true
