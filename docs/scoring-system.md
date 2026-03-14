# Scoring System

## Overview

The scoring system evaluates student answers using three independent dimensions that are combined into a single overall score. Objective question types (MCQ, true/false, fill-in-the-blank) use deterministic right/wrong evaluation. Subjective question types (essay, short answer) use the full three-dimension scoring pipeline, which combines LLM-based semantic analysis with algorithmic keyword and completeness checks.

Implementation lives in `internal/scoring/`.

## Three-Dimension Scoring Model

Every subjective answer is evaluated on three orthogonal dimensions:

| Dimension | Default Weight | Method | What It Measures |
|---|---|---|---|
| Semantic | 50% | LLM comparison | Whether the student's answer conveys the same meaning as the model answer, regardless of wording |
| Completeness | 30% | Algorithmic (key_points coverage) | How many of the expected key points the student addressed |
| Keyword | 20% | Algorithmic (stemmed term matching) | Whether domain-specific terminology and key terms appear in the answer |

Each dimension produces an independent score from 0 to 100. The overall score is a weighted combination.

## Score Ranges

All scores (per-dimension and overall) are normalized to a 0-100 scale.

| Range | Label | Interpretation |
|---|---|---|
| 90-100 | Excellent | Comprehensive, accurate, uses correct terminology |
| 75-89 | Good | Mostly correct with minor gaps in coverage or terminology |
| 60-74 | Satisfactory | Demonstrates understanding but misses key points or terms |
| 40-59 | Needs Improvement | Partial understanding, significant gaps |
| 20-39 | Poor | Minimal relevant content, major conceptual errors |
| 0-19 | Inadequate | No meaningful attempt or completely incorrect |

These labels are used in feedback generation and the student-facing score breakdown UI.

## Semantic Scoring (50%)

Semantic scoring uses the LLM to compare the meaning of the student's answer against the model answer. This is the most important dimension because it captures correct answers expressed in different words, alternative phrasings, and valid approaches the model answer did not anticipate.

### LLM Prompt Structure

The LLM receives:
1. The original question text
2. The model answer (`question.model_answer`)
3. The student's answer (`user_answer.answer_text`)
4. Instructions to return a numeric score (0-100) and a brief justification

### Scoring Criteria

The LLM evaluates:
- **Factual accuracy** -- Are the claims in the student's answer correct?
- **Conceptual alignment** -- Does the answer address the same concepts as the model answer?
- **Depth of explanation** -- Is the reasoning sufficient for the question's Bloom's level?
- **Logical coherence** -- Does the answer follow a logical structure?

### Edge Cases

- **Correct but differently worded**: Should score 80-100 on semantic. The LLM is instructed to not penalize paraphrasing.
- **Partially correct**: LLM assigns proportional credit (e.g., 40-70) based on how much of the core meaning is captured.
- **Correct answer with extra incorrect content**: LLM is instructed to penalize incorrect additions that demonstrate misunderstanding.
- **Empty or gibberish answer**: Returns 0.

### Fallback

If the LLM call fails (network error, rate limit), the system falls back to embedding-based cosine similarity between the student answer and the model answer using `nomic-embed-text`. This produces a rougher score but ensures scoring never fully fails.

## Keyword Scoring (20%)

Keyword scoring is purely algorithmic -- no LLM call required. It checks whether the student used the expected domain-specific terms from the question's `key_points` field.

### Stemming

Both the student answer and the key terms are stemmed before comparison using the Porter stemming algorithm. This handles morphological variants:

| Original Term | Stemmed |
|---|---|
| "acceleration" | "acceler" |
| "accelerating" | "acceler" |
| "accelerated" | "acceler" |
| "gravitational" | "gravit" |
| "gravity" | "gravit" |

### Algorithm

```
Input:
  answer_text: string         (student's answer)
  key_points:  []string       (from question.key_points)

Steps:
  1. Tokenize answer_text into words (lowercase, strip punctuation)
  2. Stem each token using Porter stemmer
  3. Build a set of stemmed answer tokens
  4. For each key_point:
     a. Tokenize and stem the key_point phrase
     b. Check if ALL stemmed tokens of the key_point appear in the answer token set
     c. If yes, mark this key_point as "matched"
  5. keyword_score = (matched_count / total_key_points) * 100

Output:
  keyword_score: float64 (0-100)
  matched_keywords: []string   (for feedback)
  missing_keywords: []string   (for feedback)
```

### Example

Question key_points: `["Newton's third law", "action-reaction pair", "equal and opposite force"]`

Student answer: "Every action has an equal and opposite reaction according to Newton's law."

- "Newton's third law" -- "newton", "third", "law" -- "third" is missing from answer. **Not matched.**
- "action-reaction pair" -- "action", "reaction", "pair" -- "pair" missing. **Not matched.**
- "equal and opposite force" -- "equal", "opposit", "forc" -- all present ("opposite" stems to "opposit", "force" matches). **Matched.**

keyword_score = (1 / 3) * 100 = 33.3

## Completeness Scoring (30%)

Completeness scoring measures what fraction of the expected key points the student's answer addresses. Unlike keyword scoring, completeness checks for conceptual coverage -- the student does not need to use the exact term, just express the concept.

### Algorithm

```
Input:
  answer_text: string
  key_points:  []string       (from question.key_points)

Steps:
  1. For each key_point in key_points:
     a. Check for direct textual overlap (substring match after normalization)
     b. If no direct match, check for synonym/paraphrase coverage:
        - Tokenize the key_point and the answer
        - Compute token overlap ratio
        - If overlap ratio >= 0.6, consider the key_point covered
     c. Mark as "covered" or "not covered"
  2. completeness_score = (covered_count / total_key_points) * 100

Output:
  completeness_score: float64 (0-100)
  covered_points:     []string  (for feedback)
  missing_points:     []string  (for feedback)
```

### Difference from Keyword Scoring

| Aspect | Keyword Scoring | Completeness Scoring |
|---|---|---|
| What it checks | Exact term presence (stemmed) | Conceptual coverage |
| Matching method | Token set intersection | Substring + token overlap ratio |
| Sensitivity | Strict -- requires the specific term | Lenient -- accepts paraphrases |
| Purpose | Ensures use of domain terminology | Ensures all points are addressed |

A student could score high on completeness (expressed all concepts) but low on keywords (did not use the standard terminology), or vice versa. The two dimensions are complementary.

## Overall Score Formula

```
overall_score = (semantic_weight * semantic_score)
              + (completeness_weight * completeness_score)
              + (keyword_weight * keyword_score)
```

With default weights:

```
overall_score = 0.50 * semantic_score
              + 0.30 * completeness_score
              + 0.20 * keyword_score
```

### Configurable Weights per Question Type

Weights can be adjusted per question type to reflect what matters most for that format:

| Question Type | Semantic | Completeness | Keyword | Rationale |
|---|---|---|---|---|
| `essay` | 0.50 | 0.30 | 0.20 | Default balanced weights |
| `short_answer` | 0.40 | 0.30 | 0.30 | Terminology matters more in concise answers |

Weights are stored as a configuration map in the scoring service. The sum of weights for any question type must equal 1.0.

```go
var ScoringWeights = map[string]Weights{
    "essay":        {Semantic: 0.50, Completeness: 0.30, Keyword: 0.20},
    "short_answer": {Semantic: 0.40, Completeness: 0.30, Keyword: 0.30},
}
```

If a question type has no entry in the map, the default weights (50/30/20) are used.

## Scoring by Question Type

### MCQ (`mcq`)

Deterministic evaluation. The student selects one option; it either matches `question.correct_answer` or it does not.

| Field | Value |
|---|---|
| `is_correct` | `true` if selected option matches `correct_answer`, else `false` |
| `overall_score` | 100 if correct, 0 if incorrect |
| `semantic_score` | Not set |
| `keyword_score` | Not set |
| `completeness_score` | Not set |
| `feedback` | If correct: explanation. If wrong: correct answer + explanation. |

### True/False (`true_false`)

Same as MCQ. Binary right/wrong comparison against `question.correct_answer` ("True" or "False").

| Field | Value |
|---|---|
| `is_correct` | `true` if answer matches `correct_answer`, else `false` |
| `overall_score` | 100 if correct, 0 if incorrect |
| `feedback` | Explanation of why the statement is true or false |

### Fill in the Blank (`fill_blank`)

Case-insensitive exact match against `question.correct_answer`. Trimmed of leading/trailing whitespace.

| Field | Value |
|---|---|
| `is_correct` | `true` if normalized answer matches normalized `correct_answer` |
| `overall_score` | 100 if correct, 0 if incorrect |
| `feedback` | Correct answer + explanation |

### Short Answer (`short_answer`)

Full three-dimension scoring with adjusted weights (Semantic 40%, Completeness 30%, Keyword 30%).

| Field | Value |
|---|---|
| `is_correct` | Not set (use `overall_score` instead) |
| `semantic_score` | 0-100 via LLM |
| `keyword_score` | 0-100 via stemmed matching |
| `completeness_score` | 0-100 via key_points coverage |
| `overall_score` | Weighted combination |
| `feedback` | Per-dimension feedback + missing points + missing keywords |

### Essay (`essay`)

Full three-dimension scoring with default weights (Semantic 50%, Completeness 30%, Keyword 20%).

| Field | Value |
|---|---|
| `is_correct` | Not set (use `overall_score` instead) |
| `semantic_score` | 0-100 via LLM |
| `keyword_score` | 0-100 via stemmed matching |
| `completeness_score` | 0-100 via key_points coverage |
| `overall_score` | Weighted combination |
| `feedback` | Detailed per-dimension feedback + actionable improvement suggestions |

### Match the Following (`match`)

Each pair is evaluated independently. The overall score is the percentage of correctly matched pairs.

| Field | Value |
|---|---|
| `is_correct` | `true` if all pairs match, else `false` |
| `overall_score` | `(correct_pairs / total_pairs) * 100` |
| `feedback` | Lists which pairs were correct and which were wrong |

### Assertion-Reasoning (`assertion_reasoning`)

Evaluated as a composite: the student must correctly identify (a) whether the assertion is true, (b) whether the reasoning is true, and (c) whether the reasoning correctly explains the assertion.

| Field | Value |
|---|---|
| `is_correct` | `true` only if all three sub-evaluations are correct |
| `overall_score` | 100 if fully correct, partial credit based on sub-evaluations |
| `feedback` | Breakdown of which parts were correct/incorrect + explanation |

## Feedback Generation

Every scored answer receives human-readable feedback. The feedback structure differs by question type.

### Objective Questions (MCQ, True/False, Fill-blank)

```
If correct:
  "Correct! [question.explanation]"

If incorrect:
  "The correct answer is [correct_answer]. [question.explanation]"
```

### Subjective Questions (Essay, Short Answer)

Feedback is assembled from multiple components:

1. **Overall assessment**: A 1-2 sentence summary based on the overall score and its label.
2. **Semantic feedback**: The LLM's justification for its semantic score (returned alongside the score).
3. **Completeness feedback**: "You covered N of M key points. Missing: [list of missing_points]."
4. **Keyword feedback**: "Key terms used: [matched_keywords]. Missing terms: [missing_keywords]."
5. **Improvement suggestion**: If overall_score < 75, the system adds a targeted suggestion based on the weakest dimension.

Example output:

```
Overall: Good (78/100)

Your answer demonstrates solid understanding of Newton's third law and correctly
identifies the action-reaction pair concept.

Key points covered: 3 of 4. Missing: "forces act on different objects"
Key terms used: action, reaction, equal, opposite. Missing: "pair"

Suggestion: Make sure to mention that the action and reaction forces act on
different objects, not the same object.
```

## Admin Metrics

The scoring system feeds into admin-facing analytics available via `GET /api/v1/admin/dashboard`.

### Per-Question Metrics

| Metric | Aggregation | Purpose |
|---|---|---|
| Average overall score | Mean of `user_answers.overall_score` per question | Identifies questions that are too easy or too hard |
| Average semantic score | Mean of `semantic_score` per question | Detects questions where students understand concepts but use wrong terms (high semantic, low keyword) |
| Average keyword score | Mean of `keyword_score` per question | Detects questions where key_points terms may be too specific |
| Average completeness score | Mean of `completeness_score` per question | Detects questions where key_points list may be too long |
| Attempt count | Count of `user_answers` per question | Tracks question popularity |
| Correct rate (objective) | `is_correct = true` / total attempts | For MCQ/T-F/fill_blank difficulty calibration |

### Per-Student Metrics

| Metric | Aggregation | Purpose |
|---|---|---|
| Average score by Bloom's level | Mean `overall_score` grouped by `question.bloom_level` | Identifies cognitive weak spots (e.g., strong at Remember, weak at Analyze) |
| Average score by difficulty | Mean `overall_score` grouped by `question.difficulty` | Tracks readiness for harder content |
| Score trend over time | `overall_score` ordered by `created_at` | Tracks improvement |
| Weakest dimension | Lowest average among semantic, completeness, keyword | Targeted coaching: "Work on using technical terminology" (low keyword) |
| Time per question | Average `time_taken` grouped by question type | Identifies where students struggle most |

### Platform-Wide Metrics

| Metric | Aggregation | Purpose |
|---|---|---|
| Score distribution | Histogram of `overall_score` across all answers | Monitors overall platform difficulty |
| Hardest questions | Bottom 10 by average `overall_score` | Candidates for review or hint addition |
| Easiest questions | Top 10 by average `overall_score` | May need difficulty increase |
| Average score by subject | Mean `overall_score` grouped by `book.subject` | Cross-subject difficulty comparison |
| LLM scoring latency | Time between answer submission and score return | Monitors scoring performance |
