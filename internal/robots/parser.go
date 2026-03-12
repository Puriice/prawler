package robots

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/types"
)

const (
	RobotsPath = "/robots.txt"
)

func fetchRobots(host string) (*string, error) {
	request, err := fetch.GetRequest(host + RobotsPath)

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

	return &stringBody, nil
}

func (r RobotParser) Parse(url url.URL) (*model.Robots, error) {
	url.User = nil
	url.Path = ""
	url.Fragment = ""
	url.RawQuery = ""

	host := url.String()

	if host == "" {
		return nil, types.ErrUnproccessableInput
	}

	log.Printf("Checking %s/robots.txt cached in database.", host)

	robots, err := r.repo.GetRobots(context.Background(), host)

	if err == nil {
		return robots, nil
	}

	log.Printf("%s/robots.txt not founded in the database. Try fetching the new one.", host)
	robots = new(model.Robots)
	robots.Host = host

	raw, err := fetchRobots(host)

	if err != nil || *raw == "" {
		log.Printf("Failed to fetch %s/robots.txt", host)
		return nil, ErrFailedToFetchRobots
	}

	robots.Raw = raw

	log.Printf("Saving %s/robots.txt", host)

	go func() {
		err := r.repo.AddRobots(context.Background(), host, *raw)

		if err != nil {
			log.Printf("Failed to saved %s/robots.txt. %v", host, err)
			return
		}

		log.Printf("Saved %s/robots.txt", host)
	}()

	return robots, nil
}
