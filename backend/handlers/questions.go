package handlers

import (
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// QuestionHandler handles question endpoints.
type QuestionHandler struct {
	QuestionSvc *services.QuestionService
	ProcSvc     *services.ProcessingService
}

// List handles GET /api/v1/questions
func (h *QuestionHandler) List(c *gin.Context) {
	page, limit := paginationParams(c)

	filter := bson.M{}
	if s := c.Query("book_id"); s != "" {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			filter["book_id"] = id
		}
	}
	if s := c.Query("chapter_id"); s != "" {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			filter["chapter_id"] = id
		}
	}
	if s := c.Query("question_type"); s != "" {
		filter["question_type"] = s
	}
	if s := c.Query("bloom_level"); s != "" {
		filter["bloom_level"] = s
	}
	if s := c.Query("difficulty"); s != "" {
		filter["difficulty"] = s
	}

	questions, total, err := h.QuestionSvc.ListQuestions(c.Request.Context(), filter, page, limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, paginatedResponse(questions, page, limit, total))
}

// Get handles GET /api/v1/questions/:id
func (h *QuestionHandler) Get(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	question, err := h.QuestionSvc.GetQuestion(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, question)
}

// Random handles GET /api/v1/questions/random
func (h *QuestionHandler) Random(c *gin.Context) {
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))

	filter := bson.M{}
	if s := c.Query("book_id"); s != "" {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			filter["book_id"] = id
		}
	}
	if s := c.Query("chapter_id"); s != "" {
		if id, err := bson.ObjectIDFromHex(s); err == nil {
			filter["chapter_id"] = id
		}
	}
	if s := c.Query("question_type"); s != "" {
		filter["question_type"] = s
	}
	if s := c.Query("bloom_level"); s != "" {
		filter["bloom_level"] = s
	}
	if s := c.Query("difficulty"); s != "" {
		filter["difficulty"] = s
	}

	questions, err := h.QuestionSvc.GetRandomQuestions(c.Request.Context(), filter, count)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": questions})
}

// Generate handles POST /api/v1/books/:id/generate
func (h *QuestionHandler) Generate(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	go func() {
		_ = h.ProcSvc.ProcessBook(c.Copy().Request.Context(), id)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Question generation started",
		"book_id": id.Hex(),
	})
}

// Update handles PUT /api/v1/questions/:id
func (h *QuestionHandler) Update(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	bsonUpdates := bson.M{}
	allowedFields := map[string]string{
		"questionText": "question_text", "questionType": "question_type",
		"difficulty": "difficulty", "bloomLevel": "bloom_level",
		"options": "options", "correctAnswer": "correct_answer",
		"modelAnswer": "model_answer", "keyPoints": "key_points",
		"explanation": "explanation", "tags": "tags",
	}
	for jsonKey, bsonKey := range allowedFields {
		if v, ok := updates[jsonKey]; ok {
			bsonUpdates[bsonKey] = v
		}
	}

	question, err := h.QuestionSvc.UpdateQuestion(c.Request.Context(), id, bsonUpdates)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, question)
}

// Delete handles DELETE /api/v1/questions/:id
func (h *QuestionHandler) Delete(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	if err := h.QuestionSvc.DeleteQuestion(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question deleted successfully"})
}
