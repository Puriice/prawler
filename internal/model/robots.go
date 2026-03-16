package model

type Robots struct {
	Host    string
	Raw     *string
	Rules   Rules
	Sitemap []string
}

type Rule struct {
	Allow    []string
	Disallow []string
}

type Rules map[string]*Rule
