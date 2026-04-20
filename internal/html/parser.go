package html

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
)

// parsedPage holds everything extracted from a raw HTML page.
type parsedPage struct {
	PageMetaData
	Page
	PageContent
}

type PageMetaData struct {
	Title           string
	Language        string
	MetaDescription string
	OGTitle         string
	OGDescription   string
	OGImage         string
	Author          string
	PublishedAt     *time.Time
	SchemaOrg       []string // raw JSON-LD blobs
}

type Page struct {
	CanonicalURL string
	NoIndex      bool
	NoFollow     bool
	Checksum     string // SHA-256 of extracted text

	Links []Link
}

type PageContent struct {
	RawHTML       string
	ExtractedText string
	WordCount     int
	Chunks        []Chunk
}

// Chunk is one embeddable section of the page.
type Chunk struct {
	Index          int
	SectionHeading string // nearest H1/H2/H3 above this chunk
	Content        string
	TokenEstimate  int // rough estimate: words * 1.3
}

// Link is an outbound hyperlink found on the page.
type Link struct {
	TargetURL  string
	AnchorText string
	IsNoFollow bool
	IsInternal bool
}

// Parser extracts structured data from raw HTML.
type Parser struct {
	BaseURL   *url.URL // used to resolve relative links and detect internal vs external
	ChunkSize int      // target words per chunk (default 200)
	MinChunk  int      // discard chunks shorter than this (default 30)
}

// NewParser creates a Parser for the given absolute page URL.
func NewParser(rawURL string) (*Parser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	return &Parser{
		BaseURL:   u,
		ChunkSize: 200,
		MinChunk:  30,
	}, nil
}

// Parse is the main entry point. Pass the raw HTML as a string.
func (p *Parser) Parse(rawHTML string) (*PageMetaData, *Page, *PageContent, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("html parse error: %w", err)
	}

	page := &parsedPage{}

	page.PageContent.RawHTML = rawHTML

	// 1. metadata from <head>
	p.extractHead(doc, page)

	// 2. main body text via go-readability (strips boilerplate)
	article, err := readability.FromReader(strings.NewReader(rawHTML), p.BaseURL)
	if err == nil {
		page.ExtractedText = cleanText(article.TextContent)
		if page.Title == "" {
			page.Title = article.Title
		}
		if page.Author == "" {
			page.Author = article.Byline
		}
	} else {
		// fallback: strip all tags manually
		page.ExtractedText = cleanText(extractTextFallback(doc))
	}

	// 3. checksum for deduplication
	page.Checksum = sha256Hex(page.ExtractedText)
	page.WordCount = countWords(page.ExtractedText)

	// 4. heading-aware chunking for RAG
	page.Chunks = p.chunkByHeadings(doc, page.Title)

	// 5. outbound links for URL frontier
	page.Links = p.extractLinks(doc)

	return &page.PageMetaData, &page.Page, &page.PageContent, nil
}

// ParseReader is a convenience wrapper if you have an io.Reader.
func (p *Parser) ParseReader(r io.Reader) (*PageMetaData, *Page, *PageContent, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, nil, err
	}
	return p.Parse(string(b))
}

// --------------------------------------------------------------------------
// Head extraction
// --------------------------------------------------------------------------

func (p *Parser) extractHead(doc *html.Node, page *parsedPage) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "html":
				page.Language = attr(n, "lang")
			case "title":
				if n.FirstChild != nil {
					page.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				p.handleMeta(n, page)
			case "link":
				if attrEq(n, "rel", "canonical") {
					page.CanonicalURL = resolveURL(p.BaseURL, attr(n, "href"))
				}
			case "script":
				if attrEq(n, "type", "application/ld+json") {
					if blob := textContent(n); blob != "" {
						page.SchemaOrg = append(page.SchemaOrg, blob)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if page.CanonicalURL == "" {
		page.CanonicalURL = p.BaseURL.String()
	}
}

func (p *Parser) handleMeta(n *html.Node, page *parsedPage) {
	name := strings.ToLower(attr(n, "name"))
	prop := strings.ToLower(attr(n, "property"))
	content := attr(n, "content")

	switch {
	case name == "description":
		page.MetaDescription = content
	case name == "author":
		page.Author = content
	case name == "robots" || name == "googlebot":
		lower := strings.ToLower(content)
		if strings.Contains(lower, "noindex") {
			page.NoIndex = true
		}
		if strings.Contains(lower, "nofollow") {
			page.NoFollow = true
		}
	case prop == "og:title":
		page.OGTitle = content
	case prop == "og:description":
		page.OGDescription = content
	case prop == "og:image":
		page.OGImage = content
	case name == "article:published_time" || prop == "article:published_time":
		if t, err := parseDate(content); err == nil {
			page.PublishedAt = &t
		}
	}
}

// --------------------------------------------------------------------------
// Heading-aware chunking
// --------------------------------------------------------------------------

// chunkByHeadings walks the <body> and splits content at H1/H2/H3 boundaries.
// Each chunk carries the nearest heading above it as context prefix for the embedder.
func (p *Parser) chunkByHeadings(doc *html.Node, pageTitle string) []Chunk {
	type section struct {
		heading string
		words   []string
	}

	var sections []section
	current := section{heading: pageTitle}

	headingTags := map[string]bool{"h1": true, "h2": true, "h3": true}
	skipTags := map[string]bool{
		"script": true, "style": true, "noscript": true,
		"nav": true, "header": true, "footer": true,
		"aside": true, "form": true, "button": true,
		"iframe": true, "svg": true,
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if skipTags[n.Data] {
				return
			}
			if headingTags[n.Data] {
				// flush current section before starting new one
				if len(current.words) >= p.MinChunk {
					sections = append(sections, current)
				}
				current = section{heading: cleanText(textContent(n))}
				return
			}
		}

		if n.Type == html.TextNode {
			current.words = append(current.words, strings.Fields(n.Data)...)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	body := findNode(doc, "body")
	if body == nil {
		body = doc
	}
	walk(body)

	// flush final section
	if len(current.words) >= p.MinChunk {
		sections = append(sections, current)
	}

	// split oversized sections into fixed-size windows
	var chunks []Chunk
	idx := 0
	for _, sec := range sections {
		for start := 0; start < len(sec.words); start += p.ChunkSize {
			end := start + p.ChunkSize
			if end > len(sec.words) {
				end = len(sec.words)
			}
			slice := sec.words[start:end]
			if len(slice) < p.MinChunk {
				continue
			}
			chunks = append(chunks, Chunk{
				Index:          idx,
				SectionHeading: sec.heading,
				Content:        strings.Join(slice, " "),
				TokenEstimate:  int(float64(len(slice)) * 1.3),
			})
			idx++
		}
	}

	return chunks
}

// --------------------------------------------------------------------------
// Link extraction
// --------------------------------------------------------------------------

func (p *Parser) extractLinks(doc *html.Node) []Link {
	var links []Link
	seen := map[string]bool{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attr(n, "href")
			if href == "" ||
				strings.HasPrefix(href, "#") ||
				strings.HasPrefix(href, "javascript:") ||
				strings.HasPrefix(href, "mailto:") ||
				strings.HasPrefix(href, "tel:") {
				goto children
			}

			{
				resolved := resolveURL(p.BaseURL, href)
				if resolved == "" || seen[resolved] {
					goto children
				}
				seen[resolved] = true

				rel := strings.ToLower(attr(n, "rel"))
				links = append(links, Link{
					TargetURL:  resolved,
					AnchorText: cleanText(textContent(n)),
					IsNoFollow: strings.Contains(rel, "nofollow"),
					IsInternal: sameHost(p.BaseURL, resolved),
				})
			}
		}

	children:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func attrEq(n *html.Node, key, val string) bool {
	return strings.EqualFold(attr(n, key), val)
}

// textContent returns the concatenated text of a node and all its descendants.
func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// extractTextFallback strips all tags manually when go-readability fails.
func extractTextFallback(doc *html.Node) string {
	skip := map[string]bool{
		"script": true, "style": true, "noscript": true,
		"nav": true, "header": true, "footer": true, "aside": true,
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skip[n.Data] {
			return
		}
		if n.Type == html.TextNode {
			if t := strings.TrimSpace(n.Data); t != "" {
				sb.WriteString(t)
				sb.WriteRune(' ')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return sb.String()
}

func findNode(doc *html.Node, tag string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}

func resolveURL(base *url.URL, href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	resolved.Fragment = "" // drop anchor fragments
	return resolved.String()
}

func sameHost(base *url.URL, target string) bool {
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Hostname(), base.Hostname())
}

// cleanText collapses all unicode whitespace runs to a single space.
func cleanText(s string) string {
	var prev rune
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !unicode.IsSpace(prev) {
				sb.WriteRune(' ')
			}
		} else {
			sb.WriteRune(r)
		}
		prev = r
	}
	return strings.TrimSpace(sb.String())
}

func countWords(s string) int {
	return len(strings.Fields(s))
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"January 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date: %s", s)
}
