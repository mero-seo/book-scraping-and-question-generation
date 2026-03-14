# Embeddings

## Overview

The embedding pipeline converts textual content (chapters, queries) into 768-dimensional dense vectors that enable semantic similarity search. The system uses `nomic-embed-text` running locally via Ollama for all embedding generation. Vectors are stored directly in MongoDB chapter documents and queried using MongoDB Atlas `$vectorSearch` in production or application-level cosine similarity in local development.

Implementation lives in `internal/embeddings/` and `internal/db/`.

## Model: nomic-embed-text

| Property | Value |
|---|---|
| Model name | `nomic-embed-text` |
| Provider | Nomic AI (via Ollama) |
| Dimensions | 768 |
| License | Apache 2.0 |
| Model size | ~274 MB |
| Context window | 8192 tokens |
| Hardware requirement | CPU-only (no GPU needed) |
| RAM usage | ~300 MB at inference |
| Ollama endpoint | `POST http://localhost:11434/api/embeddings` |

### Why nomic-embed-text

- **CPU-friendly**: Runs on machines without a GPU with acceptable latency (~200-500ms per chunk on a modern CPU).
- **Apache 2.0 license**: No usage restrictions, no API costs, no rate limits.
- **768 dimensions**: A good balance between embedding quality and storage/computation cost. Larger models (1024-dim, 1536-dim) provide marginal quality improvement at significantly higher storage and search cost.
- **Strong benchmark performance**: Competitive with models 2-4x its size on MTEB retrieval benchmarks.
- **Local inference**: No network dependency. Embeddings are generated entirely on the host machine via Ollama.

### Ollama API Call

```bash
curl http://localhost:11434/api/embeddings \
  -d '{"model": "nomic-embed-text", "prompt": "Newton third law states..."}'
```

Response:

```json
{
  "embedding": [0.023, -0.156, 0.089, ..., 0.041]
}
```

The `embedding` field contains exactly 768 float64 values.

## Embedding Pipeline

The full pipeline from raw content to searchable vectors:

```
Content Ingestion
    |
    v
[1] Chapter content (plain text)
    |
    v
[2] Chunk into segments (2000 tokens, 200 overlap)
    |
    v
[3] Embed each chunk via Ollama nomic-embed-text
    |
    v
[4] Average chunk embeddings into single chapter vector
    |
    v
[5] Store vector in chapter.embedding (MongoDB)
    |
    v
[6] Index via MongoDB Atlas vectorSearch (production)
```

### Step-by-step

1. **Content extraction**: The scraper produces `chapter.content` as plain text. Chapters are the primary unit of embedding -- not individual paragraphs or sentences.

2. **Chunking**: Long chapters are split into overlapping chunks to stay within the model's 8192-token context window. See the chunking strategy section below.

3. **Embedding generation**: Each chunk is sent to Ollama's `/api/embeddings` endpoint. Chunks are processed sequentially to avoid overwhelming CPU resources.

4. **Averaging**: If a chapter produced multiple chunks, their embedding vectors are averaged element-wise into a single 768-dimensional vector representing the full chapter. This is a standard approach for representing long documents with fixed-size vectors.

5. **Storage**: The averaged vector is stored in the `embedding` field of the chapter document in MongoDB.

6. **Indexing**: In production (MongoDB Atlas), a vector search index enables efficient approximate nearest neighbor (ANN) search. In local development, the application computes cosine similarity directly.

## Chunking Strategy

### Parameters

| Parameter | Value | Rationale |
|---|---|---|
| Chunk size | 2000 tokens | Well within the 8192-token context window, leaving margin for prompt overhead. Large enough to capture coherent sections of text. |
| Overlap | 200 tokens | 10% overlap ensures concepts spanning chunk boundaries are captured in at least one chunk. |
| Splitting method | Sentence-boundary aware | Chunks break at sentence boundaries (period followed by space/newline) to avoid splitting mid-sentence. |

### Algorithm

```
Input:
  text: string            (chapter.content)
  max_tokens: 2000
  overlap_tokens: 200

Steps:
  1. Estimate token count (approximation: word_count * 1.3)
  2. If total_tokens <= max_tokens:
     - Return [text] as a single chunk (no splitting needed)
  3. Else:
     - Split text into sentences
     - Build chunks greedily:
       a. Accumulate sentences until approaching max_tokens
       b. Record chunk boundary
       c. Backtrack by overlap_tokens worth of sentences
       d. Start next chunk from backtrack point
     - Return list of chunks

Output:
  []string  (list of text chunks)
```

### Example

A chapter with ~6000 tokens produces approximately:
- Chunk 1: tokens 0-2000
- Chunk 2: tokens 1800-3800 (200 overlap)
- Chunk 3: tokens 3600-5600 (200 overlap)
- Chunk 4: tokens 5400-6000 (remaining)

Each chunk is embedded independently, then the four vectors are averaged.

## MongoDB Vector Search Index

The vector search index is created on the `chapters` collection in MongoDB Atlas. This index enables the `$vectorSearch` aggregation stage for approximate nearest neighbor queries.

### Index Definition

```json
{
  "name": "chapter_vector_index",
  "type": "vectorSearch",
  "definition": {
    "fields": [
      {
        "type": "vector",
        "path": "embedding",
        "numDimensions": 768,
        "similarity": "cosine"
      },
      {
        "type": "filter",
        "path": "book_id"
      }
    ]
  }
}
```

| Field | Purpose |
|---|---|
| `path: "embedding"` | Points to the `embedding` field in chapter documents |
| `numDimensions: 768` | Must match the nomic-embed-text output dimension exactly |
| `similarity: "cosine"` | Cosine similarity for comparing normalized text embeddings |
| `filter path: "book_id"` | Allows pre-filtering search to chapters within a specific book |

### Creating the Index

The index is created via the MongoDB Atlas UI or the Atlas Admin API. It cannot be created using standard MongoDB shell commands (`createIndex`), because vector search indexes are a separate index type managed by Atlas Search.

```bash
# Via Atlas Admin API
curl -X POST \
  "https://cloud.mongodb.com/api/atlas/v2/groups/{groupId}/clusters/{clusterName}/fts/indexes" \
  -H "Content-Type: application/json" \
  -d '{
    "collectionName": "chapters",
    "database": "book_db",
    "name": "chapter_vector_index",
    "type": "vectorSearch",
    "definition": {
      "fields": [
        {"type": "vector", "path": "embedding", "numDimensions": 768, "similarity": "cosine"},
        {"type": "filter", "path": "book_id"}
      ]
    }
  }'
```

## Similarity Search

### Atlas $vectorSearch (Production)

In production with MongoDB Atlas, semantic search uses the `$vectorSearch` aggregation stage:

```javascript
db.chapters.aggregate([
  {
    $vectorSearch: {
      index: "chapter_vector_index",
      path: "embedding",
      queryVector: [0.023, -0.156, 0.089, ...],  // 768-dim query embedding
      numCandidates: 50,
      limit: 5,
      filter: {
        "book_id": ObjectId("...")
      }
    }
  },
  {
    $project: {
      title: 1,
      content: 1,
      book_id: 1,
      score: { $meta: "vectorSearchScore" }
    }
  }
])
```

| Parameter | Value | Purpose |
|---|---|---|
| `numCandidates` | 50 | Number of candidates the ANN algorithm considers. Higher values improve recall at the cost of latency. |
| `limit` | 5 | Number of results returned to the application. |
| `filter` | `{"book_id": ...}` | Pre-filters to chapters within a specific book before vector comparison. Uses the filter field in the index. |
| `$meta: "vectorSearchScore"` | 0.0-1.0 | Atlas returns a normalized similarity score. For cosine similarity, this is `(1 + cosine_similarity) / 2`. |

### Application-Level Cosine Similarity (Local Development)

Local MongoDB (non-Atlas) does not support `$vectorSearch`. The application falls back to computing cosine similarity in Go:

```
1. Fetch all chapters for the target book from MongoDB (with embeddings)
2. Compute cosine similarity between query vector and each chapter vector
3. Sort by similarity descending
4. Return top-k results
```

This approach is acceptable for development with small datasets (hundreds of chapters) but does not scale to production workloads.

## Cosine Similarity Formula

Cosine similarity measures the angle between two vectors, producing a value between -1 and 1. For normalized text embeddings, the range is typically 0 to 1.

```
                        A . B
cosine_similarity = -----------
                    ||A|| * ||B||

Where:
  A . B    = sum(A[i] * B[i]) for i = 0..767     (dot product)
  ||A||    = sqrt(sum(A[i]^2)) for i = 0..767     (L2 norm)
  ||B||    = sqrt(sum(B[i]^2)) for i = 0..767     (L2 norm)
```

### Go Implementation

```go
func CosineSimilarity(a, b []float64) float64 {
    if len(a) != len(b) {
        return 0
    }
    var dot, normA, normB float64
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    if normA == 0 || normB == 0 {
        return 0
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

### Interpreting Scores

| Cosine Similarity | Interpretation |
|---|---|
| 0.90-1.00 | Near-identical content or very close paraphrase |
| 0.75-0.89 | Strongly related -- same topic, overlapping concepts |
| 0.60-0.74 | Moderately related -- same subject area |
| 0.40-0.59 | Weakly related -- some conceptual overlap |
| < 0.40 | Unrelated content |

These ranges are empirical guidelines for nomic-embed-text. Other embedding models may produce different distributions.

## Performance Considerations

### Embedding Generation

| Factor | Impact | Mitigation |
|---|---|---|
| CPU-only inference | ~200-500ms per chunk | Acceptable for batch processing; not a bottleneck |
| Large chapters | More chunks = more Ollama calls | Chunking limits each call; averaging is O(768) |
| Concurrent requests | Ollama processes requests sequentially by default | Process chapters sequentially within a book; parallel across books only if Ollama is configured with multiple workers |
| Cold start | First embedding call after Ollama restart loads model into memory (~2-3s) | Pre-warm by embedding a dummy string at application startup |

### Storage

| Metric | Value |
|---|---|
| Vector size per chapter | 768 * 8 bytes (float64) = 6,144 bytes = ~6 KB |
| 100 chapters | ~600 KB of vector data |
| 10,000 chapters | ~60 MB of vector data |

Vector storage is negligible compared to chapter text content.

### Search Latency

| Method | Latency (approximate) | Scales to |
|---|---|---|
| Atlas $vectorSearch (ANN) | 5-20ms | Millions of documents |
| Application-level cosine (brute force) | <50ms for 1,000 chapters | Hundreds to low thousands |

For local development with typical workloads (dozens of books, hundreds of chapters), brute-force cosine similarity is fast enough. Production deployments should always use Atlas vector search.

### MongoDB Atlas Free Tier Limits (M0)

| Resource | Limit |
|---|---|
| Storage | 512 MB |
| Vector search indexes | 3 per cluster |
| $vectorSearch `numCandidates` | Max 10,000 |
| Documents with vectors | No hard limit (constrained by storage) |

With ~6 KB per vector and ~10 KB per chapter document (text + metadata), the 512 MB limit supports approximately 30,000 chapters.

## When to Re-embed

Embeddings should be regenerated in the following scenarios:

| Scenario | Action | Reason |
|---|---|---|
| Chapter content is edited | Re-embed that chapter | Embedding no longer represents the content |
| Embedding model is upgraded | Re-embed all chapters | New model produces vectors in a different space; old and new vectors are not comparable |
| Chunking parameters change | Re-embed all chapters | Different chunks produce different averaged vectors |
| Chapter is deleted | No action needed | Embedding is stored in the chapter document and is deleted with it |
| New chapter is added to existing book | Embed the new chapter only | Existing chapter embeddings remain valid |
| nomic-embed-text version changes in Ollama | Re-embed all chapters | Even minor model updates can shift the embedding space |

### Re-embedding Command

```bash
make embed              # Re-embed all chapters missing embeddings
make embed-all          # Force re-embed every chapter (use after model change)
make embed BOOK_ID=...  # Re-embed chapters for a specific book
```

The embedding service tracks whether a chapter's embedding is current by comparing the hash of `chapter.content` against a stored `embedding_content_hash` field. If the content has changed since the last embedding, the chapter is flagged for re-embedding.
