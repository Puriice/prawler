package fetch

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/purrice/prawler/internal/config"
	"github.com/purrice/prawler/internal/uri"
	"golang.org/x/time/rate"
)

type Fetcher struct {
	client   *http.Client
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func NewFecter(client *http.Client) *Fetcher {
	if client == nil {
		client = http.DefaultClient
	}

	return &Fetcher{
		client:   client,
		limiters: make(map[string]*rate.Limiter),
	}
}

func (f *Fetcher) getLimiter(origin string) *rate.Limiter {
	f.mu.RLock()
	limiter, ok := f.limiters[origin]
	f.mu.RUnlock()

	if ok {
		return limiter
	}

	delay := config.GetConfig().CrawlingPolicy.MinimumCrawlingDelayInMS
	limiter = rate.NewLimiter(rate.Every(delay), 1)

	f.mu.Lock()
	f.limiters[origin] = limiter
	f.mu.Unlock()

	return limiter
}

func (f *Fetcher) Fetch(endpoint url.URL) (*http.Response, error) {
	return f.FetchWithContext(context.Background(), endpoint)
}

func (f *Fetcher) FetchWithContext(ctx context.Context, endpoint url.URL) (*http.Response, error) {
	config := config.GetConfig()
	key := uri.OriginKey(endpoint)

	limiter := f.getLimiter(key.String())
	limiter.Wait(ctx)

	request, err := http.NewRequest("GET", endpoint.String(), nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", config.GetDisplayUserAgent())

	return f.client.Do(request)
}
