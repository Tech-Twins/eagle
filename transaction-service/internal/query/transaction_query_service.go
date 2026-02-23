package query

import (
	"context"
	"fmt"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/models"
	"github.com/eaglebank/transaction-service/internal/repository"
)

// TransactionQueryService serves transaction reads. Ownership is always checked
// against the account cache before returning results.
type TransactionQueryService struct {
	readRepo    *repository.TransactionReadRepository
	accountRepo *repository.AccountRepository
}

func NewTransactionQueryService(readRepo *repository.TransactionReadRepository, accountRepo *repository.AccountRepository) *TransactionQueryService {
	return &TransactionQueryService{readRepo: readRepo, accountRepo: accountRepo}
}

func (s *TransactionQueryService) GetTransaction(q cqrs.GetTransactionQuery) (*models.TransactionView, error) {
	ctx := context.Background()
	account, err := s.accountRepo.GetAccount(ctx, q.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("account not found")
	}
	if account.UserID != q.UserID {
		return nil, fmt.Errorf("forbidden")
	}
	view, err := s.readRepo.GetByID(ctx, q.TransactionID, q.AccountNumber)
	if err != nil {
		return nil, err
	}
	return view, nil
}

// ListTransactions returns all transactions for an account. Ownership is verified via the account cache.
func (s *TransactionQueryService) ListTransactions(q cqrs.ListTransactionsQuery) ([]models.TransactionView, error) {
	ctx := context.Background()
	account, err := s.accountRepo.GetAccount(ctx, q.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("account not found")
	}
	if account.UserID != q.UserID {
		return nil, fmt.Errorf("forbidden")
	}
	return s.readRepo.ListByAccountNumber(ctx, q.AccountNumber)
}
