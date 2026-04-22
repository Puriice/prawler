package uri

import (
	"net/url"
	"strings"
)

// OriginKey returns the standard origin (scheme + host + port)
// without modifying the input URL.
func OriginKey(uri url.URL) *url.URL {
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

	return &uri
}
