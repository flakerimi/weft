package warp

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP returns the address of the client that sent r, without its port.
// With TRUST_PROXY on it is the last address of X-Forwarded-For, which is the
// one the proxy in front of the app added; when that header is missing or its
// last entry is not an address, and always with TRUST_PROXY off, it is the
// address of the connection. Only the last entry is read, so a long header
// costs one scan.
func (a *App) ClientIP(r *http.Request) string {
	if a.cfg.TrustProxy {
		if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
			if ip, ok := parseAddr(lastEntry(values[len(values)-1])); ok {
				return ip
			}
		}
	}
	if ip, ok := parseAddr(r.RemoteAddr); ok {
		return ip
	}
	return r.RemoteAddr
}

// IsHTTPS reports whether the client reached the app over HTTPS: the
// connection is TLS, or, with TRUST_PROXY on, the last entry of
// X-Forwarded-Proto is https.
func (a *App) IsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if !a.cfg.TrustProxy {
		return false
	}
	values := r.Header.Values("X-Forwarded-Proto")
	return len(values) > 0 && strings.EqualFold(lastEntry(values[len(values)-1]), "https")
}

// lastEntry is the last comma-separated entry of a header value, trimmed.
func lastEntry(value string) string {
	return strings.TrimSpace(value[strings.LastIndexByte(value, ',')+1:])
}

// parseAddr reads an address that may carry a port ("192.0.2.1:80",
// "[2001:db8::1]:80") or not, and returns it without one.
func parseAddr(s string) (string, bool) {
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	addr, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(s, "["), "]"))
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}
