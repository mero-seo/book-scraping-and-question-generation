package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID               bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email            string        `bson:"email" json:"email"`
	Name             string        `bson:"name" json:"name"`
	PasswordHash     string        `bson:"password_hash" json:"-"`
	Role             string        `bson:"role" json:"role"`
	GradeLevel       string        `bson:"grade_level,omitempty" json:"gradeLevel,omitempty"`
	EducationSystem  string        `bson:"education_system,omitempty" json:"educationSystem,omitempty"`
	ExamPreparingFor string        `bson:"exam_preparing_for,omitempty" json:"examPreparingFor,omitempty"`
	AvatarURL        string        `bson:"avatar_url,omitempty" json:"avatarUrl,omitempty"`
	CreatedAt        time.Time     `bson:"created_at" json:"createdAt"`
	UpdatedAt        time.Time     `bson:"updated_at" json:"updatedAt"`
}

type UserAnswer struct {
	ID                bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID            bson.ObjectID `bson:"user_id" json:"userId"`
	QuestionID        bson.ObjectID `bson:"question_id" json:"questionId"`
	AnswerText        string        `bson:"answer_text" json:"answerText"`
	IsCorrect         *bool         `bson:"is_correct,omitempty" json:"isCorrect,omitempty"`
	SemanticScore     *float64      `bson:"semantic_score,omitempty" json:"semanticScore,omitempty"`
	KeywordScore      *float64      `bson:"keyword_score,omitempty" json:"keywordScore,omitempty"`
	CompletenessScore *float64      `bson:"completeness_score,omitempty" json:"completenessScore,omitempty"`
	OverallScore      *float64      `bson:"overall_score,omitempty" json:"overallScore,omitempty"`
	Feedback          string        `bson:"feedback,omitempty" json:"feedback,omitempty"`
	TimeTaken         int           `bson:"time_taken,omitempty" json:"timeTaken,omitempty"`
	CreatedAt         time.Time     `bson:"created_at" json:"createdAt"`
}

type AllowedSource struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	URLPattern string        `bson:"url_pattern" json:"urlPattern"`
	Name       string        `bson:"name" json:"name"`
	SourceType string        `bson:"source_type" json:"sourceType"`
	Enabled    bool          `bson:"enabled" json:"enabled"`
	AddedBy    bson.ObjectID `bson:"added_by,omitempty" json:"addedBy,omitempty"`
	Notes      string        `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt  time.Time     `bson:"created_at" json:"createdAt"`
	UpdatedAt  time.Time     `bson:"updated_at" json:"updatedAt"`
}

// Role constants
const (
	RoleStudent = "student"
	RoleAdmin   = "admin"
)
