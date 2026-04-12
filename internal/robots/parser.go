package robots

import (
	"context"
	"errors"
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

var errNotAllowed = errors.New("Not Allowed")

func (r RobotParser) fetchRobots(url url.URL) (*string, error) {
	url.Path = "/robots.txt"
	res, err := r.fetcher.Fetch(url)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case 404:
		return nil, nil
	case 403:
		return nil, errNotAllowed
	}

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
	robots, ok := r.cache[originStr]

	if ok && time.Now().Before(robots.Timestamp.AddDate(0, 0, 1)) {
		return &robots, nil
	}

	log.Printf("Checking if %s/robots.txt cached in database.", originStr)
	raw, updated_at, err := r.repo.GetRobots(context.Background(), originStr)

	if err == nil && updated_at != nil && time.Now().Before(updated_at.AddDate(0, 0, 1)) {
		robots.Host = origin
		robots.Raw = raw
		robots.Sitemap = grobotstxt.Sitemaps(*robots.Raw)
		robots.Timestamp = time.Now()

		r.cache[originStr] = robots

		return &robots, nil
	}

	log.Printf("%s/robots.txt not founded in the database. Try fetching the new one.", originStr)
	raw, err = r.fetchRobots(origin)

	if err != nil {
		log.Printf("Failed to fetch %s/robots.txt", originStr)
		return nil, ErrFailedToFetchRobots
	}

	if raw != nil && *raw != "" {
		go func() {
			log.Printf("Saving %s/robots.txt", originStr)
			err := r.repo.AddRobots(context.Background(), originStr, *raw)

			if err != nil {
				log.Printf("Failed to saved %s/robots.txt. %v", originStr, err)
				return
			}

			log.Printf("Saved %s/robots.txt", originStr)
		}()
	}

	robots.Host = origin
	robots.Raw = raw
	robots.Sitemap = grobotstxt.Sitemaps(*robots.Raw)
	robots.Timestamp = time.Now()

	r.cache[originStr] = robots

	return &robots, nil

}
