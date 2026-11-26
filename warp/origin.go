package warp

import (
	"net/http"
	"path"
	"strings"
)

// crossSiteMessage is the answer to a state-changing API request that a
// browser sent from another site.
const crossSiteMessage = "This request came from another site."

// newOriginProtection returns the protection every request shares: the
// standard library's, with each trusted origin added. An origin it refuses
// is returned in bad.
func newOriginProtection(trusted []string) (p *http.CrossOriginProtection, bad []string) {
	p = http.NewCrossOriginProtection()
	for _, o := range trusted {
		if err := p.AddTrustedOrigin(o); err != nil {
			bad = append(bad, o)
		}
	}
	return p, bad
}

// trustedOrigins reads TRUSTED_ORIGINS: origins separated by commas, with
// empty entries between commas left out. It adds each to one protection,
// which it returns for New to use, so each entry is checked once; bad holds
// each entry the standard library refuses as an origin.
func trustedOrigins(v string) (p *http.CrossOriginProtection, origins, bad []string) {
	p = http.NewCrossOriginProtection()
	for o := range strings.SplitSeq(v, ",") {
		if o = strings.TrimSpace(o); o == "" {
			continue
		}
		if err := p.AddTrustedOrigin(o); err != nil {
			bad = append(bad, o)
			continue
		}
		origins = append(origins, o)
	}
	return p, origins, bad
}

// isAPIPath reports whether p is /api or under it.
func isAPIPath(p string) bool {
	return p == "/api" || strings.HasPrefix(p, "/api/")
}

// refuseCrossSite answers a state-changing request under /api that came from
// another site with a 403, and runs next for every other request. A path is
// under /api as it was sent or as path.Clean writes it, which is where the
// mux would redirect it ("//api/x" is "/api/x"), so no Wrap runs for a
// request that was never checked.
func refuseCrossSite(p *http.CrossOriginProtection, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) || isAPIPath(path.Clean(r.URL.Path)) {
			if err := p.Check(r); err != nil {
				WriteError(w, r, Forbidden(crossSiteMessage))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
