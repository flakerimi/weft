package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/netip"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// How the sender tries.
const (
	tries     = 3
	tryLimit  = 30 * time.Second // a whole try: connecting, TLS, the commands and the message
	dialLimit = 10 * time.Second
)

// retryWaits are the waits before the second and the third try.
var retryWaits = [tries - 1]time.Duration{5 * time.Second, 30 * time.Second}

// The hooks a test swaps: where a connection goes, the certificates TLS
// trusts (nil is the system's), and how the sender waits between tries.
var (
	dial     = (&net.Dialer{}).DialContext
	tlsRoots *x509.CertPool
	wait     = sleep
)

// implicitTLSPort is the port that speaks TLS from the first byte.
const implicitTLSPort = "465"

// sleep waits d, or until ctx ends, and returns ctx's error then.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// outgoing is a queued message: its recipient, its subject for the log, and
// the message as it goes on the wire.
type outgoing struct {
	to      *mail.Address
	subject string
	data    []byte
}

// run is the sender: it sends each queued message in turn until Stop begins,
// then what is left in the queue, and returns; it returns at once when ctx
// ends. A message is given back its slot once it is sent or given up on, so
// the slots taken when it returns are the messages not sent.
func (m *Mailer) run(ctx context.Context) {
	defer close(m.done)
	next := func(o outgoing) bool {
		if !m.deliver(ctx, o) {
			return false
		}
		<-m.slots
		return true
	}
	for {
		select {
		case o := <-m.queue:
			if !next(o) {
				return
			}
		case <-m.stopping:
			for {
				select {
				case o := <-m.queue:
					if !next(o) {
						return
					}
				default:
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// deliver sends o, trying up to three times, and logs one line when it
// cannot. It returns false, logging nothing, when ctx ends first.
func (m *Mailer) deliver(ctx context.Context, o outgoing) bool {
	var failed *sendError
	for try := range tries {
		if try > 0 && wait(ctx, retryWaits[try-1]) != nil {
			return false
		}
		if failed = m.try(ctx, o); failed == nil {
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		if failed.permanent {
			break
		}
	}
	// The recipient and the subject, never the body, which may hold a token.
	slog.Error(fmt.Sprintf("mail to %s (%q) was not sent: %s", o.to.Address, o.subject, failed.reason))
	return true
}

// sendError is why a try failed: the sentence the log line ends with, and
// whether trying again could change it.
type sendError struct {
	reason    string
	permanent bool
}

// try sends o once, in 30 seconds at most.
func (m *Mailer) try(parent context.Context, o outgoing) *sendError {
	s := m.s
	addr := net.JoinHostPort(s.host, s.port)
	ctx, cancel := context.WithTimeout(parent, tryLimit)
	defer cancel()
	fail := func(step string, err error) *sendError { return failure(ctx, addr, step, err) }

	dctx, dcancel := context.WithTimeout(ctx, dialLimit)
	conn, err := dial(dctx, "tcp", addr)
	dcancel()
	if err != nil {
		return fail("connect", err)
	}
	defer conn.Close()
	// net/smtp knows no context: the connection's deadline ends a try that
	// runs out of time, and closing it ends one that is stopped.
	defer context.AfterFunc(ctx, func() { conn.Close() })()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	cfg := &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12, RootCAs: tlsRoots}
	implicit := s.port == implicitTLSPort
	if implicit {
		tc := tls.Client(conn, cfg)
		if err := tc.HandshakeContext(ctx); err != nil {
			return fail("tls", err)
		}
		conn = tc
	}
	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fail("greet", err)
	}
	defer c.Close()
	if err := c.Hello(s.hello); err != nil {
		return fail("hello", err)
	}
	if !implicit {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(cfg); err != nil {
				return fail("tls", err)
			}
		} else if !loopback(s.host) {
			// A loopback host without STARTTLS is used as it is, MAIL_USER and
			// MAIL_PASSWORD included, on purpose: a relay there is a development
			// server, or a TLS-terminating sidecar or SSH tunnel that needs the
			// provider's password while it does the TLS, and loopback traffic
			// does not leave the machine. Only localhost and loopback addresses
			// count, and no name is looked up, so a DNS answer cannot widen it.
			return &sendError{reason: fmt.Sprintf("the mail server at %s offers no STARTTLS; use port 465 or a server that does.", addr), permanent: true}
		}
	}
	if s.user != "" {
		if err := c.Auth(smtp.PlainAuth("", s.user, s.pass, s.host)); err != nil {
			return fail("auth", err)
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return fail("mail", err)
	}
	if err := c.Rcpt(o.to.Address); err != nil {
		return fail("rcpt", err)
	}
	w, err := c.Data()
	if err != nil {
		return fail("data", err)
	}
	if _, err := w.Write(o.data); err != nil {
		return fail("data", err)
	}
	if err := w.Close(); err != nil {
		return fail("data", err)
	}
	c.Quit() // the message was taken; how the server says goodbye does not matter
	return nil
}

// failure says why a step of a try failed, as the end of the log line.
func failure(ctx context.Context, addr, step string, err error) *sendError {
	var answer *textproto.Error
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return &sendError{reason: fmt.Sprintf("the mail server at %s did not finish within 30 s; check that it is up and takes mail on that port.", addr)}
	case errors.As(err, &answer) && answer.Code >= 400 && answer.Code < 500:
		return &sendError{reason: fmt.Sprintf("the mail server at %s answered %q to the last of %d tries; check its log for why.", addr, serverReply(answer), tries)}
	case errors.As(err, &answer) && step == "auth":
		return &sendError{reason: fmt.Sprintf("the mail server at %s refused to sign in (%q); check MAIL_USER and MAIL_PASSWORD.", addr, serverReply(answer)), permanent: true}
	case errors.As(err, &answer) && step == "rcpt":
		return &sendError{reason: fmt.Sprintf("the mail server at %s refused the address (%q); check that it is right.", addr, serverReply(answer)), permanent: true}
	case errors.As(err, &answer):
		return &sendError{reason: fmt.Sprintf("the mail server at %s answered %q; check its log for why.", addr, serverReply(answer)), permanent: true}
	case step == "connect":
		return &sendError{reason: fmt.Sprintf("the mail server at %s could not be reached (%v); check MAIL_HOST and MAIL_PORT, and that it is up.", addr, err)}
	case step == "tls":
		var unknown x509.UnknownAuthorityError
		var hostname x509.HostnameError
		var invalid x509.CertificateInvalidError
		permanent := errors.As(err, &unknown) || errors.As(err, &hostname) || errors.As(err, &invalid)
		return &sendError{reason: fmt.Sprintf("TLS with the mail server at %s failed (%v); check that MAIL_HOST is the name on its certificate.", addr, err), permanent: permanent}
	case step == "auth":
		// net/smtp's own refusal, such as a server that offers no AUTH.
		return &sendError{reason: fmt.Sprintf("the app could not sign in to the mail server at %s (%v); check MAIL_USER and MAIL_PASSWORD, or unset both.", addr, err), permanent: true}
	}
	return &sendError{reason: fmt.Sprintf("the connection to the mail server at %s broke off (%v); check its log for why.", addr, err)}
}

// loopback reports whether host is this machine, where a development relay
// may take mail without TLS: localhost, or a loopback address. No name is
// looked up.
func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	addr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	return err == nil && addr.IsLoopback()
}

// compose writes msg as it goes on the wire: the headers, then the text, or
// a multipart/alternative of the text and the HTML, each quoted-printable.
// m.mu is held.
func (m *Mailer) compose(to *mail.Address, msg Message) []byte {
	var b bytes.Buffer
	header := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	id := make([]byte, 16)
	rand.Read(id) // never fails, as crypto/rand documents
	header("From", m.s.from.String())
	header("To", to.String())
	header("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", "<"+hex.EncodeToString(id)+"@"+m.s.hello+">")
	header("MIME-Version", "1.0")
	if msg.HTML == "" {
		header("Content-Type", "text/plain; charset=utf-8")
		header("Content-Transfer-Encoding", "quoted-printable")
		b.WriteString("\r\n")
		writeQP(&b, msg.Text)
		return b.Bytes()
	}
	var body bytes.Buffer
	parts := multipart.NewWriter(&body)
	header("Content-Type", "multipart/alternative; boundary="+parts.Boundary())
	b.WriteString("\r\n")
	for _, p := range []struct{ kind, text string }{{"text/plain", msg.Text}, {"text/html", msg.HTML}} {
		w, _ := parts.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {p.kind + "; charset=utf-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		}) // a bytes.Buffer does not fail
		writeQP(w, p.text)
	}
	parts.Close()
	b.Write(body.Bytes())
	return b.Bytes()
}

// writeQP writes text to w in quoted-printable, its lines ending in CRLF.
func writeQP(w interface{ Write([]byte) (int, error) }, text string) {
	qp := quotedprintable.NewWriter(w)
	qp.Write([]byte(text))
	qp.Close()
}

// serverReply is a server's answer as it gave it, code and text, for a
// sentence to quote. It never uses textproto.Error's own Error, whose form a
// Go release may change (Go 1.26.11 quotes the text).
func serverReply(e *textproto.Error) string {
	return fmt.Sprintf("%03d %s", e.Code, e.Msg)
}
