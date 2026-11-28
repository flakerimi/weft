package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/mail"
)

// Each of the four mails parses, and renders with its link and host: the
// subject on one line, the text saying what the link does, how long it
// lasts and that the mail can be ignored, the HTML the same with the link as
// a link.
func TestTemplatesParse(t *testing.T) {
	m, err := parseMails()
	if err != nil {
		t.Fatal(err)
	}
	link := "https://shop.example.com/password#token=" + testToken
	for _, tc := range []struct {
		name    string
		tpl     *mail.Template
		subject string
		lasts   string
		does    string
	}{
		{"invite", m.invite, "You are invited to shop.example.com", "7 days", "choose a password"},
		{"register", m.register, "Finish signing up to shop.example.com", "24 hours", "finish signing up"},
		{"known", m.known, "You already have an account at shop.example.com", "1 hour", "choose a new one"},
		{"reset", m.reset, "Reset your password for shop.example.com", "1 hour", "choose a new password"},
	} {
		msg, err := tc.tpl.Message("a@example.com", mailData{Host: "shop.example.com", Link: link, Role: "member"})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if msg.To != "a@example.com" || msg.Subject != tc.subject || strings.ContainsAny(msg.Subject, "\r\n") {
			t.Errorf("%s: to %q, subject %q, want %q", tc.name, msg.To, msg.Subject, tc.subject)
		}
		for _, part := range []string{link, tc.lasts, tc.does, "ignore this mail"} {
			if !strings.Contains(msg.Text, part) {
				t.Errorf("%s: the text lacks %q:\n%s", tc.name, part, msg.Text)
			}
		}
		for _, part := range []string{`<a href="` + link + `">`, tc.lasts, "ignore this mail"} {
			if !strings.Contains(msg.HTML, part) {
				t.Errorf("%s: the HTML lacks %q:\n%s", tc.name, part, msg.HTML)
			}
		}
		if strings.Contains(msg.Text+msg.HTML, "?") {
			t.Errorf("%s: a ? in the mail, which a link must never have", tc.name)
		}
	}
	// What the data brings is escaped in the HTML and not in the text.
	msg, err := m.invite.Message("a@example.com", mailData{Host: "a<b>.example.com", Link: link, Role: "member"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg.Text, "a<b>.example.com") || strings.Contains(msg.HTML, "a<b>") || !strings.Contains(msg.HTML, "a&lt;b&gt;") {
		t.Errorf("escaping: text %q, HTML %q", msg.Text, msg.HTML)
	}
	// A template that does not parse is a mistake of the module.
	if _, err := mail.NewTemplate("{{.Host", "text", ""); err == nil {
		t.Error("a broken subject parsed")
	}
	if _, err := m.reset.Message("a@example.com", struct{}{}); err == nil {
		t.Error("a template filled without its data rendered")
	}
}

func TestTokenHash(t *testing.T) {
	token := newToken()
	sum := sha256.Sum256([]byte(token))
	if len(token) != 43 || !isToken(token) || tokenHash(token) != hex.EncodeToString(sum[:]) || len(tokenHash(token)) != 64 {
		t.Fatalf("token %q, hash %q", token, tokenHash(token))
	}
	if tokenHash(token) == tokenHash(newToken()) {
		t.Fatal("two tokens have one hash")
	}
}

// The links of the mails are <APP_URL>/password#token=<token>, and the host
// the subjects name is APP_URL's.
func TestLinkAndHost(t *testing.T) {
	for url, host := range map[string]string{
		"https://shop.example.com":      "shop.example.com",
		"http://localhost:8080":         "localhost:8080",
		"https://example.com/apps/shop": "example.com",
	} {
		m := &module{}
		m.setURL(url)
		if m.host != host || m.link("abc") != url+"/password#token=abc" {
			t.Errorf("APP_URL %s: host %q, link %q", url, m.host, m.link("abc"))
		}
	}
}

// The login needs the mail module from Register on: listed without it, the
// app has the mistake that says what to list, and with it the login keeps
// the app's Mailer.
func TestRegisterNeedsMail(t *testing.T) {
	app := warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), Module())
	want := `warp: module "auth" needs the "mail" module; list mail.Module() before it in app/modules.go.`
	if err := app.Err(); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Err = %v, want %q", err, want)
	}

	app = warp.New(warp.ConfigFrom(nil))
	app.Use(data.Module(), mail.Module(), Module())
	if err := app.Err(); err != nil {
		t.Fatalf("Err = %v", err)
	}
	if m := of(app); m.mail == nil || m.mail != mail.For(app) {
		t.Fatalf("the login keeps %p, want the app's Mailer %p", m.mail, mail.For(app))
	}
}
