package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	stored, err := hashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(stored, "correct horse battery") {
		t.Fatal("the right password does not verify")
	}
	for _, wrong := range []string{"correct horse batter", "Correct horse battery", "", "correct horse battery "} {
		if verifyPassword(stored, wrong) {
			t.Errorf("%q verifies against the hash of another password", wrong)
		}
	}
	again, err := hashPassword("correct horse battery")
	if err != nil || again == stored {
		t.Fatalf("two hashes of one password are the same (%v); the salt is not random", err)
	}

	long := strings.Repeat("é", maxPassword)
	start := time.Now()
	stored, err = hashPassword(long)
	took := time.Since(start)
	if err != nil || !verifyPassword(stored, long) {
		t.Fatalf("a %d-character password does not round-trip: %v", maxPassword, err)
	}
	if took > time.Second && !raceDetector {
		t.Fatalf("hashing a %d-character password took %v, want well under a second", maxPassword, took)
	}
	t.Logf("one hash took %v", took)
}

func TestHashFormat(t *testing.T) {
	stored, err := hashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" || parts[1] != "600000" {
		t.Fatalf("hash = %q, want pbkdf2-sha256$600000$<salt>$<hash>", stored)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) != 16 {
		t.Fatalf("salt %q decodes to %d bytes (%v), want 16", parts[2], len(salt), err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(key) != 32 {
		t.Fatalf("hash %q decodes to %d bytes (%v), want 32", parts[3], len(key), err)
	}
}

// The dummy a login for an unknown email is checked against costs what a
// real user's hash costs.
func TestDummyHasTheSameParameters(t *testing.T) {
	parts := strings.Split(dummyHash, "$")
	real, _ := hashPassword("x")
	if len(parts) != 4 || parts[0]+"$"+parts[1] != strings.Join(strings.Split(real, "$")[:2], "$") {
		t.Fatalf("dummy %q has other parameters than %q", dummyHash, real)
	}
	if _, ok := parseHash(dummyHash); !ok {
		t.Fatalf("dummy %q does not parse, so checking it would cost nothing", dummyHash)
	}
}

func TestVerifyRefusesOtherSchemes(t *testing.T) {
	good, err := hashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(good, "$")
	salt, key := parts[2], parts[3]
	for _, stored := range []string{
		"",
		"$",
		"correct horse battery",
		"bcrypt$600000$" + salt + "$" + key,
		"pbkdf2-sha1$600000$" + salt + "$" + key,
		"PBKDF2-SHA256$600000$" + salt + "$" + key,
		"pbkdf2-sha256$600000$" + salt,
		"pbkdf2-sha256$600000$" + salt + "$" + key + "$",
		"pbkdf2-sha256$$" + salt + "$" + key,
		"pbkdf2-sha256$six$" + salt + "$" + key,
		"pbkdf2-sha256$+600000$" + salt + "$" + key,
		"pbkdf2-sha256$-1$" + salt + "$" + key,
		"pbkdf2-sha256$0$" + salt + "$" + key,
		"pbkdf2-sha256$10000001$" + salt + "$" + key,
		"pbkdf2-sha256$99999999999999999999$" + salt + "$" + key,
		"pbkdf2-sha256$600000$!!!$" + key,
		"pbkdf2-sha256$600000$$" + key,
		"pbkdf2-sha256$600000$" + salt + "$!!!",
		"pbkdf2-sha256$600000$" + salt + "$" + key[:20],
		"pbkdf2-sha256$600000$" + salt + "$" + key + "AAAA",
		"pbkdf2-sha256$600000$" + strings.Repeat("A", 4000) + "$" + key,
	} {
		start := time.Now()
		if verifyPassword(stored, "correct horse battery") {
			t.Errorf("%.60q verifies", stored)
		}
		// Refusing is a parse, never a long computation an attacker could cause.
		if took := time.Since(start); took > 20*time.Millisecond {
			t.Errorf("refusing %.60q took %v", stored, took)
		}
	}
	// The iteration count is read from the string, so changing it changes the
	// key that is computed.
	fast := "pbkdf2-sha256$1$" + salt + "$" + key
	if verifyPassword(fast, "correct horse battery") {
		t.Error("the key of 600000 iterations verifies with 1")
	}
}
