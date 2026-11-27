package middleware

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
)

// LogTo sends the request log, and the lines slog's default writes once the
// app has started, to w in place of standard error.
func LogTo(w io.Writer) Option { return func(m *module) { m.out = w } }

// readLog reads LOG: text (the default), json or off.
func readLog(v string) (string, error) {
	switch f := strings.ToLower(strings.TrimSpace(v)); f {
	case "":
		return "text", nil
	case "text", "json", "off":
		return f, nil
	}
	return "", fmt.Errorf("LOG is %q; set it to text, json or off in .env.", v)
}

// logger is the logger of the format, writing to w.
func logger(format string, w io.Writer) *slog.Logger {
	if format == "json" {
		return slog.New(slog.NewJSONHandler(w, nil))
	}
	return slog.New(slog.NewTextHandler(w, nil))
}

// previous is how slog's default and the standard logger were before Start
// set them, for Stop to put back.
type previous struct {
	slog  *slog.Logger
	set   *slog.Logger // the one Start set
	out   io.Writer
	flags int
}

// takeDefault makes l slog's default, which the standard logger then writes
// through too, and returns how they were.
func takeDefault(l *slog.Logger) *previous {
	p := &previous{slog: slog.Default(), set: l, out: log.Writer(), flags: log.Flags()}
	slog.SetDefault(l)
	return p
}

// restore puts slog's default and the standard logger back, unless
// something since has made another logger the default.
func (p *previous) restore() {
	if p == nil || slog.Default() != p.set {
		return
	}
	slog.SetDefault(p.slog)
	log.SetOutput(p.out)
	log.SetFlags(p.flags)
}

// logRequests writes one line per request once next has answered: method,
// path without its query, which may hold search text, status, bytes,
// milliseconds and the client's address; at info, or error for a 5xx. Before
// Start, and with LOG=off, it writes nothing.
func (m *module) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := m.requests
		if l == nil {
			next.ServeHTTP(w, r)
			return
		}
		begun := time.Now()
		rec := &recorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		ms := math.Round(float64(time.Since(begun).Microseconds())/10) / 100
		l.LogAttrs(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int64("bytes", rec.bytes),
			slog.Float64("ms", ms),
			slog.String("ip", m.app.ClientIP(r)),
		)
	})
}

// recorder is the answer's writer, keeping its status and the bytes of its
// body. It keeps Flush, and Unwrap gives the writer it wraps, so
// http.ResponseController reaches the connection through it.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *recorder) WriteHeader(code int) {
	if r.status == 0 && code >= 200 { // a 1xx, such as Early Hints, is not the answer
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// Flush sends what is written so far, when the writer it wraps can.
func (r *recorder) Flush() {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	http.NewResponseController(r.ResponseWriter).Flush()
}

func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
