package mail

import (
	"strings"
	"testing"
)

func TestTemplateRenders(t *testing.T) {
	tpl, err := NewTemplate(
		"Reset your password for {{.App}}",
		"Hello {{.Name}},\n\nOpen {{.Link}} to reset it.",
		"<p>Hello {{.Name}},</p><p><a href=\"{{.Link}}\">Reset it</a></p>",
	)
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]string{"App": "Shop", "Name": "<b>Ann</b> & co", "Link": "https://shop.example.com/password#token=abc"}
	msg, err := tpl.Message("ann@example.com", data)
	if err != nil {
		t.Fatal(err)
	}
	if msg.To != "ann@example.com" || msg.Subject != "Reset your password for Shop" {
		t.Errorf("To, Subject = %q, %q", msg.To, msg.Subject)
	}
	if !strings.Contains(msg.Text, "Hello <b>Ann</b> & co,") {
		t.Errorf("the text escapes what it should not: %q", msg.Text)
	}
	if !strings.Contains(msg.HTML, "Hello &lt;b&gt;Ann&lt;/b&gt; &amp; co,") {
		t.Errorf("the HTML does not escape <: %q", msg.HTML)
	}
	if !strings.Contains(msg.HTML, `href="https://shop.example.com/password#token=abc"`) {
		t.Errorf("the link is not in the HTML as given: %q", msg.HTML)
	}

	textOnly, err := NewTemplate("Hi", "Just text", "")
	if err != nil {
		t.Fatal(err)
	}
	msg, err = textOnly.Message("ann@example.com", nil)
	if err != nil || msg.HTML != "" || msg.Text != "Just text" {
		t.Errorf("a template without HTML gives %+v, %v", msg, err)
	}
}

func TestTemplateMistakes(t *testing.T) {
	for _, c := range []struct{ subject, text, html, want string }{
		{"", "x", "", "mail: NewTemplate was given no subject; give every message one."},
		{"x", "", "", "mail: NewTemplate was given no text; give every message a plain-text body."},
		{"{{.A", "x", "", "mail: the subject template does not parse ("},
		{"x", "{{if}}", "", "mail: the text template does not parse ("},
		{"x", "y", "<p>{{.A</p>", "mail: the HTML template does not parse ("},
	} {
		_, err := NewTemplate(c.subject, c.text, c.html)
		if err == nil || !strings.HasPrefix(err.Error(), c.want) {
			t.Errorf("NewTemplate(%q, %q, %q) = %v, want %q", c.subject, c.text, c.html, err, c.want)
		}
		if err != nil && strings.Contains(c.want, "does not parse") && !strings.HasSuffix(err.Error(), "; fix the template.") {
			t.Errorf("%q does not say what to do", err)
		}
	}

	tpl, err := NewTemplate("Hi {{.Name}}", "Hello {{.Name}}", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = tpl.Message("a@example.com", map[string]string{})
	if want := "mail: the subject template cannot be filled ("; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("a missing key gives %v, want %q", err, want)
	}

	broken, err := NewTemplate("Hi\n{{.Name}}", "Hello", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = broken.Message("a@example.com", map[string]string{"Name": "Ann"})
	if want := "mail: the subject template wrote a line break; keep the subject on one line."; err == nil || err.Error() != want {
		t.Errorf("a subject over two lines gives %v, want %q", err, want)
	}
}
