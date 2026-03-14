package handlers

import (
	"internal/db"
	"internal/models"
	"internal/vectordb"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ChapterHandler handles chapter endpoints.
type ChapterHandler struct {
	DB       *db.MongoDB
	Pinecone *vectordb.PineconeClient
}

// List handles GET /api/v1/books/:id/chapters
func (h *ChapterHandler) List(c *gin.Context) {
	bookID, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	page, limit := paginationParams(c)
	includeContent, _ := strconv.ParseBool(c.DefaultQuery("include_content", "false"))

	filter := bson.M{"book_id": bookID}
	total, err := h.DB.Chapters().CountDocuments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	skip := int64((page - 1) * limit)

	projection := bson.M{"embedding": 0}
	if !includeContent {
		projection["content"] = 0
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "number", Value: 1}}).
		SetSkip(skip).
		SetLimit(int64(limit)).
		SetProjection(projection)

	cursor, err := h.DB.Chapters().Find(c.Request.Context(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var chapters []*models.Chapter
	if err := cursor.All(c.Request.Context(), &chapters); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, paginatedResponse(chapters, page, limit, total))
}

// Get handles GET /api/v1/books/:id/chapters/:chapterID
func (h *ChapterHandler) Get(c *gin.Context) {
	chapterID, ok := parseObjectID(c, "chapterID")
	if !ok {
		return
	}

	var chapter models.Chapter
	opts := options.FindOne().SetProjection(bson.M{"embedding": 0})
	err := h.DB.Chapters().FindOne(c.Request.Context(), bson.M{"_id": chapterID}, opts).Decode(&chapter)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found", "code": "NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, chapter)
}

// SemanticSearch handles POST /api/v1/books/:id/chapters/search
func (h *ChapterHandler) SemanticSearch(c *gin.Context) {
	bookID, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 5
	}

	if h.Pinecone == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector search not configured", "code": "SERVICE_UNAVAILABLE"})
		return
	}

	// Embed the query (using "query" input type for better search relevance)
	queryVec, err := h.Pinecone.EmbedQuery(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "embedding service unavailable", "code": "SERVICE_UNAVAILABLE"})
		return
	}

	// Search Pinecone for similar chapters
	matches, err := h.Pinecone.SearchSimilar(c.Request.Context(), queryVec, bookID.Hex(), req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "INTERNAL_ERROR"})
		return
	}

	// Fetch full chapter data from MongoDB for matched IDs
	response := make([]gin.H, 0, len(matches))
	for _, m := range matches {
		chID, err := bson.ObjectIDFromHex(m.ID)
		if err != nil {
			continue
		}
		var ch models.Chapter
		findOpts := options.FindOne().SetProjection(bson.M{"embedding": 0})
		if err := h.DB.Chapters().FindOne(c.Request.Context(), bson.M{"_id": chID}, findOpts).Decode(&ch); err != nil {
			continue
		}
		response = append(response, gin.H{
			"chapter": ch,
			"score":   m.Score,
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": response})
}
