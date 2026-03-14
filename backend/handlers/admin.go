package handlers

import (
	"internal/db"
	"internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	DB *db.MongoDB
}

// ListSources handles GET /api/v1/admin/sources
func (h *AdminHandler) ListSources(c *gin.Context) {
	page, limit := paginationParams(c)

	total, err := h.DB.AllowedSources().CountDocuments(c.Request.Context(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := h.DB.AllowedSources().Find(c.Request.Context(), bson.M{}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var sources []*models.AllowedSource
	if err := cursor.All(c.Request.Context(), &sources); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, paginatedResponse(sources, page, limit, total))
}

// CreateSource handles POST /api/v1/admin/sources
func (h *AdminHandler) CreateSource(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req struct {
		URLPattern string `json:"urlPattern" binding:"required"`
		Name       string `json:"name" binding:"required"`
		SourceType string `json:"sourceType" binding:"required"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	now := time.Now()
	source := &models.AllowedSource{
		URLPattern: req.URLPattern,
		Name:       req.Name,
		SourceType: req.SourceType,
		Enabled:    true,
		AddedBy:    userID,
		Notes:      req.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	result, err := h.DB.AllowedSources().InsertOne(c.Request.Context(), source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}
	source.ID = result.InsertedID.(bson.ObjectID)

	c.JSON(http.StatusCreated, source)
}

// UpdateSource handles PUT /api/v1/admin/sources/:id
func (h *AdminHandler) UpdateSource(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	bsonUpdates := bson.M{"updated_at": time.Now()}
	allowedFields := map[string]string{
		"urlPattern": "url_pattern", "name": "name",
		"sourceType": "source_type", "enabled": "enabled", "notes": "notes",
	}
	for jsonKey, bsonKey := range allowedFields {
		if v, ok := updates[jsonKey]; ok {
			bsonUpdates[bsonKey] = v
		}
	}

	_, err := h.DB.AllowedSources().UpdateOne(c.Request.Context(), bson.M{"_id": id}, bson.M{"$set": bsonUpdates})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	var source models.AllowedSource
	if err := h.DB.AllowedSources().FindOne(c.Request.Context(), bson.M{"_id": id}).Decode(&source); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source not found", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, source)
}

// DeleteSource handles DELETE /api/v1/admin/sources/:id
func (h *AdminHandler) DeleteSource(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	result, err := h.DB.AllowedSources().DeleteOne(c.Request.Context(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "source not found", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Source deleted successfully"})
}

// Dashboard handles GET /api/v1/admin/dashboard
func (h *AdminHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()

	totalBooks, _ := h.DB.Books().CountDocuments(ctx, bson.M{})
	totalChapters, _ := h.DB.Chapters().CountDocuments(ctx, bson.M{})
	totalQuestions, _ := h.DB.Questions().CountDocuments(ctx, bson.M{})
	totalUsers, _ := h.DB.Users().CountDocuments(ctx, bson.M{})
	totalAnswers, _ := h.DB.UserAnswers().CountDocuments(ctx, bson.M{})

	readyBooks, _ := h.DB.Books().CountDocuments(ctx, bson.M{"status": "ready"})
	processingBooks, _ := h.DB.Books().CountDocuments(ctx, bson.M{"status": "processing"})
	failedBooks, _ := h.DB.Books().CountDocuments(ctx, bson.M{"status": "failed"})

	// Average score
	var avgScore float64
	avgPipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":       nil,
			"avg_score": bson.M{"$avg": "$overall_score"},
		}},
	}
	avgCursor, err := h.DB.UserAnswers().Aggregate(ctx, avgPipeline)
	if err == nil {
		var avgResults []struct {
			AvgScore *float64 `bson:"avg_score"`
		}
		if avgCursor.All(ctx, &avgResults) == nil && len(avgResults) > 0 && avgResults[0].AvgScore != nil {
			avgScore = *avgResults[0].AvgScore
		}
		avgCursor.Close(ctx)
	}

	c.JSON(http.StatusOK, gin.H{
		"books": gin.H{
			"total":      totalBooks,
			"ready":      readyBooks,
			"processing": processingBooks,
			"failed":     failedBooks,
		},
		"chapters":  totalChapters,
		"questions": totalQuestions,
		"users":     totalUsers,
		"answers": gin.H{
			"total":        totalAnswers,
			"averageScore": avgScore,
		},
	})
}

// ListUsers handles GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, limit := paginationParams(c)

	total, err := h.DB.Users().CountDocuments(c.Request.Context(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit)).
		SetProjection(bson.M{"password_hash": 0})

	cursor, err := h.DB.Users().Find(c.Request.Context(), bson.M{}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var users []bson.M
	if err := cursor.All(c.Request.Context(), &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, paginatedResponse(users, page, limit, total))
}

// UpdateUserRole handles PUT /api/v1/admin/users/:id/role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	if req.Role != models.RoleStudent && req.Role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'student' or 'admin'", "code": "VALIDATION_ERROR"})
		return
	}

	result, err := h.DB.Users().UpdateOne(c.Request.Context(), bson.M{"_id": id}, bson.M{
		"$set": bson.M{"role": req.Role, "updated_at": time.Now()},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User role updated", "role": req.Role})
}

// Ensure mongo package is referenced
var _ = mongo.ErrNoDocuments
