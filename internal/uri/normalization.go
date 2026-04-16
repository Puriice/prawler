package uri

import (
	"net"
	"net/url"
	"path"
	"sort"
	"strings"
)

var ignoreParams = map[string]bool{
	"utm_source":   true,
	"utm_medium":   true,
	"utm_campaign": true,
	"utm_term":     true,
	"utm_content":  true,
	"session":      true,
	"ref":          true,
}

func Normalize(u url.URL) string {
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// 2. Remove default ports
	host, port, err := net.SplitHostPort(u.Host)
	if err == nil {
		if (u.Scheme == "http" && port == "80") ||
			(u.Scheme == "https" && port == "443") {
			u.Host = host
		}
	}

	// 3. Remove fragment
	u.Fragment = ""

	// 4. Normalize path
	u.Path = path.Clean(u.Path)

	// 5. Remove trailing slash (except root)
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	// 6. Normalize query parameters
	q := u.Query()
	cleanQ := url.Values{}

	for key, values := range q {
		if ignoreParams[key] {
			continue
		}
		for _, v := range values {
			cleanQ.Add(key, v)
		}
	}

	// Sort query parameters
	keys := make([]string, 0, len(cleanQ))
	for k := range cleanQ {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sortedQ := url.Values{}
	for _, k := range keys {
		vals := cleanQ[k]
		sort.Strings(vals)
		for _, v := range vals {
			sortedQ.Add(k, v)
		}
	}

	u.RawQuery = sortedQ.Encode()
	return u.String()
}
