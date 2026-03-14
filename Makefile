.PHONY: dev stop api frontend scrape embed generate test build pull-models

# Start all Docker services
dev:
	docker compose up -d

# Stop all services
stop:
	docker compose down

# Run backend API server
api:
	cd backend && go run .

# Run frontend dev server
frontend:
	cd frontend && npm run dev

# Scrape a URL (usage: make scrape URL=https://example.com/book)
scrape:
	cd scraper && go run ./cmd/scraper -url "$(URL)"

# Parse a PDF (usage: make parse PDF=./path/to/book.pdf)
parse:
	cd scraper && go run ./cmd/scraper -pdf "$(PDF)"

# Search for books (usage: make search Q="physics class 12")
search:
	cd scraper && go run ./cmd/scraper -search "$(Q)"

# Generate embeddings for all chapters
embed:
	cd backend && go run . -task embed

# Generate questions for all books
generate:
	cd backend && go run . -task generate

# Run all Go tests
test:
	go test ./internal/... ./backend/... ./scraper/...

# Build all Go binaries
build:
	cd backend && go build -o ../bin/server .
	cd scraper && go build -o ../bin/scraper ./cmd/scraper

# Pull required Ollama models
pull-models:
	ollama pull nomic-embed-text
	ollama pull llama3.2:3b

# Tidy all modules
tidy:
	cd internal && go mod tidy
	cd backend && go mod tidy
	cd scraper && go mod tidy
