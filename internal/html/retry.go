package html

import (
	"net/http"
	"strconv"
	"time"
)

func ParseRetryAfter(resp *http.Response) time.Duration {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0
	}

	// seconds format
	if secs, err := strconv.Atoi(ra); err == nil {
		return time.Duration(secs) * time.Second
	}

	// HTTP date format
	if t, err := http.ParseTime(ra); err == nil {
		return time.Until(t)
	}

	return 0
}
