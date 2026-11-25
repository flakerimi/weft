package warp

import (
	"net/http"
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
// empty entries between commas left out. bad holds each entry the standard
// library refuses as an origin.
func trustedOrigins(v string) (origins, bad []string) {
	for o := range strings.SplitSeq(v, ",") {
		if o = strings.TrimSpace(o); o == "" {
			continue
		}
		if _, refused := newOriginProtection([]string{o}); len(refused) > 0 {
			bad = append(bad, o)
			continue
		}
		origins = append(origins, o)
	}
	return origins, bad
}

// refuseCrossSite answers a state-changing request under /api that came from
// another site with a 403, and runs next for every other request.
func refuseCrossSite(p *http.CrossOriginProtection, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			if err := p.Check(r); err != nil {
				WriteError(w, r, Forbidden(crossSiteMessage))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
