package warp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const crossSiteSentence = "This request came from another site."

// originApp is an app with one route per method under /api/things and a
// counter of how many times a handler or a Wrap ran.
func originApp(t *testing.T, cfg Config) (http.Handler, *int) {
	t.Helper()
	ran := 0
	app := New(cfg)
	app.Wrap(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ran++
			next.ServeHTTP(w, r)
		})
	})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran++
		w.WriteHeader(http.StatusCreated)
	})
	for _, m := range []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"} {
		app.Handle(m+" /things", h)
	}
	app.Handle("POST /{$}", h)
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	return app.Handler(), &ran
}

func originSend(h http.Handler, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range header {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestCrossSiteRefused(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, body := range []string{"", `{"a":1}`} {
			for name, header := range map[string]map[string]string{
				"sec-fetch-site": {"Sec-Fetch-Site": "cross-site"},
				"same-site":      {"Sec-Fetch-Site": "same-site"},
				"origin":         {"Origin": "https://evil.example"},
			} {
				h, ran := originApp(t, ConfigFrom(nil))
				w := originSend(h, method, "/api/things", body, header)
				if w.Code != 403 {
					t.Errorf("%s with body %q and %s: status %d, want 403", method, body, name, w.Code)
				}
				if !strings.Contains(w.Body.String(), `"code":"forbidden"`) || !strings.Contains(w.Body.String(), crossSiteSentence) {
					t.Errorf("%s with %s: body %s", method, name, w.Body)
				}
				if ct := w.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type %q", ct)
				}
				if *ran != 0 {
					t.Errorf("%s with %s: a handler or Wrap ran", method, name)
				}
			}
		}
	}
}

func TestSameSiteAllowed(t *testing.T) {
	for name, header := range map[string]map[string]string{
		"same-origin":   {"Sec-Fetch-Site": "same-origin"},
		"none":          {"Sec-Fetch-Site": "none"},
		"no headers":    nil,
		"origin = host": {"Origin": "http://example.com"},
	} {
		h, _ := originApp(t, ConfigFrom(nil))
		if w := originSend(h, "POST", "/api/things", `{}`, header); w.Code != 201 {
			t.Errorf("%s: status %d, want 201", name, w.Code)
		}
	}
}

func TestGetIsNotChecked(t *testing.T) {
	h, _ := originApp(t, ConfigFrom(nil))
	for _, m := range []string{"GET", "HEAD", "OPTIONS"} {
		if w := originSend(h, m, "/api/things", "", map[string]string{"Sec-Fetch-Site": "cross-site"}); w.Code != 201 {
			t.Errorf("%s: status %d, want 201", m, w.Code)
		}
	}
}

func TestTrustedOrigin(t *testing.T) {
	cfg := ConfigFrom(map[string]string{"TRUSTED_ORIGINS": "https://admin.example.com, ,http://localhost:5173,"})
	if err := cfg.Err(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(cfg.TrustedOrigins, "|"); got != "https://admin.example.com|http://localhost:5173" {
		t.Fatalf("TrustedOrigins = %q", got)
	}
	h, _ := originApp(t, cfg)
	for origin, want := range map[string]int{
		"https://admin.example.com": 201,
		"http://localhost:5173":     201,
		"https://other.example.com": 403,
		"http://admin.example.com":  403,
	} {
		w := originSend(h, "POST", "/api/things", "", map[string]string{"Origin": origin, "Sec-Fetch-Site": "cross-site"})
		if w.Code != want {
			t.Errorf("Origin %s: status %d, want %d", origin, w.Code, want)
		}
	}
	// Without Sec-Fetch-Site the Origin alone is compared too.
	if w := originSend(h, "POST", "/api/things", "", map[string]string{"Origin": "https://admin.example.com"}); w.Code != 201 {
		t.Errorf("trusted Origin alone: status %d", w.Code)
	}
}

func TestTrustedOriginsMistake(t *testing.T) {
	cfg := ConfigFrom(map[string]string{"TRUSTED_ORIGINS": "https://ok.example.com,admin.example.com,https://x.example.com/path"})
	err := cfg.Err()
	if err == nil {
		t.Fatal("no problem reported")
	}
	for _, want := range []string{
		`TRUSTED_ORIGINS has "admin.example.com"; write each origin like https://admin.example.com.`,
		`TRUSTED_ORIGINS has "https://x.example.com/path"; write each origin like https://admin.example.com.`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Err() = %q, missing %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "ok.example.com") || strings.Contains(err.Error(), "warp:") {
		t.Errorf("Err() = %q", err)
	}
	if app := New(cfg); app.Err() == nil {
		t.Error("the app does not report the configuration's problem")
	}
}

func TestRefusalComesBeforeWrap(t *testing.T) {
	h, ran := originApp(t, ConfigFrom(nil))
	if w := originSend(h, "POST", "/api/nothing-here", "", map[string]string{"Sec-Fetch-Site": "cross-site"}); w.Code != 403 {
		t.Errorf("unrouted path: status %d, want 403 (before routing)", w.Code)
	}
	if *ran != 0 {
		t.Error("a Wrap ran for a refused request")
	}
	if w := originSend(h, "POST", "/api/things", "", nil); w.Code != 201 || *ran != 2 {
		t.Errorf("allowed request: status %d, ran %d, want 201 and 2", w.Code, *ran)
	}
}

func TestOutsideAPIIsNotChecked(t *testing.T) {
	h, ran := originApp(t, ConfigFrom(nil))
	for _, target := range []string{"/other", "/apix", "/ap/i"} {
		w := originSend(h, "POST", target, "", map[string]string{"Sec-Fetch-Site": "cross-site"})
		want := `{"error":{"code":"not_found","message":"Nothing is at ` + target + `."}}` + "\n"
		if w.Code != 404 || w.Body.String() != want {
			t.Errorf("%s: status %d, body %q, want 404 %q", target, w.Code, w.Body, want)
		}
	}
	if *ran != 3 {
		t.Errorf("the Wrap ran %d times, want 3", *ran)
	}
}

// A path is checked as it is and as the mux cleans it, so a doubled slash
// or a dot does not take a request past the check to the mux's redirect,
// with the app's Wraps run on the way. Every method but GET, HEAD and
// OPTIONS is checked, a made-up one included.
func TestCrossSiteCheckedOnEveryAPIPath(t *testing.T) {
	for _, target := range []string{"/api", "/api/", "/api/things", "/api/../api/things", "//api/things", "/api//things", "/./api/things", "///api"} {
		for _, method := range []string{"POST", "TRACE", "BREW"} {
			h, ran := originApp(t, ConfigFrom(nil))
			w := originSend(h, method, target, "", map[string]string{"Sec-Fetch-Site": "cross-site"})
			if w.Code != 403 || !strings.Contains(w.Body.String(), crossSiteSentence) || *ran != 0 {
				t.Errorf("%s %s: status %d, body %s, ran %d; want 403 before any Wrap", method, target, w.Code, w.Body, *ran)
			}
		}
	}
}

// TRUSTED_ORIGINS is checked once, as it is read: New uses the protection
// ConfigFrom made, unless the list was changed since.
func TestTrustedOriginsCheckedOnce(t *testing.T) {
	cfg := ConfigFrom(map[string]string{"TRUSTED_ORIGINS": "https://a.example.com,bad"})
	if cfg.xo == nil || New(cfg).xo != cfg.xo {
		t.Fatal("New made the protection again")
	}
	cfg.TrustedOrigins = []string{"https://b.example.com"}
	app := New(cfg)
	if app.xo == cfg.xo {
		t.Fatal("New kept a protection made for another list")
	}
	h := app.Handler()
	if w := originSend(h, "POST", "/api/x", "", map[string]string{"Origin": "https://b.example.com", "Sec-Fetch-Site": "cross-site"}); w.Code == 403 {
		t.Error("the changed list was not used")
	}
}
