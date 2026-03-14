# Book Scraping & Question Generation

## Project Overview

An AI-powered exam preparation platform that extracts content from books (via URL, PDF upload, or search), generates exam-style questions tagged with Bloom's Taxonomy levels, and scores student answers using multi-dimensional AI evaluation.

**Core Flow:**
1. Book content enters the system (URL scraping, PDF upload, or book search)
2. Content is processed: metadata extracted, ToC documented, chapters split, text vectorized
3. Questions generated per chapter: MCQ, essay, T/F, fill-blank, short answer, assertion-reasoning
4. Each question tagged with Bloom's level, difficulty, enrichment (what/when/how/who)
5. Students answer questions and receive AI-powered scoring with detailed feedback
6. Admins monitor quality, manage sources, view analytics

**Three book input methods:**
1. URL — scraper fetches and extracts content from a web page
2. PDF upload — user uploads a PDF, stored in R2, text extracted
3. Search — search Open Library / Google Books, import metadata

## Tech Stack

| Layer | Technology | Notes |
|---|---|---|
| Backend | Go + Gin | API server, business logic |
| Scraper | Go + Colly | **Independent module** — reusable in other projects |
| Frontend | Next.js 16 + React 19 + Tailwind 4 | App Router, Server Components |
| Database | MongoDB 7 | Documents + vector search (Atlas M0 for prod, local for dev) |
| Vector Search | MongoDB vector search | Embeddings stored in chapter documents |
| LLM | OpenRouter (free models) | Primary: `meta-llama/llama-3.1-8b-instruct:free`, fallback: Ollama |
| Embeddings | Ollama `nomic-embed-text` | 768-dim, runs on CPU, no GPU needed |
| File Storage | Cloudflare R2 | S3-compatible, 10 GB free, user-uploaded PDFs |
| Container | Docker Compose | mongo, ollama, backend, frontend |

**Everything is free and open-source. No paid APIs.**

## Monorepo Structure

```
├── go.work                      # Go workspace linking backend, scraper, internal
├── docker-compose.yml           # All services: mongo, ollama, backend, frontend
├── Makefile                     # Unified commands
├── CLAUDE.md                    # This file
├── .env.example                 # Template (committed, no secrets)
├── .env                         # Secrets (gitignored, NEVER commit)
│
├── scraper/                     # INDEPENDENT Go module (pluggable, reusable)
│   ├── go.mod                   # Own deps only — NO imports from internal/ or backend/
│   ├── types.go                 # ScrapedBook, Chapter, TOCEntry, Section, SearchResult
│   ├── scraper.go               # Public API: New(), ScrapeURL(), ParsePDF(), Search()
│   ├── fetcher.go               # URL → raw HTML (Colly-based)
│   ├── pdf.go                   # PDF → extracted text
│   ├── extractor.go             # HTML → structured ScrapedBook
│   ├── toc.go                   # Extract/generate table of contents
│   ├── search.go                # Open Library + Google Books API
│   └── cmd/scraper/main.go     # Standalone CLI entry point
│
├── internal/                    # Project-specific shared Go module
│   ├── go.mod
│   ├── models/                  # Book, Chapter, Question, User, UserAnswer, AllowedSource
│   ├── db/                      # MongoDB connection, CRUD, vector search queries
│   ├── adapter/                 # scraper.ScrapedBook → models.Book + models.Chapter
│   ├── llm/                     # OpenRouter + Ollama client (provider-agnostic interface)
│   ├── embeddings/              # Ollama nomic-embed-text embedding generation
│   ├── scoring/                 # Answer scoring: semantic, keyword, completeness
│   └── storage/                 # Cloudflare R2 (S3-compatible) client
│
├── backend/                     # Go module — API server
│   ├── go.mod
│   ├── main.go                  # Entry point: init DB, register routes, start server
│   ├── handlers/                # HTTP handlers: books, questions, answers, auth, admin
│   ├── services/                # Business logic: processing pipeline, question gen, scoring
│   └── middleware/              # Auth (JWT), CORS, rate limiting
│
├── frontend/                    # Next.js app
│   ├── package.json
│   └── src/
│       ├── app/
│       │   ├── page.tsx                 # Landing / book search
│       │   ├── books/[id]/page.tsx      # Book detail + chapters
│       │   ├── practice/[id]/page.tsx   # Question answering interface
│       │   ├── results/[id]/page.tsx    # Score breakdown + feedback
│       │   ├── dashboard/page.tsx       # Student progress
│       │   └── admin/                   # Admin panel pages
│       ├── components/                  # BookCard, QuestionCard, ScoreBreakdown, etc.
│       └── lib/
│           ├── api.ts                   # Backend API client
│           └── types.ts                 # TypeScript interfaces (mirrors Go models)
│
└── docs/                        # Comprehensive documentation
    ├── README.md                # Doc index + reading order
    ├── architecture.md          # System design, diagrams, ADRs
    ├── data-models.md           # All entities, fields, indexes
    ├── api-reference.md         # REST API endpoints
    ├── scraper-module.md        # Independent scraper docs
    ├── llm-strategy.md          # Prompts, Bloom's Taxonomy, fallback logic
    ├── scoring-system.md        # Three-dimension scoring system
    ├── embeddings.md            # Vector pipeline, MongoDB vector search
    ├── setup-guide.md           # Local dev setup
    ├── deployment.md            # Docker, production, external services
    └── contributing.md          # Code conventions, checklists
```

## Key Architectural Decisions

### Scraper is INDEPENDENT
- The `scraper/` module has zero imports from `internal/` or `backend/`
- It returns its own types (ScrapedBook, Chapter, TOCEntry)
- The project uses `internal/adapter/` to convert scraper output to app models
- This allows the scraper to be extracted to its own repo and reused in other projects

### Database: MongoDB (not Postgres)
- Semi-structured book data with embedded arrays
- MongoDB 7+ supports vector search for embeddings
- Atlas free tier (M0) for production with `$vectorSearch`
- Local mongo:7 for dev with application-level cosine similarity fallback

### LLM: OpenRouter + Ollama Fallback
- OpenRouter routes to free open-source models (Llama 3.1, Mistral, Gemma, Qwen)
- Ollama as offline/local fallback (CPU-only, no GPU needed)
- All models are open-source and free
- Fallback chain: OpenRouter → Ollama → cached/pre-generated results

### Bloom's Taxonomy for Questions
Six cognitive levels, each producing different question styles:
- **Remember**: Define, List, Recall
- **Understand**: Explain, Describe, Summarize
- **Apply**: Demonstrate, Calculate, Use
- **Analyze**: Compare, Contrast, Differentiate
- **Evaluate**: Judge, Argue, Critique
- **Create**: Design, Propose, Construct

### Three-Dimension Answer Scoring (essays)
| Dimension | Weight | What it measures |
|---|---|---|
| Semantic | 50% | LLM compares meaning of student vs model answer |
| Completeness | 30% | Coverage of key points |
| Keyword | 20% | Presence of key terms |

MCQ/T-F/Fill-blank: simple right/wrong.

### Question Enrichment
Every question includes:
- **What**: what concept/skill is being tested
- **When**: when is this relevant (exam context, real-world)
- **How**: how to approach this question
- **Who**: target audience (grade, exam type, education level)

### Question Variants
Same concept generated in multiple formats:
MCQ, True/False, Fill in the blank, Short answer, Essay, Match the following, Assertion-Reasoning

### File Storage: Cloudflare R2
- User-uploaded PDFs stored in R2 (not server filesystem)
- S3-compatible API via AWS SDK for Go
- 10 GB free, zero egress fees

## Data Model Summary

| Entity | Collection | Key Fields |
|---|---|---|
| Book | `books` | title, author, subject, grade_levels, source_type, status, toc |
| Chapter | `chapters` | book_id, number, title, content, embedding[768], topics |
| Question | `questions` | book_id, chapter_id, question_type, bloom_level, difficulty, enrichment |
| UserAnswer | `user_answers` | user_id, question_id, semantic/keyword/completeness scores, feedback |
| User | `users` | email, name, role (student/admin), grade_level, exam_preparing_for |
| AllowedSource | `allowed_sources` | url_pattern, source_type, enabled, added_by |

## Processing Pipeline

```
Book Input (URL/PDF/Search)
    → Scraper extracts content (ScrapedBook)
    → Adapter converts to Book + Chapter models
    → Store in MongoDB (status: processing)
    → Ollama generates embeddings per chapter → store in chapter.embedding
    → OpenRouter generates questions per chapter → store in questions collection
    → Status: ready
```

## Conventions

### Go
- `go.work` links all Go modules (internal, backend, scraper)
- Shared code in `internal/` — models, DB, LLM, scoring
- `context.Context` as first parameter for all I/O functions
- Errors wrapped with context: `fmt.Errorf("failed to X: %w", err)`
- Environment variables loaded from `.env` via godotenv

### Frontend
- Next.js App Router (not Pages Router)
- Server Components by default, Client Components only for interactivity
- Tailwind 4 (`@import "tailwindcss"`, not `@tailwind` directives)
- API calls via `lib/api.ts`

### Git
- Never commit `.env`
- Conventional commits: `type(scope): subject`
- Types: feat, fix, docs, style, refactor, perf, chore, test, ci, build

### Docker
- All services in root `docker-compose.yml`
- Backend needs full workspace context (imports internal/ and scraper/)
- Scraper CLI runs on-demand, not always-on

## Key Environment Variables

```
MONGODB_URL=mongodb://localhost:27017      # Local dev
MONGODB_DB=book_db
OPENROUTER_API_KEY=sk-or-...               # Free tier key
OLLAMA_URL=http://localhost:11434           # Local Ollama
CLOUDFLARE_R2_ACCOUNT_ID=...
CLOUDFLARE_R2_ACCESS_KEY_ID=...
CLOUDFLARE_R2_SECRET_ACCESS_KEY=...
CLOUDFLARE_R2_BUCKET_NAME=book-pdfs
```

## Useful Commands

```bash
make dev          # Start all Docker services
make stop         # Stop all services
make api          # Run backend only
make frontend     # Run frontend dev server
make scrape URL=  # Scrape a specific URL
make embed        # Generate embeddings for all chapters
make generate     # Generate questions for all books
make test         # Run all tests
```
