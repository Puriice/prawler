package crawler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/purrice/prawler/internal/repository"
)

type Crawler struct {
	agent         string
	webRecordRepo repository.WebRecordRepository
}

func NewCrawler(agent string, webRecordRepo repository.WebRecordRepository) Crawler {
	return Crawler{
		agent:         agent,
		webRecordRepo: webRecordRepo,
	}
}

func (c Crawler) Fetch(url string) (*string, error) {
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", c.agent)

	res, err := http.DefaultClient.Do(request)

	if err != nil {
		return nil, fmt.Errorf("Error sending request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, fmt.Errorf("Error reading body response: %v", err)
	}

	stringBody := string(body)

	return &stringBody, nil
}

func (c Crawler) Handle(data []byte) error {
	return nil
}
