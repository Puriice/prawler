package robots

import (
	"fmt"

	"github.com/purrice/prawler/internal/model"
)

func Println(robots model.Robots) {
	for agent, rule := range robots.Rules {
		fmt.Printf("User-Agent: %s\n", agent)
		for _, v := range rule.Allow {
			fmt.Printf("Allow: %s\n", v)
		}
		for _, v := range rule.Disallow {
			fmt.Printf("Disallow: %s\n", v)
		}
	}

	for _, sitemap := range robots.Sitemap {
		fmt.Printf("Sitemap: %s\n", sitemap)
	}
}
