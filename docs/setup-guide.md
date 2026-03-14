# Local Development Setup Guide

## Prerequisites

| Tool | Minimum Version | Purpose | Install |
|---|---|---|---|
| Go | 1.25+ | Backend, scraper, internal modules | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 22+ | Frontend (Next.js 16) | [nodejs.org](https://nodejs.org/) or `nvm install 22` |
| Docker | 24+ | MongoDB, Ollama, containerized builds | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/) |
| Docker Compose | v2+ (bundled with Docker Desktop) | Multi-service orchestration | Included with Docker Desktop |
| Git | 2.30+ | Version control | System package manager |

Optional but recommended:

| Tool | Purpose |
|---|---|
| `make` | Run project commands via Makefile |
| `curl` or `httpie` | Test API endpoints manually |
| `mongosh` | Inspect MongoDB data directly |

---

## Quick Start with Docker

The fastest way to get everything running. Docker Compose starts MongoDB, Ollama, the backend, and the frontend in one command.

### 1. Clone the repository

```bash
git clone https://github.com/ByteJar/book-scraping-and-question-generation.git
cd book-scraping-and-question-generation
```

### 2. Create your environment file

```bash
cp .env.example .env
```

Edit `.env` and fill in your keys (see the Environment Variables section below).

### 3. Start all services

```bash
docker compose up -d
```

This starts four containers:

| Container | Port | Image |
|---|---|---|
| `book-mongo` | 27017 | `mongo:7` |
| `book-ollama` | 11434 | `ollama/ollama:latest` |
| `book-backend` | 8080 | Custom Go build |
| `book-frontend` | 3000 | Custom Next.js build |

### 4. Pull the embedding model

Ollama needs the `nomic-embed-text` model for generating embeddings. After the Ollama container is running:

```bash
docker exec book-ollama ollama pull nomic-embed-text
```

This downloads ~300 MB. It only needs to happen once -- the model is persisted in the `ollama_data` Docker volume.

### 5. Verify everything is running

```bash
docker compose ps
```

All four services should show `Up` status.

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values. The `.env` file is gitignored and must never be committed.

| Variable | Required | Default | Description |
|---|---|---|---|
| `MONGODB_URL` | Yes | `mongodb://localhost:27017` | MongoDB connection string. Use `mongodb://mongo:27017` inside Docker network. |
| `MONGODB_DB` | Yes | `book_db` | Database name. |
| `OLLAMA_URL` | Yes | `http://localhost:11434` | Ollama API endpoint. Use `http://ollama:11434` inside Docker network. |
| `OPENROUTER_API_KEY` | Yes | (none) | Free-tier API key from [openrouter.ai](https://openrouter.ai/). Required for question generation and LLM-based scoring. |
| `CLOUDFLARE_R2_ACCOUNT_ID` | No | (none) | Cloudflare account ID. Only needed if supporting PDF uploads. |
| `CLOUDFLARE_R2_ACCESS_KEY_ID` | No | (none) | R2 API token key ID. |
| `CLOUDFLARE_R2_SECRET_ACCESS_KEY` | No | (none) | R2 API token secret. |
| `CLOUDFLARE_R2_BUCKET_NAME` | No | `book-pdfs` | R2 bucket name for uploaded PDFs. |

To get an OpenRouter API key:
1. Go to [openrouter.ai](https://openrouter.ai/) and sign up.
2. Navigate to Keys in your dashboard.
3. Create a new key. The free tier provides access to open-source models (Llama 3.1, Mistral, Gemma, Qwen) at no cost.

---

## Manual Setup (Without Docker for Services)

Use this approach if you prefer running Go and Node.js directly on your host while still using Docker for MongoDB and Ollama.

### 1. Start infrastructure services only

```bash
docker compose up -d mongo ollama
```

### 2. Pull the embedding model

```bash
docker exec book-ollama ollama pull nomic-embed-text
```

### 3. Set up the backend

The project uses a Go workspace (`go.work`) that links the `backend/` and `scraper/` modules (and `internal/` once created).

```bash
# From the project root
go work sync
cd backend && go mod download && cd ..
cd scraper && go mod download && cd ..
```

Run the backend:

```bash
cd backend
go run main.go
```

The API server starts on `http://localhost:8080`.

### 4. Set up the frontend

```bash
cd frontend
npm install
npm run dev
```

The development server starts on `http://localhost:3000` with hot reload.

### 5. Verify

```bash
# Backend health
curl http://localhost:8080/ping
# Expected: {"message":"pong"}

# Ollama health
curl http://localhost:11434/api/version
# Expected: {"version":"..."}

# Frontend
# Open http://localhost:3000 in your browser
```

---

## Running the Scraper CLI

The scraper is an independent Go module with its own CLI entry point at `scraper/cmd/scraper/main.go`. It can be run standalone without the backend.

### Run directly

```bash
cd scraper
go run main.go
```

The current `main.go` is a basic Colly example that visits `books.toscrape.com`. As the scraper module develops, the CLI will accept arguments for URL scraping, PDF parsing, and book search.

### Build and run as a binary

```bash
cd scraper
go build -o bin/scraper main.go
./bin/scraper
```

### Run via Make (once Makefile is added)

```bash
make scrape URL=https://example.com/book
```

---

## Makefile Commands

The project is designed to use a root `Makefile` for common operations. The following commands are documented in `CLAUDE.md` and will be available once the Makefile is created:

| Command | Description |
|---|---|
| `make dev` | Start all Docker services (`docker compose up -d`) |
| `make stop` | Stop all Docker services |
| `make api` | Run the backend only (Go, no Docker) |
| `make frontend` | Run the frontend dev server (Next.js, no Docker) |
| `make scrape URL=<url>` | Scrape a specific URL using the scraper CLI |
| `make embed` | Generate embeddings for all chapters missing them |
| `make generate` | Generate questions for all ready books |
| `make test` | Run all Go and frontend tests |

---

## Verify Installation

After setup, run these commands to confirm each component is healthy.

### Backend API

```bash
curl -s http://localhost:8080/ping | jq .
```

Expected response:

```json
{
  "message": "pong"
}
```

### MongoDB

```bash
# Using mongosh (if installed)
mongosh --eval "db.runCommand({ ping: 1 })"

# Or via Docker
docker exec book-mongo mongosh --eval "db.runCommand({ ping: 1 })"
```

Expected: `{ ok: 1 }`

### Ollama

```bash
curl -s http://localhost:11434/api/tags | jq '.models[].name'
```

Expected: should list `nomic-embed-text:latest` (after pulling).

Test embedding generation:

```bash
curl -s http://localhost:11434/api/embeddings \
  -d '{"model": "nomic-embed-text", "prompt": "test embedding"}' | jq '.embedding | length'
```

Expected: `768` (the embedding dimension).

### Frontend

Open `http://localhost:3000` in your browser. The default Next.js page should render.

---

## Common Issues

| Problem | Cause | Solution |
|---|---|---|
| `docker compose up` fails with port conflict on 27017 | Another MongoDB instance is running on the host | Stop the host MongoDB (`sudo systemctl stop mongod`) or change the port mapping in `docker-compose.yml` to `"27018:27017"` and update `MONGODB_URL` accordingly. |
| `docker compose up` fails with port conflict on 3000 | Another dev server is using port 3000 | Kill the other process (`lsof -ti:3000 \| xargs kill`) or change the frontend port mapping. |
| Ollama pull fails or times out | Network issues or slow connection | Retry with `docker exec book-ollama ollama pull nomic-embed-text`. The ~300 MB download may take a few minutes on slow connections. |
| `nomic-embed-text` not found after pull | Ollama container restarted without volume | Ensure the `ollama_data` volume is persisted in `docker-compose.yml` (it is by default). Re-pull if needed. |
| Backend cannot connect to MongoDB | Wrong `MONGODB_URL` for context | Inside Docker: use `mongodb://mongo:27017`. On host: use `mongodb://localhost:27017`. |
| Backend cannot connect to Ollama | Wrong `OLLAMA_URL` for context | Inside Docker: use `http://ollama:11434`. On host: use `http://localhost:11434`. |
| `go work sync` fails | Missing Go 1.25+ | Check `go version`. The project requires Go 1.25.0 or later. |
| Frontend `npm install` fails | Wrong Node.js version | Check `node --version`. Next.js 16 requires Node.js 22+. Use `nvm use 22` if using nvm. |
| OpenRouter returns 401 | Invalid or missing API key | Verify `OPENROUTER_API_KEY` in `.env`. Generate a new key at [openrouter.ai/keys](https://openrouter.ai/keys). |
| R2 upload fails | Missing or invalid R2 credentials | Cloudflare R2 credentials are optional. PDF upload will not work without them, but all other features function normally. |
| `go run` reports import cycle or missing module | `go.work` not synced | Run `go work sync` from the project root. |
| Docker build for backend fails | Build context too narrow | The backend Dockerfile must use the project root as its build context (not `./backend`) because it imports `internal/` and `scraper/` via `go.work`. This is configured correctly in `docker-compose.yml`. |

---

## Development Workflow Tips

### Hot reload

- **Frontend**: `npm run dev` includes hot reload out of the box via Next.js.
- **Backend**: Use [air](https://github.com/air-verse/air) for Go hot reload during development:
  ```bash
  go install github.com/air-verse/air@latest
  cd backend && air
  ```

### Go workspace

The `go.work` file at the project root links `./backend` and `./scraper` (and `./internal` when added). This means:
- You can import across modules without publishing them to a module proxy.
- IDE features (go-to-definition, autocomplete) work across module boundaries.
- Changes in `internal/` are immediately available to `backend/` and vice versa.

Run `go work sync` after adding new modules or changing dependencies.

### Database inspection

```bash
# Connect to MongoDB
docker exec -it book-mongo mongosh book_db

# Useful queries
db.books.find().pretty()
db.chapters.find({ book_id: ObjectId("...") }).pretty()
db.questions.countDocuments({ bloom_level: "analyze" })
```

### Testing API endpoints

```bash
# Add a book via URL
curl -X POST http://localhost:8080/api/v1/books \
  -H "Content-Type: application/json" \
  -d '{"source_url": "https://example.com/book", "source_type": "url"}'

# Search for a book
curl -X POST http://localhost:8080/api/v1/books/search \
  -H "Content-Type: application/json" \
  -d '{"query": "physics class 12"}'

# Get all books
curl http://localhost:8080/api/v1/books
```

### Git conventions

The project uses conventional commits:

```
feat(scraper): add PDF text extraction
fix(backend): handle empty chapter content
docs(setup): update environment variables table
refactor(internal): extract LLM client interface
```

Never commit the `.env` file. The `.gitignore` is already configured to exclude it.
