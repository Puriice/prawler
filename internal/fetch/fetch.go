package fetch

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/origin"
	"golang.org/x/time/rate"
)

type Fetcher struct {
	client   *http.Client
	limiters map[string]*rate.Limiter
}

func NewFecter(client *http.Client) Fetcher {
	if client == nil {
		client = http.DefaultClient
	}

	return Fetcher{
		client: client,
	}
}

func (f Fetcher) getLimiter(origin string) *rate.Limiter {
	limiter, ok := f.limiters[origin]

	if ok {
		return limiter
	}

	delay := time.Duration(config.GetConfig().CrawlingDelayInMS) * time.Millisecond
	limiter = rate.NewLimiter(rate.Every(delay), 1)

	f.limiters[origin] = limiter

	return limiter
}

func (f Fetcher) Fetch(endpoint url.URL) (*http.Response, error) {
	return f.FetchWithContext(context.Background(), endpoint)
}

func (f Fetcher) FetchWithContext(ctx context.Context, endpoint url.URL) (*http.Response, error) {
	config := config.GetConfig()
	origin := origin.GetOrigin(endpoint)

	limiter := f.getLimiter(origin.String())
	limiter.Wait(ctx)

	request, err := http.NewRequest("GET", endpoint.String(), nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", config.GetDisplayUserAgent())

	return f.client.Do(request)
}
