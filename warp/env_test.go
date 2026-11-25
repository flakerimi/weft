package warp

import (
	"os"
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
