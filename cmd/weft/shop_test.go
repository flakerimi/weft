package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// shopIgnored is what examples/shop/.gitignore leaves out, relative to the
// app: a copy of the example never takes what a run there made.
var shopIgnored = map[string]bool{
	".weft": true, "node_modules": true, "data": true, ".env": true, "shop": true, "pnpm-lock.yaml": true,
	filepath.Join("web", "dist"): true, filepath.Join("web", ".weft-types"): true,
}

// shopSource is the repository's examples/shop, found while the test binary
// still runs in cmd/weft.
var shopSource = func() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..", "examples", "shop")
}()

// copyShop copies examples/shop into dir/shop without what is git-ignored,
// and returns the copy's path.
func copyShop(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(shopSource, "main.go")); err != nil {
		return "", fmt.Errorf("examples/shop has no main.go: %v", err)
	}
	dst := filepath.Join(dir, "shop")
	return dst, filepath.WalkDir(shopSource, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(shopSource, path)
		if shopIgnored[rel] || d.Name() == ".DS_Store" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
}

// writeShopEnv writes the .env the shop needs, with a fresh secret, as
// scripts/shop.sh does.
func writeShopEnv(app string) error {
	secret := make([]byte, 48)
	rand.Read(secret)
	env := "PORT=8080\nDB_DRIVER=sqlite\nDB_URL=file:data/shop.db\nAUTH_SECRET=" + base64.RawURLEncoding.EncodeToString(secret) + "\nAUTH_OWNER_EMAIL=owner@example.com\n"
	return os.WriteFile(filepath.Join(app, ".env"), []byte(env), 0o600)
}

var (
	shopOnce               sync.Once
	shopDir                string
	shopCode               int
	shopStdout, shopStderr string
	shopCopyErr            string
)

// builtShop returns a copy of examples/shop that weft build has run in,
// made once for this test binary; tests copy it with cp -a rather than
// change it. It must be called after inTempDir, which sets WEFT_BUNDLE.
func builtShop(t *testing.T) string {
	t.Helper()
	shopOnce.Do(func() {
		dir := filepath.Join(filepath.Dir(stagedBundle), "shop-built")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			shopCopyErr = err.Error()
			return
		}
		app, err := copyShop(dir)
		if err == nil {
			err = writeShopEnv(app)
		}
		if err != nil {
			shopCopyErr = err.Error()
			return
		}
		wd, _ := os.Getwd()
		os.Chdir(app)
		defer os.Chdir(wd)
		var out, errb bytes.Buffer
		shopCode = run([]string{"build"}, &out, &errb)
		shopStdout, shopStderr, shopDir = out.String(), errb.String(), app
	})
	if shopDir == "" {
		t.Fatalf("examples/shop could not be copied: %s", shopCopyErr)
	}
	if shopCode != 0 {
		t.Fatalf("weft build in examples/shop: exit %d\n%s%s", shopCode, shopStdout, shopStderr)
	}
	return shopDir
}

// shopCopy copies the built shop into a temp folder of the test's own and
// returns the copy's path.
func shopCopy(t *testing.T) string {
	t.Helper()
	src := builtShop(t)
	dir := t.TempDir()
	if out, err := exec.Command("cp", "-a", src, dir).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v: %s", err, out)
	}
	return filepath.Join(dir, "shop")
}

// typecheck runs the pages' type check in app.
func typecheck(app string) (string, error) {
	cmd := exec.Command("pnpm", "exec", "vue-tsc", "--noEmit", "-p", "web")
	cmd.Dir = app
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// setupLink is the line auth logs while no user exists; the token is in the
// link's fragment (or, before the login's fix round, its query).
var setupLink = regexp.MustCompile(`Set up the owner at http://\S+/setup[#?]token=([A-Za-z0-9_-]+)`)

// logBuffer is the output of a running process, read while it writes.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// startShop runs the shop's binary in its folder, so it reads its .env and
// keeps its database in data/, on a free port, and returns its base URL and
// the setup token it logged. The process is stopped when the test ends.
func startShop(t *testing.T, app string) (string, string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, filepath.Join(app, "shop"))
	cmd.Dir = app
	cmd.Env = append(os.Environ(), "PORT="+strconv.Itoa(port))
	logs := &logBuffer{}
	cmd.Stdout, cmd.Stderr = logs, logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); cmd.Wait() })
	url := "http://127.0.0.1:" + strconv.Itoa(port)
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		res, err := http.Get(url + "/api/auth/me")
		if err != nil {
			continue
		}
		res.Body.Close()
		m := setupLink.FindStringSubmatch(logs.String())
		if m == nil {
			t.Fatalf("the shop answers but logged no setup link: %s", logs.String())
		}
		return url, m[1]
	}
	t.Fatalf("the shop did not answer within 15 s: %s", logs.String())
	return "", ""
}

// send sends a JSON body with client and returns the status and the body.
func send(t *testing.T, client *http.Client, method, url string, body any) (int, string) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

// The shop builds with weft build (which writes its typed client), its own Go
// tests pass, its pages type-check against the client, and the binary
// serves the pages, the run-time page's shell and the API behind its login.
func TestShop(t *testing.T) {
	inTempDir(t, "work")
	app := shopCopy(t)

	api, err := os.ReadFile(filepath.Join(app, "web", ".weft-types", "api.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)^export interface Product \{$`),
		regexp.MustCompile(`(?s)\n  products: \{\n.*?\blist\(query\?`),
		regexp.MustCompile(`(?m)^  ProductInput: \{$`),
	} {
		if !want.Match(api) {
			t.Errorf("web/.weft-types/api.ts does not match %s:\n%s", want, api)
		}
	}

	test := exec.Command("go", "test", "-count=1", "./...")
	test.Dir = app
	if out, err := test.CombinedOutput(); err != nil || !strings.Contains(string(out), "ok  \tshop/app/products") {
		t.Errorf("go test ./... in the shop: %v\n%s", err, out)
	}

	if out, err := typecheck(app); err != nil {
		t.Errorf("the pages do not type-check against the generated client: %v\n%s", err, out)
	}

	url, token := startShop(t, app)
	for path, want := range map[string]string{
		"/":            ">Shop</h1>",
		"/login":       "<weft-island",
		"/setup":       "<weft-island",
		"/products":    "<weft-island",
		"/products/42": `data-weft-route="/products/:id"`,
	} {
		if code, body := get(t, url+path); code != 200 || !strings.Contains(body, want) {
			t.Errorf("GET %s = %d, want 200 with %q: %s", path, code, want, body)
		}
	}
	if code, body := get(t, url+"/nope"); code != 404 || !strings.Contains(body, "Not found") {
		t.Errorf("GET /nope = %d %q", code, body)
	}
	if code, body := get(t, url+"/api/products"); code != 401 || !strings.Contains(body, `"code":"unauthorized"`) {
		t.Errorf("GET /api/products without a session = %d %q", code, body)
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if code, body := send(t, client, "POST", url+"/api/auth/setup", map[string]string{"token": token, "password": "a password for the test"}); code != 201 {
		t.Fatalf("setup = %d %s", code, body)
	}
	if code, body := send(t, client, "POST", url+"/api/products", map[string]any{"name": "Lamp", "price": 30, "status": "live"}); code != 201 {
		t.Fatalf("create = %d %s", code, body)
	}
	code, body := send(t, client, "GET", url+"/api/products", nil)
	var list struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if code != 200 || json.Unmarshal([]byte(body), &list) != nil || list.Total != 1 || len(list.Items) != 1 || list.Items[0].Name != "Lamp" {
		t.Errorf("list after create = %d %s", code, body)
	}
}

// A Go field renamed is a TypeScript error in the pages after weft gen: the
// design's promise, on the shop's own pages.
func TestShopRenamedFieldBreaksTypes(t *testing.T) {
	inTempDir(t, "work")
	app := shopCopy(t)
	model := filepath.Join(app, "app", "products", "model.go")
	src, err := os.ReadFile(model)
	if err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(string(src), "`json:\"price\"", "`json:\"cost\"", 1)
	if renamed == string(src) {
		t.Fatalf("app/products/model.go has no json:\"price\": %s", src)
	}
	if err := os.WriteFile(model, []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(app)
	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("weft gen after the rename: exit %d: %s%s", code, out, errs)
	}
	out, err := typecheck(app)
	if err == nil || !strings.Contains(out, "price") || !strings.Contains(out, ".vue") {
		t.Errorf("the pages type-check though Product has no price any more: %v\n%s", err, out)
	}
}
