package handlers

import (
	"backend/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AnswerHandler handles answer submission and history endpoints.
type AnswerHandler struct {
	AnswerSvc *services.AnswerService
}

// Submit handles POST /api/v1/answers
func (h *AnswerHandler) Submit(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req services.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	answer, err := h.AnswerSvc.SubmitAnswer(c.Request.Context(), userID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, answer)
}

// History handles GET /api/v1/answers
func (h *AnswerHandler) History(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	page, limit := paginationParams(c)

	answers, total, err := h.AnswerSvc.GetUserAnswers(c.Request.Context(), userID, page, limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, paginatedResponse(answers, page, limit, total))
}

// Stats handles GET /api/v1/answers/stats
func (h *AnswerHandler) Stats(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	stats, err := h.AnswerSvc.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
