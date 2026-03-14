package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"scraper"
)

func main() {
	urlFlag := flag.String("url", "", "URL to scrape book content from")
	pdfFlag := flag.String("pdf", "", "Path to a PDF file to parse")
	searchFlag := flag.String("search", "", "Search query for book lookup")
	jsonFlag := flag.Bool("json", false, "Output as JSON (default: human-readable)")
	delayFlag := flag.Duration("delay", 200*time.Millisecond, "Delay between requests")
	parallelismFlag := flag.Int("parallelism", 2, "Max concurrent requests")
	timeoutFlag := flag.Duration("timeout", 30*time.Second, "Request timeout")

	flag.Parse()

	if *urlFlag == "" && *pdfFlag == "" && *searchFlag == "" {
		fmt.Fprintln(os.Stderr, "Usage: scraper -url <URL> | -pdf <path> | -search <query>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfg := scraper.Config{
		Delay:       *delayFlag,
		Parallelism: *parallelismFlag,
		Timeout:     *timeoutFlag,
	}

	s := scraper.New(cfg)
	ctx := context.Background()

	switch {
	case *urlFlag != "":
		book, err := s.ScrapeURL(ctx, *urlFlag)
		if err != nil {
			log.Fatalf("Scrape failed: %v", err)
		}
		printBook(book, *jsonFlag)

	case *pdfFlag != "":
		f, err := os.Open(*pdfFlag)
		if err != nil {
			log.Fatalf("Failed to open PDF: %v", err)
		}
		defer f.Close()

		book, err := s.ParsePDF(ctx, f, *pdfFlag)
		if err != nil {
			log.Fatalf("PDF parse failed: %v", err)
		}
		printBook(book, *jsonFlag)

	case *searchFlag != "":
		results, err := s.Search(ctx, *searchFlag)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}
		printSearchResults(results, *jsonFlag)
	}
}

func printBook(book *scraper.ScrapedBook, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(book)
		return
	}

	fmt.Printf("Title:    %s\n", book.Title)
	fmt.Printf("Author:   %s\n", book.Author)
	fmt.Printf("Subject:  %s\n", book.Subject)
	fmt.Printf("ISBN:     %s\n", book.ISBN)
	fmt.Printf("Source:   %s (%s)\n", book.SourceURL, book.SourceType)
	fmt.Printf("Chapters: %d\n", len(book.Chapters))
	fmt.Println()

	if len(book.TOC) > 0 {
		fmt.Println("Table of Contents:")
		for _, entry := range book.TOC {
			indent := ""
			for i := 0; i < entry.Depth; i++ {
				indent += "  "
			}
			fmt.Printf("  %s%d. %s\n", indent, entry.Number, entry.Title)
		}
		fmt.Println()
	}

	for _, ch := range book.Chapters {
		words := wordCount(ch.Content)
		fmt.Printf("Chapter %d: %s (%d words)\n", ch.Number, ch.Title, words)
	}
}

func printSearchResults(results []scraper.SearchResult, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(results)
		return
	}

	fmt.Printf("Found %d results:\n\n", len(results))
	for i, r := range results {
		fmt.Printf("[%d] %s\n", i+1, r.Title)
		fmt.Printf("    Author:    %s\n", r.Author)
		fmt.Printf("    ISBN:      %s\n", r.ISBN)
		fmt.Printf("    Publisher: %s\n", r.Publisher)
		fmt.Printf("    Source:    %s\n", r.Source)
		if r.PreviewURL != "" {
			fmt.Printf("    Preview:   %s\n", r.PreviewURL)
		}
		fmt.Println()
	}
}

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
