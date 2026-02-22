package domain

import "go.mongodb.org/mongo-driver/v2/bson"

type Book struct {
	ID	bson.ObjectId	`bson:"_id,omitempty"`
	Title	string	`bson:"title"`
	Author	string	`bson:"author"`
	Summary	string	`bson:"summary"`
	TOC	[]Chapter	`bson:"toc"`
	Vector	[]float32	`bson:"vector_embedding"`
	CreatedAt	int64	`bson:"created_at_date"`
	UpdatedAt	int64	`bson:"updated_at_date"`
	DeletedAt	int64	`bson:"deleted_at_date"`
}

type Chapter struct {
	Title	string	`bson:"title"`
	Summary	string	`bson:"summary"`
	PageFrom	int	`bson:"page_from"`
	PageTo	int	`bson:"page_to"`
	TotalPage	int	`bson:"total_page"`
}
