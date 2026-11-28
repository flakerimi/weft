package mail

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// printTo is where the log driver writes each message; a test sets it.
var printTo io.Writer = os.Stderr

// print is the log driver: it writes the message to standard error as it
// is, between "--- mail to <to>: <subject> ---" and "--- end of mail to
// <to> ---", and logs one line, mail printed, with the recipient and the
// subject; then it keeps the message among the last 20. m.mu is held.
//
// The message goes to standard error itself, not through the log, so its
// lines stay lines and a link in it can be copied from a terminal, whatever
// handler the log has (the middleware module's writes a message's line
// breaks as \n). Printing is the log driver's purpose, so a link's token is
// printed as it is; no other part of Warp prints one, and the log line
// holds no body.
func (m *Mailer) print(msg Message) {
	var b strings.Builder
	b.WriteString("--- mail to " + msg.To + ": " + msg.Subject + " ---\n")
	text := strings.ReplaceAll(msg.Text, "\r\n", "\n")
	b.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("--- end of mail to " + msg.To + " ---\n")
	io.WriteString(printTo, b.String())
	slog.Info("mail printed", "to", msg.To, "subject", msg.Subject)
	if len(m.kept) == keep {
		m.kept = append(m.kept[:0], m.kept[1:]...)
	}
	m.kept = append(m.kept, msg)
	m.count++
}
