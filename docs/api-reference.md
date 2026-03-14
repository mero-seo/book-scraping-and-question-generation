# API Reference

Complete REST API reference for the Book Scraping & Question Generation platform. Backend built with Go + Gin.

---

## Conventions

### Base URL

```
http://localhost:8080
```

All versioned endpoints are prefixed with `/api/v1`. The health check endpoint (`/ping`) lives at the root.

### Content Type

All request and response bodies are JSON (`application/json`), except file upload endpoints which use `multipart/form-data`.

### Authentication

Protected endpoints require a JWT bearer token in the `Authorization` header:

```
Authorization: Bearer <token>
```

Tokens are obtained via the `/api/v1/auth/login` or `/api/v1/auth/register` endpoints and contain the user's ID and role. Tokens expire after 24 hours.

**Roles**:

| Role | Description | Access |
|---|---|---|
| `student` | Default role on registration | Read books/chapters/questions, submit answers, view own stats |
| `admin` | Assigned manually or via seed | All student permissions + manage books, generate questions, manage sources, view analytics, manage users |

Endpoints marked **Public** do not require authentication. Endpoints marked **Admin** require `role: "admin"`.

### Pagination

List endpoints support pagination via query parameters:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number (1-based) |
| `limit` | int | 20 | Items per page (max 100) |

Paginated responses include a `pagination` object:

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 142,
    "total_pages": 8
  }
}
```

### Error Format

All errors follow a consistent structure:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE"
}
```

**Standard error codes**:

| HTTP Status | Code | Description |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid request body or parameters |
| 400 | `VALIDATION_ERROR` | Field validation failed |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT token |
| 403 | `FORBIDDEN` | Insufficient permissions (wrong role) |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `CONFLICT` | Duplicate resource (e.g., email already registered) |
| 413 | `FILE_TOO_LARGE` | Uploaded file exceeds size limit |
| 415 | `UNSUPPORTED_TYPE` | Invalid file type uploaded |
| 422 | `UNPROCESSABLE_ENTITY` | Request is syntactically valid but semantically wrong |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
| 503 | `SERVICE_UNAVAILABLE` | Dependent service (LLM, DB) is down |

### Common Types

**ObjectID**: MongoDB ObjectIDs are serialized as 24-character hex strings (e.g., `"665f1a2b3c4d5e6f7a8b9c0d"`).

**Timestamps**: All timestamps are ISO 8601 format in UTC (e.g., `"2025-06-01T14:30:00Z"`).

**Enums used across endpoints**:

| Enum | Values |
|---|---|
| `question_type` | `mcq`, `essay`, `fill_blank`, `true_false`, `short_answer`, `match`, `assertion_reasoning` |
| `bloom_level` | `remember`, `understand`, `apply`, `analyze`, `evaluate`, `create` |
| `difficulty` | `easy`, `medium`, `hard` |
| `source_type` (book) | `url`, `pdf`, `search` |
| `status` (book) | `pending`, `processing`, `ready`, `failed` |
| `source_type` (allowed source) | `scrape`, `api` |
| `role` (user) | `student`, `admin` |

---

## Endpoints

### Health Check

#### GET /ping

Check that the server is running.

**Auth**: Public

**Parameters**: None

**Success Response** (`200 OK`):

```json
{
  "message": "pong"
}
```

**Example**:

```bash
curl http://localhost:8080/ping
```

```json
{
  "message": "pong"
}
```

---

### Authentication

#### POST /api/v1/auth/register

Register a new user account. Returns a JWT token on success.

**Auth**: Public

**Request Body**:

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | Valid email address (unique) |
| `name` | string | yes | Display name (2-100 chars) |
| `password` | string | yes | Password (min 8 chars) |
| `grade_level` | string | no | Current grade or education level |
| `education_system` | string | no | Education system (e.g., "CBSE", "US Common Core") |
| `exam_preparing_for` | string | no | Target exam (e.g., "JEE Mains", "SAT") |

**Success Response** (`201 Created`):

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "665f1a2b3c4d5e6f7a8b9c0d",
    "email": "student@example.com",
    "name": "Kaushal",
    "role": "student",
    "grade_level": "Grade 12",
    "education_system": "CBSE",
    "exam_preparing_for": "JEE Mains",
    "created_at": "2025-06-01T14:30:00Z"
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing required fields or invalid email format |
| 409 | `CONFLICT` | Email already registered |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "student@example.com",
    "name": "Kaushal",
    "password": "securepass123",
    "grade_level": "Grade 12",
    "education_system": "CBSE",
    "exam_preparing_for": "JEE Mains"
  }'
```

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "665f1a2b3c4d5e6f7a8b9c0d",
    "email": "student@example.com",
    "name": "Kaushal",
    "role": "student",
    "grade_level": "Grade 12",
    "education_system": "CBSE",
    "exam_preparing_for": "JEE Mains",
    "created_at": "2025-06-01T14:30:00Z"
  }
}
```

---

#### POST /api/v1/auth/login

Authenticate with email and password. Returns a JWT token.

**Auth**: Public

**Request Body**:

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | Registered email address |
| `password` | string | yes | Account password |

**Success Response** (`200 OK`):

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "665f1a2b3c4d5e6f7a8b9c0d",
    "email": "student@example.com",
    "name": "Kaushal",
    "role": "student",
    "grade_level": "Grade 12",
    "education_system": "CBSE",
    "exam_preparing_for": "JEE Mains",
    "created_at": "2025-06-01T14:30:00Z"
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing email or password |
| 401 | `UNAUTHORIZED` | Invalid email or password |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "student@example.com",
    "password": "securepass123"
  }'
```

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "665f1a2b3c4d5e6f7a8b9c0d",
    "email": "student@example.com",
    "name": "Kaushal",
    "role": "student",
    "grade_level": "Grade 12",
    "education_system": "CBSE",
    "exam_preparing_for": "JEE Mains",
    "created_at": "2025-06-01T14:30:00Z"
  }
}
```

---

### Books

#### GET /api/v1/books

List all books with optional filtering and pagination.

**Auth**: Student or Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `status` | string | - | Filter by status: `pending`, `processing`, `ready`, `failed` |
| `subject` | string | - | Filter by subject (exact match) |
| `grade_level` | string | - | Filter by grade level (matches any in array) |
| `source_type` | string | - | Filter by source type: `url`, `pdf`, `search` |
| `sort` | string | `created_at` | Sort field: `created_at`, `title`, `author` |
| `order` | string | `desc` | Sort order: `asc`, `desc` |

**Success Response** (`200 OK`):

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b9c0d",
      "title": "Physics Class 12 NCERT",
      "author": "NCERT",
      "isbn": "978-81-7450-123-4",
      "publisher": "NCERT",
      "language": "en",
      "subject": "Physics",
      "grade_levels": ["Grade 12"],
      "education_system": "CBSE",
      "source_type": "pdf",
      "source_url": null,
      "pdf_url": "https://r2.example.com/books/665f1a2b.pdf",
      "cover_image_url": "https://r2.example.com/covers/665f1a2b.jpg",
      "status": "ready",
      "processing_error": null,
      "toc": [
        {
          "number": 1,
          "title": "Electric Charges and Fields",
          "page": 1,
          "depth": 0
        },
        {
          "number": 2,
          "title": "Electrostatic Potential and Capacitance",
          "page": 33,
          "depth": 0
        }
      ],
      "metadata": {
        "pages": 350,
        "edition": "2024"
      },
      "created_by": "665f1a2b3c4d5e6f7a8b0001",
      "created_at": "2025-06-01T14:30:00Z",
      "updated_at": "2025-06-01T14:35:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid token |

**Example**:

```bash
curl http://localhost:8080/api/v1/books?status=ready&subject=Physics&page=1&limit=10 \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b9c0d",
      "title": "Physics Class 12 NCERT",
      "author": "NCERT",
      "subject": "Physics",
      "grade_levels": ["Grade 12"],
      "source_type": "pdf",
      "status": "ready",
      "toc": [
        {"number": 1, "title": "Electric Charges and Fields", "page": 1, "depth": 0}
      ],
      "created_at": "2025-06-01T14:30:00Z",
      "updated_at": "2025-06-01T14:35:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 1,
    "total_pages": 1
  }
}
```

---

#### GET /api/v1/books/:id

Get a single book by ID.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Physics Class 12 NCERT",
  "author": "NCERT",
  "isbn": "978-81-7450-123-4",
  "publisher": "NCERT",
  "language": "en",
  "subject": "Physics",
  "grade_levels": ["Grade 12"],
  "education_system": "CBSE",
  "source_type": "pdf",
  "source_url": null,
  "pdf_url": "https://r2.example.com/books/665f1a2b.pdf",
  "cover_image_url": "https://r2.example.com/covers/665f1a2b.jpg",
  "status": "ready",
  "processing_error": null,
  "toc": [
    {
      "number": 1,
      "title": "Electric Charges and Fields",
      "page": 1,
      "depth": 0
    }
  ],
  "metadata": {
    "pages": 350,
    "edition": "2024"
  },
  "created_by": "665f1a2b3c4d5e6f7a8b0001",
  "created_at": "2025-06-01T14:30:00Z",
  "updated_at": "2025-06-01T14:35:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d \
  -H "Authorization: Bearer <token>"
```

---

#### POST /api/v1/books

Create a book from a URL or search result. The backend invokes the scraper to extract content, then starts the processing pipeline (embed chapters, generate questions).

**Auth**: Admin

**Request Body**:

| Field | Type | Required | Description |
|---|---|---|---|
| `source_url` | string | yes (for URL) | URL to scrape content from |
| `source_type` | string | yes | `"url"` or `"search"` |
| `title` | string | no | Override title (otherwise extracted by scraper) |
| `author` | string | no | Override author |
| `subject` | string | yes | Primary subject |
| `grade_levels` | string[] | yes | Target grades |
| `education_system` | string | no | Education system |
| `isbn` | string | no | ISBN if known (from search) |
| `publisher` | string | no | Publisher name |
| `cover_image_url` | string | no | Cover image URL (from search) |
| `metadata` | object | no | Additional metadata |

**Success Response** (`201 Created`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Physics Class 12 NCERT",
  "author": "NCERT",
  "subject": "Physics",
  "grade_levels": ["Grade 12"],
  "source_type": "url",
  "source_url": "https://example.com/physics-12",
  "status": "processing",
  "created_by": "665f1a2b3c4d5e6f7a8b0001",
  "created_at": "2025-06-01T14:30:00Z",
  "updated_at": "2025-06-01T14:30:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing required fields |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 422 | `UNPROCESSABLE_ENTITY` | URL is not from an allowed source |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/books \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "source_url": "https://www.gutenberg.org/ebooks/12345",
    "source_type": "url",
    "subject": "Physics",
    "grade_levels": ["Grade 12", "Undergraduate"]
  }'
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Principles of Physics",
  "author": "Project Gutenberg Author",
  "subject": "Physics",
  "grade_levels": ["Grade 12", "Undergraduate"],
  "source_type": "url",
  "source_url": "https://www.gutenberg.org/ebooks/12345",
  "status": "processing",
  "created_by": "665f1a2b3c4d5e6f7a8b0001",
  "created_at": "2025-06-01T14:30:00Z",
  "updated_at": "2025-06-01T14:30:00Z"
}
```

---

#### POST /api/v1/books/upload

Upload a PDF file to create a book. The PDF is stored in Cloudflare R2, then parsed by the scraper to extract text and structure.

**Auth**: Admin

**Content-Type**: `multipart/form-data`

**Form Fields**:

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | file | yes | PDF file (max 50 MB) |
| `title` | string | no | Book title (extracted from PDF if omitted) |
| `author` | string | no | Author name |
| `subject` | string | yes | Primary subject |
| `grade_levels` | string | yes | Comma-separated grade levels |
| `education_system` | string | no | Education system |
| `isbn` | string | no | ISBN if known |

**Success Response** (`201 Created`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Physics Class 12 NCERT",
  "author": "NCERT",
  "subject": "Physics",
  "grade_levels": ["Grade 12"],
  "source_type": "pdf",
  "pdf_url": "https://r2.example.com/books/665f1a2b3c4d5e6f7a8b9c0d.pdf",
  "status": "processing",
  "created_by": "665f1a2b3c4d5e6f7a8b0001",
  "created_at": "2025-06-01T14:30:00Z",
  "updated_at": "2025-06-01T14:30:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing required fields |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 413 | `FILE_TOO_LARGE` | File exceeds 50 MB |
| 415 | `UNSUPPORTED_TYPE` | File is not a PDF |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/books/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@/path/to/physics-12.pdf" \
  -F "subject=Physics" \
  -F "grade_levels=Grade 12"
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Physics Class 12 NCERT",
  "author": "NCERT",
  "subject": "Physics",
  "grade_levels": ["Grade 12"],
  "source_type": "pdf",
  "pdf_url": "https://r2.example.com/books/665f1a2b3c4d5e6f7a8b9c0d.pdf",
  "status": "processing",
  "created_by": "665f1a2b3c4d5e6f7a8b0001",
  "created_at": "2025-06-01T14:30:00Z",
  "updated_at": "2025-06-01T14:30:00Z"
}
```

---

#### POST /api/v1/books/search

Search for books via Open Library and Google Books APIs. Returns search results that can be imported via `POST /api/v1/books` with `source_type: "search"`.

**Auth**: Student or Admin

**Request Body**:

| Field | Type | Required | Description |
|---|---|---|---|
| `query` | string | yes | Search query (title, author, ISBN, etc.) |
| `limit` | int | no | Max results to return (default 10, max 50) |

**Success Response** (`200 OK`):

```json
{
  "results": [
    {
      "title": "Concepts of Physics",
      "author": "H.C. Verma",
      "isbn": "978-81-7709-187-2",
      "publisher": "Bharati Bhawan",
      "language": "en",
      "cover_image_url": "https://covers.openlibrary.org/b/isbn/9788177091878-L.jpg",
      "description": "A comprehensive physics textbook for competitive exam preparation.",
      "source": "open_library",
      "source_url": "https://openlibrary.org/works/OL12345W"
    },
    {
      "title": "University Physics",
      "author": "Young & Freedman",
      "isbn": "978-0-321-97361-0",
      "publisher": "Pearson",
      "language": "en",
      "cover_image_url": "https://books.google.com/books/content?id=...",
      "description": "University-level physics textbook.",
      "source": "google_books",
      "source_url": "https://books.google.com/books?id=..."
    }
  ]
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing query |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 503 | `SERVICE_UNAVAILABLE` | External search APIs are unreachable |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/books/search \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"query": "physics class 12", "limit": 5}'
```

```json
{
  "results": [
    {
      "title": "Concepts of Physics",
      "author": "H.C. Verma",
      "isbn": "978-81-7709-187-2",
      "publisher": "Bharati Bhawan",
      "language": "en",
      "cover_image_url": "https://covers.openlibrary.org/b/isbn/9788177091878-L.jpg",
      "description": "A comprehensive physics textbook.",
      "source": "open_library",
      "source_url": "https://openlibrary.org/works/OL12345W"
    }
  ]
}
```

---

#### PUT /api/v1/books/:id

Update a book's metadata. Cannot change content-related fields (`toc`, `source_url`, `pdf_url`) after processing.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Request Body** (all fields optional):

| Field | Type | Description |
|---|---|---|
| `title` | string | Updated title |
| `author` | string | Updated author |
| `isbn` | string | Updated ISBN |
| `publisher` | string | Updated publisher |
| `language` | string | Updated language code |
| `subject` | string | Updated subject |
| `grade_levels` | string[] | Updated grade levels |
| `education_system` | string | Updated education system |
| `cover_image_url` | string | Updated cover image URL |
| `status` | string | Update processing status (admin override) |
| `metadata` | object | Updated metadata (merged, not replaced) |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Physics Class 12 NCERT (Updated Edition)",
  "author": "NCERT",
  "subject": "Physics",
  "grade_levels": ["Grade 11", "Grade 12"],
  "status": "ready",
  "updated_at": "2025-06-02T10:00:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 400 | `VALIDATION_ERROR` | Invalid field values |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl -X PUT http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Physics Class 12 NCERT (Updated Edition)",
    "grade_levels": ["Grade 11", "Grade 12"]
  }'
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "title": "Physics Class 12 NCERT (Updated Edition)",
  "author": "NCERT",
  "subject": "Physics",
  "grade_levels": ["Grade 11", "Grade 12"],
  "status": "ready",
  "updated_at": "2025-06-02T10:00:00Z"
}
```

---

#### DELETE /api/v1/books/:id

Delete a book and all associated chapters, questions, and user answers. Also deletes the PDF from R2 if applicable. This action is irreversible.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Success Response** (`200 OK`):

```json
{
  "message": "Book deleted successfully",
  "deleted": {
    "book": 1,
    "chapters": 12,
    "questions": 240,
    "user_answers": 1580
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl -X DELETE http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d \
  -H "Authorization: Bearer <token>"
```

```json
{
  "message": "Book deleted successfully",
  "deleted": {
    "book": 1,
    "chapters": 12,
    "questions": 240,
    "user_answers": 1580
  }
}
```

---

#### POST /api/v1/books/:id/process

Re-trigger the processing pipeline for a book. Useful if initial processing failed or if you want to re-embed chapters. Resets status to `"processing"`.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Request Body** (optional):

| Field | Type | Default | Description |
|---|---|---|---|
| `re_embed` | bool | true | Regenerate chapter embeddings |
| `re_extract` | bool | false | Re-run scraper extraction |

**Success Response** (`202 Accepted`):

```json
{
  "message": "Processing started",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "status": "processing"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Book does not exist |
| 409 | `CONFLICT` | Book is already processing |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d/process \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"re_embed": true, "re_extract": false}'
```

```json
{
  "message": "Processing started",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "status": "processing"
}
```

---

#### GET /api/v1/books/:id/status

Get the current processing status of a book. Useful for polling during asynchronous processing.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Success Response** (`200 OK`):

```json
{
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "status": "processing",
  "processing_error": null,
  "chapters_total": 12,
  "chapters_embedded": 8,
  "questions_generated": 160,
  "updated_at": "2025-06-01T14:33:00Z"
}
```

When processing is complete:

```json
{
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "status": "ready",
  "processing_error": null,
  "chapters_total": 12,
  "chapters_embedded": 12,
  "questions_generated": 240,
  "updated_at": "2025-06-01T14:35:00Z"
}
```

When processing has failed:

```json
{
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "status": "failed",
  "processing_error": "Embedding generation failed: Ollama service unavailable",
  "chapters_total": 12,
  "chapters_embedded": 3,
  "questions_generated": 0,
  "updated_at": "2025-06-01T14:32:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d/status \
  -H "Authorization: Bearer <token>"
```

```json
{
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "status": "ready",
  "processing_error": null,
  "chapters_total": 12,
  "chapters_embedded": 12,
  "questions_generated": 240,
  "updated_at": "2025-06-01T14:35:00Z"
}
```

---

### Chapters

#### GET /api/v1/books/:id/chapters

List all chapters for a book, ordered by chapter number.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `include_content` | bool | false | Include full chapter text (large payload) |

**Success Response** (`200 OK`):

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b1001",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "number": 1,
      "title": "Electric Charges and Fields",
      "summary": "This chapter covers electric charge, Coulomb's law, electric field, and Gauss's theorem.",
      "topics": ["electric charge", "Coulomb's law", "electric field", "Gauss's theorem"],
      "word_count": 4500,
      "created_at": "2025-06-01T14:30:00Z"
    },
    {
      "id": "665f1a2b3c4d5e6f7a8b1002",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "number": 2,
      "title": "Electrostatic Potential and Capacitance",
      "summary": "Covers electrostatic potential, potential difference, capacitors, and dielectrics.",
      "topics": ["potential", "capacitance", "dielectrics"],
      "word_count": 5200,
      "created_at": "2025-06-01T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 12,
    "total_pages": 1
  }
}
```

When `include_content=true`, each chapter object also includes:

```json
{
  "content": "Full chapter text here..."
}
```

The `embedding` field is never returned in API responses (it is internal to the vector search system).

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d/chapters?include_content=false \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b1001",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "number": 1,
      "title": "Electric Charges and Fields",
      "summary": "This chapter covers electric charge, Coulomb's law, electric field, and Gauss's theorem.",
      "topics": ["electric charge", "Coulomb's law", "electric field", "Gauss's theorem"],
      "word_count": 4500,
      "created_at": "2025-06-01T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 12,
    "total_pages": 1
  }
}
```

---

#### GET /api/v1/chapters/:id

Get a single chapter by ID, including full content.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Chapter ObjectID |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b1001",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "number": 1,
  "title": "Electric Charges and Fields",
  "content": "Electric charge is a fundamental property of matter...",
  "summary": "This chapter covers electric charge, Coulomb's law, electric field, and Gauss's theorem.",
  "topics": ["electric charge", "Coulomb's law", "electric field", "Gauss's theorem"],
  "word_count": 4500,
  "created_at": "2025-06-01T14:30:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Chapter does not exist |

**Example**:

```bash
curl http://localhost:8080/api/v1/chapters/665f1a2b3c4d5e6f7a8b1001 \
  -H "Authorization: Bearer <token>"
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b1001",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "number": 1,
  "title": "Electric Charges and Fields",
  "content": "Electric charge is a fundamental property of matter...",
  "summary": "This chapter covers electric charge, Coulomb's law, electric field, and Gauss's theorem.",
  "topics": ["electric charge", "Coulomb's law", "electric field", "Gauss's theorem"],
  "word_count": 4500,
  "created_at": "2025-06-01T14:30:00Z"
}
```

---

#### GET /api/v1/chapters/search

Semantic search across chapters using MongoDB vector search. The query is embedded via Ollama (nomic-embed-text) and compared against stored chapter embeddings using cosine similarity.

**Auth**: Student or Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `q` | string | (required) | Natural language search query |
| `book_id` | string | - | Restrict search to a specific book |
| `limit` | int | 10 | Number of results (max 50) |
| `min_score` | float | 0.5 | Minimum cosine similarity score (0.0-1.0) |

**Success Response** (`200 OK`):

```json
{
  "query": "Newton's third law of motion",
  "results": [
    {
      "chapter": {
        "id": "665f1a2b3c4d5e6f7a8b1005",
        "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
        "number": 5,
        "title": "Laws of Motion",
        "summary": "Covers Newton's three laws of motion, inertia, force, and action-reaction pairs.",
        "topics": ["Newton's laws", "inertia", "force", "action-reaction"],
        "word_count": 6200
      },
      "score": 0.92
    },
    {
      "chapter": {
        "id": "665f1a2b3c4d5e6f7a8b1006",
        "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
        "number": 6,
        "title": "Work, Energy and Power",
        "summary": "Discusses work done by a force, kinetic and potential energy, conservation laws.",
        "topics": ["work", "energy", "power", "conservation"],
        "word_count": 5800
      },
      "score": 0.71
    }
  ]
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing `q` parameter |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 503 | `SERVICE_UNAVAILABLE` | Ollama is unreachable (cannot generate query embedding) |

**Example**:

```bash
curl "http://localhost:8080/api/v1/chapters/search?q=Newton%27s+third+law&book_id=665f1a2b3c4d5e6f7a8b9c0d&limit=5" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "query": "Newton's third law",
  "results": [
    {
      "chapter": {
        "id": "665f1a2b3c4d5e6f7a8b1005",
        "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
        "number": 5,
        "title": "Laws of Motion",
        "summary": "Covers Newton's three laws of motion, inertia, force, and action-reaction pairs.",
        "topics": ["Newton's laws", "inertia", "force", "action-reaction"],
        "word_count": 6200
      },
      "score": 0.92
    }
  ]
}
```

---

### Questions

#### GET /api/v1/books/:id/questions

List questions for a book with filtering. Questions can be filtered by chapter, type, difficulty, and Bloom's level.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `chapter_id` | string | - | Filter by chapter |
| `question_type` | string | - | Filter: `mcq`, `essay`, `fill_blank`, `true_false`, `short_answer`, `match`, `assertion_reasoning` |
| `bloom_level` | string | - | Filter: `remember`, `understand`, `apply`, `analyze`, `evaluate`, `create` |
| `difficulty` | string | - | Filter: `easy`, `medium`, `hard` |
| `topic` | string | - | Filter by topic (text search) |

**Success Response** (`200 OK`):

For students, `model_answer`, `correct_answer`, `key_points`, and `explanation` are omitted to prevent cheating. Admins receive the full object.

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b2001",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "chapter_id": "665f1a2b3c4d5e6f7a8b1005",
      "topic": "Newton's Third Law",
      "question_text": "Which of the following best describes Newton's third law of motion?",
      "question_type": "mcq",
      "difficulty": "easy",
      "bloom_level": "remember",
      "grade_level": "Grade 12",
      "exam_type": "CBSE Board",
      "options": [
        {"text": "An object at rest stays at rest unless acted upon by a force"},
        {"text": "Force equals mass times acceleration"},
        {"text": "For every action, there is an equal and opposite reaction"},
        {"text": "Energy can neither be created nor destroyed"}
      ],
      "enrichment": {
        "what": "Recall of Newton's third law definition",
        "when": "Board exams, competitive physics exams",
        "how": "Identify the law that describes action-reaction force pairs",
        "who": "Grade 11-12 Physics students"
      },
      "related_question_ids": [
        "665f1a2b3c4d5e6f7a8b2002",
        "665f1a2b3c4d5e6f7a8b2003"
      ],
      "tags": ["mechanics", "Newton's laws", "forces"],
      "created_at": "2025-06-01T14:35:00Z"
    },
    {
      "id": "665f1a2b3c4d5e6f7a8b2002",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "chapter_id": "665f1a2b3c4d5e6f7a8b1005",
      "topic": "Newton's Third Law",
      "question_text": "Explain Newton's third law of motion with a real-world example. Describe how the action and reaction forces act on different bodies.",
      "question_type": "essay",
      "difficulty": "medium",
      "bloom_level": "understand",
      "grade_level": "Grade 12",
      "exam_type": "CBSE Board",
      "options": null,
      "enrichment": {
        "what": "Understanding of action-reaction pairs and their application",
        "when": "Board exams, competitive exams, real-world engineering",
        "how": "Identify two interacting objects, then describe equal and opposite forces on each",
        "who": "Grade 12 Physics students, JEE aspirants"
      },
      "related_question_ids": [
        "665f1a2b3c4d5e6f7a8b2001",
        "665f1a2b3c4d5e6f7a8b2003"
      ],
      "tags": ["mechanics", "Newton's laws", "forces"],
      "created_at": "2025-06-01T14:35:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 240,
    "total_pages": 12
  }
}
```

For admin users, each question additionally includes:

```json
{
  "correct_answer": "C",
  "model_answer": "Newton's third law states that for every action, there is an equal and opposite reaction...",
  "key_points": ["action-reaction", "equal and opposite", "different bodies", "simultaneous"],
  "explanation": "The third law means that forces always come in pairs..."
}
```

For MCQ questions, the admin view also includes `is_correct` on each option:

```json
{
  "options": [
    {"text": "An object at rest stays at rest unless acted upon by a force", "is_correct": false},
    {"text": "Force equals mass times acceleration", "is_correct": false},
    {"text": "For every action, there is an equal and opposite reaction", "is_correct": true},
    {"text": "Energy can neither be created nor destroyed", "is_correct": false}
  ]
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format or invalid enum value |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl "http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d/questions?question_type=mcq&bloom_level=remember&difficulty=easy&page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b2001",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "chapter_id": "665f1a2b3c4d5e6f7a8b1005",
      "topic": "Newton's Third Law",
      "question_text": "Which of the following best describes Newton's third law of motion?",
      "question_type": "mcq",
      "difficulty": "easy",
      "bloom_level": "remember",
      "grade_level": "Grade 12",
      "options": [
        {"text": "An object at rest stays at rest unless acted upon by a force"},
        {"text": "Force equals mass times acceleration"},
        {"text": "For every action, there is an equal and opposite reaction"},
        {"text": "Energy can neither be created nor destroyed"}
      ],
      "enrichment": {
        "what": "Recall of Newton's third law definition",
        "when": "Board exams, competitive physics exams",
        "how": "Identify the law that describes action-reaction force pairs",
        "who": "Grade 11-12 Physics students"
      },
      "tags": ["mechanics", "Newton's laws", "forces"],
      "created_at": "2025-06-01T14:35:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 15,
    "total_pages": 2
  }
}
```

---

#### GET /api/v1/questions/:id

Get a single question by ID.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Question ObjectID |

**Success Response** (`200 OK`):

Returns the same question object structure as the list endpoint. Students do not receive `model_answer`, `correct_answer`, `key_points`, `explanation`, or `is_correct` on options. Admins receive the full object.

Student view:

```json
{
  "id": "665f1a2b3c4d5e6f7a8b2002",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "chapter_id": "665f1a2b3c4d5e6f7a8b1005",
  "topic": "Newton's Third Law",
  "question_text": "Explain Newton's third law of motion with a real-world example.",
  "question_type": "essay",
  "difficulty": "medium",
  "bloom_level": "understand",
  "grade_level": "Grade 12",
  "exam_type": "CBSE Board",
  "options": null,
  "enrichment": {
    "what": "Understanding of action-reaction pairs and their application",
    "when": "Board exams, competitive exams, real-world engineering",
    "how": "Identify two interacting objects, then describe equal and opposite forces on each",
    "who": "Grade 12 Physics students, JEE aspirants"
  },
  "related_question_ids": ["665f1a2b3c4d5e6f7a8b2001"],
  "tags": ["mechanics", "Newton's laws", "forces"],
  "created_at": "2025-06-01T14:35:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Question does not exist |

**Example**:

```bash
curl http://localhost:8080/api/v1/questions/665f1a2b3c4d5e6f7a8b2002 \
  -H "Authorization: Bearer <token>"
```

---

#### POST /api/v1/books/:id/questions/generate

Trigger AI question generation for a book. Generates questions across all chapters for specified Bloom's levels, difficulty levels, and question types. Processing happens asynchronously.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Request Body** (all fields optional -- defaults generate a balanced set):

| Field | Type | Default | Description |
|---|---|---|---|
| `chapter_ids` | string[] | all chapters | Restrict to specific chapters |
| `question_types` | string[] | all types | Types to generate |
| `bloom_levels` | string[] | all levels | Bloom's levels to target |
| `difficulties` | string[] | all levels | Difficulty levels to target |
| `questions_per_chapter` | int | 20 | Approx. questions per chapter |
| `grade_level` | string | from book | Target grade level |
| `exam_type` | string | - | Target exam (e.g., "JEE Mains") |
| `generate_variants` | bool | true | Generate same concept in multiple question formats |

**Success Response** (`202 Accepted`):

```json
{
  "message": "Question generation started",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "chapters_to_process": 12,
  "estimated_questions": 240
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID or invalid enum values |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Book does not exist |
| 409 | `CONFLICT` | Generation already in progress for this book |
| 422 | `UNPROCESSABLE_ENTITY` | Book status is not `ready` (chapters not yet processed) |
| 503 | `SERVICE_UNAVAILABLE` | LLM service (OpenRouter/Ollama) is unreachable |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d/questions/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "question_types": ["mcq", "essay", "true_false"],
    "bloom_levels": ["remember", "understand", "apply"],
    "difficulties": ["easy", "medium"],
    "questions_per_chapter": 15,
    "exam_type": "CBSE Board",
    "generate_variants": true
  }'
```

```json
{
  "message": "Question generation started",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "chapters_to_process": 12,
  "estimated_questions": 180
}
```

---

#### PUT /api/v1/questions/:id

Edit a question. Admins can correct generated questions, update options, change classification, etc.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Question ObjectID |

**Request Body** (all fields optional):

| Field | Type | Description |
|---|---|---|
| `question_text` | string | Updated question text |
| `question_type` | string | Updated type (enum) |
| `difficulty` | string | Updated difficulty (enum) |
| `bloom_level` | string | Updated Bloom's level (enum) |
| `grade_level` | string | Updated grade level |
| `exam_type` | string | Updated exam type |
| `topic` | string | Updated topic |
| `options` | Option[] | Updated MCQ options (array of `{text, is_correct}`) |
| `correct_answer` | string | Updated correct answer |
| `model_answer` | string | Updated model answer |
| `key_points` | string[] | Updated key points |
| `explanation` | string | Updated explanation |
| `enrichment` | Enrichment | Updated enrichment (`{what, when, how, who}`) |
| `tags` | string[] | Updated tags |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b2001",
  "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "chapter_id": "665f1a2b3c4d5e6f7a8b1005",
  "topic": "Newton's Third Law",
  "question_text": "Which statement correctly describes Newton's third law of motion?",
  "question_type": "mcq",
  "difficulty": "easy",
  "bloom_level": "remember",
  "grade_level": "Grade 12",
  "exam_type": "CBSE Board",
  "options": [
    {"text": "An object at rest stays at rest unless acted upon by a force", "is_correct": false},
    {"text": "Force equals mass times acceleration", "is_correct": false},
    {"text": "For every action, there is an equal and opposite reaction", "is_correct": true},
    {"text": "Energy can neither be created nor destroyed", "is_correct": false}
  ],
  "correct_answer": "C",
  "model_answer": null,
  "key_points": ["action-reaction", "equal and opposite"],
  "explanation": "Newton's third law describes how forces always occur in pairs.",
  "enrichment": {
    "what": "Recall of Newton's third law definition",
    "when": "Board exams, competitive physics exams",
    "how": "Identify the law that describes action-reaction force pairs",
    "who": "Grade 11-12 Physics students"
  },
  "tags": ["mechanics", "Newton's laws"],
  "created_at": "2025-06-01T14:35:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID or invalid enum values |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Question does not exist |

**Example**:

```bash
curl -X PUT http://localhost:8080/api/v1/questions/665f1a2b3c4d5e6f7a8b2001 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "question_text": "Which statement correctly describes Newton'\''s third law of motion?",
    "difficulty": "medium",
    "tags": ["mechanics", "Newton'\''s laws", "board exam"]
  }'
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b2001",
  "question_text": "Which statement correctly describes Newton's third law of motion?",
  "question_type": "mcq",
  "difficulty": "medium",
  "bloom_level": "remember",
  "tags": ["mechanics", "Newton's laws", "board exam"],
  "created_at": "2025-06-01T14:35:00Z"
}
```

---

#### DELETE /api/v1/questions/:id

Delete a question and all associated user answers.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Question ObjectID |

**Success Response** (`200 OK`):

```json
{
  "message": "Question deleted successfully",
  "deleted": {
    "question": 1,
    "user_answers": 45
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Question does not exist |

**Example**:

```bash
curl -X DELETE http://localhost:8080/api/v1/questions/665f1a2b3c4d5e6f7a8b2001 \
  -H "Authorization: Bearer <token>"
```

```json
{
  "message": "Question deleted successfully",
  "deleted": {
    "question": 1,
    "user_answers": 45
  }
}
```

---

#### GET /api/v1/books/:id/questions/random

Get a random set of questions for practice. Useful for generating a quiz or practice session.

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Book ObjectID |

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `count` | int | 10 | Number of random questions (max 50) |
| `chapter_id` | string | - | Restrict to a specific chapter |
| `question_type` | string | - | Filter by question type |
| `bloom_level` | string | - | Filter by Bloom's level |
| `difficulty` | string | - | Filter by difficulty |
| `exclude_answered` | bool | false | Exclude questions the user has already answered |

**Success Response** (`200 OK`):

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b2001",
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "chapter_id": "665f1a2b3c4d5e6f7a8b1005",
      "topic": "Newton's Third Law",
      "question_text": "Which of the following best describes Newton's third law of motion?",
      "question_type": "mcq",
      "difficulty": "easy",
      "bloom_level": "remember",
      "grade_level": "Grade 12",
      "options": [
        {"text": "An object at rest stays at rest unless acted upon by a force"},
        {"text": "Force equals mass times acceleration"},
        {"text": "For every action, there is an equal and opposite reaction"},
        {"text": "Energy can neither be created nor destroyed"}
      ],
      "enrichment": {
        "what": "Recall of Newton's third law definition",
        "when": "Board exams, competitive physics exams",
        "how": "Identify the law that describes action-reaction force pairs",
        "who": "Grade 11-12 Physics students"
      },
      "tags": ["mechanics", "Newton's laws"],
      "created_at": "2025-06-01T14:35:00Z"
    }
  ],
  "count": 10
}
```

Student view: `model_answer`, `correct_answer`, `key_points`, `explanation`, and `is_correct` on options are omitted.

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID or invalid enum values |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Book does not exist |

**Example**:

```bash
curl "http://localhost:8080/api/v1/books/665f1a2b3c4d5e6f7a8b9c0d/questions/random?count=5&difficulty=medium&bloom_level=apply" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b2010",
      "question_text": "A 5 kg object is pushed with a force of 20 N. Calculate the acceleration.",
      "question_type": "short_answer",
      "difficulty": "medium",
      "bloom_level": "apply",
      "grade_level": "Grade 12",
      "enrichment": {
        "what": "Application of F=ma to calculate acceleration",
        "when": "Numerical problems in board exams and competitive exams",
        "how": "Use Newton's second law: a = F/m",
        "who": "Grade 12 Physics students"
      },
      "tags": ["mechanics", "Newton's laws", "numericals"],
      "created_at": "2025-06-01T14:35:00Z"
    }
  ],
  "count": 5
}
```

---

### Answers & Scoring

#### POST /api/v1/questions/:id/answer

Submit an answer to a question and receive scoring with feedback. Scoring logic depends on the question type:

- **MCQ, true_false, fill_blank**: Exact match comparison against `correct_answer`. Returns `is_correct` (boolean) and `overall_score` (0 or 100).
- **essay, short_answer**: Three-dimensional scoring via LLM + algorithmic evaluation.
- **match**: Pair-wise comparison of matched items.
- **assertion_reasoning**: Evaluates both the assertion/reasoning correctness and the relationship between them.

**Scoring dimensions (essay/short_answer)**:

| Dimension | Weight | Method | Description |
|---|---|---|---|
| `semantic_score` | 50% | LLM comparison | Meaning similarity between student answer and model answer |
| `completeness_score` | 30% | Key point coverage | How many key points from `key_points` are addressed |
| `keyword_score` | 20% | Term matching | Presence of expected domain-specific terms |

**Formula**: `overall_score = 0.5 * semantic_score + 0.3 * completeness_score + 0.2 * keyword_score`

**Auth**: Student or Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Question ObjectID |

**Request Body**:

| Field | Type | Required | Description |
|---|---|---|---|
| `answer_text` | string | yes | Student's answer. For MCQ: option letter ("A", "B", "C", "D"). For true_false: "True" or "False". For fill_blank: the word/phrase. For essay/short_answer: free text. |
| `time_taken` | int | no | Time spent in seconds |

**Success Response -- MCQ / true_false / fill_blank** (`200 OK`):

```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3001",
  "question_id": "665f1a2b3c4d5e6f7a8b2001",
  "is_correct": true,
  "overall_score": 100.0,
  "correct_answer": "C",
  "explanation": "Newton's third law states that for every action, there is an equal and opposite reaction. The forces act on different objects simultaneously.",
  "time_taken": 15
}
```

When incorrect:

```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3002",
  "question_id": "665f1a2b3c4d5e6f7a8b2001",
  "is_correct": false,
  "overall_score": 0.0,
  "correct_answer": "C",
  "explanation": "Newton's third law states that for every action, there is an equal and opposite reaction. Option A describes the first law (inertia), and option B describes the second law (F=ma).",
  "time_taken": 22
}
```

**Success Response -- essay / short_answer** (`200 OK`):

```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3003",
  "question_id": "665f1a2b3c4d5e6f7a8b2002",
  "is_correct": null,
  "semantic_score": 78.5,
  "keyword_score": 65.0,
  "completeness_score": 80.0,
  "overall_score": 75.5,
  "feedback": "Good explanation of the action-reaction principle. You correctly identified that forces act on different bodies. However, you missed mentioning that the forces are simultaneous and did not use the term 'interaction pair'. Consider adding a more specific real-world example like rocket propulsion or swimming.",
  "model_answer": "Newton's third law states that for every action, there is an equal and opposite reaction. When object A exerts a force on object B, object B simultaneously exerts an equal force in the opposite direction on object A. These forces act on different objects and are called action-reaction pairs or interaction pairs. Example: When you push against a wall, the wall pushes back on you with equal force.",
  "key_points_hit": ["action-reaction", "equal and opposite", "different bodies"],
  "key_points_missed": ["simultaneous", "interaction pair"],
  "time_taken": 180
}
```

**Success Response -- match** (`200 OK`):

For match questions, `answer_text` is a JSON string of pairs:

Request:
```json
{
  "answer_text": "{\"Newton's First Law\": \"Inertia\", \"Newton's Second Law\": \"F=ma\", \"Newton's Third Law\": \"Action-Reaction\"}",
  "time_taken": 45
}
```

Response:
```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3004",
  "question_id": "665f1a2b3c4d5e6f7a8b2005",
  "is_correct": true,
  "overall_score": 100.0,
  "correct_pairs": {
    "Newton's First Law": "Inertia",
    "Newton's Second Law": "F=ma",
    "Newton's Third Law": "Action-Reaction"
  },
  "explanation": "All pairs matched correctly.",
  "time_taken": 45
}
```

**Success Response -- assertion_reasoning** (`200 OK`):

For assertion-reasoning questions, `answer_text` is one of: "A", "B", "C", "D" corresponding to standard assertion-reasoning options (both correct and reason explains, both correct but reason does not explain, assertion correct reason wrong, both wrong).

```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3005",
  "question_id": "665f1a2b3c4d5e6f7a8b2006",
  "is_correct": true,
  "overall_score": 100.0,
  "correct_answer": "A",
  "explanation": "Both the assertion and the reason are correct, and the reason correctly explains the assertion. Newton's third law (reason) directly explains why the wall pushes back when you push it (assertion).",
  "time_taken": 30
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 400 | `VALIDATION_ERROR` | Missing `answer_text` or invalid format for question type |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 404 | `NOT_FOUND` | Question does not exist |
| 503 | `SERVICE_UNAVAILABLE` | LLM service unavailable (for essay/short_answer scoring) |

**Example -- MCQ**:

```bash
curl -X POST http://localhost:8080/api/v1/questions/665f1a2b3c4d5e6f7a8b2001/answer \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "answer_text": "C",
    "time_taken": 15
  }'
```

```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3001",
  "question_id": "665f1a2b3c4d5e6f7a8b2001",
  "is_correct": true,
  "overall_score": 100.0,
  "correct_answer": "C",
  "explanation": "Newton's third law states that for every action, there is an equal and opposite reaction.",
  "time_taken": 15
}
```

**Example -- Essay**:

```bash
curl -X POST http://localhost:8080/api/v1/questions/665f1a2b3c4d5e6f7a8b2002/answer \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "answer_text": "Newton'\''s third law says that for every action there is an equal and opposite reaction. When you push a wall, the wall pushes back on you. The forces are equal in magnitude but opposite in direction, and they act on different bodies.",
    "time_taken": 180
  }'
```

```json
{
  "answer_id": "665f1a2b3c4d5e6f7a8b3003",
  "question_id": "665f1a2b3c4d5e6f7a8b2002",
  "is_correct": null,
  "semantic_score": 78.5,
  "keyword_score": 65.0,
  "completeness_score": 80.0,
  "overall_score": 75.5,
  "feedback": "Good explanation of the action-reaction principle. You correctly identified that forces act on different bodies. However, you missed mentioning that the forces are simultaneous and did not use the term 'interaction pair'.",
  "model_answer": "Newton's third law states that for every action, there is an equal and opposite reaction...",
  "key_points_hit": ["action-reaction", "equal and opposite", "different bodies"],
  "key_points_missed": ["simultaneous", "interaction pair"],
  "time_taken": 180
}
```

---

#### GET /api/v1/users/me/answers

Get the authenticated user's answer history with scores and feedback.

**Auth**: Student or Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `book_id` | string | - | Filter by book |
| `chapter_id` | string | - | Filter by chapter |
| `question_type` | string | - | Filter by question type |
| `sort` | string | `created_at` | Sort field: `created_at`, `overall_score` |
| `order` | string | `desc` | Sort order: `asc`, `desc` |

**Success Response** (`200 OK`):

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b3003",
      "user_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "question_id": "665f1a2b3c4d5e6f7a8b2002",
      "question": {
        "id": "665f1a2b3c4d5e6f7a8b2002",
        "question_text": "Explain Newton's third law of motion with a real-world example.",
        "question_type": "essay",
        "bloom_level": "understand",
        "difficulty": "medium",
        "topic": "Newton's Third Law"
      },
      "answer_text": "Newton's third law says that for every action there is an equal and opposite reaction...",
      "is_correct": null,
      "semantic_score": 78.5,
      "keyword_score": 65.0,
      "completeness_score": 80.0,
      "overall_score": 75.5,
      "feedback": "Good explanation of the action-reaction principle...",
      "time_taken": 180,
      "created_at": "2025-06-01T15:00:00Z"
    },
    {
      "id": "665f1a2b3c4d5e6f7a8b3001",
      "user_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "question_id": "665f1a2b3c4d5e6f7a8b2001",
      "question": {
        "id": "665f1a2b3c4d5e6f7a8b2001",
        "question_text": "Which of the following best describes Newton's third law of motion?",
        "question_type": "mcq",
        "bloom_level": "remember",
        "difficulty": "easy",
        "topic": "Newton's Third Law"
      },
      "answer_text": "C",
      "is_correct": true,
      "semantic_score": null,
      "keyword_score": null,
      "completeness_score": null,
      "overall_score": 100.0,
      "feedback": null,
      "time_taken": 15,
      "created_at": "2025-06-01T14:55:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 47,
    "total_pages": 3
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid token |

**Example**:

```bash
curl "http://localhost:8080/api/v1/users/me/answers?book_id=665f1a2b3c4d5e6f7a8b9c0d&sort=overall_score&order=asc&page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b3010",
      "question_id": "665f1a2b3c4d5e6f7a8b2015",
      "question": {
        "question_text": "Derive the expression for kinetic energy.",
        "question_type": "essay",
        "bloom_level": "apply",
        "difficulty": "hard",
        "topic": "Work-Energy Theorem"
      },
      "answer_text": "Kinetic energy is the energy of motion...",
      "overall_score": 42.0,
      "feedback": "Your answer describes what kinetic energy is but does not derive the expression...",
      "created_at": "2025-06-01T16:10:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 47,
    "total_pages": 5
  }
}
```

---

#### GET /api/v1/users/me/stats

Get the authenticated user's aggregated performance statistics.

**Auth**: Student or Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `book_id` | string | - | Stats for a specific book |
| `chapter_id` | string | - | Stats for a specific chapter |
| `period` | string | `all` | Time period: `today`, `week`, `month`, `all` |

**Success Response** (`200 OK`):

```json
{
  "user_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "period": "all",
  "total_answers": 247,
  "average_score": 72.3,
  "total_time_spent": 18450,
  "by_question_type": {
    "mcq": {
      "total": 120,
      "correct": 96,
      "accuracy": 80.0,
      "average_score": 80.0
    },
    "essay": {
      "total": 45,
      "correct": null,
      "accuracy": null,
      "average_score": 68.5,
      "average_semantic_score": 72.1,
      "average_keyword_score": 60.3,
      "average_completeness_score": 71.0
    },
    "true_false": {
      "total": 40,
      "correct": 35,
      "accuracy": 87.5,
      "average_score": 87.5
    },
    "fill_blank": {
      "total": 22,
      "correct": 15,
      "accuracy": 68.2,
      "average_score": 68.2
    },
    "short_answer": {
      "total": 15,
      "correct": null,
      "accuracy": null,
      "average_score": 65.2,
      "average_semantic_score": 68.0,
      "average_keyword_score": 58.5,
      "average_completeness_score": 66.1
    },
    "match": {
      "total": 3,
      "correct": 2,
      "accuracy": 66.7,
      "average_score": 66.7
    },
    "assertion_reasoning": {
      "total": 2,
      "correct": 1,
      "accuracy": 50.0,
      "average_score": 50.0
    }
  },
  "by_bloom_level": {
    "remember": {"total": 60, "average_score": 85.2},
    "understand": {"total": 55, "average_score": 76.8},
    "apply": {"total": 50, "average_score": 70.1},
    "analyze": {"total": 40, "average_score": 65.3},
    "evaluate": {"total": 25, "average_score": 58.7},
    "create": {"total": 17, "average_score": 52.4}
  },
  "by_difficulty": {
    "easy": {"total": 90, "average_score": 82.1},
    "medium": {"total": 100, "average_score": 70.5},
    "hard": {"total": 57, "average_score": 58.3}
  },
  "recent_activity": [
    {"date": "2025-06-01", "answers": 12, "average_score": 74.2},
    {"date": "2025-05-31", "answers": 8, "average_score": 69.5},
    {"date": "2025-05-30", "answers": 15, "average_score": 71.0}
  ],
  "weak_areas": [
    {"topic": "Electromagnetic Induction", "bloom_level": "analyze", "average_score": 42.0, "attempts": 8},
    {"topic": "Thermodynamics", "bloom_level": "evaluate", "average_score": 45.5, "attempts": 6}
  ],
  "strong_areas": [
    {"topic": "Newton's Laws", "bloom_level": "remember", "average_score": 95.0, "attempts": 15},
    {"topic": "Electric Charges", "bloom_level": "understand", "average_score": 88.2, "attempts": 10}
  ]
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid token |

**Example**:

```bash
curl "http://localhost:8080/api/v1/users/me/stats?book_id=665f1a2b3c4d5e6f7a8b9c0d&period=month" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "user_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "period": "month",
  "total_answers": 85,
  "average_score": 74.1,
  "total_time_spent": 7200,
  "by_question_type": {
    "mcq": {"total": 40, "correct": 34, "accuracy": 85.0, "average_score": 85.0},
    "essay": {"total": 20, "correct": null, "accuracy": null, "average_score": 70.2}
  },
  "by_bloom_level": {
    "remember": {"total": 25, "average_score": 88.0},
    "understand": {"total": 20, "average_score": 75.5},
    "apply": {"total": 18, "average_score": 68.3},
    "analyze": {"total": 12, "average_score": 62.1},
    "evaluate": {"total": 7, "average_score": 55.0},
    "create": {"total": 3, "average_score": 48.0}
  },
  "by_difficulty": {
    "easy": {"total": 30, "average_score": 84.5},
    "medium": {"total": 35, "average_score": 72.0},
    "hard": {"total": 20, "average_score": 58.8}
  },
  "recent_activity": [
    {"date": "2025-06-01", "answers": 12, "average_score": 74.2}
  ],
  "weak_areas": [
    {"topic": "Electromagnetic Induction", "bloom_level": "analyze", "average_score": 42.0, "attempts": 8}
  ],
  "strong_areas": [
    {"topic": "Newton's Laws", "bloom_level": "remember", "average_score": 95.0, "attempts": 15}
  ]
}
```

---

### Admin

All admin endpoints require `role: "admin"` in the JWT token.

#### GET /api/v1/admin/sources

List all allowed book sources.

**Auth**: Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `enabled` | bool | - | Filter by enabled status |
| `source_type` | string | - | Filter: `scrape`, `api` |

**Success Response** (`200 OK`):

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b4001",
      "url_pattern": "*.gutenberg.org/*",
      "name": "Project Gutenberg",
      "source_type": "scrape",
      "enabled": true,
      "added_by": "665f1a2b3c4d5e6f7a8b0001",
      "notes": "Public domain books, no rate limit needed",
      "created_at": "2025-06-01T10:00:00Z",
      "updated_at": "2025-06-01T10:00:00Z"
    },
    {
      "id": "665f1a2b3c4d5e6f7a8b4002",
      "url_pattern": "*.ncert.nic.in/*",
      "name": "NCERT Official",
      "source_type": "scrape",
      "enabled": true,
      "added_by": "665f1a2b3c4d5e6f7a8b0001",
      "notes": "Indian government textbooks, freely available",
      "created_at": "2025-06-01T10:00:00Z",
      "updated_at": "2025-06-01T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |

**Example**:

```bash
curl http://localhost:8080/api/v1/admin/sources?enabled=true \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b4001",
      "url_pattern": "*.gutenberg.org/*",
      "name": "Project Gutenberg",
      "source_type": "scrape",
      "enabled": true,
      "notes": "Public domain books, no rate limit needed",
      "created_at": "2025-06-01T10:00:00Z",
      "updated_at": "2025-06-01T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 3,
    "total_pages": 1
  }
}
```

---

#### POST /api/v1/admin/sources

Create a new allowed source.

**Auth**: Admin

**Request Body**:

| Field | Type | Required | Description |
|---|---|---|---|
| `url_pattern` | string | yes | URL glob pattern or domain (e.g., `"*.gutenberg.org/*"`) |
| `name` | string | yes | Human-readable name |
| `source_type` | string | yes | `"scrape"` or `"api"` |
| `enabled` | bool | no | Default `true` |
| `notes` | string | no | Admin notes |

**Success Response** (`201 Created`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b4003",
  "url_pattern": "*.openstax.org/*",
  "name": "OpenStax",
  "source_type": "scrape",
  "enabled": true,
  "added_by": "665f1a2b3c4d5e6f7a8b0001",
  "notes": "Free peer-reviewed textbooks",
  "created_at": "2025-06-02T12:00:00Z",
  "updated_at": "2025-06-02T12:00:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing required fields |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 409 | `CONFLICT` | URL pattern already exists |

**Example**:

```bash
curl -X POST http://localhost:8080/api/v1/admin/sources \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "url_pattern": "*.openstax.org/*",
    "name": "OpenStax",
    "source_type": "scrape",
    "notes": "Free peer-reviewed textbooks"
  }'
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b4003",
  "url_pattern": "*.openstax.org/*",
  "name": "OpenStax",
  "source_type": "scrape",
  "enabled": true,
  "added_by": "665f1a2b3c4d5e6f7a8b0001",
  "notes": "Free peer-reviewed textbooks",
  "created_at": "2025-06-02T12:00:00Z",
  "updated_at": "2025-06-02T12:00:00Z"
}
```

---

#### PUT /api/v1/admin/sources/:id

Update an allowed source.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | AllowedSource ObjectID |

**Request Body** (all fields optional):

| Field | Type | Description |
|---|---|---|
| `url_pattern` | string | Updated URL pattern |
| `name` | string | Updated name |
| `source_type` | string | Updated source type |
| `enabled` | bool | Enable or disable |
| `notes` | string | Updated notes |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b4001",
  "url_pattern": "*.gutenberg.org/*",
  "name": "Project Gutenberg",
  "source_type": "scrape",
  "enabled": false,
  "added_by": "665f1a2b3c4d5e6f7a8b0001",
  "notes": "Temporarily disabled for maintenance",
  "created_at": "2025-06-01T10:00:00Z",
  "updated_at": "2025-06-02T14:00:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Source does not exist |
| 409 | `CONFLICT` | Updated `url_pattern` conflicts with an existing source |

**Example**:

```bash
curl -X PUT http://localhost:8080/api/v1/admin/sources/665f1a2b3c4d5e6f7a8b4001 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false,
    "notes": "Temporarily disabled for maintenance"
  }'
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b4001",
  "url_pattern": "*.gutenberg.org/*",
  "name": "Project Gutenberg",
  "source_type": "scrape",
  "enabled": false,
  "notes": "Temporarily disabled for maintenance",
  "updated_at": "2025-06-02T14:00:00Z"
}
```

---

#### DELETE /api/v1/admin/sources/:id

Delete an allowed source. Does not affect books already scraped from this source.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | AllowedSource ObjectID |

**Success Response** (`200 OK`):

```json
{
  "message": "Source deleted successfully"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | Source does not exist |

**Example**:

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/sources/665f1a2b3c4d5e6f7a8b4003 \
  -H "Authorization: Bearer <token>"
```

```json
{
  "message": "Source deleted successfully"
}
```

---

#### GET /api/v1/admin/dashboard

Get platform-wide analytics for the admin dashboard. Aggregates data across all users, books, and questions.

**Auth**: Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `period` | string | `month` | Time period: `today`, `week`, `month`, `all` |

**Success Response** (`200 OK`):

```json
{
  "period": "month",
  "overview": {
    "total_users": 156,
    "new_users": 23,
    "total_books": 12,
    "books_processing": 1,
    "books_ready": 10,
    "books_failed": 1,
    "total_chapters": 142,
    "total_questions": 2840,
    "total_answers": 18450
  },
  "scoring": {
    "average_overall_score": 68.7,
    "average_semantic_score": 71.2,
    "average_keyword_score": 62.4,
    "average_completeness_score": 69.8,
    "mcq_accuracy": 76.3,
    "true_false_accuracy": 82.1
  },
  "popular_books": [
    {
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "title": "Physics Class 12 NCERT",
      "total_answers": 5200,
      "unique_users": 89,
      "average_score": 72.1
    },
    {
      "book_id": "665f1a2b3c4d5e6f7a8b9c0e",
      "title": "Chemistry Class 12 NCERT",
      "total_answers": 3100,
      "unique_users": 67,
      "average_score": 65.8
    }
  ],
  "question_distribution": {
    "by_type": {
      "mcq": 1200,
      "essay": 480,
      "true_false": 400,
      "fill_blank": 320,
      "short_answer": 240,
      "match": 120,
      "assertion_reasoning": 80
    },
    "by_bloom_level": {
      "remember": 600,
      "understand": 550,
      "apply": 500,
      "analyze": 450,
      "evaluate": 400,
      "create": 340
    },
    "by_difficulty": {
      "easy": 900,
      "medium": 1100,
      "hard": 840
    }
  },
  "hardest_questions": [
    {
      "question_id": "665f1a2b3c4d5e6f7a8b2050",
      "question_text": "Derive the expression for electric field due to a uniformly charged sphere...",
      "question_type": "essay",
      "bloom_level": "create",
      "difficulty": "hard",
      "average_score": 28.5,
      "attempts": 45
    }
  ],
  "weakest_topics": [
    {
      "topic": "Electromagnetic Induction",
      "average_score": 45.2,
      "total_attempts": 320,
      "book_title": "Physics Class 12 NCERT"
    },
    {
      "topic": "Organic Chemistry Reactions",
      "average_score": 48.7,
      "total_attempts": 280,
      "book_title": "Chemistry Class 12 NCERT"
    }
  ],
  "daily_activity": [
    {"date": "2025-06-01", "new_users": 3, "answers_submitted": 450, "average_score": 69.2},
    {"date": "2025-05-31", "new_users": 5, "answers_submitted": 380, "average_score": 71.0},
    {"date": "2025-05-30", "new_users": 2, "answers_submitted": 420, "average_score": 67.5}
  ]
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |

**Example**:

```bash
curl http://localhost:8080/api/v1/admin/dashboard?period=week \
  -H "Authorization: Bearer <token>"
```

```json
{
  "period": "week",
  "overview": {
    "total_users": 156,
    "new_users": 8,
    "total_books": 12,
    "books_ready": 10,
    "total_questions": 2840,
    "total_answers": 4200
  },
  "scoring": {
    "average_overall_score": 70.1,
    "mcq_accuracy": 78.0
  },
  "popular_books": [
    {
      "book_id": "665f1a2b3c4d5e6f7a8b9c0d",
      "title": "Physics Class 12 NCERT",
      "total_answers": 1200,
      "unique_users": 45,
      "average_score": 72.1
    }
  ]
}
```

---

#### GET /api/v1/admin/users

List all registered users with their activity summary.

**Auth**: Admin

**Query Parameters**:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `role` | string | - | Filter by role: `student`, `admin` |
| `search` | string | - | Search by name or email |
| `sort` | string | `created_at` | Sort field: `created_at`, `name`, `email` |
| `order` | string | `desc` | Sort order: `asc`, `desc` |

**Success Response** (`200 OK`):

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b9c0d",
      "email": "student@example.com",
      "name": "Kaushal",
      "role": "student",
      "grade_level": "Grade 12",
      "education_system": "CBSE",
      "exam_preparing_for": "JEE Mains",
      "avatar_url": null,
      "total_answers": 247,
      "average_score": 72.3,
      "last_active": "2025-06-01T15:00:00Z",
      "created_at": "2025-05-01T10:00:00Z",
      "updated_at": "2025-06-01T15:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 156,
    "total_pages": 8
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |

**Example**:

```bash
curl "http://localhost:8080/api/v1/admin/users?role=student&search=kaushal&page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

```json
{
  "data": [
    {
      "id": "665f1a2b3c4d5e6f7a8b9c0d",
      "email": "student@example.com",
      "name": "Kaushal",
      "role": "student",
      "grade_level": "Grade 12",
      "total_answers": 247,
      "average_score": 72.3,
      "last_active": "2025-06-01T15:00:00Z",
      "created_at": "2025-05-01T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 1,
    "total_pages": 1
  }
}
```

---

#### GET /api/v1/admin/users/:id

Get detailed information about a specific user, including their full stats.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | User ObjectID |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "email": "student@example.com",
  "name": "Kaushal",
  "role": "student",
  "grade_level": "Grade 12",
  "education_system": "CBSE",
  "exam_preparing_for": "JEE Mains",
  "avatar_url": null,
  "stats": {
    "total_answers": 247,
    "average_score": 72.3,
    "total_time_spent": 18450,
    "by_question_type": {
      "mcq": {"total": 120, "correct": 96, "accuracy": 80.0},
      "essay": {"total": 45, "average_score": 68.5}
    },
    "by_bloom_level": {
      "remember": {"total": 60, "average_score": 85.2},
      "understand": {"total": 55, "average_score": 76.8}
    }
  },
  "created_at": "2025-05-01T10:00:00Z",
  "updated_at": "2025-06-01T15:00:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | User does not exist |

**Example**:

```bash
curl http://localhost:8080/api/v1/admin/users/665f1a2b3c4d5e6f7a8b9c0d \
  -H "Authorization: Bearer <token>"
```

---

#### PUT /api/v1/admin/users/:id

Update a user's profile or role. Primary use case: promoting a user to admin or updating their profile.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | User ObjectID |

**Request Body** (all fields optional):

| Field | Type | Description |
|---|---|---|
| `name` | string | Updated name |
| `role` | string | Updated role: `student` or `admin` |
| `grade_level` | string | Updated grade level |
| `education_system` | string | Updated education system |
| `exam_preparing_for` | string | Updated target exam |

**Success Response** (`200 OK`):

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "email": "student@example.com",
  "name": "Kaushal",
  "role": "admin",
  "grade_level": "Grade 12",
  "education_system": "CBSE",
  "exam_preparing_for": "JEE Mains",
  "updated_at": "2025-06-02T16:00:00Z"
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID or invalid role value |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin |
| 404 | `NOT_FOUND` | User does not exist |

**Example**:

```bash
curl -X PUT http://localhost:8080/api/v1/admin/users/665f1a2b3c4d5e6f7a8b9c0d \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

```json
{
  "id": "665f1a2b3c4d5e6f7a8b9c0d",
  "email": "student@example.com",
  "name": "Kaushal",
  "role": "admin",
  "updated_at": "2025-06-02T16:00:00Z"
}
```

---

#### DELETE /api/v1/admin/users/:id

Delete a user account and all their answer history. This action is irreversible.

**Auth**: Admin

**Path Parameters**:

| Parameter | Type | Description |
|---|---|---|
| `id` | string | User ObjectID |

**Success Response** (`200 OK`):

```json
{
  "message": "User deleted successfully",
  "deleted": {
    "user": 1,
    "user_answers": 247
  }
}
```

**Error Responses**:

| Status | Code | Cause |
|---|---|---|
| 400 | `BAD_REQUEST` | Invalid ObjectID format |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | User is not an admin, or attempting to delete own account |
| 404 | `NOT_FOUND` | User does not exist |

**Example**:

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/users/665f1a2b3c4d5e6f7a8b9c0d \
  -H "Authorization: Bearer <token>"
```

```json
{
  "message": "User deleted successfully",
  "deleted": {
    "user": 1,
    "user_answers": 247
  }
}
```

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/ping` | Public | Health check |
| POST | `/api/v1/auth/register` | Public | Register a new user |
| POST | `/api/v1/auth/login` | Public | Log in and get JWT |
| GET | `/api/v1/books` | Student | List books with filters |
| GET | `/api/v1/books/:id` | Student | Get book details |
| POST | `/api/v1/books` | Admin | Create book from URL/search |
| POST | `/api/v1/books/upload` | Admin | Create book from PDF upload |
| POST | `/api/v1/books/search` | Student | Search external book APIs |
| PUT | `/api/v1/books/:id` | Admin | Update book metadata |
| DELETE | `/api/v1/books/:id` | Admin | Delete book and all related data |
| POST | `/api/v1/books/:id/process` | Admin | Re-trigger processing pipeline |
| GET | `/api/v1/books/:id/status` | Student | Get processing status |
| GET | `/api/v1/books/:id/chapters` | Student | List chapters for a book |
| GET | `/api/v1/chapters/:id` | Student | Get chapter with full content |
| GET | `/api/v1/chapters/search` | Student | Semantic search across chapters |
| GET | `/api/v1/books/:id/questions` | Student | List questions with filters |
| GET | `/api/v1/questions/:id` | Student | Get a single question |
| POST | `/api/v1/books/:id/questions/generate` | Admin | Generate questions via AI |
| PUT | `/api/v1/questions/:id` | Admin | Edit a question |
| DELETE | `/api/v1/questions/:id` | Admin | Delete a question |
| GET | `/api/v1/books/:id/questions/random` | Student | Get random questions for practice |
| POST | `/api/v1/questions/:id/answer` | Student | Submit answer and get score |
| GET | `/api/v1/users/me/answers` | Student | View own answer history |
| GET | `/api/v1/users/me/stats` | Student | View own performance stats |
| GET | `/api/v1/admin/sources` | Admin | List allowed sources |
| POST | `/api/v1/admin/sources` | Admin | Create an allowed source |
| PUT | `/api/v1/admin/sources/:id` | Admin | Update an allowed source |
| DELETE | `/api/v1/admin/sources/:id` | Admin | Delete an allowed source |
| GET | `/api/v1/admin/dashboard` | Admin | Platform analytics |
| GET | `/api/v1/admin/users` | Admin | List users |
| GET | `/api/v1/admin/users/:id` | Admin | Get user details + stats |
| PUT | `/api/v1/admin/users/:id` | Admin | Update user profile/role |
| DELETE | `/api/v1/admin/users/:id` | Admin | Delete a user |
