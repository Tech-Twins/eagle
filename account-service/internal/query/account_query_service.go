package query

import (
	"context"

	"github.com/eaglebank/account-service/internal/repository"
	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/models"
)

type AccountQueryService struct {
	readRepo *repository.AccountReadRepository
}

func NewAccountQueryService(readRepo *repository.AccountReadRepository) *AccountQueryService {
	return &AccountQueryService{readRepo: readRepo}
}

// GetAccount fetches a single account view and enforces ownership.
func (s *AccountQueryService) GetAccount(q cqrs.GetAccountQuery) (*models.AccountView, error) {
	ctx := context.Background()
	view, err := s.readRepo.GetByAccountNumber(ctx, q.AccountNumber)
	if err != nil {
		return nil, err
	}

	// Ownership check: the AccountView carries UserID (json:"-") for this purpose.
	if view.UserID != q.RequestingUserID {
		return nil, &forbiddenError{}
	}

	return view, nil
}

func (s *AccountQueryService) ListAccounts(q cqrs.ListAccountsQuery) ([]models.AccountView, error) {
	ctx := context.Background()
	return s.readRepo.ListByUserID(ctx, q.UserID)
}

// forbiddenError signals that the requesting user does not own the resource.
type forbiddenError struct{}

func (e *forbiddenError) Error() string { return "forbidden" }
