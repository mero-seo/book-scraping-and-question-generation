package scraper

import "time"

// Config holds the scraper configuration.
type Config struct {
	// UserAgent is the HTTP User-Agent header sent with requests.
	UserAgent string

	// Delay between consecutive requests to the same domain.
	Delay time.Duration

	// Parallelism is the max number of concurrent requests.
	Parallelism int

	// Timeout for individual HTTP requests.
	Timeout time.Duration

	// MaxRetries is the number of times to retry failed requests.
	MaxRetries int

	// AllowedDomains restricts scraping to these domains.
	// Empty slice means all domains are allowed.
	AllowedDomains []string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		UserAgent:   "BookScraper/1.0",
		Delay:       200 * time.Millisecond,
		Parallelism: 2,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
	}
}

// ScrapedBook is the scraper's output type. It contains all extracted
// book data in a generic, project-agnostic format.
type ScrapedBook struct {
	Title       string
	Author      string
	ISBN        string
	Publisher   string
	Language    string
	Subject     string
	Description string
	CoverURL    string
	SourceURL   string
	SourceType  string // "url", "pdf", "search"
	TOC         []TOCEntry
	Chapters    []Chapter
	RawMetadata map[string]string
}

// Chapter represents a single chapter extracted from a book.
type Chapter struct {
	Number   int
	Title    string
	Content  string
	Sections []Section
}

// TOCEntry represents a single entry in the table of contents.
type TOCEntry struct {
	Number int
	Title  string
	Page   int
	Depth  int // 0 = top level
}

// Section represents a sub-section within a chapter.
type Section struct {
	Title   string
	Content string
}

// SearchResult represents a book found via search APIs.
type SearchResult struct {
	Title      string
	Author     string
	ISBN       string
	Publisher  string
	CoverURL   string
	PreviewURL string
	Source     string // "openlibrary" or "googlebooks"
}
