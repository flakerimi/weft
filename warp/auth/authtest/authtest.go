// Package authtest helps the tests of an app that signs people in with
// warp/auth.
package authtest

import (
	"testing"

	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/warptest"
)

// Password is the password Owner sets for the owner.
const Password = "the owner's password, for tests"

// Owner makes the app's owner through the setup route, with the setup token
// of the started app and Password, leaving c signed in as the owner, and
// returns the owner. It fails the test when the app has no setup token, which
// is when a user already exists or the auth module is not listed.
func Owner(t testing.TB, c *warptest.Client) auth.User {
	t.Helper()
	token := auth.SetupToken(c.App)
	if token == "" {
		t.Fatalf("authtest: the app has no setup token, so it has a user already or no auth module; call Owner once, on an app started with auth.Module() and an empty users table.")
	}
	var u auth.User
	c.Post("/api/auth/setup", map[string]string{"token": token, "password": Password}).Want(201).JSON(&u)
	return u
}
