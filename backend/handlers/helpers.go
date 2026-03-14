package handlers

import (
	"backend/services"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// handleServiceError maps service error types to HTTP responses.
func handleServiceError(c *gin.Context, err error) {
	var notFound services.ErrNotFound
	if errors.As(err, &notFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": notFound.Message, "code": "NOT_FOUND"})
		return
	}

	var conflict services.ErrConflict
	if errors.As(err, &conflict) {
		c.JSON(http.StatusConflict, gin.H{"error": conflict.Message, "code": "CONFLICT"})
		return
	}

	var unauthorized services.ErrUnauthorized
	if errors.As(err, &unauthorized) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": unauthorized.Message, "code": "UNAUTHORIZED"})
		return
	}

	var validation services.ErrValidation
	if errors.As(err, &validation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": validation.Message, "code": "VALIDATION_ERROR"})
		return
	}

	var forbidden services.ErrForbidden
	if errors.As(err, &forbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": forbidden.Message, "code": "FORBIDDEN"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
}

// parseObjectID parses a hex string into a bson.ObjectID.
func parseObjectID(c *gin.Context, param string) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param(param))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID format", "code": "BAD_REQUEST"})
		return bson.ObjectID{}, false
	}
	return id, true
}

// getUserID extracts the user ID from the gin context (set by auth middleware).
func getUserID(c *gin.Context) (bson.ObjectID, bool) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated", "code": "UNAUTHORIZED"})
		return bson.ObjectID{}, false
	}
	id, err := bson.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID in token", "code": "INTERNAL_ERROR"})
		return bson.ObjectID{}, false
	}
	return id, true
}

// paginationParams extracts page and limit from query params.
func paginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

// paginatedResponse returns a standard paginated response.
func paginatedResponse(data interface{}, page, limit int, total int64) gin.H {
	totalPages := int64(math.Ceil(float64(total) / float64(limit)))
	return gin.H{
		"data": data,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}
}
