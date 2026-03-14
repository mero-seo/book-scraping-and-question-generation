package scraper

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// extractTOCFromHTML tries to find a table of contents in the HTML.
func extractTOCFromHTML(doc *goquery.Selection) []TOCEntry {
	var entries []TOCEntry

	// Look for common TOC containers
	tocSelectors := []string{
		".toc",
		"#toc",
		"#table-of-contents",
		".table-of-contents",
		"nav.toc",
		"[role='directory']",
	}

	for _, sel := range tocSelectors {
		toc := doc.Find(sel)
		if toc.Length() == 0 {
			continue
		}

		num := 0
		toc.Find("a, li").Each(func(_ int, item *goquery.Selection) {
			title := strings.TrimSpace(item.Text())
			if title == "" {
				return
			}
			num++
			depth := 0
			// Check nesting level
			if item.ParentsFiltered("ul ul, ol ol").Length() > 0 {
				depth = 1
			}
			entries = append(entries, TOCEntry{
				Number: num,
				Title:  title,
				Depth:  depth,
			})
		})

		if len(entries) > 0 {
			return entries
		}
	}

	return nil
}

// extractTOCFromText tries to find a table of contents in plain text.
func extractTOCFromText(text string) []TOCEntry {
	var entries []TOCEntry
	lines := strings.Split(text, "\n")

	inTOC := false
	num := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Detect start of TOC
		if !inTOC {
			if lower == "table of contents" || lower == "contents" {
				inTOC = true
				continue
			}
			continue
		}

		// Empty line might end TOC
		if trimmed == "" {
			if num > 2 { // If we already found entries, stop at blank line
				break
			}
			continue
		}

		// Stop at common post-TOC markers
		if lower == "introduction" || lower == "preface" || lower == "foreword" {
			// This is likely the first chapter heading, not part of TOC format
			if num > 0 {
				break
			}
		}

		num++
		entries = append(entries, TOCEntry{
			Number: num,
			Title:  trimmed,
			Depth:  0,
		})
	}

	return entries
}
