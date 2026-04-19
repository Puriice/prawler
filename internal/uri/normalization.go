package uri

import (
	"net"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/purrice/prawler/internal/set"
)

var ignoreParams = set.NewSet(
	"utm_source",
	"utm_medium",
	"utm_campaign",
	"utm_term",
	"session",
	"ref",
)

var defaultIndexPath = set.NewSet(
	"/index.html",
	"/index.htm",
	"/index.php",
	"/index.asp",
	"/index.aspx",
)

func Normalize(u url.URL) *url.URL {
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	host, port, err := net.SplitHostPort(u.Host)
	if err == nil {
		if (u.Scheme == "http" && port == "80") ||
			(u.Scheme == "https" && port == "443") {
			u.Host = host
		}
	}

	u.Fragment = ""

	u.Path = path.Clean(u.Path)

	if defaultIndexPath.Contains(u.Path) {
		u.Path = "/"
	} else if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	q := u.Query()
	cleanQ := url.Values{}

	for key, values := range q {
		if ignoreParams.Contains(key) {
			continue
		}
		for _, v := range values {
			cleanQ.Add(key, v)
		}
	}

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
	return &u
}
