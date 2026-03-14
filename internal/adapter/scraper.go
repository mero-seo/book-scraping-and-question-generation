package adapter

import (
	"strings"
	"time"

	"internal/models"

	"scraper"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ConvertBook converts a scraper.ScrapedBook into app models.
// Returns the Book and a slice of Chapters (stored as separate documents).
func ConvertBook(scraped *scraper.ScrapedBook, createdBy bson.ObjectID) (*models.Book, []*models.Chapter) {
	now := time.Now()

	book := &models.Book{
		Title:         scraped.Title,
		Author:        scraped.Author,
		ISBN:          scraped.ISBN,
		Publisher:     scraped.Publisher,
		Language:      scraped.Language,
		Subject:       scraped.Subject,
		SourceType:    scraped.SourceType,
		SourceURL:     scraped.SourceURL,
		CoverImageURL: scraped.CoverURL,
		Status:        models.BookStatusPending,
		TOC:           convertTOC(scraped.TOC),
		Metadata:      scraped.RawMetadata,
		CreatedBy:     createdBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if book.Language == "" {
		book.Language = "en"
	}
	if book.Metadata == nil {
		book.Metadata = make(map[string]string)
	}
	if scraped.Description != "" {
		book.Metadata["description"] = scraped.Description
	}

	chapters := make([]*models.Chapter, len(scraped.Chapters))
	for i, ch := range scraped.Chapters {
		content := ch.Content
		// Append section content if chapters have sections
		if len(ch.Sections) > 0 {
			var parts []string
			parts = append(parts, content)
			for _, sec := range ch.Sections {
				if sec.Title != "" {
					parts = append(parts, "\n\n## "+sec.Title+"\n\n"+sec.Content)
				} else {
					parts = append(parts, sec.Content)
				}
			}
			content = strings.Join(parts, "")
		}

		chapters[i] = &models.Chapter{
			Number:    ch.Number,
			Title:     ch.Title,
			Content:   content,
			WordCount: countWords(content),
			CreatedAt: now,
		}
	}

	return book, chapters
}

// convertTOC converts scraper TOC entries to app model TOC entries.
func convertTOC(entries []scraper.TOCEntry) []models.TOCEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]models.TOCEntry, len(entries))
	for i, e := range entries {
		result[i] = models.TOCEntry{
			Number: e.Number,
			Title:  e.Title,
			Page:   e.Page,
			Depth:  e.Depth,
		}
	}
	return result
}

// countWords returns the approximate word count of a string.
func countWords(s string) int {
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
