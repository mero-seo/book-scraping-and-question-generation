# Documentation

## Book Scraping & Question Generation — AI-Powered Exam Prep Platform

An open-source platform that extracts content from books (via URL, PDF upload, or search), generates exam-style questions using open-source LLMs, and scores student answers with multi-dimensional AI evaluation.

---

## Reading Order

| Audience | Start here |
|---|---|
| **New developer** | [Setup Guide](setup-guide.md) → [Architecture](architecture.md) → [Data Models](data-models.md) |
| **Building features** | [Data Models](data-models.md) → [API Reference](api-reference.md) → [Contributing](contributing.md) |
| **Working on scraper** | [Scraper Module](scraper-module.md) |
| **Working on LLM/questions** | [LLM Strategy](llm-strategy.md) → [Scoring System](scoring-system.md) |
| **Working on search** | [Embeddings](embeddings.md) |
| **DevOps/deployment** | [Setup Guide](setup-guide.md) → [Deployment](deployment.md) |

---

## Documentation Index

### Architecture & Design
- [**architecture.md**](architecture.md) — System design, component diagram, data flows, architectural decisions
- [**data-models.md**](data-models.md) — All entities, fields, relationships, MongoDB indexes

### API
- [**api-reference.md**](api-reference.md) — Complete REST API endpoint reference with request/response schemas

### Subsystems
- [**scraper-module.md**](scraper-module.md) — Independent scraper module (reusable in other projects)
- [**llm-strategy.md**](llm-strategy.md) — OpenRouter + Ollama fallback, prompt templates, Bloom's Taxonomy
- [**scoring-system.md**](scoring-system.md) — Semantic, keyword, and completeness scoring for answers
- [**embeddings.md**](embeddings.md) — Vector embeddings pipeline and MongoDB vector search

### Operations
- [**setup-guide.md**](setup-guide.md) — Local development setup from zero
- [**deployment.md**](deployment.md) — Docker, production configuration, external services

### Contributing
- [**contributing.md**](contributing.md) — Code conventions, git workflow, adding features
