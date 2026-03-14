package scraper

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// extractFromPDF reads a PDF and extracts structured book content.
func extractFromPDF(r io.Reader, filename string) (*ScrapedBook, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty PDF file")
	}

	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		return nil, fmt.Errorf("not a valid PDF file")
	}

	// Extract text using pdfcpu to a temp directory
	fullText, err := extractTextWithPdfcpu(data)
	if err != nil {
		// Fallback: basic text extraction from raw bytes
		fullText = extractTextFallback(data)
		if fullText == "" {
			return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
		}
	}

	if strings.TrimSpace(fullText) == "" {
		return nil, fmt.Errorf("PDF contains no extractable text (possibly scanned/image-based)")
	}

	chapters := splitTextIntoChapters(fullText)

	title := strings.TrimSuffix(filename, ".pdf")
	title = strings.TrimSuffix(title, ".PDF")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	book := &ScrapedBook{
		Title:      title,
		SourceType: "pdf",
		Chapters:   chapters,
		RawMetadata: map[string]string{
			"pages": fmt.Sprintf("%d", estimatePages(fullText)),
		},
	}

	for _, ch := range chapters {
		book.TOC = append(book.TOC, TOCEntry{
			Number: ch.Number,
			Title:  ch.Title,
		})
	}

	return book, nil
}

// extractTextWithPdfcpu uses pdfcpu to extract content streams to temp files,
// then reads the text from those files.
func extractTextWithPdfcpu(data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdf-extract-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	reader := bytes.NewReader(data)
	conf := model.NewDefaultConfiguration()

	err = api.ExtractContent(reader, tmpDir, "content", nil, conf)
	if err != nil {
		return "", fmt.Errorf("pdfcpu content extraction failed: %w", err)
	}

	// Read all extracted content files
	var allText strings.Builder
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", fmt.Errorf("failed to read temp dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, entry.Name()))
		if err != nil {
			continue
		}
		allText.Write(content)
		allText.WriteString("\n")
	}

	return allText.String(), nil
}

// extractTextFallback does basic text extraction from raw PDF bytes.
func extractTextFallback(data []byte) string {
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindAllSubmatch(data, -1)

	var parts []string
	for _, m := range matches {
		if len(m) > 1 {
			text := string(m[1])
			if len(text) > 1 && !strings.ContainsAny(text, "\x00\x01\x02\x03\x04\x05") {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, " ")
}

func estimatePages(text string) int {
	pages := len(text) / 3000
	if pages < 1 {
		pages = 1
	}
	return pages
}

// splitTextIntoChapters splits extracted text into chapters using heading detection.
func splitTextIntoChapters(text string) []Chapter {
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
		if len(matches) >= 2 {
			submatches := pat.FindAllStringSubmatch(text, -1)
			for i, match := range matches {
				title := strings.TrimSpace(submatches[i][0])
				boundaries = append(boundaries, boundary{
					start: match[0],
					title: title,
					num:   i + 1,
				})
			}
			break
		}
	}

	if len(boundaries) == 0 {
		if strings.TrimSpace(text) != "" {
			return []Chapter{
				{Number: 1, Title: "Full Text", Content: strings.TrimSpace(text)},
			}
		}
		return nil
	}

	var chapters []Chapter
	for i, b := range boundaries {
		var content string
		if i+1 < len(boundaries) {
			content = text[b.start:boundaries[i+1].start]
		} else {
			content = text[b.start:]
		}

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
