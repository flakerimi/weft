package auth

import (
	"context"
	"errors"
	"fmt"
	htemplate "html/template"
	"strings"
	ttemplate "text/template"

	"github.com/flakerimi/weft/warp"
)

// The login mails its links through the mail module, warp/mail (Task B1 of
// the accounts plan). Until that package is merged into this branch, this
// file stands in for the three parts of it the login uses, with their
// shapes: mail.Message is letter, (*mail.Mailer).Send is postman, and
// mail.NewTemplate is newLetterTemplate. Merging B1 replaces the file:
//
//   - letter becomes mail.Message and *letterTemplate *mail.Template;
//   - Register calls mail.For(app) before anything else and keeps the
//     *mail.Mailer, so an app without the mail module has the mistake
//     `warp: module "auth" needs the "mail" module; list mail.Module() before
//     it in app/modules.go.`, and send calls its Send;
//   - parseMails calls mail.NewTemplate, and message is (*mail.Template).Message.
//
// Until then the postman is looked for when the login first sends, so an app
// that lists no mail module, as none can yet, registers and starts.

// letter has the shape of mail.Message: one recipient; HTML may be "".
type letter = struct{ To, Subject, Text, HTML string }

// postman has the shape of the one method of *mail.Mailer the login uses.
// Unnamed, it is one type wherever it is written, so a test's mail module
// can provide it.
type postman = interface {
	Send(ctx context.Context, msg letter) error
}

// send queues msg with the app's mail module.
func (m *module) send(ctx context.Context, msg letter) error {
	m.postOnce.Do(func() { m.post = warp.Need[postman](m.app, "mail") })
	if m.post == nil {
		return errors.New("auth: no mail module sends the login's mail; list mail.Module() before auth.Module() in app/modules.go.")
	}
	return m.post.Send(ctx, msg)
}

// letterTemplate has the shape of mail.Template: the subject and the text
// are text/template, the HTML html/template, and a key the data lacks is a
// mistake.
type letterTemplate struct {
	subject, text *ttemplate.Template
	html          *htemplate.Template
}

// newLetterTemplate parses the three parts, as mail.NewTemplate does.
func newLetterTemplate(subject, text, html string) (*letterTemplate, error) {
	t := &letterTemplate{}
	var err error
	if t.subject, err = ttemplate.New("subject").Option("missingkey=error").Parse(subject); err != nil {
		return nil, fmt.Errorf("the subject template does not parse (%v)", err)
	}
	if t.text, err = ttemplate.New("text").Option("missingkey=error").Parse(text); err != nil {
		return nil, fmt.Errorf("the text template does not parse (%v)", err)
	}
	if html != "" {
		if t.html, err = htemplate.New("html").Option("missingkey=error").Parse(html); err != nil {
			return nil, fmt.Errorf("the HTML template does not parse (%v)", err)
		}
	}
	return t, nil
}

// message fills the template for to, as (*mail.Template).Message does: the
// subject trimmed and on one line.
func (t *letterTemplate) message(to string, data any) (letter, error) {
	var subject, text, html strings.Builder
	if err := t.subject.Execute(&subject, data); err != nil {
		return letter{}, err
	}
	if err := t.text.Execute(&text, data); err != nil {
		return letter{}, err
	}
	if t.html != nil {
		if err := t.html.Execute(&html, data); err != nil {
			return letter{}, err
		}
	}
	s := strings.TrimSpace(subject.String())
	if strings.ContainsAny(s, "\r\n") {
		return letter{}, errors.New("the subject template wrote a line break")
	}
	return letter{To: to, Subject: s, Text: text.String(), HTML: html.String()}, nil
}
