package mail

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTP is a mail server for one test, on a port of its own: it offers
// STARTTLS or not, speaks TLS from the first byte or not, refuses the first
// fails messages with a 451, or stalls, never greeting.
type fakeSMTP struct {
	t        *testing.T
	ln       net.Listener
	cfg      *tls.Config
	startTLS bool
	implicit bool
	fails    int
	stall    bool

	mu       sync.Mutex
	seen     []string // the commands, in order, each with " (tls)" when over TLS
	messages []string
	tries    int
	got      chan struct{}
}

// testCert is a certificate for smtp.example.com, localhost and 127.0.0.1,
// made once, and a pool that trusts it.
var testCert = sync.OnceValues(func() (tls.Certificate, *x509.CertPool) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "smtp.example.com"},
		DNSNames:              []string{"smtp.example.com", "localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
})

// newFakeSMTP starts a fake server that offers STARTTLS, and points the
// package's dialling and trust at it until the test ends, whatever host and
// port the settings name.
func newFakeSMTP(t *testing.T, set ...func(*fakeSMTP)) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cert, pool := testCert()
	s := &fakeSMTP{t: t, ln: ln, cfg: &tls.Config{Certificates: []tls.Certificate{cert}}, startTLS: true, got: make(chan struct{}, 200)}
	for _, f := range set {
		f(s)
	}
	t.Cleanup(func() { ln.Close() })
	go s.serve()
	swapDial(t, func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, ln.Addr().String())
	})
	roots := tlsRoots
	tlsRoots = pool
	t.Cleanup(func() { tlsRoots = roots })
	waits := wait
	wait = func(ctx context.Context, d time.Duration) error { return ctx.Err() }
	t.Cleanup(func() { wait = waits })
	return s
}

// swapDial puts d in the place of the package's dial until the test ends.
func swapDial(t *testing.T, d func(ctx context.Context, network, addr string) (net.Conn, error)) {
	old := dial
	dial = d
	t.Cleanup(func() { dial = old })
}

// vars are settings that send mail to smtp.example.com, which the fake is.
func (s *fakeSMTP) vars() map[string]string {
	return map[string]string{
		"MAIL": "smtp", "MAIL_HOST": "smtp.example.com", "MAIL_FROM": "Shop <noreply@shop.example.com>",
		"MAIL_USER": "shop", "MAIL_PASSWORD": "secret", "APP_URL": "https://shop.example.com",
	}
}

func (s *fakeSMTP) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.session(conn)
	}
}

func (s *fakeSMTP) note(cmd string, overTLS bool) {
	if overTLS {
		cmd += " (tls)"
	}
	s.mu.Lock()
	s.seen = append(s.seen, cmd)
	s.mu.Unlock()
}

func (s *fakeSMTP) session(conn net.Conn) {
	defer conn.Close()
	if s.stall {
		io.Copy(io.Discard, conn)
		return
	}
	overTLS := false
	if s.implicit {
		conn = tls.Server(conn, s.cfg)
		overTLS = true
	}
	tp := textproto.NewConn(conn)
	tp.PrintfLine("220 smtp.example.com ESMTP fake")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb := strings.ToUpper(strings.Fields(line + " x")[0])
		switch verb {
		case "EHLO", "HELO":
			s.note(line, overTLS)
			if s.startTLS && !overTLS {
				tp.PrintfLine("250-smtp.example.com")
				tp.PrintfLine("250-STARTTLS")
			} else {
				tp.PrintfLine("250-smtp.example.com")
			}
			tp.PrintfLine("250 AUTH PLAIN")
		case "STARTTLS":
			s.note(verb, overTLS)
			tp.PrintfLine("220 go ahead")
			tc := tls.Server(conn, s.cfg)
			if err := tc.Handshake(); err != nil {
				return
			}
			conn, overTLS = tc, true
			tp = textproto.NewConn(conn)
		case "AUTH":
			s.note("AUTH", overTLS)
			tp.PrintfLine("235 2.7.0 signed in")
		case "MAIL", "RCPT":
			s.note(line, overTLS)
			tp.PrintfLine("250 ok")
		case "DATA":
			s.note(verb, overTLS)
			tp.PrintfLine("354 go ahead")
			b, err := tp.ReadDotBytes()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.tries++
			failing := s.tries <= s.fails
			if !failing {
				s.messages = append(s.messages, string(b))
			}
			s.mu.Unlock()
			if failing {
				tp.PrintfLine("451 4.3.0 try again later")
			} else {
				tp.PrintfLine("250 queued")
				s.got <- struct{}{}
			}
		case "QUIT":
			tp.PrintfLine("221 bye")
			return
		default:
			tp.PrintfLine("250 ok")
		}
	}
}

func (s *fakeSMTP) received() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.messages...)
}

func (s *fakeSMTP) commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

// await waits for n messages, failing the test after five seconds.
func (s *fakeSMTP) await(n int) []string {
	s.t.Helper()
	for range n {
		select {
		case <-s.got:
		case <-time.After(5 * time.Second):
			s.t.Fatalf("the fake mail server got %d messages, want %d; it saw %q", len(s.received()), n, s.commands())
		}
	}
	return s.received()
}

func TestSMTPStartTLSAndAuth(t *testing.T) {
	captureLog(t)
	srv := newFakeSMTP(t)
	_, m := started(t, srv.vars())
	err := m.Send(context.Background(), Message{
		To: "Ann <ann@example.com>", Subject: "Réinitialiser le mot de passe",
		Text: "Open https://shop.example.com/password#token=abc to reset it.\n.\nA line of a dot alone.",
		HTML: "<p>Open <a href=\"https://shop.example.com/password#token=abc\">the link</a>.</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := srv.await(1)
	want := []string{
		"EHLO shop.example.com", "STARTTLS", "EHLO shop.example.com (tls)", "AUTH (tls)",
		"MAIL FROM:<noreply@shop.example.com> (tls)", "RCPT TO:<ann@example.com> (tls)", "DATA (tls)",
	}
	if c := srv.commands(); strings.Join(c, "|") != strings.Join(want, "|") {
		t.Errorf("the server saw\n%q\nwant\n%q", c, want)
	}

	msg, err := mail.ReadMessage(strings.NewReader(got[0]))
	if err != nil {
		t.Fatalf("the message does not parse: %v\n%s", err, got[0])
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || subject != "Réinitialiser le mot de passe" {
		t.Errorf("Subject %q decodes to %q, %v", msg.Header.Get("Subject"), subject, err)
	}
	if msg.Header.Get("Subject") == subject {
		t.Error("a subject that is not ASCII is sent as it is, not Q-encoded")
	}
	for k, v := range map[string]string{"From": `"Shop" <noreply@shop.example.com>`, "To": `"Ann" <ann@example.com>`, "Mime-Version": "1.0"} {
		if got := msg.Header.Get(k); got != v {
			t.Errorf("%s is %q, want %q", k, got, v)
		}
	}
	if _, err := msg.Header.Date(); err != nil {
		t.Errorf("Date %q does not parse: %v", msg.Header.Get("Date"), err)
	}
	if id := msg.Header.Get("Message-Id"); !strings.HasSuffix(id, "@shop.example.com>") || len(id) != len("<@shop.example.com>")+32 {
		t.Errorf("Message-ID is %q, want <32 hex@shop.example.com>", id)
	}
	media, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || media != "multipart/alternative" {
		t.Fatalf("Content-Type is %q, want multipart/alternative", msg.Header.Get("Content-Type"))
	}
	parts := multipart.NewReader(msg.Body, params["boundary"])
	var types, bodies []string
	for {
		p, err := parts.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(p)
		types = append(types, p.Header.Get("Content-Type"))
		bodies = append(bodies, string(b))
	}
	if len(types) != 2 || types[0] != "text/plain; charset=utf-8" || types[1] != "text/html; charset=utf-8" {
		t.Fatalf("the parts are %q, want the text and then the HTML", types)
	}
	if want := "Open https://shop.example.com/password#token=abc to reset it.\n.\nA line of a dot alone."; strings.ReplaceAll(bodies[0], "\r\n", "\n") != want {
		t.Errorf("the text part is %q, want %q", bodies[0], want)
	}
	if !strings.Contains(bodies[1], `href="https://shop.example.com/password#token=abc"`) {
		t.Errorf("the HTML part is %q", bodies[1])
	}
}

func TestSMTPTextAlone(t *testing.T) {
	captureLog(t)
	srv := newFakeSMTP(t)
	_, m := started(t, srv.vars())
	if err := m.Send(context.Background(), Message{To: "ann@example.com", Subject: "Hi", Text: "Hello, Ann."}); err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(strings.NewReader(srv.await(1)[0]))
	if err != nil {
		t.Fatal(err)
	}
	if ct, cte := msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"); ct != "text/plain; charset=utf-8" || cte != "quoted-printable" {
		t.Errorf("a text alone is %q in %q, want text/plain in quoted-printable", ct, cte)
	}
	if subject := msg.Header.Get("Subject"); subject != "Hi" {
		t.Errorf("an ASCII subject is sent as %q", subject)
	}
}

func TestSMTPImplicitTLS(t *testing.T) {
	captureLog(t)
	srv := newFakeSMTP(t, func(s *fakeSMTP) { s.implicit = true })
	vars := srv.vars()
	vars["MAIL_PORT"] = "465"
	_, m := started(t, vars)
	if err := m.Send(context.Background(), Message{To: "ann@example.com", Subject: "Hi", Text: "x"}); err != nil {
		t.Fatal(err)
	}
	srv.await(1)
	want := []string{"EHLO shop.example.com (tls)", "AUTH (tls)", "MAIL FROM:<noreply@shop.example.com> (tls)", "RCPT TO:<ann@example.com> (tls)", "DATA (tls)"}
	if c := srv.commands(); strings.Join(c, "|") != strings.Join(want, "|") {
		t.Errorf("the server saw\n%q\nwant\n%q", c, want)
	}
}

func TestSMTPRefusesNoStartTLS(t *testing.T) {
	logs := captureLog(t)
	srv := newFakeSMTP(t, func(s *fakeSMTP) { s.startTLS = false })
	app, m := started(t, srv.vars())
	if err := m.Send(context.Background(), Message{To: "a@example.com", Subject: "Reset your password", Text: "#token=secret"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := `mail to a@example.com ("Reset your password") was not sent: the mail server at smtp.example.com:587 offers no STARTTLS; use port 465 or a server that does.`
	if !strings.Contains(logs.String(), want) {
		t.Errorf("the log has\n%s\nwant\n%s", logs, want)
	}
	if strings.Contains(logs.String(), "secret") {
		t.Errorf("the log has the body: %s", logs)
	}
	if c := srv.commands(); strings.Join(c, "|") != "EHLO shop.example.com" {
		t.Errorf("the server saw %q, want only EHLO: nothing in the clear", c)
	}
}

func TestSMTPPlainOnLoopback(t *testing.T) {
	captureLog(t)
	srv := newFakeSMTP(t, func(s *fakeSMTP) { s.startTLS = false })
	vars := srv.vars()
	vars["MAIL_HOST"] = "localhost"
	delete(vars, "MAIL_USER")
	delete(vars, "MAIL_PASSWORD")
	_, m := started(t, vars)
	if err := m.Send(context.Background(), Message{To: "ann@example.com", Subject: "Hi", Text: "x"}); err != nil {
		t.Fatal(err)
	}
	srv.await(1)
	want := []string{"EHLO shop.example.com", "MAIL FROM:<noreply@shop.example.com>", "RCPT TO:<ann@example.com>", "DATA"}
	if c := srv.commands(); strings.Join(c, "|") != strings.Join(want, "|") {
		t.Errorf("the server saw\n%q\nwant\n%q", c, want)
	}

	if !loopback("127.0.0.1") || !loopback("::1") || !loopback("LocalHost") || loopback("smtp.example.com") || loopback("10.0.0.1") {
		t.Error("loopback tells hosts apart wrongly")
	}
}

func TestSMTPRetries(t *testing.T) {
	logs := captureLog(t)
	srv := newFakeSMTP(t, func(s *fakeSMTP) { s.fails = 2 })
	var mu sync.Mutex
	var waited []time.Duration
	wait = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		waited = append(waited, d)
		mu.Unlock()
		return ctx.Err()
	}
	_, m := started(t, srv.vars())
	if err := m.Send(context.Background(), Message{To: "ann@example.com", Subject: "Hi", Text: "x"}); err != nil {
		t.Fatal(err)
	}
	srv.await(1)
	srv.mu.Lock()
	tries := srv.tries
	srv.mu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	if tries != 3 || fmt.Sprint(waited) != "[5s 30s]" {
		t.Errorf("%d tries with waits %v, want 3 with [5s 30s]", tries, waited)
	}
	if strings.Contains(logs.String(), "not sent") {
		t.Errorf("a message that went on its third try is logged: %s", logs)
	}

	// Three failures: one line.
	logs = captureLog(t)
	srv2 := newFakeSMTP(t, func(s *fakeSMTP) { s.fails = 3 })
	app, m := started(t, srv2.vars())
	m.Send(context.Background(), Message{To: "bob@example.com", Subject: "Welcome", Text: "x"})
	app.Stop(context.Background())
	want := `mail to bob@example.com ("Welcome") was not sent: the mail server at smtp.example.com:587 answered "451 4.3.0 try again later" to the last of 3 tries; check its log for why.`
	if !strings.Contains(logs.String(), want) || strings.Count(logs.String(), "was not sent") != 1 {
		t.Errorf("the log has\n%s\nwant once\n%s", logs, want)
	}
}
