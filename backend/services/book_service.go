package services

import (
	"context"
	"fmt"
	"internal/adapter"
	"internal/db"
	"internal/models"
	"internal/storage"
	"io"
	"log"
	"scraper"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// BookService handles book CRUD operations.
type BookService struct {
	DB      *db.MongoDB
	Scraper *scraper.Scraper
	Storage *storage.R2Client
	ProcSvc *ProcessingService
}

// CreateBookRequest holds fields for creating a book from URL/search.
type CreateBookRequest struct {
	SourceURL       string            `json:"sourceUrl"`
	SourceType      string            `json:"sourceType" binding:"required"`
	Title           string            `json:"title"`
	Author          string            `json:"author"`
	Subject         string            `json:"subject" binding:"required"`
	GradeLevels     []string          `json:"gradeLevels" binding:"required"`
	EducationSystem string            `json:"educationSystem"`
	ISBN            string            `json:"isbn"`
	Publisher       string            `json:"publisher"`
	CoverImageURL   string            `json:"coverImageUrl"`
	Metadata        map[string]string `json:"metadata"`
}

// CreateFromURL scrapes a URL and creates a book with its chapters.
func (s *BookService) CreateFromURL(ctx context.Context, req CreateBookRequest, userID bson.ObjectID) (*models.Book, error) {
	if req.SourceURL == "" {
		return nil, ErrValidation{Message: "source_url is required for URL source type"}
	}

	scraped, err := s.Scraper.ScrapeURL(ctx, req.SourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape URL: %w", err)
	}

	book, chapters := adapter.ConvertBook(scraped, userID)

	// Apply overrides from request
	if req.Title != "" {
		book.Title = req.Title
	}
	if req.Author != "" {
		book.Author = req.Author
	}
	book.Subject = req.Subject
	book.GradeLevels = req.GradeLevels
	book.EducationSystem = req.EducationSystem
	if req.ISBN != "" {
		book.ISBN = req.ISBN
	}
	if req.Publisher != "" {
		book.Publisher = req.Publisher
	}
	if req.CoverImageURL != "" {
		book.CoverImageURL = req.CoverImageURL
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			book.Metadata[k] = v
		}
	}
	book.SourceType = models.SourceTypeURL
	book.SourceURL = req.SourceURL

	// Insert book
	result, err := s.DB.Books().InsertOne(ctx, book)
	if err != nil {
		return nil, fmt.Errorf("failed to insert book: %w", err)
	}
	book.ID = result.InsertedID.(bson.ObjectID)

	// Insert chapters
	if len(chapters) > 0 {
		docs := make([]interface{}, len(chapters))
		for i, ch := range chapters {
			ch.BookID = book.ID
			docs[i] = ch
		}
		_, err = s.DB.Chapters().InsertMany(ctx, docs)
		if err != nil {
			return nil, fmt.Errorf("failed to insert chapters: %w", err)
		}
	}

	// Start processing in background
	go func() {
		if err := s.ProcSvc.ProcessBook(context.Background(), book.ID); err != nil {
			log.Printf("Background processing failed for book %s: %v", book.ID.Hex(), err)
		}
	}()

	return book, nil
}

// CreateFromPDF uploads a PDF to R2, parses it, and creates a book.
func (s *BookService) CreateFromPDF(ctx context.Context, file io.ReadSeeker, filename string, req CreateBookRequest, userID bson.ObjectID) (*models.Book, error) {
	// Upload PDF to R2
	key := fmt.Sprintf("books/%s/%s", bson.NewObjectID().Hex(), filename)
	pdfURL, err := s.Storage.Upload(ctx, key, file, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to upload PDF: %w", err)
	}

	// Reset reader for parsing
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to reset file reader: %w", err)
	}

	scraped, err := s.Scraper.ParsePDF(ctx, file, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PDF: %w", err)
	}

	book, chapters := adapter.ConvertBook(scraped, userID)

	if req.Title != "" {
		book.Title = req.Title
	}
	if req.Author != "" {
		book.Author = req.Author
	}
	book.Subject = req.Subject
	book.GradeLevels = req.GradeLevels
	book.EducationSystem = req.EducationSystem
	if req.ISBN != "" {
		book.ISBN = req.ISBN
	}
	book.SourceType = models.SourceTypePDF
	book.PDFURL = pdfURL
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			book.Metadata[k] = v
		}
	}

	result, err := s.DB.Books().InsertOne(ctx, book)
	if err != nil {
		return nil, fmt.Errorf("failed to insert book: %w", err)
	}
	book.ID = result.InsertedID.(bson.ObjectID)

	if len(chapters) > 0 {
		docs := make([]interface{}, len(chapters))
		for i, ch := range chapters {
			ch.BookID = book.ID
			docs[i] = ch
		}
		_, err = s.DB.Chapters().InsertMany(ctx, docs)
		if err != nil {
			return nil, fmt.Errorf("failed to insert chapters: %w", err)
		}
	}

	go func() {
		if err := s.ProcSvc.ProcessBook(context.Background(), book.ID); err != nil {
			log.Printf("Background processing failed for book %s: %v", book.ID.Hex(), err)
		}
	}()

	return book, nil
}

// SearchBooks searches external APIs for books.
func (s *BookService) SearchBooks(ctx context.Context, query string, limit int) ([]scraper.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	results, err := s.Scraper.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("book search failed: %w", err)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// ListBooks returns paginated books with optional filters.
func (s *BookService) ListBooks(ctx context.Context, filter bson.M, page, limit int, sort string, order string) ([]*models.Book, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if sort == "" {
		sort = "created_at"
	}

	sortOrder := -1
	if order == "asc" {
		sortOrder = 1
	}

	total, err := s.DB.Books().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count books: %w", err)
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: sort, Value: sortOrder}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := s.DB.Books().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find books: %w", err)
	}
	defer cursor.Close(ctx)

	var books []*models.Book
	if err := cursor.All(ctx, &books); err != nil {
		return nil, 0, fmt.Errorf("failed to decode books: %w", err)
	}

	return books, total, nil
}

// GetBook returns a single book by ID.
func (s *BookService) GetBook(ctx context.Context, id bson.ObjectID) (*models.Book, error) {
	var book models.Book
	err := s.DB.Books().FindOne(ctx, bson.M{"_id": id}).Decode(&book)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound{Message: "book not found"}
		}
		return nil, fmt.Errorf("failed to find book: %w", err)
	}
	return &book, nil
}

// UpdateBook updates a book's metadata.
func (s *BookService) UpdateBook(ctx context.Context, id bson.ObjectID, updates bson.M) (*models.Book, error) {
	updates["updated_at"] = time.Now()
	_, err := s.DB.Books().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	if err != nil {
		return nil, fmt.Errorf("failed to update book: %w", err)
	}
	return s.GetBook(ctx, id)
}

// DeleteBook removes a book and all associated data.
func (s *BookService) DeleteBook(ctx context.Context, id bson.ObjectID) (map[string]int64, error) {
	book, err := s.GetBook(ctx, id)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)

	// Collect question IDs for cascading delete of answers
	cursor, _ := s.DB.Questions().Find(ctx, bson.M{"book_id": id})
	var qIDs []bson.ObjectID
	if cursor != nil {
		for cursor.Next(ctx) {
			var q models.Question
			if err := cursor.Decode(&q); err == nil {
				qIDs = append(qIDs, q.ID)
			}
		}
		cursor.Close(ctx)
	}

	if len(qIDs) > 0 {
		result, _ := s.DB.UserAnswers().DeleteMany(ctx, bson.M{"question_id": bson.M{"$in": qIDs}})
		if result != nil {
			counts["user_answers"] = result.DeletedCount
		}
	}

	result, _ := s.DB.Questions().DeleteMany(ctx, bson.M{"book_id": id})
	if result != nil {
		counts["questions"] = result.DeletedCount
	}

	result, _ = s.DB.Chapters().DeleteMany(ctx, bson.M{"book_id": id})
	if result != nil {
		counts["chapters"] = result.DeletedCount
	}

	_, err = s.DB.Books().DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return counts, fmt.Errorf("failed to delete book: %w", err)
	}
	counts["book"] = 1

	if book.PDFURL != "" && s.Storage != nil {
		_ = s.Storage.Delete(ctx, book.PDFURL)
	}

	return counts, nil
}
