package scraper

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// extractFromPDF reads a PDF and extracts structured book content.
// TODO: Integrate pdfcpu or pdf library for actual PDF text extraction.
// For now, this is a placeholder that will be implemented when we add
// the PDF parsing dependency.
func extractFromPDF(r io.Reader, filename string) (*ScrapedBook, error) {
	// Read all bytes from the reader
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty PDF file")
	}

	// Check PDF magic bytes
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		return nil, fmt.Errorf("not a valid PDF file")
	}

	// TODO: Replace with actual PDF text extraction
	// For now, return an error indicating PDF support needs the library
	return nil, fmt.Errorf("PDF text extraction not yet implemented — add pdfcpu dependency")
}

// splitTextIntoChapters splits extracted text into chapters using heading detection.
func splitTextIntoChapters(text string) []Chapter {
	// Chapter detection patterns (in priority order)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?mi)^chapter\s+(\d+)[:\s]*(.*)$`),
		regexp.MustCompile(`(?mi)^CHAPTER\s+(\d+)[:\s]*(.*)$`),
		regexp.MustCompile(`(?mi)^(\d+)\.\s+(.+)$`),
		regexp.MustCompile(`(?mi)^Part\s+(\d+)[:\s]*(.*)$`),
	}

	type boundary struct {
		start int
		title string
		num   int
	}

	var boundaries []boundary

	for _, pat := range patterns {
		matches := pat.FindAllStringIndex(text, -1)
		if len(matches) >= 2 { // Need at least 2 chapters to be useful
			submatches := pat.FindAllStringSubmatch(text, -1)
			for i, match := range matches {
				title := strings.TrimSpace(submatches[i][0])
				boundaries = append(boundaries, boundary{
					start: match[0],
					title: title,
					num:   i + 1,
				})
			}
			break // Use the first pattern that works
		}
	}

	if len(boundaries) == 0 {
		// No chapters detected — return whole text as one chapter
		if strings.TrimSpace(text) != "" {
			return []Chapter{
				{Number: 1, Title: "Full Text", Content: strings.TrimSpace(text)},
			}
		}
		return nil
	}

	// Split text at boundaries
	var chapters []Chapter
	for i, b := range boundaries {
		var content string
		if i+1 < len(boundaries) {
			content = text[b.start:boundaries[i+1].start]
		} else {
			content = text[b.start:]
		}

		// Remove the heading line from content
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) > 1 {
			content = strings.TrimSpace(lines[1])
		}

		if content != "" {
			chapters = append(chapters, Chapter{
				Number:  b.num,
				Title:   b.title,
				Content: content,
			})
		}
	}

	return chapters
}
