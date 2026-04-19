package robots

import (
	"context"
	"errors"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/jimsmart/grobotstxt"
	"github.com/purrice/prawler/internal/types"
	"github.com/purrice/prawler/internal/uri"
)

const (
	RobotsPath = "/robots.txt"
	kilobytes  = 1000
	readLimit  = 512 * kilobytes
)

var ErrNotAllowed = errors.New("Not Allowed")

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
		return nil, ErrNotAllowed
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
	origin := uri.OriginKey(url)
	originString := origin.String()

	if originString == "" {
		return nil, types.ErrUnproccessableInput
	}
	robots, ok := r.cache[originString]

	if ok && time.Now().Before(robots.Timestamp.AddDate(0, 0, 1)) {
		log.Printf("Found %s/robots.txt cached in local.", originString)
		return &robots, nil
	}

	raw, updated_at, err := r.repo.GetRobots(context.Background(), origin)

	if err == nil && updated_at != nil && time.Now().Before(updated_at.AddDate(0, 0, 1)) {
		log.Printf("Found %s/robots.txt cached in database.", originString)
		robots.Host = origin
		robots.Raw = raw
		robots.Sitemap = grobotstxt.Sitemaps(*robots.Raw)
		robots.Timestamp = time.Now()

		r.cache[originString] = robots

		return &robots, nil
	}

	log.Printf("%s/robots.txt not founded in the database. Try fetching the new one.", originString)
	raw, err = r.fetchRobots(origin)

	if raw == nil {
		empty := ""
		raw = &empty
	}

	go func() {
		log.Printf("Saving %s/robots.txt", originString)

		err := r.repo.AddRobots(context.Background(), origin, *raw)

		if err != nil {
			log.Printf("Failed to saved %s/robots.txt. %v", originString, err)
			return
		}

		log.Printf("Saved %s/robots.txt", originString)
	}()

	robots.Host = origin
	robots.Raw = raw

	if *raw != "" {
		robots.Sitemap = grobotstxt.Sitemaps(*raw)
	}
	robots.Timestamp = time.Now()

	r.cache[originString] = robots

	return &robots, err

}
