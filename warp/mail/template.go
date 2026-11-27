package mail

import (
	"errors"
	"fmt"
	htemplate "html/template"
	"strings"
	ttemplate "text/template"
)

// Template makes messages of one kind, such as a password reset, from data.
// The subject and the text are text/template, which escapes nothing; the
// HTML is html/template, which escapes what the data puts in it. A key the
// data does not have is a mistake, not an empty value.
type Template struct {
	subject *ttemplate.Template
	text    *ttemplate.Template
	html    *htemplate.Template // nil when the messages have no HTML
}

// NewTemplate parses the three parts of a template; html may be "", for
// messages of text alone.
func NewTemplate(subject, text, html string) (*Template, error) {
	switch {
	case subject == "":
		return nil, errors.New("mail: NewTemplate was given no subject; give every message one.")
	case text == "":
		return nil, errors.New("mail: NewTemplate was given no text; give every message a plain-text body.")
	}
	t := &Template{}
	var err error
	if t.subject, err = ttemplate.New("subject").Option("missingkey=error").Parse(subject); err != nil {
		return nil, fmt.Errorf("mail: the subject template does not parse (%v); fix the template.", err)
	}
	if t.text, err = ttemplate.New("text").Option("missingkey=error").Parse(text); err != nil {
		return nil, fmt.Errorf("mail: the text template does not parse (%v); fix the template.", err)
	}
	if html != "" {
		if t.html, err = htemplate.New("html").Option("missingkey=error").Parse(html); err != nil {
			return nil, fmt.Errorf("mail: the HTML template does not parse (%v); fix the template.", err)
		}
	}
	return t, nil
}

// Message fills the template with data and returns the message to to. The
// subject is trimmed, and must come out on one line.
func (t *Template) Message(to string, data any) (Message, error) {
	var subject, text, html strings.Builder
	if err := t.subject.Execute(&subject, data); err != nil {
		return Message{}, fmt.Errorf("mail: the subject template cannot be filled (%v); give it the data it reads.", err)
	}
	if err := t.text.Execute(&text, data); err != nil {
		return Message{}, fmt.Errorf("mail: the text template cannot be filled (%v); give it the data it reads.", err)
	}
	if t.html != nil {
		if err := t.html.Execute(&html, data); err != nil {
			return Message{}, fmt.Errorf("mail: the HTML template cannot be filled (%v); give it the data it reads.", err)
		}
	}
	s := strings.TrimSpace(subject.String())
	if strings.ContainsAny(s, "\r\n") {
		return Message{}, errors.New("mail: the subject template wrote a line break; keep the subject on one line.")
	}
	return Message{To: to, Subject: s, Text: text.String(), HTML: html.String()}, nil
}
