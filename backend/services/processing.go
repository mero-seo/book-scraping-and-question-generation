package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"internal/db"
	"internal/llm"
	"internal/models"
	"internal/vectordb"
	"log"
	"strings"
	"text/template"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ProcessingService handles the book processing pipeline:
// embed chapters, extract topics, summarize, generate questions.
type ProcessingService struct {
	DB       *db.MongoDB
	LLM      llm.LLMClient
	Pinecone *vectordb.PineconeClient
}

// BloomConfig holds the distribution and metadata for a Bloom's level.
type BloomConfig struct {
	Level       string
	Percentage  float64
	Description string
	Stems       string
}

var bloomConfigs = []BloomConfig{
	{
		Level:       "remember",
		Percentage:  0.20,
		Description: "Recall facts, terms, basic concepts, and answers.",
		Stems:       "Define, List, State, Name, Identify, Label, Recognize, Recall, Select, Match, What is, Where is, When did, Who was, How many",
	},
	{
		Level:       "understand",
		Percentage:  0.20,
		Description: "Demonstrate understanding of facts and ideas by organizing, comparing, translating, interpreting, giving descriptions, and stating main ideas.",
		Stems:       "Explain, Describe, Summarize, Paraphrase, Classify, Discuss, Interpret, Illustrate, Review, Distinguish, Predict, What does X mean, Give an example of",
	},
	{
		Level:       "apply",
		Percentage:  0.20,
		Description: "Use acquired knowledge to solve problems in new situations. Apply information, methods, concepts, and theories in new contexts.",
		Stems:       "Calculate, Demonstrate, Solve, Use, Apply, Show, Implement, Compute, Determine, Predict the outcome of, What would happen if, How would you use X to",
	},
	{
		Level:       "analyze",
		Percentage:  0.15,
		Description: "Examine and break information into component parts, determine how the parts relate, identify motives or causes, make inferences, find evidence to support generalizations.",
		Stems:       "Compare, Contrast, Differentiate, Distinguish, Examine, Categorize, Analyze, Investigate, Why does X differ from Y, What is the relationship between, What evidence supports, What are the causes of",
	},
	{
		Level:       "evaluate",
		Percentage:  0.15,
		Description: "Present and defend opinions by making judgments about information, validity of ideas, or quality of work based on a set of criteria.",
		Stems:       "Judge, Argue, Evaluate, Assess, Critique, Justify, Defend, Rate, Prioritize, Do you agree that, What is the most important, Which is more effective, Is X justified",
	},
	{
		Level:       "create",
		Percentage:  0.10,
		Description: "Compile information in a different way by combining elements in a new pattern, proposing alternative solutions, designing experiments or models.",
		Stems:       "Design, Propose, Construct, Formulate, Develop, Plan, Compose, Create, Invent, How would you improve, What would you design to, Propose a solution for, Devise an experiment to test",
	},
}

const questionsPerChapter = 20

const questionGenSystemPrompt = `You are an expert exam question generator for educational content. You create high-quality exam questions that accurately test student understanding at specific cognitive levels based on Bloom's Taxonomy.

Rules:
- Generate questions ONLY from the provided chapter content. Do not use external knowledge.
- Each question must be answerable using only the information in the chapter.
- Match the specified Bloom's Taxonomy level precisely. Use the provided question stems as guidance.
- Match the specified difficulty level.
- For MCQ questions, provide exactly 4 options with exactly 1 correct answer. Distractors must be plausible, not obviously wrong.
- For true_false questions, provide the statement and whether it is "True" or "False".
- For fill_blank questions, use "___" to mark the blank in the question text.
- For essay and short_answer questions, provide a detailed model answer and at least 3 key points that a student's answer should cover.
- For assertion_reasoning questions, provide an Assertion (A) and a Reason (R) with options about their relationship.
- For match questions, provide two columns of items to match.
- Include enrichment metadata for every question: what concept is tested, when it is relevant, how to approach it, and who the target audience is.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside the JSON array.`

const topicExtractionSystemPrompt = `You are an expert at identifying and extracting key topics from educational content. You produce concise, specific topic labels suitable for tagging and categorization.

Rules:
- Extract between 3 and 15 topics depending on chapter length and complexity.
- Topics should be specific enough to be useful for search and filtering. Use "Newton's Third Law" not "Physics". Use "photosynthesis light reactions" not "biology".
- Topics should be noun phrases, not full sentences.
- Order topics from most prominent to least prominent in the chapter.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside the JSON object.`

const summarizationSystemPrompt = `You are an expert at summarizing educational content. You produce clear, concise summaries that capture the key concepts, definitions, and relationships presented in a chapter.

Rules:
- Write 3-5 sentences for short chapters (under 2000 words) and 5-8 sentences for longer chapters.
- Focus on concepts, definitions, and relationships -- not narrative or stylistic elements.
- Use precise academic language appropriate for the subject and grade level.
- Do not introduce information not present in the chapter.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside the JSON object.`

// ProcessBook runs the full processing pipeline for a book.
func (s *ProcessingService) ProcessBook(ctx context.Context, bookID bson.ObjectID) error {
	// Update status to processing
	_, err := s.DB.Books().UpdateOne(ctx, bson.M{"_id": bookID}, bson.M{
		"$set": bson.M{"status": models.BookStatusProcessing, "updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to update book status: %w", err)
	}

	// Get chapters
	cursor, err := s.DB.Chapters().Find(ctx, bson.M{"book_id": bookID})
	if err != nil {
		return s.failBook(ctx, bookID, fmt.Errorf("failed to find chapters: %w", err))
	}
	defer cursor.Close(ctx)

	var chapters []*models.Chapter
	if err := cursor.All(ctx, &chapters); err != nil {
		return s.failBook(ctx, bookID, fmt.Errorf("failed to decode chapters: %w", err))
	}

	if len(chapters) == 0 {
		return s.failBook(ctx, bookID, fmt.Errorf("no chapters found"))
	}

	// Get book for metadata
	var book models.Book
	err = s.DB.Books().FindOne(ctx, bson.M{"_id": bookID}).Decode(&book)
	if err != nil {
		return s.failBook(ctx, bookID, fmt.Errorf("failed to find book: %w", err))
	}

	// Process each chapter
	for _, chapter := range chapters {
		if err := s.processChapter(ctx, &book, chapter); err != nil {
			log.Printf("Failed to process chapter %d (%s): %v", chapter.Number, chapter.Title, err)
			continue // Don't fail the whole book for one chapter
		}
	}

	// Mark as ready
	_, err = s.DB.Books().UpdateOne(ctx, bson.M{"_id": bookID}, bson.M{
		"$set": bson.M{"status": models.BookStatusReady, "updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to update book status to ready: %w", err)
	}

	return nil
}

func (s *ProcessingService) processChapter(ctx context.Context, book *models.Book, chapter *models.Chapter) error {
	// Step 1: Generate embedding and store in Pinecone
	if s.Pinecone != nil && chapter.Content != "" {
		embedding, err := s.Pinecone.EmbedChapter(ctx, chapter.Content)
		if err != nil {
			log.Printf("Embedding failed for chapter %d: %v", chapter.Number, err)
		} else {
			err = s.Pinecone.UpsertChapter(ctx, chapter.ID.Hex(), book.ID.Hex(), chapter.Title, embedding)
			if err != nil {
				log.Printf("Failed to store embedding in Pinecone for chapter %d: %v", chapter.Number, err)
			}
		}
	}

	// Step 2: Extract topics
	if len(chapter.Topics) == 0 {
		topics, err := s.extractTopics(ctx, book, chapter)
		if err != nil {
			log.Printf("Topic extraction failed for chapter %d: %v", chapter.Number, err)
		} else {
			chapter.Topics = topics
			_, _ = s.DB.Chapters().UpdateOne(ctx, bson.M{"_id": chapter.ID}, bson.M{
				"$set": bson.M{"topics": topics},
			})
		}
	}

	// Step 3: Generate summary
	if chapter.Summary == "" {
		summary, err := s.summarizeChapter(ctx, book, chapter)
		if err != nil {
			log.Printf("Summarization failed for chapter %d: %v", chapter.Number, err)
		} else {
			chapter.Summary = summary
			_, _ = s.DB.Chapters().UpdateOne(ctx, bson.M{"_id": chapter.ID}, bson.M{
				"$set": bson.M{"summary": summary},
			})
		}
	}

	// Step 4: Generate questions per Bloom's level
	gradeLevel := ""
	if len(book.GradeLevels) > 0 {
		gradeLevel = book.GradeLevels[0]
	}
	examType := book.Metadata["exam_type"]

	for _, bloom := range bloomConfigs {
		numQuestions := int(float64(questionsPerChapter) * bloom.Percentage)
		if numQuestions < 1 {
			numQuestions = 1
		}

		questions, err := s.generateQuestions(ctx, book, chapter, bloom, numQuestions, gradeLevel, examType)
		if err != nil {
			log.Printf("Question generation failed for chapter %d, bloom %s: %v", chapter.Number, bloom.Level, err)
			continue
		}

		// Insert questions
		if len(questions) > 0 {
			docs := make([]interface{}, len(questions))
			for i, q := range questions {
				docs[i] = q
			}
			_, err = s.DB.Questions().InsertMany(ctx, docs)
			if err != nil {
				log.Printf("Failed to insert questions for chapter %d: %v", chapter.Number, err)
			}
		}

		// Batch delay between LLM calls
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (s *ProcessingService) extractTopics(ctx context.Context, book *models.Book, chapter *models.Chapter) ([]string, error) {
	content := truncateContent(chapter.Content, 6000)

	userPrompt := fmt.Sprintf(`Extract the key topics from this chapter.

CHAPTER: "%s"
BOOK: "%s"
SUBJECT: %s

CHAPTER CONTENT:
---
%s
---

Respond with this exact JSON structure:

{
  "topics": ["Topic 1", "Topic 2", "Topic 3"]
}

Return ONLY the JSON object. No other text.`, chapter.Title, book.Title, book.Subject, content)

	var result struct {
		Topics []string `json:"topics"`
	}
	err := s.LLM.CompleteJSON(ctx, llm.CompletionRequest{
		SystemPrompt: topicExtractionSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    512,
	}, &result)
	if err != nil {
		return nil, err
	}

	return result.Topics, nil
}

func (s *ProcessingService) summarizeChapter(ctx context.Context, book *models.Book, chapter *models.Chapter) (string, error) {
	content := truncateContent(chapter.Content, 6000)
	gradeLevel := ""
	if len(book.GradeLevels) > 0 {
		gradeLevel = book.GradeLevels[0]
	}

	userPrompt := fmt.Sprintf(`Summarize the following chapter.

CHAPTER: "%s"
BOOK: "%s" by %s
SUBJECT: %s
GRADE LEVEL: %s

CHAPTER CONTENT:
---
%s
---

Respond with this exact JSON structure:

{
  "summary": "The concise summary of the chapter content."
}

Return ONLY the JSON object. No other text.`, chapter.Title, book.Title, book.Author, book.Subject, gradeLevel, content)

	var result struct {
		Summary string `json:"summary"`
	}
	err := s.LLM.CompleteJSON(ctx, llm.CompletionRequest{
		SystemPrompt: summarizationSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    512,
	}, &result)
	if err != nil {
		return "", err
	}

	return result.Summary, nil
}

// ParsedQuestion represents a question as parsed from LLM JSON output.
type ParsedQuestion struct {
	QuestionText string `json:"question_text"`
	QuestionType string `json:"question_type"`
	Difficulty   string `json:"difficulty"`
	BloomLevel   string `json:"bloom_level"`
	Topic        string `json:"topic"`
	Options      []struct {
		Text      string `json:"text"`
		IsCorrect bool   `json:"is_correct"`
	} `json:"options,omitempty"`
	CorrectAnswer string `json:"correct_answer,omitempty"`
	ModelAnswer   string `json:"model_answer,omitempty"`
	KeyPoints     []string `json:"key_points,omitempty"`
	Explanation   string `json:"explanation,omitempty"`
	Enrichment    struct {
		What string `json:"what"`
		When string `json:"when"`
		How  string `json:"how"`
		Who  string `json:"who"`
	} `json:"enrichment"`
	Tags []string `json:"tags,omitempty"`
}

var questionGenUserTmpl = template.Must(template.New("qgen").Parse(`Generate {{.NumQuestions}} exam questions from the following chapter content.

CHAPTER: "{{.ChapterTitle}}"
BOOK: "{{.BookTitle}}" by {{.BookAuthor}}
SUBJECT: {{.Subject}}
GRADE LEVEL: {{.GradeLevel}}
EXAM TYPE: {{.ExamType}}

BLOOM'S TAXONOMY LEVEL: {{.BloomLevel}}
Definition: {{.BloomDescription}}
Question stems for this level: {{.BloomStems}}

DIFFICULTY: mixed (easy, medium, hard)
QUESTION TYPES TO GENERATE: mcq, short_answer, essay, true_false, fill_blank

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
    "bloom_level": "{{.BloomLevel}}",
    "topic": "Specific topic within the chapter",
    "options": [{"text": "Option A", "is_correct": false}, {"text": "Option B", "is_correct": true}, {"text": "Option C", "is_correct": false}, {"text": "Option D", "is_correct": false}],
    "correct_answer": "The correct answer",
    "model_answer": "For essay/short_answer: complete ideal answer",
    "key_points": ["Key concept 1", "Key concept 2", "Key concept 3"],
    "explanation": "Why this answer is correct",
    "enrichment": {"what": "concept tested", "when": "relevance", "how": "approach strategy", "who": "target audience"},
    "tags": ["tag1", "tag2"]
  }
]

Field rules by question type:
- MCQ: include "options" (exactly 4) and "correct_answer". Omit "model_answer".
- essay / short_answer: include "model_answer" and "key_points" (at least 3). Omit "options".
- true_false: include "correct_answer" ("True" or "False"). Omit "options" and "model_answer".
- fill_blank: include "correct_answer". Omit "options" and "model_answer".

Return ONLY the JSON array. No other text.`))

func (s *ProcessingService) generateQuestions(ctx context.Context, book *models.Book, chapter *models.Chapter, bloom BloomConfig, numQuestions int, gradeLevel, examType string) ([]*models.Question, error) {
	content := truncateContent(chapter.Content, 6000)

	var buf bytes.Buffer
	err := questionGenUserTmpl.Execute(&buf, map[string]interface{}{
		"NumQuestions":     numQuestions,
		"ChapterTitle":    chapter.Title,
		"BookTitle":       book.Title,
		"BookAuthor":      book.Author,
		"Subject":         book.Subject,
		"GradeLevel":      gradeLevel,
		"ExamType":        examType,
		"BloomLevel":      bloom.Level,
		"BloomDescription": bloom.Description,
		"BloomStems":      bloom.Stems,
		"ChapterContent":  content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	raw, err := s.LLM.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: questionGenSystemPrompt,
		UserPrompt:   buf.String(),
		Temperature:  0.3,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse the JSON array
	parsed, err := llm.ParseJSONResponse[[]ParsedQuestion](raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse questions JSON: %w", err)
	}

	// Convert to models and validate
	now := time.Now()
	var questions []*models.Question
	for _, pq := range parsed {
		if err := validateParsedQuestion(&pq); err != nil {
			log.Printf("Skipping invalid question: %v", err)
			continue
		}

		q := &models.Question{
			BookID:        book.ID,
			ChapterID:     chapter.ID,
			Topic:         pq.Topic,
			QuestionText:  pq.QuestionText,
			QuestionType:  pq.QuestionType,
			Difficulty:    pq.Difficulty,
			BloomLevel:    pq.BloomLevel,
			GradeLevel:    gradeLevel,
			ExamType:      examType,
			CorrectAnswer: pq.CorrectAnswer,
			ModelAnswer:   pq.ModelAnswer,
			KeyPoints:     pq.KeyPoints,
			Explanation:   pq.Explanation,
			Enrichment: models.Enrichment{
				What: pq.Enrichment.What,
				When: pq.Enrichment.When,
				How:  pq.Enrichment.How,
				Who:  pq.Enrichment.Who,
			},
			Tags:      pq.Tags,
			CreatedAt: now,
		}

		// Convert options
		if len(pq.Options) > 0 {
			q.Options = make([]models.Option, len(pq.Options))
			for i, o := range pq.Options {
				q.Options[i] = models.Option{Text: o.Text, IsCorrect: o.IsCorrect}
			}
		}

		questions = append(questions, q)
	}

	return questions, nil
}

func validateParsedQuestion(pq *ParsedQuestion) error {
	if pq.QuestionText == "" {
		return fmt.Errorf("question_text is required")
	}
	if !llm.ValidateQuestionType(pq.QuestionType) {
		return fmt.Errorf("invalid question_type: %s", pq.QuestionType)
	}
	if !llm.ValidateBloomLevel(pq.BloomLevel) {
		return fmt.Errorf("invalid bloom_level: %s", pq.BloomLevel)
	}
	if !llm.ValidateDifficulty(pq.Difficulty) {
		return fmt.Errorf("invalid difficulty: %s", pq.Difficulty)
	}
	if pq.QuestionType == "mcq" && len(pq.Options) != 4 {
		return fmt.Errorf("MCQ must have exactly 4 options, got %d", len(pq.Options))
	}
	if (pq.QuestionType == "essay" || pq.QuestionType == "short_answer") && pq.ModelAnswer == "" {
		return fmt.Errorf("essay/short_answer requires model_answer")
	}
	return nil
}

func (s *ProcessingService) failBook(ctx context.Context, bookID bson.ObjectID, origErr error) error {
	_, err := s.DB.Books().UpdateOne(ctx, bson.M{"_id": bookID}, bson.M{
		"$set": bson.M{
			"status":           models.BookStatusFailed,
			"processing_error": origErr.Error(),
			"updated_at":       time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update book status: %w (original: %v)", err, origErr)
	}
	return origErr
}

// truncateContent limits content to approximately maxTokens (estimated as len/4).
func truncateContent(content string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}
	// Try to cut at a sentence boundary
	truncated := content[:maxChars]
	lastPeriod := strings.LastIndex(truncated, ". ")
	if lastPeriod > maxChars/2 {
		return truncated[:lastPeriod+1]
	}
	return truncated
}

// GetBookStatus returns processing status info for a book.
func (s *ProcessingService) GetBookStatus(ctx context.Context, bookID bson.ObjectID) (map[string]interface{}, error) {
	var book models.Book
	err := s.DB.Books().FindOne(ctx, bson.M{"_id": bookID}).Decode(&book)
	if err != nil {
		return nil, fmt.Errorf("book not found: %w", err)
	}

	totalChapters, _ := s.DB.Chapters().CountDocuments(ctx, bson.M{"book_id": bookID})

	// Count chapters with embeddings using aggregation
	pipeline := bson.A{
		bson.M{"$match": bson.M{"book_id": bookID, "embedding": bson.M{"$exists": true, "$ne": bson.A{}}}},
		bson.M{"$count": "count"},
	}
	embeddedCursor, _ := s.DB.Chapters().Aggregate(ctx, pipeline)
	var embeddedCount int64
	if embeddedCursor != nil {
		var result []struct {
			Count int64 `bson:"count"`
		}
		if embeddedCursor.All(ctx, &result) == nil && len(result) > 0 {
			embeddedCount = result[0].Count
		}
		embeddedCursor.Close(ctx)
	}

	questionsGenerated, _ := s.DB.Questions().CountDocuments(ctx, bson.M{"book_id": bookID})

	status := map[string]interface{}{
		"book_id":             bookID.Hex(),
		"status":              book.Status,
		"processing_error":    book.ProcessingError,
		"chapters_total":      totalChapters,
		"chapters_embedded":   embeddedCount,
		"questions_generated": questionsGenerated,
		"updated_at":          book.UpdatedAt,
	}

	return status, nil
}

// Ensure json package is used (for ParsedQuestion tags)
var _ = json.Unmarshal
