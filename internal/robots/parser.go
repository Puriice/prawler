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
	host := origin.GetOrigin(url)
	hostString := host.String()

	if host.String() == "" {
		return nil, types.ErrUnproccessableInput
	}

	log.Printf("Checking %s/robots.txt cached in database.", hostString)

	raw, updated_at, err := r.repo.GetRobots(context.Background(), hostString)

	var robots Robots

	if err != nil || (updated_at != nil && time.Now().After(updated_at.AddDate(0, 0, 1))) {
		log.Printf("%s/robots.txt not founded in the database. Try fetching the new one.", hostString)
		robots.Host = host

		raw, err := r.fetchRobots(host)

		if err != nil || *raw == "" {
			log.Printf("Failed to fetch %s/robots.txt", hostString)
			return nil, ErrFailedToFetchRobots
		}

		robots.Raw = raw
		log.Printf("Saving %s/robots.txt", hostString)

		go func() {
			err := r.repo.AddRobots(context.Background(), hostString, *raw)

			if err != nil {
				log.Printf("Failed to saved %s/robots.txt. %v", hostString, err)
				return
			}

			log.Printf("Saved %s/robots.txt", hostString)
		}()
	} else {
		robots.Host = host
		robots.Raw = raw
	}

	robots.Sitemap = grobotstxt.Sitemaps(*robots.Raw)

	return &robots, nil
}
