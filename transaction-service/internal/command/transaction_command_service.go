package command

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/events"
	"github.com/eaglebank/shared/models"
	"github.com/eaglebank/shared/utils"
	"github.com/eaglebank/transaction-service/internal/repository"
)

// TransactionCommandService creates transactions. It checks account ownership and
// balance against the Redis cache before writing to Postgres.
type TransactionCommandService struct {
	writeRepo   *repository.TransactionWriteRepository
	readRepo    *repository.TransactionReadRepository
	accountRepo *repository.AccountRepository
	publisher   *events.Publisher
}

func NewTransactionCommandService(
	writeRepo *repository.TransactionWriteRepository,
	readRepo *repository.TransactionReadRepository,
	accountRepo *repository.AccountRepository,
	publisher *events.Publisher,
) *TransactionCommandService {
	return &TransactionCommandService{
		writeRepo:   writeRepo,
		readRepo:    readRepo,
		accountRepo: accountRepo,
		publisher:   publisher,
	}
}

func (s *TransactionCommandService) CreateTransaction(cmd cqrs.CreateTransactionCommand) (*models.Transaction, error) {
	if cmd.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	ctx := context.Background()
	account, err := s.accountRepo.GetAccount(ctx, cmd.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("account not found")
	}
	if account.UserID != cmd.UserID {
		return nil, fmt.Errorf("forbidden")
	}
	if cmd.Type == "withdrawal" && account.Balance < cmd.Amount {
		return nil, fmt.Errorf("insufficient funds")
	}
	transaction := &models.Transaction{
		ID:            utils.GenerateID("tan"),
		AccountNumber: cmd.AccountNumber,
		UserID:        cmd.UserID,
		Amount:        cmd.Amount,
		Currency:      cmd.Currency,
		Type:          cmd.Type,
		Reference:     cmd.Reference,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.writeRepo.Create(transaction); err != nil {
		return nil, err
	}
	s.readRepo.CacheTransactionView(ctx, txToView(transaction))
	if err := s.publisher.Publish(ctx, events.TransactionEventsStream, events.TransactionCreated, events.TransactionCreatedEvent{
		TransactionID: transaction.ID,
		AccountNumber: cmd.AccountNumber,
		UserID:        cmd.UserID,
		Amount:        cmd.Amount,
		Type:          cmd.Type,
		Currency:      cmd.Currency,
	}); err != nil {
		log.Printf("Failed to publish transaction.created event: %v", err)
	}
	return transaction, nil
}

// txToView converts the write model to a read view model.
func txToView(t *models.Transaction) *models.TransactionView {
	return &models.TransactionView{
		ID:            t.ID,
		AccountNumber: t.AccountNumber,
		UserID:        t.UserID,
		Amount:        t.Amount,
		Currency:      t.Currency,
		Type:          t.Type,
		Reference:     t.Reference,
		CreatedAt:     t.CreatedAt,
	}
}
