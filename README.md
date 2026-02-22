# Book Scraping and Question Generation

A tool to scrape books, extract text, and generate questions using AI.

## Project Structure

```text
book-scraping-and-question-generation
├── cmd/
│   └── main.go           # Entry point: Wires everything together
├── internal/             # Private code: Logic only for this project
│   ├── ai/
│   │   ├── client.go     # Unified interface for Gemini/Groq/DeepSeek
│   │   └── models.go     # Prompt templates (TOC generation, Summary)
│   ├── extractor/
│   │   ├── fetcher.go    # Web downloading logic (HTTP)
│   │   └── parser.go     # PDF/Text extraction (using pdfcpu/fitz)
│   ├── search/
│   │   └── searxng.go    # SearXNG client & question fetching
│   ├── storage/
│   │   ├── mongo.go      # MongoDB connection & CRUD
│   │   └── vector.go     # Vector index & $vectorSearch logic
│   └── models/
│       └── domain.go     # Shared Go structs (Book, Chapter, Question)
├── scripts/              # Automation (Database migrations, Docker setup)
├── .env                  # Secret API Keys (ignored by Git)
├── .gitignore            # Ensures .env and large PDFs aren't pushed
├── docker-compose.yml    # Sets up MongoDB, SearXNG, and Firecrawl
├── go.mod                # Module definition & dependencies
└── go.sum                # Checksums for security
```

## Setup

- [ ] Install dependencies
- [ ] Configure `.env`
- [ ] Run `docker-compose up`

## Usage

```bash
go run cmd/main.go
```
