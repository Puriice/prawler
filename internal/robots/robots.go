package robots

import (
	"io"
	"net/http"
	"net/url"

	"github.com/purrice/prawler/internal/fetch"
)

const (
	RobotsPath = "/robots.txt"
)

type Robots struct {
	raw *string
}

func Parse(url *url.URL) (*Robots, error) {
	request, err := fetch.GetRequest(url.Host + RobotsPath)

	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(request)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	stringBody := string(body)

	return &Robots{
		raw: &stringBody,
	}, nil
}
