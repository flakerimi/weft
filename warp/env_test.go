package warp

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func writeDotEnv(t *testing.T, body string) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".env", []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDotEnvGrammar(t *testing.T) {
	writeDotEnv(t, "# c\n\nexport A=1\nB = two words  # note\nC=\"x # y=z\"\nD='q'\nE=\n")
	cfg := FromEnv()
	want := map[string]string{"A": "1", "B": "two words", "C": "x # y=z", "D": "q", "E": ""}
	for k, v := range want {
		if got := cfg.Get(k); got != v {
			t.Errorf("Get(%q) = %q, want %q", k, got, v)
		}
	}
	if err := cfg.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestEnvironmentWins(t *testing.T) {
	writeDotEnv(t, "PORT=3000\nHOST=1.2.3.4\n")
	t.Setenv("PORT", "4000")
	cfg := FromEnv()
	if cfg.Port != "4000" {
		t.Errorf("Port = %q, want 4000", cfg.Port)
	}
	if cfg.Host != "1.2.3.4" {
		t.Errorf("Host = %q, want 1.2.3.4", cfg.Host)
	}
}

func TestNoDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PORT", "")
	cfg := FromEnv().withDefaults()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if err := cfg.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestBadLine(t *testing.T) {
	writeDotEnv(t, "A=1\n\nFOO\nB=2\n")
	cfg := FromEnv()
	want := `.env line 3 is "FOO"; write it as KEY=value, or start it with # to make it a comment.`
	if err := cfg.Err(); err == nil || err.Error() != want {
		t.Errorf("Err() = %v, want %q", err, want)
	}
	if cfg.Get("A") != "1" || cfg.Get("B") != "2" {
		t.Errorf("other lines not read: A=%q B=%q", cfg.Get("A"), cfg.Get("B"))
	}
}

func TestBadPort(t *testing.T) {
	for _, p := range []string{"abc", "0", "70000"} {
		err := ConfigFrom(map[string]string{"PORT": p}).Err()
		want := `PORT is "` + p + `"; set it to a number from 1 to 65535 in .env or the environment.`
		if err == nil || err.Error() != want {
			t.Errorf("PORT=%s: Err() = %v, want %q", p, err, want)
		}
	}
	if err := ConfigFrom(map[string]string{"PORT": "65535"}).Err(); err != nil {
		t.Errorf("PORT=65535: Err() = %v", err)
	}
}

func TestBadMaxBody(t *testing.T) {
	for _, p := range []string{"big", "0", "-2", "1.5"} {
		err := ConfigFrom(map[string]string{"MAX_BODY_MB": p}).Err()
		want := `MAX_BODY_MB is "` + p + `"; set it to a whole number of megabytes, 1 or more, in .env or the environment.`
		if err == nil || err.Error() != want {
			t.Errorf("MAX_BODY_MB=%s: Err() = %v, want %q", p, err, want)
		}
	}
	if got := ConfigFrom(map[string]string{"MAX_BODY_MB": "5"}).MaxBodyBytes; got != 5<<20 {
		t.Errorf("MaxBodyBytes = %d, want %d", got, 5<<20)
	}
}

func TestEveryProblemReported(t *testing.T) {
	writeDotEnv(t, "FOO\nPORT=abc\nMAX_BODY_MB=big\n")
	err := FromEnv().Err()
	if err == nil {
		t.Fatal("Err() = nil")
	}
	for _, s := range []string{`.env line 1 is "FOO"`, `PORT is "abc"`, `MAX_BODY_MB is "big"`} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("Err() = %q, missing %q", err, s)
		}
	}
}

func TestOn(t *testing.T) {
	for _, v := range []string{"on", "ON", "true", "1", "yes"} {
		if !ConfigFrom(map[string]string{"K": v}).On("K") {
			t.Errorf("%q should be on", v)
		}
	}
	for _, v := range []string{"off", "", "0", "no"} {
		if ConfigFrom(map[string]string{"K": v}).On("K") {
			t.Errorf("%q should be off", v)
		}
	}
	if (Config{}).On("MISSING") {
		t.Error("unset key should be off")
	}
}

func TestZeroConfig(t *testing.T) {
	cfg := New(Config{}).Config()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.MaxBodyBytes != 1<<20 {
		t.Errorf("MaxBodyBytes = %d, want %d", cfg.MaxBodyBytes, 1<<20)
	}
	if cfg.Get("X") != "" || cfg.Err() != nil {
		t.Error("zero Config should have no vars and no error")
	}
}

func TestNegativeBodyLimitIsTheDefault(t *testing.T) {
	if got := New(Config{MaxBodyBytes: -1}).Config().MaxBodyBytes; got != 1<<20 {
		t.Fatalf("MaxBodyBytes = %d, want %d", got, 1<<20)
	}
}

func TestDotEnvCannotBeRead(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir(".env", 0o755); err != nil {
		t.Fatal(err)
	}
	err := FromEnv().Err()
	if err == nil || !strings.HasPrefix(err.Error(), ".env cannot be read (") || !strings.HasSuffix(err.Error(), "); fix its permissions or remove it.") {
		t.Fatalf("Err() = %v", err)
	}
}

func TestDotEnvExport(t *testing.T) {
	writeDotEnv(t, "export\tA=1\nexport FOO\nexporter=2\n")
	cfg := FromEnv()
	if cfg.Get("A") != "1" || cfg.Get("exporter") != "2" {
		t.Errorf("A=%q exporter=%q", cfg.Get("A"), cfg.Get("exporter"))
	}
	want := `.env line 2 is "export FOO"; write it as KEY=value, or start it with # to make it a comment.`
	if err := cfg.Err(); err == nil || err.Error() != want {
		t.Errorf("Err() = %v, want %q", err, want)
	}
}

func TestDotEnvCRLF(t *testing.T) {
	writeDotEnv(t, "A=1\r\nB=\"two\"\r\n# c\r\n\r\nC=3 # note\r\n")
	cfg := FromEnv()
	if cfg.Get("A") != "1" || cfg.Get("B") != "two" || cfg.Get("C") != "3" || cfg.Err() != nil {
		t.Errorf("A=%q B=%q C=%q Err=%v", cfg.Get("A"), cfg.Get("B"), cfg.Get("C"), cfg.Err())
	}
}

func TestEmptyPortIsUnset(t *testing.T) {
	cfg := ConfigFrom(map[string]string{"PORT": ""})
	if cfg.Err() != nil || New(cfg).Config().Port != "8080" {
		t.Fatalf("Err() = %v, Port %q", cfg.Err(), New(cfg).Config().Port)
	}
}

func TestMaxBodyUpperBound(t *testing.T) {
	top := strconv.FormatInt((1<<62)>>20, 10)
	if cfg := ConfigFrom(map[string]string{"MAX_BODY_MB": top}); cfg.Err() != nil || cfg.MaxBodyBytes != 1<<62 {
		t.Fatalf("MAX_BODY_MB=%s: Err() = %v, MaxBodyBytes %d", top, cfg.Err(), cfg.MaxBodyBytes)
	}
	over := strconv.FormatInt((1<<62)>>20+1, 10)
	if err := ConfigFrom(map[string]string{"MAX_BODY_MB": over}).Err(); err == nil {
		t.Fatalf("MAX_BODY_MB=%s was taken", over)
	}
}

func TestAppURL(t *testing.T) {
	if got := New(Config{}).Config().URL; got != "http://localhost:8080" {
		t.Errorf("default URL %q", got)
	}
	if got := New(ConfigFrom(map[string]string{"PORT": "3000"})).Config().URL; got != "http://localhost:3000" {
		t.Errorf("URL with PORT=3000 %q", got)
	}
	for in, want := range map[string]string{
		"https://shop.example.com":  "https://shop.example.com",
		"https://shop.example.com/": "https://shop.example.com",
		"http://localhost:5173":     "http://localhost:5173",
		"HTTP://127.0.0.1:9000/":    "http://127.0.0.1:9000",
	} {
		cfg := ConfigFrom(map[string]string{"APP_URL": in})
		if cfg.Err() != nil || New(cfg).Config().URL != want {
			t.Errorf("APP_URL=%s: Err() = %v, URL %q, want %q", in, cfg.Err(), New(cfg).Config().URL, want)
		}
	}
	for _, in := range []string{"shop.example.com", "ftp://shop.example.com", "https://", "https://shop.example.com/app", "https://shop.example.com?a=1", "https://shop.example.com/#top", "/setup", "http://%zz", "https://:8080", "http://:80/"} {
		want := `APP_URL is "` + in + `"; set it to an address like https://shop.example.com.`
		if err := ConfigFrom(map[string]string{"APP_URL": in}).Err(); errText(err) != want {
			t.Errorf("APP_URL=%s: Err() = %v, want %q", in, err, want)
		}
	}
}

func TestTrustProxy(t *testing.T) {
	if ConfigFrom(nil).TrustProxy || (Config{}).TrustProxy {
		t.Error("TrustProxy is on by default")
	}
	if !ConfigFrom(map[string]string{"TRUST_PROXY": "on"}).TrustProxy {
		t.Error("TRUST_PROXY=on did not turn TrustProxy on")
	}
	if ConfigFrom(map[string]string{"TRUST_PROXY": "off"}).TrustProxy {
		t.Error("TRUST_PROXY=off turned TrustProxy on")
	}
}
