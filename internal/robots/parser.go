package robots

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/purrice/prawler/internal/fetch"
	"github.com/purrice/prawler/internal/model"
	"github.com/purrice/prawler/internal/types"
)

const (
	RobotsPath = "/robots.txt"
	kilobytes  = 1000
	readLimit  = 512 * kilobytes
)

func (r RobotParser) fetchRobots(host string) (*string, error) {
	r.limiter.Wait(context.Background())

	request, err := fetch.GetRequest(host + RobotsPath)

	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(request)

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

func parse(raw string) (model.Rules, []string) {
	lines := strings.Split(raw, "\n")
	log.Println(lines)

	rules := make(map[string]*model.Rule, 5)
	sitemap := []string{}

	currentRule := &model.Rule{
		Allow:    []string{},
		Disallow: []string{},
	}

	rules["*"] = currentRule

	for _, line := range lines {
		log.Printf("Parsing: %s\n", line)
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		splitedLine := strings.SplitN(line, ":", 2)

		key, value := strings.ToLower(splitedLine[0]), strings.TrimSpace(splitedLine[1])

		value = strings.Split(value, "#")[0]

		switch key {
		case "user-agent":
			rule, ok := rules[value]

			if !ok {
				rule = &model.Rule{
					Allow:    []string{},
					Disallow: []string{},
				}

				rules[value] = rule
			}

			currentRule = rule
		case "allow":
			currentRule.Allow = append(currentRule.Allow, value)
		case "disallow":
			currentRule.Disallow = append(currentRule.Disallow, value)
		case "sitemap":
			sitemap = append(sitemap, value)
		}
	}

	log.Println(rules)

	return rules, sitemap
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

	if err != nil {
		log.Printf("%s/robots.txt not founded in the database. Try fetching the new one.", host)
		robots = new(model.Robots)
		robots.Host = host

		raw, err := r.fetchRobots(host)

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
	}

	log.Printf("Parsing %s/robots.txt", host)
	rules, sitemap := parse(*robots.Raw)
	log.Printf("Parsed %s/robots.txt", host)

	robots.Rules = rules
	robots.Sitemap = sitemap

	return robots, nil
}
