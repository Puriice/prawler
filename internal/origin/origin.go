package origin

import "net/url"

func GetOrigin(url url.URL) url.URL {
	url.User = nil
	url.Path = ""
	url.Fragment = ""
	url.RawQuery = ""

	if url.Scheme == "" {
		url.Scheme = "https"
	}

	return url
}
