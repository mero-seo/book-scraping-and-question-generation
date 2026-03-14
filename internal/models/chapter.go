package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Chapter struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	BookID    bson.ObjectID `bson:"book_id" json:"bookId"`
	Number    int           `bson:"number" json:"number"`
	Title     string        `bson:"title" json:"title"`
	Content   string        `bson:"content" json:"content"`
	Summary   string        `bson:"summary,omitempty" json:"summary,omitempty"`
	Topics    []string      `bson:"topics,omitempty" json:"topics,omitempty"`
	Embedding []float64     `bson:"embedding,omitempty" json:"-"`
	WordCount int           `bson:"word_count,omitempty" json:"wordCount,omitempty"`
	CreatedAt time.Time     `bson:"created_at" json:"createdAt"`
}
