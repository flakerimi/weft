---
title: Your first Go route
description: Add a typed route that checks its input to the app's Go module, test it, and build the binary.
order: 13
---

On this page you add a second route to the app's Go side: one that takes a JSON body, checks it, and answers. You test it, call it while `weft dev` runs, and build the binary. Run every command in the app's folder, `myapp`.

## The module you have

The app's Go code is a `warp.App` (from `github.com/flakerimi/weft/warp`) with a list of modules, in `app/modules.go`. The first module is `app/hello/routes.go`. Its `Register` adds one route:

```go
func (module) Register(app *warp.App) error {
	warp.Get(app, "/hello", hello)
	return nil
}
```

`warp.Get` registers `hello` for `GET /api/hello`: the app puts `/api` in front of every route. `hello` is a typed function, `func(ctx context.Context, _ warp.None) (Message, error)`: it takes nothing, and what it returns is answered as JSON.

## Add a route

In `app/hello/routes.go`, register a second route, `POST /api/greetings`, in `Register`:

```go
func (module) Register(app *warp.App) error {
	warp.Get(app, "/hello", hello)
	warp.Post(app, "/greetings", greet)
	return nil
}
```

and add its input and its function at the end of the file:

```go
// Greeting is the body of POST /api/greetings.
type Greeting struct {
	Name  string `json:"name" warp:"required,max=40"`
	Email string `json:"email" warp:"email"`
}

func greet(ctx context.Context, in Greeting) (Message, error) {
	return Message{Message: "Hello, " + in.Name}, nil
}
```

- `json:"name"` says the field comes from the JSON body, under the key `name`.
- `warp:"required,max=40"` are its rules: it must be there, not blank, and at most 40 characters. `email` checks an address, and lets an empty one pass, so `email` is optional.
- Warp fills `Greeting` from the body and checks the rules before `greet` runs. When a rule fails, `greet` is not called, and the answer is a 422 that names each field.
- `warp.Post` answers 201.

## Test it

Add a test to `app/hello/hello_test.go`, under `TestHello`:

```go
func TestGreet(t *testing.T) {
	app := warp.New(warp.Config{})
	app.Use(Module())
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		body   string
		status int
		want   string
	}{
		{`{"name":"Ada"}`, 201, `{"message":"Hello, Ada"}`},
		{`{"name":" ","email":"ada"}`, 422, `{"error":{"code":"validation","message":"2 fields need attention.","fields":{"email":"Enter an email address.","name":"This is required."}}}`},
	} {
		req := httptest.NewRequest("POST", "/api/greetings", strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if got := strings.TrimSpace(rec.Body.String()); rec.Code != c.status || got != c.want {
			t.Errorf("%s: %d %s, want %d %s", c.body, rec.Code, got, c.status, c.want)
		}
	}
}
```

`app.Handler()` is the app's API alone, so the test needs no server. `app.Err()` reports any mistake made while registering, such as a field with no tag, one sentence each. Run the tests:

```sh
go test -tags dev ./...
```

`-tags dev` builds the app as `weft dev` does, without the embedded pages, so the tests run before any `weft build`.

## Call it

With `weft dev` running, send a greeting:

```sh
curl -i -H 'Content-Type: application/json' -d '{"name":"Ada"}' http://localhost:5173/api/greetings
```

It answers `201 Created` with `{"message":"Hello, Ada"}`. Send `{"name":"","email":"ada"}` and it answers 422 with the two messages, in the shape every Warp error has. A body sent without `Content-Type: application/json` is a 415, and a key the route does not take, such as `nme`, is a 400 that names it.

## Build one binary

```sh
weft build
./myapp
```

`myapp` is the whole app: the pages, the API and the server. Copy it to a machine and run it; it needs no Node and no files beside it. [Publishing](../guides/publishing.md) says more.

## Next

- [Warp, the Go side](../guides/warp.md): modules, every tag and rule, the errors, and `.env`.
- [Pages and islands](../guides/pages-and-islands.md): the parts of a page that run in the browser, which can call this route.
