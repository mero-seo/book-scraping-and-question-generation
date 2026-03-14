package handlers

import (
	"backend/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// BookHandler handles book endpoints.
type BookHandler struct {
	BookSvc *services.BookService
	ProcSvc *services.ProcessingService
}

// List handles GET /api/v1/books
func (h *BookHandler) List(c *gin.Context) {
	page, limit := paginationParams(c)

	filter := bson.M{}
	if s := c.Query("status"); s != "" {
		filter["status"] = s
	}
	if s := c.Query("subject"); s != "" {
		filter["subject"] = s
	}
	if s := c.Query("grade_level"); s != "" {
		filter["grade_levels"] = s
	}
	if s := c.Query("source_type"); s != "" {
		filter["source_type"] = s
	}

	sort := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	books, total, err := h.BookSvc.ListBooks(c.Request.Context(), filter, page, limit, sort, order)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, paginatedResponse(books, page, limit, total))
}

// Get handles GET /api/v1/books/:id
func (h *BookHandler) Get(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	book, err := h.BookSvc.GetBook(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, book)
}

// Create handles POST /api/v1/books
func (h *BookHandler) Create(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req services.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	book, err := h.BookSvc.CreateFromURL(c.Request.Context(), req, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, book)
}

// Upload handles POST /api/v1/books/upload
func (h *BookHandler) Upload(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required", "code": "VALIDATION_ERROR"})
		return
	}
	defer file.Close()

	// Validate file size (50 MB)
	if header.Size > 50*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds 50 MB limit", "code": "FILE_TOO_LARGE"})
		return
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "only PDF files are accepted", "code": "UNSUPPORTED_TYPE"})
		return
	}

	gradeLevels := strings.Split(c.PostForm("grade_levels"), ",")
	for i := range gradeLevels {
		gradeLevels[i] = strings.TrimSpace(gradeLevels[i])
	}

	req := services.CreateBookRequest{
		SourceType:      "pdf",
		Title:           c.PostForm("title"),
		Author:          c.PostForm("author"),
		Subject:         c.PostForm("subject"),
		GradeLevels:     gradeLevels,
		EducationSystem: c.PostForm("education_system"),
		ISBN:            c.PostForm("isbn"),
	}

	if req.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject is required", "code": "VALIDATION_ERROR"})
		return
	}

	book, err := h.BookSvc.CreateFromPDF(c.Request.Context(), file, header.Filename, req, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, book)
}

// Search handles POST /api/v1/books/search
func (h *BookHandler) Search(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	results, err := h.BookSvc.SearchBooks(c.Request.Context(), req.Query, req.Limit)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error(), "code": "SERVICE_UNAVAILABLE"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// Update handles PUT /api/v1/books/:id
func (h *BookHandler) Update(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	// Convert to bson.M
	bsonUpdates := bson.M{}
	allowedFields := map[string]string{
		"title": "title", "author": "author", "isbn": "isbn",
		"publisher": "publisher", "language": "language", "subject": "subject",
		"gradeLevels": "grade_levels", "educationSystem": "education_system",
		"coverImageUrl": "cover_image_url", "status": "status", "metadata": "metadata",
	}

	for jsonKey, bsonKey := range allowedFields {
		if v, ok := updates[jsonKey]; ok {
			bsonUpdates[bsonKey] = v
		}
	}

	book, err := h.BookSvc.UpdateBook(c.Request.Context(), id, bsonUpdates)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, book)
}

// Delete handles DELETE /api/v1/books/:id
func (h *BookHandler) Delete(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	counts, err := h.BookSvc.DeleteBook(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Book deleted successfully",
		"deleted": counts,
	})
}

// Process handles POST /api/v1/books/:id/process
func (h *BookHandler) Process(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	// Check book exists and is not already processing
	book, err := h.BookSvc.GetBook(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	if book.Status == "processing" {
		c.JSON(http.StatusConflict, gin.H{"error": "book is already processing", "code": "CONFLICT"})
		return
	}

	go func() {
		_ = h.ProcSvc.ProcessBook(c.Copy().Request.Context(), id)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Processing started",
		"book_id": id.Hex(),
		"status":  "processing",
	})
}

// Status handles GET /api/v1/books/:id/status
func (h *BookHandler) Status(c *gin.Context) {
	id, ok := parseObjectID(c, "id")
	if !ok {
		return
	}

	status, err := h.ProcSvc.GetBookStatus(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}
