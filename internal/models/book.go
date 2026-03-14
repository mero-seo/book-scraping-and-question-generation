package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Book struct {
	ID              bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	Title           string            `bson:"title" json:"title"`
	Author          string            `bson:"author" json:"author"`
	ISBN            string            `bson:"isbn,omitempty" json:"isbn,omitempty"`
	Publisher       string            `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Language        string            `bson:"language,omitempty" json:"language,omitempty"`
	Subject         string            `bson:"subject" json:"subject"`
	GradeLevels     []string          `bson:"grade_levels" json:"gradeLevels"`
	EducationSystem string            `bson:"education_system,omitempty" json:"educationSystem,omitempty"`
	SourceType      string            `bson:"source_type" json:"sourceType"`
	SourceURL       string            `bson:"source_url,omitempty" json:"sourceUrl,omitempty"`
	PDFURL          string            `bson:"pdf_url,omitempty" json:"pdfUrl,omitempty"`
	CoverImageURL   string            `bson:"cover_image_url,omitempty" json:"coverImageUrl,omitempty"`
	Status          string            `bson:"status" json:"status"`
	ProcessingError string            `bson:"processing_error,omitempty" json:"processingError,omitempty"`
	TOC             []TOCEntry        `bson:"toc,omitempty" json:"toc,omitempty"`
	Metadata        map[string]string `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedBy       bson.ObjectID     `bson:"created_by,omitempty" json:"createdBy,omitempty"`
	CreatedAt       time.Time         `bson:"created_at" json:"createdAt"`
	UpdatedAt       time.Time         `bson:"updated_at" json:"updatedAt"`
}

type TOCEntry struct {
	Number int    `bson:"number" json:"number"`
	Title  string `bson:"title" json:"title"`
	Page   int    `bson:"page,omitempty" json:"page,omitempty"`
	Depth  int    `bson:"depth" json:"depth"`
}

// Book status constants
const (
	BookStatusPending    = "pending"
	BookStatusProcessing = "processing"
	BookStatusReady      = "ready"
	BookStatusFailed     = "failed"
)

// Source type constants
const (
	SourceTypePDF    = "pdf"
	SourceTypeURL    = "url"
	SourceTypeSearch = "search"
)
