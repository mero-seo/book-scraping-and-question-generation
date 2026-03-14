package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Question struct {
	ID                 bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	BookID             bson.ObjectID   `bson:"book_id" json:"bookId"`
	ChapterID          bson.ObjectID   `bson:"chapter_id" json:"chapterId"`
	Topic              string          `bson:"topic" json:"topic"`
	QuestionText       string          `bson:"question_text" json:"questionText"`
	QuestionType       string          `bson:"question_type" json:"questionType"`
	Difficulty         string          `bson:"difficulty" json:"difficulty"`
	BloomLevel         string          `bson:"bloom_level" json:"bloomLevel"`
	GradeLevel         string          `bson:"grade_level" json:"gradeLevel"`
	ExamType           string          `bson:"exam_type,omitempty" json:"examType,omitempty"`
	Options            []Option        `bson:"options,omitempty" json:"options,omitempty"`
	CorrectAnswer      string          `bson:"correct_answer,omitempty" json:"correctAnswer,omitempty"`
	ModelAnswer        string          `bson:"model_answer,omitempty" json:"modelAnswer,omitempty"`
	KeyPoints          []string        `bson:"key_points,omitempty" json:"keyPoints,omitempty"`
	Explanation        string          `bson:"explanation,omitempty" json:"explanation,omitempty"`
	Enrichment         Enrichment      `bson:"enrichment,omitempty" json:"enrichment,omitempty"`
	RelatedQuestionIDs []bson.ObjectID `bson:"related_question_ids,omitempty" json:"relatedQuestionIds,omitempty"`
	Tags               []string        `bson:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt          time.Time       `bson:"created_at" json:"createdAt"`
}

type Option struct {
	Text      string `bson:"text" json:"text"`
	IsCorrect bool   `bson:"is_correct" json:"isCorrect"`
}

type Enrichment struct {
	What string `bson:"what" json:"what"`
	When string `bson:"when" json:"when"`
	How  string `bson:"how" json:"how"`
	Who  string `bson:"who" json:"who"`
}

// Question type constants
const (
	QuestionTypeMCQ                = "mcq"
	QuestionTypeEssay              = "essay"
	QuestionTypeFillBlank          = "fill_blank"
	QuestionTypeTrueFalse          = "true_false"
	QuestionTypeShortAnswer        = "short_answer"
	QuestionTypeMatch              = "match"
	QuestionTypeAssertionReasoning = "assertion_reasoning"
)

// Bloom's Taxonomy level constants
const (
	BloomRemember   = "remember"
	BloomUnderstand = "understand"
	BloomApply      = "apply"
	BloomAnalyze    = "analyze"
	BloomEvaluate   = "evaluate"
	BloomCreate     = "create"
)

// Difficulty constants
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)
