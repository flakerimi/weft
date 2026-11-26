package warp

import (
	"crypto/tls"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientIPAndHTTPS(t *testing.T) {
	long := strings.Repeat("10.0.0.1, ", 100_000) + "203.0.113.9"
	for _, c := range []struct {
		name   string
		trust  bool
		remote string
		tls    bool
		header []string // name, value pairs
		ip     string
		https  bool
	}{
		{name: "the connection", remote: "192.0.2.1:4321", ip: "192.0.2.1"},
		{name: "IPv6 connection", remote: "[2001:db8::1]:4321", ip: "2001:db8::1"},
		{name: "TLS connection", remote: "192.0.2.1:4321", tls: true, ip: "192.0.2.1", https: true},
		{name: "forged headers ignored", remote: "192.0.2.1:4321",
			header: []string{"X-Forwarded-For", "203.0.113.9", "X-Forwarded-Proto", "https"}, ip: "192.0.2.1"},
		{name: "trusted", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "198.51.100.1, 203.0.113.9", "X-Forwarded-Proto", "https"}, ip: "203.0.113.9", https: true},
		{name: "trusted, last of several lines", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "198.51.100.1", "X-Forwarded-For", "203.0.113.7 , 203.0.113.9"}, ip: "203.0.113.9"},
		{name: "trusted, a port stripped", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "203.0.113.9:5555"}, ip: "203.0.113.9"},
		{name: "trusted, IPv6", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "2001:db8::7"}, ip: "2001:db8::7"},
		{name: "trusted, IPv6 with a port", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "[2001:db8::7]:5555"}, ip: "2001:db8::7"},
		{name: "trusted, not an address", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "203.0.113.9, nonsense"}, ip: "10.0.0.2"},
		{name: "trusted, empty", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", ""}, ip: "10.0.0.2"},
		{name: "trusted, a long chain", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", long}, ip: "203.0.113.9"},
		{name: "trusted, no headers", trust: true, remote: "10.0.0.2:80", ip: "10.0.0.2"},
		{name: "trusted, http", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-Proto", "http"}, ip: "10.0.0.2"},
		{name: "trusted, HTTPS in capitals", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-Proto", "HTTPS"}, ip: "10.0.0.2", https: true},
		{name: "trusted, TLS without a header", trust: true, remote: "10.0.0.2:80", tls: true, ip: "10.0.0.2", https: true},
		{name: "a remote address without a port", remote: "192.0.2.1", ip: "192.0.2.1"},
		{name: "IPv6 with a zone", remote: "[fe80::1%eth0]:4321", ip: "fe80::1"},
		{name: "IPv6 with a zone, no port", remote: "fe80::1%eth0", ip: "fe80::1"},
		{name: "IPv4 in IPv6", remote: "[::ffff:1.2.3.4]:4321", ip: "1.2.3.4"},
		{name: "trusted, IPv6 with a zone", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "fe80::7%25en0, fe80::9%en0"}, ip: "fe80::9"},
		{name: "trusted, IPv4 in IPv6", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "::ffff:1.2.3.4"}, ip: "1.2.3.4"},
		{name: "trusted, a trailing comma", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "203.0.113.9,"}, ip: "10.0.0.2"},
		{name: "trusted, unknown", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "unknown"}, ip: "10.0.0.2"},
		{name: "trusted, an opening bracket alone", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "[2001:db8::7"}, ip: "10.0.0.2"},
		{name: "trusted, a closing bracket alone", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "2001:db8::7]"}, ip: "10.0.0.2"},
		{name: "trusted, brackets without a port", trust: true, remote: "10.0.0.2:80",
			header: []string{"X-Forwarded-For", "[2001:db8::7]"}, ip: "2001:db8::7"},
	} {
		t.Run(c.name, func(t *testing.T) {
			vars := map[string]string{}
			if c.trust {
				vars["TRUST_PROXY"] = "on"
			}
			app := New(ConfigFrom(vars))
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = c.remote
			if c.tls {
				r.TLS = &tls.ConnectionState{}
			}
			for i := 0; i+1 < len(c.header); i += 2 {
				r.Header.Add(c.header[i], c.header[i+1])
			}
			if got := app.ClientIP(r); got != c.ip {
				t.Errorf("ClientIP = %q, want %q", got, c.ip)
			}
			if got := app.IsHTTPS(r); got != c.https {
				t.Errorf("IsHTTPS = %v, want %v", got, c.https)
			}
		})
	}
}

// ClientKey is the address for IPv4 and the /64 for IPv6, since one client
// is given a whole /64.
func TestClientKey(t *testing.T) {
	for remote, want := range map[string]string{
		"192.0.2.1:4321":              "192.0.2.1",
		"[2001:db8:1:2:3:4:5:6]:4321": "2001:db8:1:2::/64",
		"[2001:db8:1:2:ffff::1]:80":   "2001:db8:1:2::/64",
		"[fe80::1%eth0]:80":           "fe80::/64",
		"[::ffff:1.2.3.4]:80":         "1.2.3.4",
		"pipe":                        "pipe",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		if got := New(Config{}).ClientKey(r); got != want {
			t.Errorf("ClientKey(%s) = %q, want %q", remote, got, want)
		}
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.2:80"
	r.Header.Set("X-Forwarded-For", "2001:db8:9:9::5")
	if got := New(ConfigFrom(map[string]string{"TRUST_PROXY": "on"})).ClientKey(r); got != "2001:db8:9:9::/64" {
		t.Errorf("ClientKey behind a proxy = %q", got)
	}
}
