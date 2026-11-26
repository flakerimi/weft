package warp

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP returns the address of the client that sent r, without its port
// or an IPv6 zone, an IPv4 address inside IPv6 (::ffff:1.2.3.4) written as
// IPv4. With TRUST_PROXY on it is the last address of X-Forwarded-For, which
// is the one the proxy in front of the app added; when that header is
// missing or its last entry is not an address (empty, "unknown"), and always
// with TRUST_PROXY off, it is the address of the connection. Only the last
// entry is read, so a long header costs one scan.
func (a *App) ClientIP(r *http.Request) string {
	if addr, ok := a.clientAddr(r); ok {
		return addr.String()
	}
	return r.RemoteAddr
}

// ClientKey returns what a limiter counts the client that sent r by: its
// address, as ClientIP gives it, for IPv4, and the /64 its address is in for
// IPv6 ("2001:db8:1:2::/64"), since one client is commonly given a whole /64
// and can send from any address in it. When the address cannot be read it is
// ClientIP's answer.
func (a *App) ClientKey(r *http.Request) string {
	addr, ok := a.clientAddr(r)
	switch {
	case !ok:
		return r.RemoteAddr
	case addr.Is4():
		return addr.String()
	}
	prefix, _ := addr.Prefix(64)
	return prefix.String()
}

// clientAddr is ClientIP's address, when there is one.
func (a *App) clientAddr(r *http.Request) (netip.Addr, bool) {
	if a.cfg.TrustProxy {
		if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
			if addr, ok := parseAddr(lastEntry(values[len(values)-1])); ok {
				return addr, true
			}
		}
	}
	return parseAddr(r.RemoteAddr)
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
// "[2001:db8::1]:80") or not, in brackets or not, and returns it without its
// port and zone, an IPv4 address inside IPv6 as IPv4. A bracket is taken
// only with its pair.
func parseAddr(s string) (netip.Addr, bool) {
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	} else if len(s) > 1 && s[0] == '[' && s[len(s)-1] == ']' {
		s = s[1 : len(s)-1]
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.WithZone("").Unmap(), true
}
