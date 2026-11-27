package mail

import (
	"log"
	"strings"
)

// print is the log driver: it writes "mail to <to>: <subject>" and the text,
// each of its lines indented two spaces, through the standard logger, and
// keeps the message among the last 20. m.mu is held.
//
// Printing is the log driver's purpose, so a link's token is printed as it
// is; no other part of Warp prints one.
func (m *Mailer) print(msg Message) {
	var b strings.Builder
	b.WriteString("mail to ")
	b.WriteString(msg.To)
	b.WriteString(": ")
	b.WriteString(msg.Subject)
	for line := range strings.SplitSeq(strings.ReplaceAll(msg.Text, "\r\n", "\n"), "\n") {
		b.WriteString("\n  ")
		b.WriteString(line)
	}
	log.Print(b.String())
	if len(m.kept) == keep {
		m.kept = append(m.kept[:0], m.kept[1:]...)
	}
	m.kept = append(m.kept, msg)
}
