package uri

import (
	"net/url"

	"golang.org/x/net/publicsuffix"
)

// StickySessionKey returns a normalized key for session affinity.
// It ignores scheme differences (http/https) and treats subdomains
// as the same site (e.g. api.example.com → example.com).
func StickySessionKey(uri url.URL) string {
	uri.Scheme = ""

	host := uri.Hostname()

	eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err == nil {
		host = eTLDPlusOne
	}

	uri.Host = host
	uri.Path = ""
	uri.RawQuery = ""
	uri.Fragment = ""

	return uri.String()
}
