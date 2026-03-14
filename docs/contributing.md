# Contributing

Guidelines for working on the Book Scraping & Question Generation codebase. Read this before opening your first PR.

---

## Table of Contents

- [Code Organization](#code-organization)
- [Go Conventions](#go-conventions)
- [Frontend Conventions](#frontend-conventions)
- [Git Conventions](#git-conventions)
- [Testing](#testing)
- [Checklist: Adding a New API Endpoint](#checklist-adding-a-new-api-endpoint)
- [Checklist: Adding a New Question Type](#checklist-adding-a-new-question-type)
- [Checklist: Adding a New Scraper Source](#checklist-adding-a-new-scraper-source)

---

## Code Organization

The repository is a monorepo with three Go modules linked by `go.work` and one Node.js application.

```
go.work              # workspace: ./backend, ./scraper
backend/             # API server (Go + Gin)
scraper/             # Independent scraper module (Go + Colly)
internal/            # Shared project code (models, DB, LLM, scoring, adapter)
frontend/            # Next.js 16 + React 19 + Tailwind 4
docs/                # Project documentation
docker-compose.yml   # mongo, ollama, backend, frontend
```

### Where each type of code goes

| Code type | Location | Notes |
|---|---|---|
| MongoDB models (Go structs) | `internal/models/` | One file per entity: `book.go`, `question.go`, etc. |
| Database queries and CRUD | `internal/db/` | All MongoDB access. Nothing else talks to the driver directly. |
| Scraper-to-model conversion | `internal/adapter/` | Converts `scraper.ScrapedBook` to `models.Book` + `models.Chapter`. |
| LLM client (OpenRouter, Ollama) | `internal/llm/` | Provider-agnostic interface, prompt templates. |
| Embedding generation | `internal/embeddings/` | Ollama `nomic-embed-text` wrapper. |
| Answer scoring logic | `internal/scoring/` | Semantic, keyword, completeness dimensions. |
| File storage (R2) | `internal/storage/` | Cloudflare R2 / S3-compatible client. |
| HTTP handlers | `backend/handlers/` | Thin layer: parse request, call service, write response. |
| Business logic | `backend/services/` | Processing pipeline, question generation, scoring orchestration. |
| Auth, CORS, rate limiting | `backend/middleware/` | Gin middleware functions. |
| Scraping, fetching, parsing | `scraper/` | **Zero imports from `internal/` or `backend/`**. Own types only. |
| Scraper CLI | `scraper/cmd/scraper/` | Standalone entry point for running the scraper independently. |
| Pages and routes | `frontend/src/app/` | Next.js App Router. One directory per route segment. |
| Reusable UI components | `frontend/src/components/` | `BookCard`, `QuestionCard`, `ScoreBreakdown`, etc. |
| API client and types | `frontend/src/lib/` | `api.ts` for backend calls, `types.ts` for TypeScript interfaces. |

### Dependency rules

- `scraper/` must NEVER import from `internal/` or `backend/`. It is an independent, reusable module.
- `backend/` imports from `internal/` and `scraper/` (via `go.work`).
- `internal/` may import from `scraper/` (for the adapter) but never from `backend/`.
- `frontend/` communicates with `backend/` only over HTTP (via `lib/api.ts`).

---

## Go Conventions

### Error handling

- Always check errors. Never use `_` to discard an error.
- Wrap errors with context using `fmt.Errorf`:

```go
user, err := db.FindUser(ctx, id)
if err != nil {
    return fmt.Errorf("finding user %s: %w", id, err)
}
```

- Use `%w` (not `%v`) so callers can use `errors.Is` and `errors.As`.
- Return errors to the caller rather than logging and continuing. Let the top-level handler decide what to log.

### Naming

- Use `MixedCaps` / `mixedCaps`. No underscores in Go names.
- Exported names start with an uppercase letter. Keep unexported names as the default.
- Acronyms are all-caps: `ID`, `URL`, `HTTP`, `LLM`, `PDF`, `TOC`.
- Receiver names: short (one or two letters), consistent across methods of the same type.

```go
func (s *BookService) ProcessBook(ctx context.Context, id string) error { ... }
```

- Package names: short, lowercase, singular. `handler` not `handlers` at the package level (directory names may be plural for clarity, but the `package` declaration should be singular where practical).

### Interfaces

- Define interfaces where they are consumed, not where they are implemented.
- Keep interfaces small (one to three methods).
- Name single-method interfaces after the method with an `-er` suffix: `Reader`, `Scorer`, `Embedder`.

```go
// In backend/services/ (the consumer), not in internal/llm/ (the provider)
type QuestionGenerator interface {
    Generate(ctx context.Context, content string, bloomLevel string) ([]models.Question, error)
}
```

### Package dependencies

- No circular imports. The dependency graph flows downward:
  `backend/handlers` -> `backend/services` -> `internal/*` -> standard library
- `scraper/` is a leaf module with only external dependencies.

### Context

- `context.Context` is always the first parameter for functions that do I/O (database, HTTP, LLM calls).
- Never store `context.Context` in a struct.

### Environment variables

- Loaded from `.env` via `godotenv` at startup in `main.go`.
- Access with `os.Getenv`. Define all expected variables in `.env.example`.
- Never hardcode secrets or API keys.

---

## Frontend Conventions

### App Router (Next.js 16)

- Use the App Router (`src/app/`), not the Pages Router.
- Each route is a directory with a `page.tsx` file.
- Layouts go in `layout.tsx` within the appropriate route segment.
- Loading states use `loading.tsx`, error boundaries use `error.tsx`.

### Server Components vs. Client Components

- Server Components are the default. Do not add `"use client"` unless the component needs browser APIs, event handlers, or React state/effects.
- Data fetching happens in Server Components using `async` functions.
- Client Components are for interactivity: forms, click handlers, `useState`, `useEffect`.

### Data fetching

- Server-side: call the backend API directly in Server Components using `fetch` or the `lib/api.ts` client.
- Client-side: use React hooks (`useEffect` or a data-fetching library) only when the data must be loaded after user interaction.
- Always define response types in `lib/types.ts` (mirroring Go models).

### Tailwind 4

- Import Tailwind with `@import "tailwindcss"` in `globals.css`. Do not use the old `@tailwind base/components/utilities` directives.
- Use Tailwind utility classes directly in JSX. Avoid custom CSS unless absolutely necessary.
- Use Tailwind's built-in responsive (`sm:`, `md:`, `lg:`) and state (`hover:`, `focus:`, `disabled:`) variants.

### TypeScript

- Strict mode is enabled. Do not use `any` unless there is no alternative.
- Define all API response types in `frontend/src/lib/types.ts`.
- Use `interface` for object shapes, `type` for unions and intersections.
- Props interfaces are named `ComponentNameProps`:

```tsx
interface BookCardProps {
  title: string;
  author: string;
  chapterCount: number;
}

export function BookCard({ title, author, chapterCount }: BookCardProps) {
  // ...
}
```

### File naming

- Components: `PascalCase.tsx` (e.g., `BookCard.tsx`).
- Utilities and lib files: `camelCase.ts` (e.g., `api.ts`, `types.ts`).
- Route files follow Next.js conventions: `page.tsx`, `layout.tsx`, `loading.tsx`, `error.tsx`.

---

## Git Conventions

### Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): subject
```

**Types:**

| Type | When to use |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, whitespace (no logic change) |
| `refactor` | Code restructuring (no feature or fix) |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `chore` | Build, CI, tooling, dependency updates |
| `ci` | CI/CD configuration |
| `build` | Build system or external dependencies |

**Scopes** (optional): `backend`, `frontend`, `scraper`, `internal`, `db`, `llm`, `docs`, `docker`.

Examples:

```
feat(backend): add POST /api/books endpoint for URL scraping
fix(scraper): handle relative links in TOC extraction
refactor(internal): extract embedding logic into embeddings package
test(backend): add handler tests for question generation
docs: update API reference with scoring endpoints
chore(docker): pin Ollama image to specific version
```

### Branch naming

```
type/short-description
```

Examples: `feat/book-upload`, `fix/pdf-parsing-crash`, `refactor/scoring-interface`.

### What NOT to commit

- **Never commit `.env`**. It contains secrets (API keys, R2 credentials). It is listed in `.gitignore`.
- Do not commit `node_modules/`, `bin/`, `vendor/`, `.next/`, `ollama_data/`.
- Do not commit IDE-specific files (`.idea/`, `.vscode/`).
- If you add a new environment variable, add it to `.env.example` with a placeholder value.

---

## Testing

### Go tests

- Test files live alongside the code they test: `handler_test.go` next to `handler.go`.
- Use the standard `testing` package. Name test functions `TestFunctionName_Scenario`:

```go
func TestProcessBook_InvalidURL(t *testing.T) { ... }
func TestScoreAnswer_PerfectMatch(t *testing.T) { ... }
```

- Use table-driven tests for functions with multiple input/output cases:

```go
func TestNormalizeTitle(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {"simple", "Hello World", "hello-world"},
        {"extra spaces", "  too  many  spaces  ", "too-many-spaces"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := NormalizeTitle(tt.input)
            if got != tt.want {
                t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

- Use interfaces for dependencies so you can inject test doubles. Do not call real databases or external APIs in unit tests.
- Run Go tests: `go test ./...` from the module root, or `make test` from the repo root.

### Scraper tests

- The scraper module has its own tests, run independently: `cd scraper && go test ./...`
- Test with fixture HTML files (stored in `scraper/testdata/`) rather than making live HTTP requests.
- Use `httptest.NewServer` to simulate web servers when testing fetcher behavior.

### Frontend tests

- Use [Vitest](https://vitest.dev/) as the test runner (or Jest if configured).
- Test files use `.test.tsx` or `.test.ts` suffix.
- Test components with [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/):

```tsx
import { render, screen } from "@testing-library/react";
import { BookCard } from "./BookCard";

test("renders book title", () => {
  render(<BookCard title="Test Book" author="Author" chapterCount={5} />);
  expect(screen.getByText("Test Book")).toBeInTheDocument();
});
```

- Test `lib/api.ts` functions by mocking `fetch`.
- Run frontend tests: `cd frontend && npm test`.

### Integration tests

- Integration tests verify that multiple components work together (e.g., handler -> service -> database).
- Use a dedicated test MongoDB instance (via Docker or [testcontainers](https://golang.testcontainers.org/)).
- Tag integration tests with a build tag so they do not run by default:

```go
//go:build integration

package handlers_test
```

- Run integration tests: `go test -tags=integration ./...`
- Integration tests should clean up after themselves (drop test collections or use unique collection names per test run).

---

## Checklist: Adding a New API Endpoint

Use this step-by-step checklist every time you add a new endpoint to the backend.

- [ ] **1. Define or update the model** in `internal/models/` if the endpoint requires a new or modified entity.
- [ ] **2. Add database operations** in `internal/db/` for any new queries, inserts, or updates the endpoint needs.
- [ ] **3. Write the service function** in `backend/services/`. This contains the business logic. Accept interfaces for external dependencies (DB, LLM, storage).
- [ ] **4. Write the handler** in `backend/handlers/`. Keep it thin:
  - Parse and validate the request (path params, query params, JSON body).
  - Call the service function.
  - Return the appropriate HTTP status and JSON response.
  - Handle errors with consistent error response format.
- [ ] **5. Register the route** in `backend/main.go` (or a dedicated router file if one exists). Group related endpoints under a common prefix.
- [ ] **6. Add middleware** if the endpoint needs authentication, authorization, or rate limiting.
- [ ] **7. Write tests**:
  - Unit test the service function with mocked dependencies.
  - Unit test the handler with a mocked service (use `httptest.NewRecorder`).
  - Add an integration test if the endpoint has complex DB interactions.
- [ ] **8. Update `docs/api-reference.md`** with the endpoint's method, path, request schema, response schema, and example.
- [ ] **9. Update `frontend/src/lib/api.ts`** with a function to call the new endpoint.
- [ ] **10. Update `frontend/src/lib/types.ts`** if the endpoint introduces new response types.

---

## Checklist: Adding a New Question Type

The platform generates questions in multiple formats (MCQ, True/False, Fill in the blank, Short answer, Essay, Match the following, Assertion-Reasoning). To add a new type:

- [ ] **1. Add the type constant** to the question type enum/constants in `internal/models/question.go`. Use a short, lowercase, hyphenated name (e.g., `"sequence-ordering"`).
- [ ] **2. Update the Question model** in `internal/models/` if the new type requires additional fields (e.g., a list of items to order). Add the fields with appropriate `bson` and `json` tags.
- [ ] **3. Write the generation prompt** for the LLM. Add a prompt template in `internal/llm/` that produces questions of the new type, including Bloom's Taxonomy level and enrichment fields (what/when/how/who).
- [ ] **4. Update the question generation service** in `backend/services/` to include the new type in the generation pipeline. Make sure it respects the per-chapter question distribution logic.
- [ ] **5. Update the scoring logic** in `internal/scoring/`:
  - If the new type is objective (single correct answer): add it to the simple right/wrong scoring path.
  - If the new type is subjective: implement the three-dimension scoring (semantic, keyword, completeness) or define a custom scoring strategy.
- [ ] **6. Add the TypeScript type** to `frontend/src/lib/types.ts`. Update the `QuestionType` union and add any new fields.
- [ ] **7. Build the UI component** in `frontend/src/components/` for rendering and answering the new question type.
- [ ] **8. Update the practice page** (`frontend/src/app/practice/[id]/page.tsx`) to render the new question type component.
- [ ] **9. Update the results page** (`frontend/src/app/results/[id]/page.tsx`) if scoring display differs for the new type.
- [ ] **10. Write tests**:
  - Test the LLM prompt produces valid output (mock the LLM, validate structure).
  - Test the scoring logic for the new type.
  - Test the frontend component renders correctly and handles user input.
- [ ] **11. Update documentation**: add the new type to `docs/data-models.md` (Question entity) and `docs/llm-strategy.md` (prompt templates).

---

## Checklist: Adding a New Scraper Source

The scraper supports multiple input methods (URL scraping, PDF upload, book search). To add a new source (e.g., EPUB files, a new book API):

- [ ] **1. Add the source type constant** to `scraper/types.go`. The scraper defines its own types -- do not import from `internal/`.
- [ ] **2. Implement the source handler** in a new file in `scraper/` (e.g., `epub.go`). The function must return `ScrapedBook` (the scraper's own type), populated with title, author, chapters, and TOC.
- [ ] **3. Expose the public API** by adding a method to the scraper's main struct in `scraper/scraper.go` (e.g., `ParseEPUB(ctx context.Context, path string) (*ScrapedBook, error)`).
- [ ] **4. Add any new dependencies** to `scraper/go.mod`. Run `cd scraper && go mod tidy`.
- [ ] **5. Verify independence**: confirm that `scraper/go.mod` has zero references to `internal/` or `backend/`. The scraper must remain standalone.
- [ ] **6. Update the adapter** in `internal/adapter/` if the new source produces data that maps differently to `models.Book` / `models.Chapter` (e.g., different metadata fields).
- [ ] **7. Add the backend endpoint or modify existing ones** in `backend/handlers/` and `backend/services/` to accept the new source type as input.
- [ ] **8. Update the scraper CLI** (`scraper/cmd/scraper/main.go`) to support the new source via command-line flags.
- [ ] **9. Write tests**:
  - Unit test the source handler with fixture files in `scraper/testdata/`.
  - Test the adapter conversion for the new source type.
  - Test the backend endpoint with a mocked scraper.
- [ ] **10. Update documentation**: add the new source to `docs/scraper-module.md` and update `CLAUDE.md` if the monorepo structure section needs changes.
- [ ] **11. Update `.env.example`** if the new source requires additional environment variables (e.g., API keys for a new book service).
