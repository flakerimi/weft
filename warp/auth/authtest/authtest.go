// Package authtest helps the tests of an app that signs people in with
// warp/auth.
package authtest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

// Password is the password Owner sets for the owner, and As for each user it
// makes.
const Password = "the owner's password, for tests"

// Owner makes the app's owner through the setup route, with the setup token
// of the started app and Password, leaving c signed in as the owner, and
// returns the owner. It fails the test when the app has no setup token, which
// is when a user already exists or the auth module is not listed. Call it
// before As, since a user As makes ends the setup.
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

// As makes a new user of role with auth.AddUser, with the email
// <role>-<n>@example.com and Password, signs a fork of c in as that user, and
// returns the fork and the user; c is left as it was. It fails the test when
// the user cannot be made or cannot sign in.
func As(t testing.TB, c *warptest.Client, role string) (*warptest.Client, auth.User) {
	t.Helper()
	ctx := context.Background()
	var n int
	if err := data.For(c.App).QueryRow(ctx, `SELECT count(*) FROM "users"`).Scan(&n); err != nil {
		t.Fatalf("authtest: the users table cannot be read (%v); call As on an app started with data.Module() and auth.Module().", err)
	}
	for {
		n++
		email := fmt.Sprintf("%s-%d@example.com", role, n)
		u, err := auth.AddUser(ctx, c.App, email, Password, role)
		var e *warp.Error
		if errors.As(err, &e) && e.Fields["email"] == "This is already taken." {
			continue // a user was removed, so the count is behind
		}
		if err != nil {
			t.Fatalf("authtest: As could not make a user of role %q (%v); give it owner, administrator, member or viewer.", role, err)
		}
		f := c.Fork()
		f.Post("/api/auth/login", map[string]string{"email": email, "password": Password}).Want(200)
		return f, u
	}
}
