package services

import (
	"context"
	"fmt"
	"internal/db"
	"internal/models"
	"internal/scoring"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AnswerService handles answer submission and scoring.
type AnswerService struct {
	DB          *db.MongoDB
	Scorer      *scoring.Scorer
	QuestionSvc *QuestionService
}

// SubmitAnswerRequest holds the fields for submitting an answer.
type SubmitAnswerRequest struct {
	QuestionID string `json:"questionId" binding:"required"`
	AnswerText string `json:"answerText" binding:"required"`
	TimeTaken  int    `json:"timeTaken"`
}

// SubmitAnswer scores an answer and stores the result.
func (s *AnswerService) SubmitAnswer(ctx context.Context, userID bson.ObjectID, req SubmitAnswerRequest) (*models.UserAnswer, error) {
	qID, err := bson.ObjectIDFromHex(req.QuestionID)
	if err != nil {
		return nil, ErrValidation{Message: "invalid question ID"}
	}

	question, err := s.QuestionSvc.GetQuestion(ctx, qID)
	if err != nil {
		return nil, err
	}

	// Score the answer
	result, err := s.Scorer.ScoreAnswer(ctx, question, req.AnswerText)
	if err != nil {
		return nil, fmt.Errorf("scoring failed: %w", err)
	}

	answer := &models.UserAnswer{
		UserID:            userID,
		QuestionID:        qID,
		AnswerText:        req.AnswerText,
		IsCorrect:         result.IsCorrect,
		SemanticScore:     result.SemanticScore,
		KeywordScore:      result.KeywordScore,
		CompletenessScore: result.CompletenessScore,
		OverallScore:      &result.OverallScore,
		Feedback:          result.Feedback,
		TimeTaken:         req.TimeTaken,
		CreatedAt:         time.Now(),
	}

	insertResult, err := s.DB.UserAnswers().InsertOne(ctx, answer)
	if err != nil {
		return nil, fmt.Errorf("failed to insert answer: %w", err)
	}
	answer.ID = insertResult.InsertedID.(bson.ObjectID)

	return answer, nil
}

// GetUserAnswers returns a user's answer history with pagination.
func (s *AnswerService) GetUserAnswers(ctx context.Context, userID bson.ObjectID, page, limit int) ([]*models.UserAnswer, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := bson.M{"user_id": userID}
	total, err := s.DB.UserAnswers().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count answers: %w", err)
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := s.DB.UserAnswers().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find answers: %w", err)
	}
	defer cursor.Close(ctx)

	var answers []*models.UserAnswer
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, 0, fmt.Errorf("failed to decode answers: %w", err)
	}

	return answers, total, nil
}

// UserStats holds aggregated statistics for a user.
type UserStats struct {
	TotalAnswered       int64              `json:"totalAnswered"`
	AverageScore        float64            `json:"averageScore"`
	ScoreByBloom        map[string]float64 `json:"scoreByBloom"`
	ScoreByDifficulty   map[string]float64 `json:"scoreByDifficulty"`
	CorrectRate         float64            `json:"correctRate"`
	AverageTimeTaken    float64            `json:"averageTimeTaken"`
}

// GetUserStats returns aggregated user performance statistics.
func (s *AnswerService) GetUserStats(ctx context.Context, userID bson.ObjectID) (*UserStats, error) {
	stats := &UserStats{
		ScoreByBloom:      make(map[string]float64),
		ScoreByDifficulty: make(map[string]float64),
	}

	// Total answered
	total, err := s.DB.UserAnswers().CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to count answers: %w", err)
	}
	stats.TotalAnswered = total

	if total == 0 {
		return stats, nil
	}

	// Average score and correct rate using aggregation
	pipeline := bson.A{
		bson.M{"$match": bson.M{"user_id": userID}},
		bson.M{"$group": bson.M{
			"_id": nil,
			"avg_score": bson.M{"$avg": "$overall_score"},
			"avg_time":  bson.M{"$avg": "$time_taken"},
			"correct_count": bson.M{"$sum": bson.M{
				"$cond": bson.A{"$is_correct", 1, 0},
			}},
			"total": bson.M{"$sum": 1},
		}},
	}

	cursor, err := s.DB.UserAnswers().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		AvgScore     *float64 `bson:"avg_score"`
		AvgTime      *float64 `bson:"avg_time"`
		CorrectCount int64    `bson:"correct_count"`
		Total        int64    `bson:"total"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	if len(results) > 0 {
		r := results[0]
		if r.AvgScore != nil {
			stats.AverageScore = *r.AvgScore
		}
		if r.AvgTime != nil {
			stats.AverageTimeTaken = *r.AvgTime
		}
		if r.Total > 0 {
			stats.CorrectRate = float64(r.CorrectCount) / float64(r.Total) * 100
		}
	}

	// Score by Bloom's level - requires joining with questions collection
	bloomPipeline := bson.A{
		bson.M{"$match": bson.M{"user_id": userID}},
		bson.M{"$lookup": bson.M{
			"from":         "questions",
			"localField":   "question_id",
			"foreignField": "_id",
			"as":           "question",
		}},
		bson.M{"$unwind": "$question"},
		bson.M{"$group": bson.M{
			"_id":       "$question.bloom_level",
			"avg_score": bson.M{"$avg": "$overall_score"},
		}},
	}

	bloomCursor, err := s.DB.UserAnswers().Aggregate(ctx, bloomPipeline)
	if err == nil {
		var bloomResults []struct {
			ID       string   `bson:"_id"`
			AvgScore *float64 `bson:"avg_score"`
		}
		if bloomCursor.All(ctx, &bloomResults) == nil {
			for _, br := range bloomResults {
				if br.AvgScore != nil {
					stats.ScoreByBloom[br.ID] = *br.AvgScore
				}
			}
		}
		bloomCursor.Close(ctx)
	}

	// Score by difficulty
	diffPipeline := bson.A{
		bson.M{"$match": bson.M{"user_id": userID}},
		bson.M{"$lookup": bson.M{
			"from":         "questions",
			"localField":   "question_id",
			"foreignField": "_id",
			"as":           "question",
		}},
		bson.M{"$unwind": "$question"},
		bson.M{"$group": bson.M{
			"_id":       "$question.difficulty",
			"avg_score": bson.M{"$avg": "$overall_score"},
		}},
	}

	diffCursor, err := s.DB.UserAnswers().Aggregate(ctx, diffPipeline)
	if err == nil {
		var diffResults []struct {
			ID       string   `bson:"_id"`
			AvgScore *float64 `bson:"avg_score"`
		}
		if diffCursor.All(ctx, &diffResults) == nil {
			for _, dr := range diffResults {
				if dr.AvgScore != nil {
					stats.ScoreByDifficulty[dr.ID] = *dr.AvgScore
				}
			}
		}
		diffCursor.Close(ctx)
	}

	return stats, nil
}
