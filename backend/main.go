package main

import (
	"context"
	"log"
	"os"

	"backend/handlers"
	"backend/middleware"
	"backend/services"
	"internal/db"
	"internal/llm"
	"internal/scoring"
	"internal/storage"
	"internal/vectordb"
	"scraper"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file — try parent directory (project root) first, then current dir
	_ = godotenv.Load("../.env")
	_ = godotenv.Load()

	ctx := context.Background()

	// Connect to MongoDB
	mongoDB, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Disconnect(ctx)
	log.Println("Connected to MongoDB")

	// Initialize LLM client (OpenRouter primary, Ollama cloud fallback)
	llmClient := llm.NewFallbackClient()
	log.Printf("LLM client initialized: %s", llmClient.Name())

	// Initialize Pinecone (vector embeddings + search)
	var pinecone *vectordb.PineconeClient
	if os.Getenv("PINECONE_API_KEY") != "" {
		pinecone, err = vectordb.NewPineconeClient()
		if err != nil {
			log.Printf("Pinecone not configured: %v", err)
		} else {
			log.Println("Pinecone initialized")
		}
	} else {
		log.Println("Pinecone not configured (PINECONE_API_KEY not set)")
	}

	// Initialize scorer
	scorer := scoring.NewScorer(llmClient)

	// Initialize R2 storage (optional - for PDF uploads)
	var r2Client *storage.R2Client
	if os.Getenv("CLOUDFLARE_R2_ACCOUNT_ID") != "" {
		r2Client, err = storage.NewR2Client(ctx)
		if err != nil {
			log.Printf("R2 storage not configured: %v", err)
		} else {
			log.Println("R2 storage initialized")
		}
	}

	// Initialize scraper
	scr := scraper.New(scraper.Config{})

	// Initialize services
	procSvc := &services.ProcessingService{
		DB:       mongoDB,
		LLM:      llmClient,
		Pinecone: pinecone,
	}

	bookSvc := &services.BookService{
		DB:      mongoDB,
		Scraper: scr,
		Storage: r2Client,
		ProcSvc: procSvc,
	}

	userSvc := &services.UserService{DB: mongoDB}
	questionSvc := &services.QuestionService{DB: mongoDB}
	answerSvc := &services.AnswerService{
		DB:          mongoDB,
		Scorer:      scorer,
		QuestionSvc: questionSvc,
	}

	// Initialize handlers
	authHandler := &handlers.AuthHandler{UserSvc: userSvc}
	bookHandler := &handlers.BookHandler{BookSvc: bookSvc, ProcSvc: procSvc}
	chapterHandler := &handlers.ChapterHandler{DB: mongoDB, Pinecone: pinecone}
	questionHandler := &handlers.QuestionHandler{QuestionSvc: questionSvc, ProcSvc: procSvc}
	answerHandler := &handlers.AnswerHandler{AnswerSvc: answerSvc}
	adminHandler := &handlers.AdminHandler{DB: mongoDB}

	// Setup router
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Health check with provider status
	r.GET("/api/v1/health", func(c *gin.Context) {
		pineconeOk := pinecone != nil && pinecone.Available(c.Request.Context())
		c.JSON(200, gin.H{
			"status":   "ok",
			"llm":      llmClient.Available(c.Request.Context()),
			"pinecone": pineconeOk,
			"database": true,
		})
	})

	// API v1
	v1 := r.Group("/api/v1")

	// Public routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Authenticated routes
	authenticated := v1.Group("")
	authenticated.Use(middleware.AuthMiddleware())
	{
		// Books
		authenticated.GET("/books", bookHandler.List)
		authenticated.GET("/books/:id", bookHandler.Get)
		authenticated.GET("/books/:id/status", bookHandler.Status)
		authenticated.POST("/books/search", bookHandler.Search)

		// Chapters
		authenticated.GET("/books/:id/chapters", chapterHandler.List)
		authenticated.GET("/books/:id/chapters/:chapterID", chapterHandler.Get)
		authenticated.POST("/books/:id/chapters/search", chapterHandler.SemanticSearch)

		// Questions
		authenticated.GET("/questions", questionHandler.List)
		authenticated.GET("/questions/:id", questionHandler.Get)
		authenticated.GET("/questions/random", questionHandler.Random)

		// Answers
		authenticated.POST("/answers", answerHandler.Submit)
		authenticated.GET("/answers", answerHandler.History)
		authenticated.GET("/answers/stats", answerHandler.Stats)
	}

	// Admin routes
	admin := v1.Group("")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		// Books management
		admin.POST("/books", bookHandler.Create)
		admin.POST("/books/upload", bookHandler.Upload)
		admin.PUT("/books/:id", bookHandler.Update)
		admin.DELETE("/books/:id", bookHandler.Delete)
		admin.POST("/books/:id/process", bookHandler.Process)
		admin.POST("/books/:id/generate", questionHandler.Generate)

		// Questions management
		admin.PUT("/questions/:id", questionHandler.Update)
		admin.DELETE("/questions/:id", questionHandler.Delete)

		// Admin panel
		admin.GET("/admin/dashboard", adminHandler.Dashboard)
		admin.GET("/admin/sources", adminHandler.ListSources)
		admin.POST("/admin/sources", adminHandler.CreateSource)
		admin.PUT("/admin/sources/:id", adminHandler.UpdateSource)
		admin.DELETE("/admin/sources/:id", adminHandler.DeleteSource)
		admin.GET("/admin/users", adminHandler.ListUsers)
		admin.PUT("/admin/users/:id/role", adminHandler.UpdateUserRole)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
