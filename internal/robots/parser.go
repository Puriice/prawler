package robots

import (
	"context"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/jimsmart/grobotstxt"
	"github.com/purrice/prawler/internal/origin"
	"github.com/purrice/prawler/internal/types"
)

const (
	RobotsPath = "/robots.txt"
	kilobytes  = 1000
	readLimit  = 512 * kilobytes
)

func (r RobotParser) fetchRobots(url url.URL) (*string, error) {
	url.Path = "/robots.txt"
	res, err := r.fetcher.Fetch(url)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	reader := io.LimitReader(res.Body, readLimit)
	body, err := io.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	stringBody := string(body)

	return &stringBody, nil
}

func (r RobotParser) Parse(url url.URL) (*Robots, error) {
	origin := origin.GetOrigin(url)
	originStr := origin.String()

	if origin.String() == "" {
		return nil, types.ErrUnproccessableInput
	}
	log.Printf("Checking if %s/robots.txt cached in local.", originStr)

	var robots Robots

	robots, ok := r.cache[originStr]

	if ok {
		return &robots, nil
	}

	log.Printf("Checking if %s/robots.txt cached in database.", originStr)

	raw, updated_at, err := r.repo.GetRobots(context.Background(), originStr)

	if err != nil || (updated_at != nil && time.Now().After(updated_at.AddDate(0, 0, 1))) {
		log.Printf("%s/robots.txt not founded in the database. Try fetching the new one.", originStr)
		robots.Host = origin

		raw, err := r.fetchRobots(origin)

		if err != nil || *raw == "" {
			log.Printf("Failed to fetch %s/robots.txt", originStr)
			return nil, ErrFailedToFetchRobots
		}

		robots.Raw = raw
		log.Printf("Saving %s/robots.txt", originStr)

		go func() {
			err := r.repo.AddRobots(context.Background(), originStr, *raw)

			if err != nil {
				log.Printf("Failed to saved %s/robots.txt. %v", originStr, err)
				return
			}

			log.Printf("Saved %s/robots.txt", originStr)
		}()
	} else {
		robots.Host = origin
		robots.Raw = raw
	}

	robots.Sitemap = grobotstxt.Sitemaps(*robots.Raw)

	r.cache[originStr] = robots

	return &robots, nil
}
