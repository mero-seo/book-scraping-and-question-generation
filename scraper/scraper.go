package scraper

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gocolly/colly/v2"
)

// Scraper is the main entry point for content extraction.
type Scraper struct {
	cfg       Config
	collector *colly.Collector
}

// New creates a new Scraper with the given configuration.
func New(cfg Config) *Scraper {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultConfig().UserAgent
	}
	if cfg.Delay == 0 {
		cfg.Delay = DefaultConfig().Delay
	}
	if cfg.Parallelism == 0 {
		cfg.Parallelism = DefaultConfig().Parallelism
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultConfig().MaxRetries
	}

	opts := []colly.CollectorOption{
		colly.UserAgent(cfg.UserAgent),
	}
	if len(cfg.AllowedDomains) > 0 {
		opts = append(opts, colly.AllowedDomains(cfg.AllowedDomains...))
	}

	c := colly.NewCollector(opts...)
	c.SetRequestTimeout(cfg.Timeout)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.Parallelism,
		Delay:       cfg.Delay,
	})

	return &Scraper{
		cfg:       cfg,
		collector: c,
	}
}

// ScrapeURL fetches the given URL and extracts structured book content.
func (s *Scraper) ScrapeURL(ctx context.Context, url string) (*ScrapedBook, error) {
	book, err := fetchAndExtract(s.collector.Clone(), url)
	if err != nil {
		return nil, fmt.Errorf("scraper: failed to scrape %s: %w", url, err)
	}

	book.SourceURL = url
	book.SourceType = "url"

	if len(book.TOC) == 0 && len(book.Chapters) > 0 {
		book.TOC = generateTOCFromChapters(book.Chapters)
	}

	return book, nil
}

// ParsePDF reads a PDF file and extracts structured book content.
func (s *Scraper) ParsePDF(ctx context.Context, r io.Reader, filename string) (*ScrapedBook, error) {
	book, err := extractFromPDF(r, filename)
	if err != nil {
		return nil, fmt.Errorf("scraper: failed to parse PDF %s: %w", filename, err)
	}

	book.SourceType = "pdf"

	if len(book.TOC) == 0 && len(book.Chapters) > 0 {
		book.TOC = generateTOCFromChapters(book.Chapters)
	}

	return book, nil
}

// Search queries Open Library and Google Books for matching books.
func (s *Scraper) Search(ctx context.Context, query string) ([]SearchResult, error) {
	results, err := searchBooks(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("scraper: search failed for %q: %w", query, err)
	}
	return results, nil
}

// generateTOCFromChapters builds a TOC from chapter titles.
func generateTOCFromChapters(chapters []Chapter) []TOCEntry {
	entries := make([]TOCEntry, len(chapters))
	for i, ch := range chapters {
		entries[i] = TOCEntry{
			Number: ch.Number,
			Title:  ch.Title,
			Depth:  0,
		}
	}
	return entries
}

// wordCount returns the approximate word count of a string.
func wordCount(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

// timestamp returns the current time (for testing, this can be overridden).
var timestamp = func() time.Time {
	return time.Now()
}
