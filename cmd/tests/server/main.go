package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Configuration ────────────────────────────────────────────────────────────

var ports = []int{8081, 8082, 8083}

const (
	maxDepth      = 6 // 0-based, so pages exist at depth 0 → 6 (7 levels)
	linksPerPage  = 5 // number of links rendered on each page
	pagesPerDepth = 4 // how many distinct pages exist per depth per port
)

// ─── HTML Template ────────────────────────────────────────────────────────────

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Title}}</title>
  <style>
    body { font-family: Georgia, serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #222; }
    h1   { border-bottom: 2px solid #444; padding-bottom: 8px; }
    .meta { color: #666; font-size: 0.85em; margin-bottom: 24px; }
    ul   { line-height: 2; }
    a    { color: #1a6a9a; }
    a:hover { color: #c0392b; }
    .tag { display:inline-block; background:#eef; border:1px solid #aac;
           border-radius:4px; padding:1px 6px; font-size:0.75em; margin-left:6px; }
    footer { margin-top: 48px; font-size: 0.8em; color: #999; border-top:1px solid #ddd; padding-top:8px; }
  </style>
</head>
<body>
  <h1>{{.Title}}</h1>
  <p class="meta">
    Port: <strong>{{.Port}}</strong> &nbsp;|&nbsp;
    Depth: <strong>{{.Depth}}</strong> / {{.MaxDepth}} &nbsp;|&nbsp;
    Page ID: <strong>{{.PageID}}</strong> &nbsp;|&nbsp;
    Generated: {{.Now}}
  </p>

  <p>{{.Body}}</p>

  <h2>Links</h2>
  <ul>
    {{range .Links}}
    <li>
      <a href="{{.URL}}">{{.Label}}</a>
      <span class="tag">{{.Kind}}</span>
    </li>
    {{end}}
  </ul>

  {{if .IsLeaf}}
  <p><em>⛔ This is a leaf page — maximum crawl depth reached.</em></p>
  {{end}}

  <footer>Crawl-test server &bull; port {{.Port}} &bull; depth {{.Depth}} &bull; page {{.PageID}}</footer>
</body>
</html>
`))

// ─── Data Structures ──────────────────────────────────────────────────────────

type LinkData struct {
	URL   string
	Label string
	Kind  string // "internal", "cross-port", "recursive"
}

type PageData struct {
	Title    string
	Port     int
	Depth    int
	MaxDepth int
	PageID   int
	Body     string
	Links    []LinkData
	IsLeaf   bool
	Now      string
}

// ─── Seeded random per request (not global — avoids lock contention) ──────────

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// ─── URL helpers ──────────────────────────────────────────────────────────────

// pageURL builds the path used by our router: /page/{depth}/{id}
func pageURL(port, depth, id int) string {
	return fmt.Sprintf("http://localhost:%d/page/%d/%d", port, depth, id)
}

// recursiveURL points back to a shallower page to simulate recursive crawling.
func recursiveURL(port, currentDepth int, rng *rand.Rand) string {
	targetDepth := rng.Intn(currentDepth) // 0 … currentDepth-1
	targetID := rng.Intn(pagesPerDepth)
	return pageURL(port, targetDepth, targetID)
}

// ─── Link generation ─────────────────────────────────────────────────────────

func buildLinks(port, depth int, rng *rand.Rand) []LinkData {
	links := make([]LinkData, 0, linksPerPage)

	for i := 0; i < linksPerPage; i++ {
		roll := rng.Intn(10) // 0-9

		switch {
		// ── recursive back-link (only sensible when depth > 0) ──────────────
		case depth > 0 && roll < 2:
			u := recursiveURL(port, depth, rng)
			links = append(links, LinkData{
				URL:   u,
				Label: fmt.Sprintf("↩ Back-link to a shallower page (%s)", u),
				Kind:  "recursive",
			})

		// ── cross-port link ──────────────────────────────────────────────────
		case roll < 5:
			otherPorts := make([]int, 0, len(ports)-1)
			for _, p := range ports {
				if p != port {
					otherPorts = append(otherPorts, p)
				}
			}
			targetPort := otherPorts[rng.Intn(len(otherPorts))]
			targetDepth := rng.Intn(maxDepth + 1)
			targetID := rng.Intn(pagesPerDepth)
			u := pageURL(targetPort, targetDepth, targetID)
			links = append(links, LinkData{
				URL:   u,
				Label: fmt.Sprintf("→ Cross-port link → port %d depth %d page %d", targetPort, targetDepth, targetID),
				Kind:  "cross-port",
			})

		// ── internal deeper link (only when not at leaf) ─────────────────────
		default:
			nextDepth := depth + 1
			if nextDepth > maxDepth {
				nextDepth = maxDepth
			}
			targetID := rng.Intn(pagesPerDepth)
			u := pageURL(port, nextDepth, targetID)
			links = append(links, LinkData{
				URL:   u,
				Label: fmt.Sprintf("↓ Internal link → depth %d page %d", nextDepth, targetID),
				Kind:  "internal",
			})
		}
	}

	return links
}

// ─── Handler factory ─────────────────────────────────────────────────────────

var lorem = []string{
	"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
	"Pellentesque habitant morbi tristique senectus et netus et malesuada fames.",
	"Vestibulum ante ipsum primis in faucibus orci luctus et ultrices.",
	"Curabitur pretium tincidunt lacus. Nulla gravida orci a odio.",
	"Nullam varius, turpis molestie dictum semper, est enim aliquet sapien.",
	"Phasellus molestie magna non est bibendum non venenatis nisl tempor.",
	"Suspendisse dictum feugiat nisl ut dapibus. Mauris iaculis porttitor.",
}

func makeHandler(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rng := newRand()

		// ── route: /  →  index page ──────────────────────────────────────────
		if r.URL.Path == "/" || r.URL.Path == "" {
			serveIndexPage(w, port, rng)
			return
		}

		// ── route: /page/{depth}/{id} ────────────────────────────────────────
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(parts) == 3 && parts[0] == "page" {
			depth, err1 := strconv.Atoi(parts[1])
			id, err2 := strconv.Atoi(parts[2])
			if err1 != nil || err2 != nil || depth < 0 || depth > maxDepth || id < 0 || id >= pagesPerDepth {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			serveContentPage(w, port, depth, id, rng)
			return
		}

		http.NotFound(w, r)
	}
}

// ─── Index page ───────────────────────────────────────────────────────────────

func serveIndexPage(w http.ResponseWriter, port int, rng *rand.Rand) {
	links := make([]LinkData, 0, pagesPerDepth)
	for i := 0; i < pagesPerDepth; i++ {
		u := pageURL(port, 1, i)
		links = append(links, LinkData{
			URL:   u,
			Label: fmt.Sprintf("Section %d — depth 1, page %d", i+1, i),
			Kind:  "internal",
		})
	}
	// add a couple of cross-port index links
	for _, p := range ports {
		if p != port {
			links = append(links, LinkData{
				URL:   fmt.Sprintf("http://localhost:%d/", p),
				Label: fmt.Sprintf("→ Index of port %d", p),
				Kind:  "cross-port",
			})
		}
	}

	data := PageData{
		Title:    fmt.Sprintf("Index — Port %d", port),
		Port:     port,
		Depth:    0,
		MaxDepth: maxDepth,
		PageID:   0,
		Body:     lorem[rng.Intn(len(lorem))] + " " + lorem[rng.Intn(len(lorem))],
		Links:    links,
		IsLeaf:   false,
		Now:      time.Now().Format(time.RFC1123),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

// ─── Content page ─────────────────────────────────────────────────────────────

func serveContentPage(w http.ResponseWriter, port, depth, id int, rng *rand.Rand) {
	isLeaf := depth >= maxDepth

	var links []LinkData
	if isLeaf {
		// leaf pages only have back-links and cross-port links
		for i := 0; i < linksPerPage; i++ {
			roll := rng.Intn(2)
			if roll == 0 && depth > 0 {
				u := recursiveURL(port, depth, rng)
				links = append(links, LinkData{URL: u, Label: "↩ Recursive back-link", Kind: "recursive"})
			} else {
				otherPorts := make([]int, 0)
				for _, p := range ports {
					if p != port {
						otherPorts = append(otherPorts, p)
					}
				}
				tp := otherPorts[rng.Intn(len(otherPorts))]
				td := rng.Intn(maxDepth + 1)
				ti := rng.Intn(pagesPerDepth)
				u := pageURL(tp, td, ti)
				links = append(links, LinkData{URL: u, Label: fmt.Sprintf("→ Cross-port port %d", tp), Kind: "cross-port"})
			}
		}
	} else {
		links = buildLinks(port, depth, rng)
	}

	body := lorem[rng.Intn(len(lorem))] + " " + lorem[rng.Intn(len(lorem))]

	data := PageData{
		Title:    fmt.Sprintf("Port %d — Depth %d — Page %d", port, depth, id),
		Port:     port,
		Depth:    depth,
		MaxDepth: maxDepth,
		PageID:   id,
		Body:     body,
		Links:    links,
		IsLeaf:   isLeaf,
		Now:      time.Now().Format(time.RFC1123),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	var wg sync.WaitGroup

	for _, port := range ports {
		port := port // capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			mux := http.NewServeMux()
			mux.HandleFunc("/", makeHandler(port))

			addr := fmt.Sprintf(":%d", port)
			log.Printf("Starting crawl-test server on http://localhost%s", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Fatalf("port %d: %v", port, err)
			}
		}()
	}

	wg.Wait()
}
