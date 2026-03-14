package services

import (
	"context"
	"fmt"
	"internal/db"
	"internal/models"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user registration, authentication, and profile management.
type UserService struct {
	DB *db.MongoDB
}

// RegisterRequest holds the fields for registering a new user.
type RegisterRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Name             string `json:"name" binding:"required,min=2,max=100"`
	Password         string `json:"password" binding:"required,min=8"`
	GradeLevel       string `json:"gradeLevel"`
	EducationSystem  string `json:"educationSystem"`
	ExamPreparingFor string `json:"examPreparingFor"`
}

// LoginRequest holds the fields for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register creates a new user account.
func (s *UserService) Register(ctx context.Context, req RegisterRequest) (*models.User, error) {
	// Check if email already exists
	count, err := s.DB.Users().CountDocuments(ctx, bson.M{"email": req.Email})
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if count > 0 {
		return nil, ErrConflict{Message: "email already registered"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		Email:            req.Email,
		Name:             req.Name,
		PasswordHash:     string(hash),
		Role:             models.RoleStudent,
		GradeLevel:       req.GradeLevel,
		EducationSystem:  req.EducationSystem,
		ExamPreparingFor: req.ExamPreparingFor,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	result, err := s.DB.Users().InsertOne(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	user.ID = result.InsertedID.(bson.ObjectID)
	return user, nil
}

// Login authenticates a user and returns the user record.
func (s *UserService) Login(ctx context.Context, req LoginRequest) (*models.User, error) {
	var user models.User
	err := s.DB.Users().FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUnauthorized{Message: "invalid email or password"}
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrUnauthorized{Message: "invalid email or password"}
	}

	return &user, nil
}

// GetByID returns a user by their ID.
func (s *UserService) GetByID(ctx context.Context, id bson.ObjectID) (*models.User, error) {
	var user models.User
	err := s.DB.Users().FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound{Message: "user not found"}
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}
