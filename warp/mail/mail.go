// Package mail sends a Warp app's mail: printed to the log while an app is
// made, sent over SMTP once it is configured to.
//
// It is configured in .env, read when the app starts: MAIL is log (the
// default), which prints each message and keeps the last 20 for mailtest, or
// smtp, which sends them through MAIL_HOST at MAIL_PORT (default 587),
// signing in with MAIL_USER and MAIL_PASSWORD when both are set, from
// MAIL_FROM. Port 465 is TLS from the first byte; any other port must offer
// STARTTLS, except on a loopback host. Start does not connect, so an app
// starts while its mail server is down.
//
// Under smtp, Send puts a message on a queue in memory and returns: one
// sender sends them in turn, giving each try 30 seconds and three tries, 5
// and then 30 seconds apart, and logs one line naming the recipient and the
// subject of a message that could not go.
package mail

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/mail"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/flakerimi/weft/warp"
)

// Limits on a message and on what the mailer holds.
const (
	maxSubject = 200       // bytes
	maxBody    = 256 << 10 // bytes of Text, and of HTML
	queueSize  = 100       // messages waiting or being sent
	keep       = 20        // messages the log driver keeps
)

// Message is one message to one recipient. To is one address that net/mail
// reads, as "ann@example.com" or "Ann <ann@example.com>"; Subject is one
// line; Text is the plain-text body every message has; HTML, when not "",
// is sent beside it as the other form of the same message.
type Message struct{ To, Subject, Text, HTML string }

// Module is the mail module, named "mail". Registering provides the Mailer
// and reads nothing; its Start reads the MAIL settings and starts the sender,
// and its Stop sends what is queued, for as long as the app gives it.
func Module() warp.Module { return &module{mailer: &Mailer{}} }

// For returns the app's Mailer: what the mail module provided. Without the
// module it is nil, and the app has a mistake that says to list it.
func For(app *warp.App) *Mailer { return warp.Need[*Mailer](app, "mail") }

type module struct{ mailer *Mailer }

func (m *module) Name() string { return "mail" }

func (m *module) Register(app *warp.App) error {
	m.mailer.app = app
	warp.Provide(app, m.mailer)
	return nil
}

func (m *module) Start(ctx context.Context) error { return m.mailer.start(ctx) }

func (m *module) Stop(ctx context.Context) error { return m.mailer.stop(ctx) }

// Mailer sends the app's mail. Get it with For.
type Mailer struct {
	app *warp.App

	mu    sync.Mutex
	state int // notStarted, running or stopped
	s     settings
	kept  []Message // the log driver's last messages, oldest first

	// The smtp driver's queue and sender.
	queue    chan outgoing
	slots    chan struct{} // one per message waiting or being sent
	stopping chan struct{} // closed when Stop begins
	done     chan struct{} // closed when the sender returns
	cancel   context.CancelFunc
}

const (
	notStarted = iota
	running
	stopped
)

// settings are the MAIL values, read at Start.
type settings struct {
	driver     string // "log" or "smtp"
	host, port string
	user, pass string
	from       *mail.Address
	hello      string // APP_URL's host: the name the app gives itself in EHLO and in Message-ID
}

// readSettings reads the MAIL values from cfg. Every problem is its own
// sentence, all returned together.
func readSettings(cfg warp.Config) (settings, error) {
	s := settings{driver: "log", port: "587", hello: "localhost"}
	if u, err := url.Parse(cfg.URL); err == nil && u.Hostname() != "" {
		s.hello = u.Hostname()
	}
	var problems []error
	switch v := strings.ToLower(strings.TrimSpace(cfg.Get("MAIL"))); v {
	case "", "log":
	case "smtp":
		s.driver = "smtp"
	default:
		return s, fmt.Errorf("MAIL is %q; set it to log, which prints mail, or smtp, which sends it.", cfg.Get("MAIL"))
	}
	if s.driver == "smtp" {
		if s.host = strings.TrimSpace(cfg.Get("MAIL_HOST")); s.host == "" {
			problems = append(problems, errors.New("MAIL_HOST is not set; set it to your mail server, such as smtp.example.com."))
		}
		if v := cfg.Get("MAIL_PORT"); v != "" {
			if n, err := strconv.Atoi(v); err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != v {
				problems = append(problems, fmt.Errorf("MAIL_PORT is %q; set it to a number from 1 to 65535, such as 587.", v))
			} else {
				s.port = v
			}
		}
		s.user, s.pass = cfg.Get("MAIL_USER"), cfg.Get("MAIL_PASSWORD")
		switch {
		case s.user != "" && s.pass == "":
			problems = append(problems, errors.New("MAIL_USER is set and MAIL_PASSWORD is not; set both, or neither for a server that takes mail without signing in."))
		case s.user == "" && s.pass != "":
			problems = append(problems, errors.New("MAIL_PASSWORD is set and MAIL_USER is not; set both, or neither for a server that takes mail without signing in."))
		}
	}
	switch v := cfg.Get("MAIL_FROM"); {
	case v != "":
		if a, err := mail.ParseAddress(v); err == nil {
			s.from = a
		} else {
			problems = append(problems, fmt.Errorf("MAIL_FROM is %q; set it to the address mail comes from, such as Shop <noreply@shop.example.com>.", v))
		}
	case s.driver == "smtp":
		problems = append(problems, errors.New("MAIL_FROM is not set; set it to the address mail comes from, such as Shop <noreply@shop.example.com>."))
	default:
		s.from = &mail.Address{Address: "noreply@" + s.hello}
	}
	return s, errors.Join(problems...)
}

// start reads the settings and, under smtp, starts the sender. It connects
// to nothing.
func (m *Mailer) start(ctx context.Context) error {
	s, err := readSettings(m.app.Config())
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.s, m.state = s, running
	if s.driver == "log" {
		log.Print("mail is printed, not sent; set MAIL=smtp in .env to send it.")
		return nil
	}
	m.queue = make(chan outgoing, queueSize)
	m.slots = make(chan struct{}, queueSize)
	m.stopping = make(chan struct{})
	m.done = make(chan struct{})
	// The sender outlives Start's context, which Serve ends at the first
	// signal: Stop decides when it ends.
	run, cancel := context.WithCancel(context.WithoutCancel(ctx))
	m.cancel = cancel
	go m.run(run)
	return nil
}

// stop sends what is queued until ctx ends, then logs how many messages
// were not sent.
func (m *Mailer) stop(ctx context.Context) error {
	m.mu.Lock()
	was, driver := m.state, m.s.driver
	m.state = stopped
	m.mu.Unlock()
	if was != running || driver != "smtp" {
		return nil
	}
	close(m.stopping)
	select {
	case <-m.done:
	case <-ctx.Done():
		m.cancel()
		<-m.done
	}
	m.cancel()
	switch n := len(m.slots); n {
	case 0:
	case 1:
		slog.Error("mail: 1 message was not sent, since the app stopped.")
	default:
		slog.Error(fmt.Sprintf("mail: %d messages were not sent, since the app stopped.", n))
	}
	return nil
}

// Send checks msg and sends it: under log it prints and keeps it at once;
// under smtp it queues it and returns, and the sender sends it. A message
// the checks refuse is an error that says which part to fix, a mistake of
// the code that made it. A full queue is a warp.Unavailable, 503, which a
// handler returns as it is.
func (m *Mailer) Send(ctx context.Context, msg Message) error {
	if m == nil {
		return errors.New("mail: Send was called on no Mailer; list mail.Module() in app/modules.go.")
	}
	to, err := check(msg)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.state {
	case notStarted:
		return errors.New("mail: Send was called before the app started; send from a handler, or after Start.")
	case stopped:
		return errors.New("mail: Send was called after the app stopped; send while it runs.")
	}
	if m.s.driver == "log" {
		m.print(msg)
		return nil
	}
	select {
	case m.slots <- struct{}{}:
	default:
		return warp.Unavailable("The server is busy sending mail; try again in a moment.")
	}
	m.queue <- outgoing{to: to, subject: msg.Subject, data: m.compose(to, msg)} // never blocks: a slot was free
	return nil
}

// check refuses a message that cannot be sent as it is, and returns its
// recipient.
func check(msg Message) (*mail.Address, error) {
	to, err := mail.ParseAddress(msg.To)
	if err != nil || strings.ContainsAny(msg.To, "\r\n") {
		return nil, fmt.Errorf("mail: the message's To is %q; give it one email address.", msg.To)
	}
	switch {
	case msg.Subject == "":
		return nil, errors.New("mail: the message has no Subject; give it one.")
	case strings.ContainsAny(msg.Subject, "\r\n"):
		return nil, errors.New("mail: the message's Subject has a line break; keep it on one line.")
	case len(msg.Subject) > maxSubject:
		return nil, fmt.Errorf("mail: the message's Subject is %d bytes; keep it to %d.", len(msg.Subject), maxSubject)
	case msg.Text == "":
		return nil, errors.New("mail: the message has no Text; give it a plain-text body, which every message needs.")
	case len(msg.Text) > maxBody:
		return nil, fmt.Errorf("mail: the message's Text is %d bytes; keep it to 256 KiB.", len(msg.Text))
	case len(msg.HTML) > maxBody:
		return nil, fmt.Errorf("mail: the message's HTML is %d bytes; keep it to 256 KiB.", len(msg.HTML))
	}
	return to, nil
}

// Kept returns the last 20 messages the log driver printed, oldest first:
// what mailtest reads. Under smtp it is empty.
func (m *Mailer) Kept() []Message {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.kept)
}
