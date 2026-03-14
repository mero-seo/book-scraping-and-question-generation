# Scraper Module

## Overview

The scraper is a **fully independent, reusable Go module** that extracts structured book content from multiple sources: web pages, PDF files, and online book APIs. It lives at `scraper/` in the monorepo and has its own `go.mod` with zero imports from any other module in the project.

The scraper knows nothing about MongoDB, questions, scoring, users, or the application that consumes it. It accepts a URL, a PDF reader, or a search query and returns plain Go structs. Any Go project can import it as a library or run it as a standalone CLI.

**Module path**: `scraper`
**Go version**: 1.25+
**Primary dependency**: [Colly v2](https://github.com/gocolly/colly) for HTTP fetching and HTML traversal.

---

## Independence Contract

The scraper module enforces a hard boundary:

1. **Zero imports from `internal/`, `backend/`, or any sibling module.** The `go.mod` file contains only third-party dependencies (Colly, goquery, a PDF library). No application-specific packages appear anywhere in the import graph.
2. **No database drivers.** The scraper does not import `mongo-driver`, `pgx`, `sqlx`, or any persistence library.
3. **No LLM or AI dependencies.** No Ollama client, no OpenRouter calls, no embedding generation.
4. **No application models.** The scraper defines its own types (`ScrapedBook`, `Chapter`, `TOCEntry`, `Section`, `SearchResult`). These are generic content-extraction types, not application entities.
5. **No configuration coupling.** The scraper accepts a `Config` struct at construction time. It does not read `.env` files, environment variables, or shared config packages.

If you can run `cd scraper && go build ./...` without the rest of the monorepo present, the contract holds.

### Why This Matters

- The module can be extracted to its own repository and published on a Go module proxy at any time.
- Any developer can `go get` it for an unrelated project (a different web app, a CLI tool, a data pipeline) without pulling in MongoDB, Gin, Ollama, or anything else.
- Changes to the application's data models, API routes, or infrastructure never require changes to the scraper.

---

## Module Structure

```
scraper/
├── go.mod                   # Own module definition and dependencies
├── go.sum                   # Dependency checksums
├── types.go                 # All exported types: Config, ScrapedBook, Chapter, TOCEntry, Section, SearchResult
├── scraper.go               # Scraper struct and public API: New, ScrapeURL, ParsePDF, Search
├── fetcher.go               # Colly-based HTTP fetcher — URL to raw HTML
├── pdf.go                   # PDF reader — io.Reader to extracted text
├── extractor.go             # HTML to structured data (title, author, chapters, metadata)
├── toc.go                   # Table of contents detection and extraction
├── search.go                # Open Library + Google Books API clients
└── cmd/
    └── scraper/
        └── main.go          # Standalone CLI entry point
```

Each file has a single responsibility. The dependency flow within the module is:

```
scraper.go  (public API, orchestrator)
    ├── fetcher.go    (HTTP / Colly)
    ├── pdf.go        (PDF parsing)
    ├── extractor.go  (HTML → structured data)
    ├── toc.go        (ToC extraction)
    └── search.go     (external book APIs)
```

`types.go` is imported by all other files but imports nothing from the module itself.

---

## Types

All types live in `scraper/types.go`. They are exported and carry no BSON, JSON, or ORM struct tags beyond basic `json` tags for serialization convenience.

### Config

```go
// Config controls scraper behavior. Pass it to New().
type Config struct {
    // UserAgent is the HTTP User-Agent header sent with every request.
    // Default: "BookScraper/1.0"
    UserAgent string

    // MaxDepth limits how many links deep Colly will follow from the
    // initial URL. 0 means only the given page. Default: 2.
    MaxDepth int

    // AllowedDomains restricts which domains the fetcher will visit.
    // An empty slice means no restriction (use with caution).
    AllowedDomains []string

    // RequestTimeout is the per-request HTTP timeout.
    // Default: 30s.
    RequestTimeout time.Duration

    // RateLimit is the minimum delay between requests to the same
    // domain. Prevents aggressive scraping. Default: 1s.
    RateLimit time.Duration

    // Parallelism is the number of concurrent requests Colly may
    // issue. Default: 2.
    Parallelism int

    // MaxRetries is the number of times to retry failed requests.
    // Default: 3.
    MaxRetries int

    // CacheDir, if set, tells Colly to cache HTTP responses on disk
    // so repeated scrapes of the same URL are instant.
    CacheDir string

    // Verbose enables debug logging to stdout. Default: false.
    Verbose bool
}
```

### ScrapedBook

```go
// ScrapedBook is the top-level result of any scraping or parsing
// operation. It contains everything the scraper was able to extract
// from the source, with no interpretation or enrichment.
type ScrapedBook struct {
    // Title of the book. May be empty if extraction failed.
    Title string

    // Author name(s) as a single string.
    Author string

    // ISBN-10 or ISBN-13, if found.
    ISBN string

    // Publisher name, if found.
    Publisher string

    // Language code (e.g., "en"). Defaults to "en" if undetectable.
    Language string

    // Subject or category (e.g., "Physics", "History").
    Subject string

    // Description or summary blurb.
    Description string

    // CoverURL is a direct link to the book's cover image.
    CoverURL string

    // SourceURL is the original URL that was scraped (empty for PDFs).
    SourceURL string

    // SourceType indicates how the book was obtained: "url", "pdf", or "search".
    SourceType string

    // TOC is the extracted table of contents. May be empty.
    TOC []TOCEntry

    // Chapters contains the extracted content, split into chapters.
    // If the source has no clear chapter structure, the entire
    // content may appear as a single chapter.
    Chapters []Chapter

    // RawMetadata holds any extra key-value pairs the extractor
    // found (edition, page count, publication year, etc.) that
    // do not have a dedicated field above.
    RawMetadata map[string]string
}
```

### Chapter

```go
// Chapter represents a single chapter or major section of a book.
type Chapter struct {
    // Number is the 1-based chapter ordinal.
    Number int

    // Title is the chapter heading.
    Title string

    // Content is the full text of the chapter as a single string,
    // with sections concatenated if present.
    Content string

    // Sections breaks the chapter into named subsections, if the
    // source provides that level of structure.
    Sections []Section
}
```

### TOCEntry

```go
// TOCEntry represents one line in a table of contents.
type TOCEntry struct {
    // Number is the chapter/section number.
    Number int

    // Title is the human-readable heading.
    Title string

    // Page is the page number in the source (0 if unknown or
    // not applicable, e.g., for web content).
    Page int

    // Depth is the nesting level. 0 = top-level chapter,
    // 1 = section within a chapter, 2 = subsection, etc.
    Depth int
}
```

### Section

```go
// Section is a subsection within a chapter.
type Section struct {
    // Title is the section heading.
    Title string

    // Content is the text of this section.
    Content string
}
```

### SearchResult

```go
// SearchResult represents a single book found by the Search method.
// It contains enough metadata to display to a user and enough
// identifiers to fetch the full content later.
type SearchResult struct {
    // Title of the found book.
    Title string

    // Author name(s).
    Author string

    // ISBN, if available.
    ISBN string

    // Publisher name.
    Publisher string

    // Description or summary.
    Description string

    // CoverURL is a link to the cover image.
    CoverURL string

    // InfoURL is a link to more information (e.g., Open Library
    // book page, Google Books info page).
    InfoURL string

    // Source identifies which API returned this result:
    // "openlibrary" or "googlebooks".
    Source string
}
```

---

## Public API

The entire public surface of the scraper is four methods on a single `Scraper` struct, plus the `New` constructor. All methods return `(result, error)` pairs following Go convention.

### New

```go
// New creates a Scraper with the given configuration.
// Pass a zero-value Config to use all defaults.
func New(cfg Config) *Scraper
```

Initializes the Colly collector, applies rate limiting, sets the user agent, configures allowed domains, and prepares internal state. This is the only way to create a `Scraper`.

**Example**:

```go
s := scraper.New(scraper.Config{
    MaxDepth:       1,
    AllowedDomains: []string{"www.gutenberg.org"},
    RateLimit:      2 * time.Second,
})
```

### ScrapeURL

```go
// ScrapeURL fetches the given URL, follows links up to MaxDepth,
// extracts structured book content, and returns a ScrapedBook.
//
// Returns an error if the URL is unreachable, the response is not
// HTML, or content extraction fails completely.
func (s *Scraper) ScrapeURL(ctx context.Context, url string) (ScrapedBook, error)
```

This is the primary entry point for web-based scraping. Internally it calls the fetcher, then the extractor, then the ToC generator.

**Example**:

```go
book, err := s.ScrapeURL(ctx, "https://www.gutenberg.org/files/84/84-h/84-h.htm")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Title: %s, Chapters: %d\n", book.Title, len(book.Chapters))
```

**What it does internally**:

1. Fetches the URL using Colly (respects rate limiting, allowed domains, retries on 5xx).
2. Identifies the main content area (strips nav, ads, headers, footers).
3. Extracts metadata (title, author) from meta tags, headings, Open Graph, JSON-LD.
4. Detects chapter boundaries from headings (`<h1>`, `<h2>`, `<h3>`).
5. Splits content into `[]Chapter`.
6. Extracts or generates a table of contents.
7. Returns a populated `ScrapedBook` with `SourceType: "url"`.

### ParsePDF

```go
// ParsePDF reads a PDF from the given io.Reader, extracts text
// content, detects chapter boundaries, and returns a ScrapedBook.
//
// The reader must provide a complete, valid PDF. Encrypted or
// image-only PDFs will return an error.
func (s *Scraper) ParsePDF(ctx context.Context, r io.Reader) (ScrapedBook, error)
```

Accepts any `io.Reader` -- an `os.File`, an HTTP response body, a `bytes.Reader`, etc. This makes it composable with any storage backend (local filesystem, S3, R2, GCS) without the scraper knowing which one.

**Example**:

```go
f, _ := os.Open("textbook.pdf")
defer f.Close()

book, err := s.ParsePDF(ctx, f)
if err != nil {
    log.Fatal(err)
}
for _, ch := range book.Chapters {
    fmt.Printf("Chapter %d: %s (%d bytes)\n", ch.Number, ch.Title, len(ch.Content))
}
```

**What it does internally**:

1. Reads the entire PDF stream.
2. Extracts text from all pages, preserving paragraph structure where possible.
3. Looks for a "Table of Contents" or "Contents" page. If found, parses it.
4. If not found, detects chapter headings using pattern matching ("Chapter N", "CHAPTER N", ALL CAPS lines, PDF bookmarks/outline).
5. Splits text at chapter boundaries.
6. Falls back to page-based splitting if no chapters detected (every N pages becomes one chapter).
7. Returns a populated `ScrapedBook` with `SourceType: "pdf"`.

### Search

```go
// Search queries Open Library and Google Books for books matching
// the given query string. Results from both sources are merged and
// deduplicated by ISBN.
//
// Returns an empty slice (not an error) if no results are found.
// Returns an error only if both APIs are unreachable.
func (s *Scraper) Search(ctx context.Context, query string) ([]SearchResult, error)
```

**Example**:

```go
results, err := s.Search(ctx, "physics class 12 NCERT")
if err != nil {
    log.Fatal(err)
}
for _, r := range results {
    fmt.Printf("[%s] %s by %s\n", r.Source, r.Title, r.Author)
}
```

**What it does internally**:

1. Queries Open Library (`openlibrary.org/search.json?q=<query>`) and Google Books (`googleapis.com/books/v1/volumes?q=<query>`) in parallel.
2. Parses both responses, maps fields to `SearchResult` structs.
3. Deduplicates by ISBN (prefers the result with more complete metadata).
4. Returns the merged, deduplicated list.

---

## Internal Components

### Fetcher (`fetcher.go`)

The fetcher is a Colly-based HTTP client responsible for turning a URL into raw HTML. It is not exported; only `scraper.go` calls it.

**Responsibilities**:

- Configure the Colly collector from `Config` (user agent, allowed domains, rate limit, parallelism, cache, depth).
- Register `OnRequest`, `OnResponse`, `OnError`, and `OnHTML` callbacks.
- Respect `robots.txt` (Colly does this by default).
- Retry on 5xx errors (up to `MaxRetries`).
- Respect HTTP 429 (rate limit) responses and `Retry-After` headers.
- Return the collected HTML body and any response metadata (final URL after redirects, content type, status code).

**Key implementation detail**: The fetcher does not interpret HTML. It hands the raw bytes to the extractor.

### PDF Parser (`pdf.go`)

Reads a PDF from an `io.Reader` and produces raw text, page by page.

**Responsibilities**:

- Open the PDF stream and iterate through pages.
- Extract text content from each page, preserving paragraph structure where possible.
- Detect chapter boundaries using heuristics (in priority order):
  1. PDF bookmarks/outline (if the PDF has them).
  2. "Table of Contents" page parsing.
  3. Pattern matching: "Chapter N", "CHAPTER N", "Part N".
  4. Heading detection: lines in ALL CAPS or larger font sizes.
  5. Fallback: split every 10 pages into a chapter.
- Return the segmented text to `scraper.go`, which wraps it in `ScrapedBook` and `[]Chapter`.

**Limitations**:

- Scanned/image-only PDFs are not supported (no OCR). The parser returns an error for these.
- Complex layouts (multi-column, sidebars) may produce garbled text ordering.
- Mathematical formulas may lose formatting.
- Encrypted PDFs require the password or will fail.

### Extractor (`extractor.go`)

Converts raw HTML into the structured fields of `ScrapedBook`.

**Responsibilities**:

- Parse HTML using goquery (pulled in transitively via Colly).
- Extract metadata: `<title>`, `<meta name="author">`, `<meta name="description">`, Open Graph tags (`og:title`, `og:image`), schema.org JSON-LD, Dublin Core meta tags.
- Identify the main content area (skip navbars, footers, sidebars) using common HTML5 semantic elements (`<main>`, `<article>`, `.content`, `#content`) and heuristic scoring.
- Strip non-content elements: `<nav>`, `<header>`, `<footer>`, `.sidebar`, `<script>`, `<style>`.
- Split content into chapters by detecting heading patterns (`<h1>`, `<h2>`, numbered headings).
- Within each chapter, detect sections by `<h3>`/`<h4>` headings.
- Strip HTML tags from content, preserving paragraph breaks.
- Handle relative URLs (images, links) by resolving against the source URL.
- Collect any additional metadata into `RawMetadata`.

**Site-specific extractors**: The extractor supports domain-specific logic. When a URL matches a known domain (e.g., `gutenberg.org`), a specialized extraction function is used instead of the generic one. The generic extractor handles all unknown domains.

### ToC Generator (`toc.go`)

Builds the `[]TOCEntry` from either an explicit table of contents found in the source or from the headings extracted by the extractor.

**Strategies** (tried in order):

1. **Explicit ToC element**: Look for an HTML element with id/class containing "toc", "table-of-contents", or "contents". Parse its list items into `TOCEntry` structs.
2. **PDF outline**: If parsing a PDF, use the document's outline/bookmark tree.
3. **Heading inference**: Walk the extracted headings from the extractor, assign depth based on heading level (`<h1>` = 0, `<h2>` = 1, etc.), and build a ToC from those.

If no ToC can be constructed, the field is left as an empty slice -- this is not an error.

### Search (`search.go`)

Queries external book APIs and returns `[]SearchResult`.

**Open Library** (`openlibrary.org/search.json`):

- Free, no API key required.
- Returns title, author, ISBN, cover ID (converted to a cover URL via `https://covers.openlibrary.org/b/id/{cover_id}-L.jpg`), publisher.
- Rate limit: be polite (the scraper's `RateLimit` config applies here too).

**Google Books** (`www.googleapis.com/books/v1/volumes`):

- Free tier, no API key required for basic queries (limited quota).
- Returns title, authors, description, ISBN identifiers, cover thumbnail, info link.
- The scraper maps Google's `volumeInfo` fields to `SearchResult`.

**Deduplication**: Results from both sources are merged. If two results share the same ISBN, the one with more complete metadata wins.

**Graceful degradation**: If one API is down, results from the other are returned without error. An error is returned only if both APIs fail.

---

## Standalone CLI Usage

The scraper ships with a CLI at `scraper/cmd/scraper/main.go` that can be built and used without the rest of the project.

### Build

```bash
cd scraper
go build -o bin/scraper ./cmd/scraper
```

Or from the monorepo root:

```bash
go build -o bin/scraper ./scraper/cmd/scraper
```

### Commands

**Scrape a URL**:

```bash
./bin/scraper scrape --url "https://www.gutenberg.org/files/84/84-h/84-h.htm"
```

Outputs JSON-formatted `ScrapedBook` to stdout. Pipe it to `jq` for pretty-printing:

```bash
./bin/scraper scrape --url "..." | jq .
```

**Parse a PDF**:

```bash
./bin/scraper parse --file textbook.pdf
```

**Search for books**:

```bash
./bin/scraper search --query "physics class 12"
```

**Common flags** (apply to all commands):

| Flag | Default | Description |
|---|---|---|
| `--user-agent` | `BookScraper/1.0` | HTTP User-Agent string |
| `--max-depth` | `2` | Link-following depth for URL scraping |
| `--timeout` | `30s` | Per-request HTTP timeout |
| `--rate-limit` | `1s` | Minimum delay between requests |
| `--parallelism` | `2` | Max concurrent requests |
| `--cache-dir` | (none) | Directory for Colly's HTTP response cache |
| `--verbose` | `false` | Print debug information to stderr |
| `--output` | `-` (stdout) | Output file path; `-` for stdout |

### Use in a Script

```bash
# Scrape a book and save to a file
./bin/scraper scrape --url "https://example.com/book" --output book.json

# Search and pick the first result's ISBN
ISBN=$(./bin/scraper search --query "calculus" | jq -r '.[0].isbn')
echo "Found ISBN: $ISBN"
```

---

## Using the Scraper in a Different Project

Because the scraper is a standalone Go module, you can import it into any Go project.

### Step 1: Add the dependency

If the module is published:

```bash
go get scraper@latest
```

If you are working from a local checkout:

```go
// go.mod
require scraper v0.0.0

replace scraper => /path/to/scraper
```

### Step 2: Use the API

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "scraper"
)

func main() {
    s := scraper.New(scraper.Config{
        MaxDepth:       1,
        AllowedDomains: []string{"www.gutenberg.org"},
        RequestTimeout: 15 * time.Second,
        RateLimit:      2 * time.Second,
    })

    ctx := context.Background()

    // Scrape a URL
    book, err := s.ScrapeURL(ctx, "https://www.gutenberg.org/files/84/84-h/84-h.htm")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Book: %s by %s\n", book.Title, book.Author)
    fmt.Printf("Chapters: %d\n", len(book.Chapters))
    fmt.Printf("ToC entries: %d\n", len(book.TOC))

    for _, ch := range book.Chapters {
        fmt.Printf("  Chapter %d: %s (%d chars)\n", ch.Number, ch.Title, len(ch.Content))
    }
}
```

The code above works in a completely different repository with no knowledge of this project's backend, database, or AI pipeline.

---

## Extending for New Sources

To add support for a new content source (e.g., an EPUB parser, a specific publisher's API, or a sitemap crawler):

### 1. Add a new internal file

Create a new file in `scraper/` (e.g., `epub.go`). The file should export nothing -- only `scraper.go` calls it.

```go
// epub.go
package scraper // same package, not a sub-package

import "io"

// parseEPUB reads an EPUB from the given reader and returns
// extracted content in the same format as the PDF parser.
func parseEPUB(r io.Reader) (ScrapedBook, error) {
    // Implementation here.
    // Return a ScrapedBook with the same fields the rest of
    // the module expects. No special types needed.
}
```

### 2. Add a public method to `scraper.go`

```go
// ParseEPUB reads an EPUB from the given io.Reader and returns
// a ScrapedBook.
func (s *Scraper) ParseEPUB(ctx context.Context, r io.Reader) (ScrapedBook, error) {
    book, err := parseEPUB(r)
    if err != nil {
        return ScrapedBook{}, fmt.Errorf("epub parsing failed: %w", err)
    }
    book.SourceType = "epub"
    return book, nil
}
```

### 3. Add a CLI subcommand (optional)

In `cmd/scraper/main.go`, add a new subcommand that calls `s.ParseEPUB(ctx, file)`.

### 4. Add a site-specific extractor (for new websites)

For a domain with non-standard HTML structure, add a specialized extractor:

```go
// extractor_gutenberg.go
package scraper

import "github.com/PuerkitoBio/goquery"

func extractGutenberg(doc *goquery.Document, sourceURL string) (ScrapedBook, error) {
    // Site-specific extraction logic
    // Gutenberg has its own HTML structure for book content
}
```

Register it by domain pattern so `ScrapeURL` routes automatically:

```go
var extractors = map[string]extractorFunc{
    "gutenberg.org":  extractGutenberg,
    "openlibrary.org": extractOpenLibrary,
    // default: extractGeneric (used when no domain match)
}
```

### Rules for new sources

- The new source **must** return `ScrapedBook`. Do not create new top-level result types.
- The new source must not import anything from outside the `scraper` module.
- If the source requires an API key, accept it through `Config`, not through an environment variable.
- If the source requires a new third-party dependency, add it to `scraper/go.mod` only -- never to the root or backend `go.mod`.

---

## Adapter Pattern

The scraper produces generic types. The application consumes application-specific models (with MongoDB ObjectIDs, status fields, embedding vectors, etc.). The translation between the two happens in `internal/adapter/`, which is **outside** the scraper module.

### How It Works

```
scraper module                 internal/adapter/              application
(independent)                  (project-specific)             (models + DB)

  ScrapeURL(url)
  ParsePDF(reader)     ──>     Convert(ScrapedBook)    ──>    models.Book
  Search(query)                  maps fields                  models.Chapter
                                 sets defaults                (stored in MongoDB)
                                 adds app context
```

The adapter is a pure mapping function. It:

1. Takes a `scraper.ScrapedBook`.
2. Returns a `models.Book` and `[]models.Chapter`.
3. Sets application-specific defaults (e.g., `status = "pending"`).
4. Moves scraper fields that have no dedicated app field into `metadata`.

### Field Mapping Table

| Scraper Type | Scraper Field | App Model | App Field | Mapping Notes |
|---|---|---|---|---|
| `ScrapedBook` | `.Title` | `Book` | `.title` | Direct copy |
| `ScrapedBook` | `.Author` | `Book` | `.author` | Direct copy |
| `ScrapedBook` | `.ISBN` | `Book` | `.isbn` | Direct copy |
| `ScrapedBook` | `.Publisher` | `Book` | `.publisher` | Direct copy |
| `ScrapedBook` | `.Language` | `Book` | `.language` | Direct copy |
| `ScrapedBook` | `.Subject` | `Book` | `.subject` | Direct copy |
| `ScrapedBook` | `.Description` | `Book` | `.metadata["description"]` | Stored in flexible metadata object |
| `ScrapedBook` | `.CoverURL` | `Book` | `.cover_image_url` | Direct copy |
| `ScrapedBook` | `.SourceURL` | `Book` | `.source_url` | Direct copy |
| `ScrapedBook` | `.SourceType` | `Book` | `.source_type` | Direct copy |
| `ScrapedBook` | `.TOC` | `Book` | `.toc` | `[]TOCEntry` copied as embedded array |
| `ScrapedBook` | `.Chapters` | `Chapter` | (separate collection) | Each element becomes its own `Chapter` document |
| `ScrapedBook` | `.RawMetadata` | `Book` | `.metadata` | Merged into the metadata object |
| `Chapter` | `.Number` | `Chapter` | `.number` | Direct copy |
| `Chapter` | `.Title` | `Chapter` | `.title` | Direct copy |
| `Chapter` | `.Content` | `Chapter` | `.content` | Direct copy |
| `Chapter` | `.Sections` | `Chapter` | `.content` | Sections are concatenated into the chapter's content string |
| `TOCEntry` | `.Number` | `Book.toc[]` | `.number` | Direct copy |
| `TOCEntry` | `.Title` | `Book.toc[]` | `.title` | Direct copy |
| `TOCEntry` | `.Page` | `Book.toc[]` | `.page` | Direct copy |
| `TOCEntry` | `.Depth` | `Book.toc[]` | `.depth` | Direct copy |

### Fields Set by the Application (Not from Scraper)

These fields exist on the application models but have no corresponding scraper field. The scraper is intentionally unaware of them:

| App Field | Set By | Purpose |
|---|---|---|
| `Book.status` | Adapter (default: `"pending"`) | Processing pipeline state |
| `Book.grade_levels` | User or admin input | Target audience |
| `Book.education_system` | User or admin input | Curriculum framework |
| `Book.created_by` | Auth middleware | User who added the book |
| `Book.pdf_url` | Upload handler | Cloudflare R2 storage URL |
| `Book.processing_error` | Processing pipeline | Error message on failure |
| `Chapter.embedding` | Embedding service (Ollama) | 768-dim vector for semantic search |
| `Chapter.summary` | LLM service | AI-generated chapter summary |
| `Chapter.topics` | LLM service | Extracted topic tags |
| `Chapter.word_count` | Adapter (computed) | `len(strings.Fields(content))` |

### Writing Your Own Adapter

If you are using the scraper in a different project with different data models, write your own adapter. The pattern is straightforward:

```go
package myadapter

import "scraper"

// ToMyBook converts a scraped book to your application's book type.
func ToMyBook(sb scraper.ScrapedBook) MyBook {
    return MyBook{
        Name:     sb.Title,
        Writer:   sb.Author,
        Chapters: convertChapters(sb.Chapters),
        // Map whatever fields you need.
        // Ignore what you do not need.
    }
}

func convertChapters(chapters []scraper.Chapter) []MyChapter {
    out := make([]MyChapter, len(chapters))
    for i, ch := range chapters {
        out[i] = MyChapter{
            Index:   ch.Number,
            Heading: ch.Title,
            Body:    ch.Content,
        }
    }
    return out
}
```

The scraper does not care what you do with its output. It returns data; you decide how to store, transform, or display it.

---

## Error Handling

The scraper uses standard Go error handling. All public methods return `(result, error)`.

### Error Categories

| Category | Example | Returned By | Retried? |
|---|---|---|---|
| **Network errors** | DNS failure, connection refused, timeout | `ScrapeURL`, `Search` | Yes (up to `MaxRetries`) |
| **HTTP 5xx** | 500 Internal Server Error, 502 Bad Gateway | `ScrapeURL` | Yes (with backoff) |
| **HTTP 4xx** | 404 Not Found, 403 Forbidden | `ScrapeURL` | No |
| **HTTP 429** | Rate limited | `ScrapeURL` | Yes (respects `Retry-After` header) |
| **Parse errors** | Malformed HTML, no content extractable | `ScrapeURL` | No |
| **PDF errors** | Corrupted file, encrypted PDF, image-only PDF | `ParsePDF` | No |
| **API errors** | Open Library / Google Books returning errors | `Search` | No (graceful degradation) |
| **Invalid input** | Empty URL, nil reader | `ScrapeURL`, `ParsePDF` | No |

### Error Wrapping

All errors are wrapped with context using `fmt.Errorf("...: %w", err)` so callers can use `errors.Is` and `errors.As` for programmatic handling:

```go
book, err := s.ScrapeURL(ctx, url)
if err != nil {
    var netErr *net.OpError
    if errors.As(err, &netErr) {
        // Handle network-level failure
    }
    // Or just log/return it
    return fmt.Errorf("failed to scrape %s: %w", url, err)
}
```

### Partial Success

The scraper tries to extract as much as possible even when some parts fail:

- If the title cannot be extracted, `ScrapedBook.Title` is empty but no error is returned.
- If chapter detection fails, the entire content is returned as a single `Chapter` with `Number: 1` and `Title: "Full Content"`.
- If the ToC cannot be built, `ScrapedBook.TOC` is `nil` (empty slice) -- not an error.
- `Search` returns results from whichever APIs succeeded. An error is returned only when **all** APIs are unreachable.

### Context Cancellation

All public methods accept a `context.Context`. Passing a context with a deadline or cancellation allows callers to abort long-running scrapes:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

book, err := s.ScrapeURL(ctx, url)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Scrape timed out")
}
```
