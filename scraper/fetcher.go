package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// fetchAndExtract uses Colly to fetch a URL and extract book content.
func fetchAndExtract(c *colly.Collector, url string) (*ScrapedBook, error) {
	book := &ScrapedBook{
		RawMetadata: make(map[string]string),
	}

	var extractErr error

	c.OnHTML("html", func(e *colly.HTMLElement) {
		// Try site-specific extractor first
		domain := e.Request.URL.Hostname()
		if extractor, ok := siteExtractors[domain]; ok {
			result, err := extractor(e)
			if err != nil {
				extractErr = err
				return
			}
			*book = *result
			return
		}

		// Generic extraction
		result, err := extractGeneric(e)
		if err != nil {
			extractErr = err
			return
		}
		*book = *result
	})

	c.OnError(func(r *colly.Response, err error) {
		extractErr = fmt.Errorf("HTTP error %d: %w", r.StatusCode, err)
	})

	if err := c.Visit(url); err != nil {
		return nil, err
	}
	c.Wait()

	if extractErr != nil {
		return nil, extractErr
	}

	if book.Title == "" {
		return nil, fmt.Errorf("no content extracted from %s", url)
	}

	return book, nil
}

// ExtractorFunc is a function that extracts book data from an HTML element.
type ExtractorFunc func(e *colly.HTMLElement) (*ScrapedBook, error)

// siteExtractors maps domains to site-specific extractors.
var siteExtractors = map[string]ExtractorFunc{
	// Add site-specific extractors here:
	// "www.gutenberg.org": extractGutenberg,
}

// extractGeneric attempts to extract book content from any HTML page.
func extractGeneric(e *colly.HTMLElement) (*ScrapedBook, error) {
	book := &ScrapedBook{
		RawMetadata: make(map[string]string),
	}

	// Extract metadata from meta tags
	book.Title = extractTitle(e)
	book.Author = extractMeta(e, "author")
	book.Description = extractMeta(e, "description")
	book.Language = extractMetaProperty(e, "og:locale")
	book.CoverURL = extractMetaProperty(e, "og:image")

	if book.CoverURL != "" {
		book.CoverURL = e.Request.AbsoluteURL(book.CoverURL)
	}

	// Extract ISBN from common patterns
	book.ISBN = extractMeta(e, "isbn")

	// Extract content from the main content area
	doc := e.DOM

	// Find main content container
	content := findContentArea(doc)
	if content == nil {
		return nil, fmt.Errorf("could not find main content area")
	}

	// Extract chapters from headings
	book.Chapters = extractChaptersFromHTML(content)

	// If no chapters found, create one chapter from all content
	if len(book.Chapters) == 0 {
		text := strings.TrimSpace(content.Text())
		if text != "" {
			book.Chapters = []Chapter{
				{
					Number:  1,
					Title:   book.Title,
					Content: text,
				},
			}
		}
	}

	return book, nil
}

// extractTitle gets the page title from various sources.
func extractTitle(e *colly.HTMLElement) string {
	// Try og:title first
	if title := extractMetaProperty(e, "og:title"); title != "" {
		return title
	}
	// Try h1
	if title := e.ChildText("h1"); title != "" {
		return strings.TrimSpace(title)
	}
	// Fall back to <title>
	return strings.TrimSpace(e.ChildText("title"))
}

// extractMeta gets a meta tag content by name.
func extractMeta(e *colly.HTMLElement, name string) string {
	var content string
	e.ForEach(fmt.Sprintf("meta[name='%s']", name), func(_ int, el *colly.HTMLElement) {
		content = el.Attr("content")
	})
	return content
}

// extractMetaProperty gets a meta tag content by property.
func extractMetaProperty(e *colly.HTMLElement, property string) string {
	var content string
	e.ForEach(fmt.Sprintf("meta[property='%s']", property), func(_ int, el *colly.HTMLElement) {
		content = el.Attr("content")
	})
	return content
}

// findContentArea locates the main content area of the page.
func findContentArea(doc *goquery.Selection) *goquery.Selection {
	// Try common content selectors in order of specificity
	selectors := []string{
		"article",
		"main",
		"[role='main']",
		".content",
		"#content",
		".post-content",
		".entry-content",
		".article-body",
	}

	for _, sel := range selectors {
		if found := doc.Find(sel); found.Length() > 0 {
			// Remove non-content elements
			found.Find("nav, header, footer, .sidebar, script, style, .ads, .comments").Remove()
			return found
		}
	}

	// Fall back to body
	body := doc.Find("body")
	body.Find("nav, header, footer, .sidebar, script, style").Remove()
	return body
}

// extractChaptersFromHTML splits content into chapters based on headings.
func extractChaptersFromHTML(content *goquery.Selection) []Chapter {
	var chapters []Chapter
	chapterNum := 0

	// Look for h2 headings as chapter boundaries (h1 is usually the page title)
	headings := content.Find("h2")
	if headings.Length() == 0 {
		// Try h3 if no h2
		headings = content.Find("h3")
	}

	if headings.Length() == 0 {
		return nil
	}

	headings.Each(func(i int, heading *goquery.Selection) {
		chapterNum++
		title := strings.TrimSpace(heading.Text())

		// Collect all content between this heading and the next
		var contentParts []string
		next := heading.Next()
		for next.Length() > 0 {
			tagName := goquery.NodeName(next)
			if tagName == "h2" || tagName == "h3" || tagName == "h1" {
				break
			}
			text := strings.TrimSpace(next.Text())
			if text != "" {
				contentParts = append(contentParts, text)
			}
			next = next.Next()
		}

		if len(contentParts) > 0 {
			chapters = append(chapters, Chapter{
				Number:  chapterNum,
				Title:   title,
				Content: strings.Join(contentParts, "\n\n"),
			})
		}
	})

	return chapters
}
