# LLM Strategy

## Overview

All LLM interactions in this platform use free, open-source models exclusively. No paid APIs. The system uses a dual-provider architecture: OpenRouter as the primary cloud provider (routing to free open-source models) and Ollama as a local fallback. A provider-agnostic Go interface in `internal/llm/` abstracts both providers so business logic never depends on a specific backend.

LLM is used for five tasks:
1. **Question generation** -- produce exam-style questions from chapter content, tagged with Bloom's Taxonomy
2. **Question variant generation** -- reformat an existing question into different question types
3. **Semantic answer scoring** -- compare a student answer to a model answer and score meaning similarity
4. **Topic extraction** -- extract key topics from a chapter
5. **Chapter summarization** -- produce a concise summary of chapter content

---

## Provider Architecture

### Interface Design

The `internal/llm/` package defines a provider-agnostic interface that both OpenRouter and Ollama implement:

```go
// internal/llm/client.go

// LLMClient is the provider-agnostic interface for all LLM interactions.
type LLMClient interface {
    // Complete sends a prompt and returns the raw text completion.
    Complete(ctx context.Context, req CompletionRequest) (string, error)

    // CompleteJSON sends a prompt and parses the response as JSON into dst.
    CompleteJSON(ctx context.Context, req CompletionRequest, dst interface{}) error

    // Name returns the provider name for logging ("openrouter", "ollama").
    Name() string

    // Available returns true if the provider is reachable and ready.
    Available(ctx context.Context) bool
}

type CompletionRequest struct {
    SystemPrompt string  // System-level instructions
    UserPrompt   string  // The actual prompt with content
    Temperature  float64 // 0.0 = deterministic, 1.0 = creative
    MaxTokens    int     // Maximum tokens in the response
}
```

### Provider Chain

```
Request
  |
  v
OpenRouterClient (primary, cloud)
  |-- success --> return response
  |-- timeout (>10s) --> fall through
  |-- HTTP 429 (rate limit) --> retry once after Retry-After delay, then rotate model, then fall through
  |-- HTTP 5xx / network error --> fall through
  |
  v
OllamaClient (fallback, local CPU)
  |-- success --> return response
  |-- error --> return error to caller
```

The chain is implemented in a `FallbackClient` wrapper:

```go
// internal/llm/fallback.go

type FallbackClient struct {
    Primary  LLMClient
    Fallback LLMClient
    Timeout  time.Duration // 10s default for primary
}

func (f *FallbackClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
    primaryCtx, cancel := context.WithTimeout(ctx, f.Timeout)
    defer cancel()

    resp, err := f.Primary.Complete(primaryCtx, req)
    if err == nil {
        return resp, nil
    }

    log.Printf("LLM primary (%s) failed: %v, falling back to %s",
        f.Primary.Name(), err, f.Fallback.Name())

    return f.Fallback.Complete(ctx, req)
}
```

---

## OpenRouter Configuration

### Connection Details

| Setting | Value |
|---|---|
| Base URL | `https://openrouter.ai/api/v1/chat/completions` |
| Auth header | `Authorization: Bearer $OPENROUTER_API_KEY` |
| Required header | `HTTP-Referer: https://github.com/ByteJar/book-scraping-and-question-generation` |
| Required header | `X-Title: BookQGen` |

OpenRouter requires a free API key (no credit card). Sign up at https://openrouter.ai and generate a key. Store it in `.env` as `OPENROUTER_API_KEY`.

### Free Models

All models below use the `:free` suffix on OpenRouter, meaning zero cost with rate limits.

| Model ID | Parameters | Context Window | Best For |
|---|---|---|---|
| `meta-llama/llama-3.1-8b-instruct:free` | 8B | 131,072 | Primary. Strong instruction following, reliable JSON output. |
| `mistralai/mistral-7b-instruct:free` | 7B | 32,768 | Good at structured generation, concise answers. |
| `google/gemma-2-9b-it:free` | 9B | 8,192 | Strong reasoning, shorter context window. |
| `qwen/qwen-2.5-7b-instruct:free` | 7B | 32,768 | Multilingual support, good at analysis tasks. |

**Model selection strategy**: The system defaults to `meta-llama/llama-3.1-8b-instruct:free`. If a request returns HTTP 429 or fails, the system retries with the next model in the rotation list before falling through to Ollama.

Model rotation order:
1. `meta-llama/llama-3.1-8b-instruct:free`
2. `mistralai/mistral-7b-instruct:free`
3. `google/gemma-2-9b-it:free`
4. `qwen/qwen-2.5-7b-instruct:free`

### OpenRouter Request Format

```json
{
  "model": "meta-llama/llama-3.1-8b-instruct:free",
  "messages": [
    {"role": "system", "content": "<system prompt>"},
    {"role": "user", "content": "<user prompt>"}
  ],
  "temperature": 0.3,
  "max_tokens": 4096
}
```

### Rate Limit Management

OpenRouter free tier enforces per-model and per-key rate limits. The system handles this with a layered strategy:

1. **Per-request retry**: On HTTP 429, read the `Retry-After` header (seconds). Wait that duration (capped at 60s), then retry once with the same model.
2. **Model rotation**: If the retry also returns 429, rotate to the next model in the list.
3. **Batch throttling**: When generating questions for an entire book (many chapters, many Bloom's levels), insert a 2-second delay between consecutive API calls to stay under burst limits.
4. **Concurrency limit**: At most 2 concurrent OpenRouter requests at any time, enforced by a buffered channel semaphore in the client.
5. **Fallback to Ollama**: If all four OpenRouter models are rate-limited, fall through to the Ollama client.

```go
type OpenRouterClient struct {
    APIKey     string
    BaseURL    string
    HTTPClient *http.Client
    Models     []string          // rotation list
    Semaphore  chan struct{}      // buffered channel, cap 2
    BatchDelay time.Duration     // 2s between calls in batch mode
}
```

---

## Ollama Configuration

### Connection Details

| Setting | Value |
|---|---|
| Base URL | `http://localhost:11434` (or `OLLAMA_URL` env var) |
| Chat endpoint | `POST /api/chat` |
| Embed endpoint | `POST /api/embeddings` |
| Auth | None (local service) |

Ollama runs as a Docker container defined in `docker-compose.yml`. It serves both embeddings (always) and LLM completions (fallback only).

### Models Needed

Pull these models after starting the Ollama container:

```bash
# LLM for question generation and scoring (fallback only)
ollama pull llama3.2:3b

# Embeddings (always used, not a fallback -- primary for all vector operations)
ollama pull nomic-embed-text
```

| Model | Parameters | Disk | RAM | Purpose |
|---|---|---|---|---|
| `llama3.2:3b` | 3B | ~2 GB | ~3 GB | Fallback LLM for all tasks. Runs on CPU without GPU. |
| `nomic-embed-text` | 137M | ~274 MB | ~300 MB | Embedding generation (768-dim). Always used for vector search. |

`llama3.2:3b` is the largest model that runs comfortably on CPU-only machines with 8 GB RAM alongside the embedding model. Since Ollama is a fallback (OpenRouter handles the primary load), 3B is an acceptable quality/resource trade-off. Expect ~30s per completion on CPU versus ~2s on OpenRouter.

### Ollama Request Format

```json
{
  "model": "llama3.2:3b",
  "messages": [
    {"role": "system", "content": "<system prompt>"},
    {"role": "user", "content": "<user prompt>"}
  ],
  "stream": false,
  "options": {
    "temperature": 0.3,
    "num_predict": 4096
  }
}
```

---

## Fallback Logic Details

### Timeout and Retry Behavior

| Scenario | Behavior |
|---|---|
| OpenRouter responds within 10s | Use response. Done. |
| OpenRouter times out (>10s) | Cancel request, fall through to Ollama. |
| OpenRouter returns HTTP 429 | Read `Retry-After`, wait (max 60s), retry once with same model. If still 429, rotate to next model. If all models exhausted, fall through to Ollama. |
| OpenRouter returns HTTP 5xx | Fall through to Ollama immediately (no retry on server errors). |
| OpenRouter network error | Fall through to Ollama immediately. |
| Ollama not running | Return error to caller. The processing pipeline marks the book/chapter as failed. |
| Both providers fail | Return error. The pipeline retries the entire task later with exponential backoff (max 3 attempts). |

### Health Check

On application startup and every 60 seconds, the `FallbackClient` pings both providers:

- **OpenRouter**: `GET https://openrouter.ai/api/v1/models` -- returns 200 if key is valid
- **Ollama**: `GET http://localhost:11434/api/tags` -- returns 200 if running

Health status is logged and exposed at `GET /api/v1/health` in the backend API response.

---

## Bloom's Taxonomy Integration

### The Six Cognitive Levels

Every generated question is tagged with exactly one Bloom's Taxonomy level. The LLM prompt includes explicit definitions and example question stems for each level to guide accurate classification.

| Level | Code | Cognitive Process | Question Stems |
|---|---|---|---|
| **Remember** | `remember` | Recall facts and basic concepts | Define..., List..., State..., Name..., Identify..., Label..., Recognize..., Recall..., Select..., Match..., What is...?, Where is...?, When did...?, Who was...?, How many...? |
| **Understand** | `understand` | Explain ideas or concepts | Explain..., Describe..., Summarize..., Paraphrase..., Classify..., Discuss..., Interpret..., Illustrate..., Review..., Distinguish..., Predict..., What does X mean?, Give an example of... |
| **Apply** | `apply` | Use information in new situations | Calculate..., Demonstrate..., Solve..., Use..., Apply..., Show..., Implement..., Compute..., Determine..., Predict the outcome of..., What would happen if...?, How would you use X to...? |
| **Analyze** | `analyze` | Draw connections among ideas | Compare..., Contrast..., Differentiate..., Distinguish..., Examine..., Categorize..., Analyze..., Investigate..., Why does X differ from Y?, What is the relationship between...?, What evidence supports...?, What are the causes of...? |
| **Evaluate** | `evaluate` | Justify a stance or decision | Judge..., Argue..., Evaluate..., Assess..., Critique..., Justify..., Defend..., Rate..., Prioritize..., Do you agree that...?, What is the most important...?, Which is more effective...?, Is X justified? |
| **Create** | `create` | Produce new or original work | Design..., Propose..., Construct..., Formulate..., Develop..., Plan..., Compose..., Create..., Invent..., How would you improve...?, What would you design to...?, Propose a solution for..., Devise an experiment to test... |

### Bloom's Level Distribution Per Chapter

When generating questions for a chapter, the system requests questions across all six levels with this target distribution:

| Level | Target % | Rationale |
|---|---|---|
| Remember | 20% | Foundation -- ensures basic recall is covered |
| Understand | 20% | Verifies comprehension of core ideas |
| Apply | 20% | Tests practical usage of concepts |
| Analyze | 15% | Higher-order thinking, connections between ideas |
| Evaluate | 15% | Critical judgment and argumentation |
| Create | 10% | Most challenging cognitive level, fewer needed |

The system makes one LLM call per Bloom's level per chapter (6 calls per chapter). Each call requests a number of questions proportional to the target distribution.

### Difficulty vs Bloom's Level

These are independent dimensions. Examples showing the cross-product:

- **Easy Remember**: "What is Newton's first law called?" (simple recall of a name)
- **Hard Remember**: "State all three of Newton's laws with their mathematical forms." (detailed recall)
- **Easy Analyze**: "Compare the SI units of force and mass." (straightforward comparison)
- **Hard Analyze**: "Analyze why the coefficient of static friction is always greater than kinetic friction at the molecular level." (deep structural analysis)

### Full Bloom's Reference (used in prompt templates)

**Remember:**
```
Description: Recall facts, terms, basic concepts, and answers.
Stems: Define, List, State, Name, Identify, Label, Recognize, Recall, Select, Match,
       What is, Where is, When did, Who was, How many
Example: "What is Newton's first law of motion?"
```

**Understand:**
```
Description: Demonstrate understanding of facts and ideas by organizing, comparing,
translating, interpreting, giving descriptions, and stating main ideas.
Stems: Explain, Describe, Summarize, Paraphrase, Classify, Discuss, Interpret,
       Illustrate, Review, Distinguish, Predict, What does X mean, Give an example of
Example: "Explain the relationship between force and acceleration in your own words."
```

**Apply:**
```
Description: Use acquired knowledge to solve problems in new situations. Apply
information, methods, concepts, and theories in new contexts.
Stems: Calculate, Demonstrate, Solve, Use, Apply, Show, Implement, Compute,
       Determine, Predict the outcome of, What would happen if, How would you use X to
Example: "Calculate the net force on a 5 kg object accelerating at 3 m/s^2."
```

**Analyze:**
```
Description: Examine and break information into component parts, determine how the
parts relate, identify motives or causes, make inferences, find evidence to support
generalizations.
Stems: Compare, Contrast, Differentiate, Distinguish, Examine, Categorize, Analyze,
       Investigate, Why does X differ from Y, What is the relationship between,
       What evidence supports, What are the causes of
Example: "Compare the effects of friction on stationary versus moving objects."
```

**Evaluate:**
```
Description: Present and defend opinions by making judgments about information, validity
of ideas, or quality of work based on a set of criteria.
Stems: Judge, Argue, Evaluate, Assess, Critique, Justify, Defend, Rate, Prioritize,
       Do you agree that, What is the most important, Which is more effective,
       Is X justified
Example: "Evaluate whether Newton's laws are sufficient to describe all types of motion.
Justify your answer."
```

**Create:**
```
Description: Compile information in a different way by combining elements in a new
pattern, proposing alternative solutions, designing experiments or models.
Stems: Design, Propose, Construct, Formulate, Develop, Plan, Compose, Create, Invent,
       How would you improve, What would you design to, Propose a solution for,
       Devise an experiment to test
Example: "Design an experiment to demonstrate Newton's third law using everyday materials."
```

---

## Prompt Templates

All prompts use a system/user message pair. The system message sets the role and output constraints. The user message contains the specific content and parameters. Temperature is 0.3 for all generation tasks (low creativity, high consistency) and 0.2 for scoring tasks (more deterministic).

Go `text/template` syntax is used below. Variables like `{{.ChapterTitle}}` are resolved at call time.

### 1. Question Generation

Called once per (chapter, bloom_level) pair. This is the highest-volume prompt in the system.

**System prompt:**

```
You are an expert exam question generator for educational content. You create
high-quality exam questions that accurately test student understanding at specific
cognitive levels based on Bloom's Taxonomy.

Rules:
- Generate questions ONLY from the provided chapter content. Do not use external
  knowledge.
- Each question must be answerable using only the information in the chapter.
- Match the specified Bloom's Taxonomy level precisely. Use the provided question
  stems as guidance.
- Match the specified difficulty level.
- For MCQ questions, provide exactly 4 options with exactly 1 correct answer.
  Distractors must be plausible, not obviously wrong.
- For true_false questions, provide the statement and whether it is "True" or "False".
- For fill_blank questions, use "___" to mark the blank in the question text.
- For essay and short_answer questions, provide a detailed model answer and at least
  3 key points that a student's answer should cover.
- For assertion_reasoning questions, provide an Assertion (A) and a Reason (R) with
  options about their relationship.
- For match questions, provide two columns of items to match.
- Include enrichment metadata for every question: what concept is tested, when it is
  relevant, how to approach it, and who the target audience is.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside
  the JSON array.
```

**User prompt:**

```
Generate {{.NumQuestions}} exam questions from the following chapter content.

CHAPTER: "{{.ChapterTitle}}"
BOOK: "{{.BookTitle}}" by {{.BookAuthor}}
SUBJECT: {{.Subject}}
GRADE LEVEL: {{.GradeLevel}}
EXAM TYPE: {{.ExamType}}

BLOOM'S TAXONOMY LEVEL: {{.BloomLevel}}
Definition: {{.BloomDescription}}
Question stems for this level: {{.BloomStems}}

DIFFICULTY: {{.Difficulty}}
QUESTION TYPES TO GENERATE: {{.QuestionTypes}}

CHAPTER CONTENT:
---
{{.ChapterContent}}
---

Respond with a JSON array. Each element must have this exact structure:

[
  {
    "question_text": "The full question text",
    "question_type": "mcq|essay|fill_blank|true_false|short_answer|match|assertion_reasoning",
    "difficulty": "easy|medium|hard",
    "bloom_level": "remember|understand|apply|analyze|evaluate|create",
    "topic": "Specific topic within the chapter this question covers",
    "options": [
      {"text": "Option A text", "is_correct": false},
      {"text": "Option B text", "is_correct": true},
      {"text": "Option C text", "is_correct": false},
      {"text": "Option D text", "is_correct": false}
    ],
    "correct_answer": "For MCQ: the correct option letter. For true_false: True or False. For fill_blank: the word/phrase that fills the blank.",
    "model_answer": "For essay/short_answer: a complete ideal answer demonstrating what a perfect response looks like.",
    "key_points": ["Key concept 1", "Key concept 2", "Key concept 3"],
    "explanation": "Why this answer is correct. For MCQ, also explain why each distractor is wrong.",
    "enrichment": {
      "what": "What concept or skill this question tests",
      "when": "When this knowledge is relevant -- exam context, real-world application, prerequisites for advanced topics",
      "how": "How a student should approach this question -- strategy, steps, common mistakes to avoid",
      "who": "Target audience -- grade level, exam type, education level, career relevance"
    },
    "tags": ["tag1", "tag2"]
  }
]

Field rules by question type:
- MCQ: include "options" (exactly 4) and "correct_answer". Omit "model_answer".
- essay / short_answer: include "model_answer" and "key_points" (at least 3). Omit "options".
- true_false: include "correct_answer" ("True" or "False"). Omit "options" and "model_answer".
- fill_blank: include "correct_answer" (the word/phrase). Omit "options" and "model_answer".
- assertion_reasoning: structure question_text as "Assertion (A): ... Reason (R): ..." and include 4 options about the A-R relationship.
- match: structure question_text with Column A and Column B items. correct_answer maps the pairings.

Return ONLY the JSON array. No other text.
```

**Template variable resolution:**

| Variable | Source | Example Value |
|---|---|---|
| `{{.NumQuestions}}` | Calculated from Bloom's distribution and total target | `4` |
| `{{.ChapterTitle}}` | `chapter.title` from MongoDB | `"Laws of Motion"` |
| `{{.BookTitle}}` | `book.title` from MongoDB | `"Physics Class 12 NCERT"` |
| `{{.BookAuthor}}` | `book.author` from MongoDB | `"H.C. Verma"` |
| `{{.Subject}}` | `book.subject` from MongoDB | `"Physics"` |
| `{{.GradeLevel}}` | `book.grade_levels[0]` or user-specified | `"Grade 12"` |
| `{{.ExamType}}` | `book.metadata["exam_type"]` or user-specified | `"CBSE Board"` |
| `{{.BloomLevel}}` | Current level being generated | `"Analyze"` |
| `{{.BloomDescription}}` | Hardcoded per level (see Bloom's Reference above) | `"Examine and break information into component parts..."` |
| `{{.BloomStems}}` | Hardcoded per level (see Bloom's Reference above) | `"Compare, Contrast, Differentiate, Distinguish..."` |
| `{{.Difficulty}}` | `"easy"`, `"medium"`, or `"hard"` | `"medium"` |
| `{{.QuestionTypes}}` | Comma-separated list | `"mcq, short_answer, essay"` |
| `{{.ChapterContent}}` | `chapter.content` from MongoDB, truncated if needed | Full or truncated chapter text |

**Content truncation**: If chapter content exceeds 6000 tokens (~24,000 characters), truncate to the first 6000 tokens. For Gemma-2 with its 8192-token context, truncate more aggressively to 3000 tokens. Token count is estimated as `len(text) / 4`.

**Temperature**: 0.3

### 2. Question Variant Generation

Given an existing question, generate the same concept in different question formats. Called after initial generation to create cross-format variants.

**System prompt:**

```
You are an expert at reformulating exam questions into different formats while
preserving the same underlying concept being tested. You maintain the same Bloom's
Taxonomy level, difficulty, and topic across all variants.

Rules:
- The new questions must test the EXACT SAME concept as the original question.
- Each variant must be a different question_type from the original.
- Maintain the same difficulty level and Bloom's Taxonomy level.
- Ensure all variants are independently answerable from the same chapter content.
- For MCQ variants, create plausible distractors that relate to the topic.
- For essay/short_answer variants, provide a complete model answer.
- Include enrichment metadata for every variant.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside
  the JSON array.
```

**User prompt:**

```
Given the following original question, generate variants in the specified formats.

ORIGINAL QUESTION:
- Text: "{{.OriginalQuestionText}}"
- Type: {{.OriginalType}}
- Bloom's Level: {{.BloomLevel}}
- Difficulty: {{.Difficulty}}
- Topic: {{.Topic}}
- Model Answer: "{{.ModelAnswer}}"
- Key Points: {{.KeyPointsJSON}}

CHAPTER CONTEXT (for reference):
---
{{.ChapterExcerpt}}
---

GENERATE VARIANTS IN THESE FORMATS: {{.TargetTypes}}

Respond with a JSON array. Each element must have this structure:

[
  {
    "question_text": "The reformulated question in the new format",
    "question_type": "mcq|essay|fill_blank|true_false|short_answer|match|assertion_reasoning",
    "difficulty": "{{.Difficulty}}",
    "bloom_level": "{{.BloomLevel}}",
    "topic": "{{.Topic}}",
    "options": [
      {"text": "Option text", "is_correct": false},
      {"text": "Option text", "is_correct": true},
      {"text": "Option text", "is_correct": false},
      {"text": "Option text", "is_correct": false}
    ],
    "correct_answer": "The correct answer for this format",
    "model_answer": "Complete ideal answer (for essay/short_answer only)",
    "key_points": ["Key concept 1", "Key concept 2", "Key concept 3"],
    "explanation": "Why this answer is correct",
    "enrichment": {
      "what": "What concept or skill this question tests",
      "when": "When this knowledge is relevant",
      "how": "How to approach this question",
      "who": "Target audience"
    },
    "tags": ["tag1", "tag2"]
  }
]

Return ONLY the JSON array. No other text.
```

**Template variable resolution:**

| Variable | Source |
|---|---|
| `{{.OriginalQuestionText}}` | `question_text` of the source question |
| `{{.OriginalType}}` | `question_type` of the source question |
| `{{.BloomLevel}}` | Carried over from the source question |
| `{{.Difficulty}}` | Carried over from the source question |
| `{{.Topic}}` | Carried over from the source question |
| `{{.ModelAnswer}}` | `model_answer` of the source question (empty string for MCQ/T-F) |
| `{{.KeyPointsJSON}}` | JSON-encoded array of `key_points` from the source question |
| `{{.ChapterExcerpt}}` | First 2000 characters of the chapter content for context |
| `{{.TargetTypes}}` | Comma-separated list of desired output types, excluding the original type |

**Temperature**: 0.3

### 3. Semantic Answer Scoring

Compares a student answer to the model answer and returns a similarity score with actionable feedback. Called for every essay and short_answer submission.

**System prompt:**

```
You are a precise academic answer evaluator. You compare a student's answer against
a model answer and score how well the student's response captures the intended
meaning, regardless of exact wording.

Rules:
- Score from 0 to 100 based on semantic similarity to the model answer.
- 90-100: Excellent -- captures all key ideas, may use different words but meaning
  is equivalent.
- 70-89: Good -- captures most key ideas with minor omissions or imprecisions.
- 50-69: Partial -- captures some key ideas but misses important aspects.
- 30-49: Weak -- shows some understanding but has significant gaps.
- 0-29: Poor -- does not demonstrate understanding of the concept.
- Be lenient about wording differences. "force equals mass times acceleration" and
  "F=ma" should score equally.
- Be strict about factual accuracy. Incorrect facts must reduce the score
  significantly.
- Provide specific, actionable feedback telling the student exactly what they got
  right and what they missed.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside
  the JSON object.
```

**User prompt:**

```
Evaluate the student's answer against the model answer for this question.

QUESTION: "{{.QuestionText}}"
BLOOM'S LEVEL: {{.BloomLevel}}
TOPIC: {{.Topic}}

MODEL ANSWER:
"{{.ModelAnswer}}"

KEY POINTS EXPECTED:
{{range .KeyPoints}}- {{.}}
{{end}}

STUDENT'S ANSWER:
"{{.StudentAnswer}}"

Respond with this exact JSON structure:

{
  "semantic_score": <integer 0-100>,
  "feedback": "Specific feedback explaining what the student got right, what they missed, and how to improve. Reference specific key points by name.",
  "key_points_covered": ["list of key points the student addressed"],
  "key_points_missed": ["list of key points the student did not address"],
  "factual_errors": ["list of any factually incorrect statements in the student's answer, empty array if none"]
}

Return ONLY the JSON object. No other text.
```

**Temperature**: 0.2 (more deterministic for consistent, reproducible scoring)

**Scoring integration**: The `semantic_score` from this response feeds into the three-dimension scoring system:

| Dimension | Weight | Source |
|---|---|---|
| Semantic score | 50% | `semantic_score` from this LLM response |
| Completeness score | 30% | `len(key_points_covered) / len(all_key_points) * 100`, cross-validated with keyword matching |
| Keyword score | 20% | Algorithmic -- checks presence of `key_points` terms in student answer text (no LLM needed) |

Final score: `overall = 0.50 * semantic + 0.30 * completeness + 0.20 * keyword`

### 4. Topic Extraction

Extracts key topics from a chapter for tagging and search. Called once per chapter during the processing pipeline.

**System prompt:**

```
You are an expert at identifying and extracting key topics from educational content.
You produce concise, specific topic labels suitable for tagging and categorization.

Rules:
- Extract between 3 and 15 topics depending on chapter length and complexity.
- Topics should be specific enough to be useful for search and filtering. Use
  "Newton's Third Law" not "Physics". Use "photosynthesis light reactions" not
  "biology".
- Topics should be noun phrases, not full sentences.
- Order topics from most prominent to least prominent in the chapter.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside
  the JSON object.
```

**User prompt:**

```
Extract the key topics from this chapter.

CHAPTER: "{{.ChapterTitle}}"
BOOK: "{{.BookTitle}}"
SUBJECT: {{.Subject}}

CHAPTER CONTENT:
---
{{.ChapterContent}}
---

Respond with this exact JSON structure:

{
  "topics": ["Topic 1", "Topic 2", "Topic 3"]
}

Return ONLY the JSON object. No other text.
```

**Temperature**: 0.3

**Usage**: The extracted topics are stored in `chapter.topics` (string array) and used to populate `question.tags` during question generation.

### 5. Chapter Summarization

Generates a concise summary of each chapter for display and quick reference. Called once per chapter during the processing pipeline.

**System prompt:**

```
You are an expert at summarizing educational content. You produce clear, concise
summaries that capture the key concepts, definitions, and relationships presented
in a chapter.

Rules:
- Write 3-5 sentences for short chapters (under 2000 words) and 5-8 sentences for
  longer chapters.
- Focus on concepts, definitions, and relationships -- not narrative or stylistic
  elements.
- Use precise academic language appropriate for the subject and grade level.
- Do not introduce information not present in the chapter.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside
  the JSON object.
```

**User prompt:**

```
Summarize the following chapter.

CHAPTER: "{{.ChapterTitle}}"
BOOK: "{{.BookTitle}}" by {{.BookAuthor}}
SUBJECT: {{.Subject}}
GRADE LEVEL: {{.GradeLevel}}

CHAPTER CONTENT:
---
{{.ChapterContent}}
---

Respond with this exact JSON structure:

{
  "summary": "The concise summary of the chapter content."
}

Return ONLY the JSON object. No other text.
```

**Temperature**: 0.3

**Usage**: Stored in `chapter.summary` and displayed on the frontend book detail page.

---

## Output JSON Parsing Strategy

All LLM responses are expected to be valid JSON (either an array or an object depending on the task). The parsing pipeline handles the reality that LLMs sometimes return malformed output.

### Parsing Pipeline

```
Raw LLM Response
  |
  v
Step 1: Strip markdown fences
  Remove ```json ... ``` or ``` ... ``` wrappers if present.
  |
  v
Step 2: Find JSON boundaries
  Locate the first '[' or '{' and the last matching ']' or '}'.
  Extract only that substring, discarding any surrounding text.
  |
  v
Step 3: Standard JSON parse
  json.Unmarshal into the target Go struct.
  |-- success --> validate and return
  |-- error --> Step 4
  |
  v
Step 4: Repair common issues
  - Remove trailing commas before ] or }
  - Replace single quotes with double quotes in key/value positions
  - Fix unescaped newlines inside string values
  - Remove control characters (0x00-0x1F except \n \r \t)
  Retry json.Unmarshal.
  |-- success --> validate and return
  |-- error --> Step 5
  |
  v
Step 5: Retry with LLM
  Send the malformed output back to the LLM with a repair prompt
  (temperature 0.1):
  "The following JSON is malformed. Fix it and return valid JSON only: <raw>"
  Parse the repaired output through Steps 1-4 only (no recursive retries).
  |-- success --> validate and return
  |-- error --> return parse error, skip this item
```

### Implementation

```go
// internal/llm/parse.go

func ParseJSONResponse[T any](raw string) (T, error) {
    var result T

    // Step 1: Strip markdown fences
    cleaned := raw
    cleaned = strings.TrimSpace(cleaned)
    if strings.HasPrefix(cleaned, "```json") {
        cleaned = strings.TrimPrefix(cleaned, "```json")
    } else if strings.HasPrefix(cleaned, "```") {
        cleaned = strings.TrimPrefix(cleaned, "```")
    }
    if strings.HasSuffix(cleaned, "```") {
        cleaned = strings.TrimSuffix(cleaned, "```")
    }
    cleaned = strings.TrimSpace(cleaned)

    // Step 2: Find JSON boundaries
    startArr := strings.Index(cleaned, "[")
    startObj := strings.Index(cleaned, "{")
    start := -1
    if startArr >= 0 && (startObj < 0 || startArr < startObj) {
        start = startArr
    } else if startObj >= 0 {
        start = startObj
    }
    if start > 0 {
        cleaned = cleaned[start:]
    }

    // Step 3: Standard parse
    if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
        return result, nil
    }

    // Step 4: Repair common issues
    repaired := repairJSON(cleaned)
    if err := json.Unmarshal([]byte(repaired), &result); err != nil {
        return result, fmt.Errorf("JSON parse failed after repair: %w", err)
    }

    return result, nil
}
```

### Validation After Parsing

After successful JSON parsing, each question is validated before storage:

```go
func ValidateQuestion(q *ParsedQuestion) error {
    if q.QuestionText == "" {
        return errors.New("question_text is required")
    }
    if !isValidQuestionType(q.QuestionType) {
        return fmt.Errorf("invalid question_type: %s", q.QuestionType)
    }
    if !isValidBloomLevel(q.BloomLevel) {
        return fmt.Errorf("invalid bloom_level: %s", q.BloomLevel)
    }
    if !isValidDifficulty(q.Difficulty) {
        return fmt.Errorf("invalid difficulty: %s", q.Difficulty)
    }
    if q.QuestionType == "mcq" && len(q.Options) != 4 {
        return fmt.Errorf("MCQ must have exactly 4 options, got %d", len(q.Options))
    }
    if q.QuestionType == "mcq" && countCorrect(q.Options) != 1 {
        return errors.New("MCQ must have exactly 1 correct option")
    }
    if (q.QuestionType == "essay" || q.QuestionType == "short_answer") && q.ModelAnswer == "" {
        return errors.New("essay/short_answer requires model_answer")
    }
    if q.Enrichment.What == "" || q.Enrichment.When == "" ||
       q.Enrichment.How == "" || q.Enrichment.Who == "" {
        return errors.New("all enrichment fields (what/when/how/who) are required")
    }
    return nil
}
```

Valid enum values:
- `question_type`: `mcq`, `essay`, `fill_blank`, `true_false`, `short_answer`, `match`, `assertion_reasoning`
- `bloom_level`: `remember`, `understand`, `apply`, `analyze`, `evaluate`, `create`
- `difficulty`: `easy`, `medium`, `hard`

Invalid questions are logged and discarded. The system does not fail the entire batch if a single question fails validation.

---

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `OPENROUTER_API_KEY` | Yes (for primary) | None | Free OpenRouter API key |
| `OPENROUTER_BASE_URL` | No | `https://openrouter.ai/api/v1` | Override for testing |
| `OPENROUTER_DEFAULT_MODEL` | No | `meta-llama/llama-3.1-8b-instruct:free` | Primary model |
| `OPENROUTER_TIMEOUT` | No | `10s` | Per-request timeout before fallback |
| `OPENROUTER_MAX_RETRIES` | No | `1` | Retries on 429 per model before rotating |
| `OPENROUTER_BATCH_DELAY` | No | `2s` | Delay between consecutive batch calls |
| `OLLAMA_URL` | No | `http://localhost:11434` | Ollama base URL |
| `OLLAMA_MODEL` | No | `llama3.2:3b` | Ollama model for LLM tasks |
| `LLM_TEMPERATURE` | No | `0.3` | Default temperature for generation |
| `LLM_MAX_TOKENS` | No | `4096` | Default max tokens per response |

### Temperature Settings by Task

| Task | Temperature | Rationale |
|---|---|---|
| Question generation | 0.3 | Some variety in questions, but consistent JSON structure |
| Variant generation | 0.3 | Same concept in a new format -- needs structural consistency |
| Semantic scoring | 0.2 | Scoring must be deterministic and reproducible |
| Topic extraction | 0.3 | Minor variation acceptable |
| Chapter summarization | 0.3 | Consistent, factual output preferred |
| JSON repair (retry) | 0.1 | Purely mechanical fix, minimize creativity |

---

## Processing Pipeline Integration

This shows how LLM calls fit into the overall book processing pipeline:

```
Book stored in MongoDB (status: processing)
  |
  v
For each chapter:
  |
  +--> [Ollama] Generate embedding (nomic-embed-text, 768-dim)
  |       Store in chapter.embedding
  |
  +--> [LLM] Extract topics
  |       Store in chapter.topics
  |
  +--> [LLM] Generate summary
  |       Store in chapter.summary
  |
  +--> For each Bloom's level (6 iterations):
  |       |
  |       +--> [LLM] Generate questions (N per level, mixed difficulty and types)
  |       |       Parse JSON --> validate --> store in questions collection
  |       |
  |       +--> [2s delay if using OpenRouter batch mode]
  |
  +--> [LLM] Generate variants for selected questions
  |       Link originals and variants via related_question_ids
  |
  v
Update book status: ready
```

### Call Volume Estimate

Total LLM calls per chapter (approximate):
- 1 topic extraction call
- 1 summarization call
- 6 question generation calls (one per Bloom's level)
- 2-3 variant generation calls (for selected high-quality questions)
- **Total: ~10-11 LLM calls per chapter**

For a book with 10 chapters, that is roughly 100-110 LLM calls. With the 2-second batch delay on OpenRouter, the entire generation process takes approximately 4-5 minutes. If Ollama fallback is used for the full book (CPU-only, ~30s per call), expect 50-55 minutes.

### Batch Processing Strategy

For processing a new book with N chapters:

1. **Phase 1 -- Metadata** (fast, N calls each): Extract topics and generate summaries for all chapters. These are short responses and rarely hit rate limits.
2. **Phase 2 -- Questions** (slow, 6N calls): Generate questions per chapter, one Bloom's level at a time. Throttle to respect rate limits. Save progress after each chapter so the pipeline can resume on failure.
3. **Phase 3 -- Variants** (moderate, 2-3N calls): Generate variants for selected questions. Run after all base questions are stored.

Progress is tracked per-chapter. If the pipeline fails mid-book, it resumes from the last incomplete chapter rather than restarting from the beginning.
