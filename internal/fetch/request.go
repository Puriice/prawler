package fetch

import (
	"net/http"

	"github.com/purrice/prawler/internal/config"
)

func GetRequest(url string) (*http.Request, error) {
	config := config.GetConfig()

	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", config.GetDisplayUserAgent())

	return request, nil
}
