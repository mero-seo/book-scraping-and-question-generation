package services

import (
	"context"
	"fmt"
	"internal/db"
	"internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// QuestionService handles question operations.
type QuestionService struct {
	DB *db.MongoDB
}

// ListQuestions returns paginated questions with optional filters.
func (s *QuestionService) ListQuestions(ctx context.Context, filter bson.M, page, limit int) ([]*models.Question, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	total, err := s.DB.Questions().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := s.DB.Questions().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find questions: %w", err)
	}
	defer cursor.Close(ctx)

	var questions []*models.Question
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, 0, fmt.Errorf("failed to decode questions: %w", err)
	}

	return questions, total, nil
}

// GetQuestion returns a single question by ID.
func (s *QuestionService) GetQuestion(ctx context.Context, id bson.ObjectID) (*models.Question, error) {
	var question models.Question
	err := s.DB.Questions().FindOne(ctx, bson.M{"_id": id}).Decode(&question)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound{Message: "question not found"}
		}
		return nil, fmt.Errorf("failed to find question: %w", err)
	}
	return &question, nil
}

// GetRandomQuestions returns random questions for practice using MongoDB $sample.
func (s *QuestionService) GetRandomQuestions(ctx context.Context, filter bson.M, count int) ([]*models.Question, error) {
	if count < 1 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	pipeline := bson.A{}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}
	pipeline = append(pipeline, bson.M{"$sample": bson.M{"size": count}})

	cursor, err := s.DB.Questions().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get random questions: %w", err)
	}
	defer cursor.Close(ctx)

	var questions []*models.Question
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, fmt.Errorf("failed to decode questions: %w", err)
	}

	return questions, nil
}

// UpdateQuestion updates a question's fields.
func (s *QuestionService) UpdateQuestion(ctx context.Context, id bson.ObjectID, updates bson.M) (*models.Question, error) {
	_, err := s.DB.Questions().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	if err != nil {
		return nil, fmt.Errorf("failed to update question: %w", err)
	}
	return s.GetQuestion(ctx, id)
}

// DeleteQuestion removes a question by ID.
func (s *QuestionService) DeleteQuestion(ctx context.Context, id bson.ObjectID) error {
	result, err := s.DB.Questions().DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound{Message: "question not found"}
	}
	return nil
}
