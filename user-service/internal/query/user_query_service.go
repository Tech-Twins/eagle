package query

import (
	"context"
	"fmt"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/models"
	"github.com/eaglebank/user-service/internal/repository"
)

// UserQueryService reads user views from the Redis cache (with a Postgres fallback).
type UserQueryService struct {
	readRepo *repository.UserReadRepository
}

func NewUserQueryService(readRepo *repository.UserReadRepository) *UserQueryService {
	return &UserQueryService{readRepo: readRepo}
}

func (s *UserQueryService) GetUser(q cqrs.GetUserQuery) (*models.UserView, error) {
	if q.UserID != q.RequestingUserID {
		return nil, fmt.Errorf("forbidden")
	}
	ctx := context.Background()
	return s.readRepo.GetByID(ctx, q.UserID)
}
