package key

import (
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// StickySessionKey returns a normalized key for session affinity.
// It ignores scheme differences (http/https) and treats subdomains
// as the same site (e.g. api.example.com → example.com).
func SiteKey(uri url.URL) string {
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

// OriginKey returns the standard origin (scheme + host + port)
// without modifying the input URL.
func OriginKey(uri url.URL) url.URL {
	uri.Path = ""
	uri.RawQuery = ""
	uri.Fragment = ""
	uri.User = nil

	// Normalize host: ensure port is explicit if needed
	host := uri.Hostname()
	port := uri.Port()

	// Add default ports if missing
	if port == "" {
		switch strings.ToLower(uri.Scheme) {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	if port != "" {
		uri.Host = host + ":" + port
	} else {
		uri.Host = host
	}

	return uri
}
