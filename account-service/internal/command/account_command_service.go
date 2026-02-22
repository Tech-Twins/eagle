package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/eaglebank/account-service/internal/repository"
	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/events"
	"github.com/eaglebank/shared/models"
	"github.com/eaglebank/shared/utils"
)

// AccountCommandService writes account state and keeps the read model in sync.
type AccountCommandService struct {
	writeRepo *repository.AccountWriteRepository
	readRepo  *repository.AccountReadRepository
	publisher *events.Publisher
}

func NewAccountCommandService(
	writeRepo *repository.AccountWriteRepository,
	readRepo *repository.AccountReadRepository,
	publisher *events.Publisher,
) *AccountCommandService {
	return &AccountCommandService{
		writeRepo: writeRepo,
		readRepo:  readRepo,
		publisher: publisher,
	}
}

func (s *AccountCommandService) CreateAccount(cmd cqrs.CreateAccountCommand) (*models.Account, error) {
	account := &models.Account{
		AccountNumber: utils.GenerateAccountNumber(),
		UserID:        cmd.UserID,
		SortCode:      "10-10-10",
		Name:          cmd.Name,
		AccountType:   cmd.AccountType,
		Balance:       0.00,
		Currency:      "GBP",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.writeRepo.Create(account); err != nil {
		return nil, err
	}
	ctx := context.Background()
	s.readRepo.CacheAccountView(ctx, accountToView(account))
	if err := s.publisher.Publish(ctx, events.AccountEventsStream, events.AccountCreated, events.AccountCreatedEvent{
		AccountNumber: account.AccountNumber,
		UserID:        account.UserID,
		Name:          account.Name,
		AccountType:   account.AccountType,
	}); err != nil {
		log.Printf("Failed to publish account.created event: %v", err)
	}
	return account, nil
}

func (s *AccountCommandService) UpdateAccount(cmd cqrs.UpdateAccountCommand) (*models.AccountView, error) {
	account, err := s.writeRepo.GetByAccountNumber(cmd.AccountNumber)
	if err != nil {
		return nil, err
	}
	if account.UserID != cmd.RequestingUserID {
		return nil, fmt.Errorf("forbidden")
	}
	account.Name = cmd.Name
	account.AccountType = cmd.AccountType
	account.UpdatedAt = time.Now().UTC()
	if err := s.writeRepo.Update(account); err != nil {
		return nil, err
	}
	updated, err := s.writeRepo.GetByAccountNumber(cmd.AccountNumber)
	if err != nil {
		return nil, err
	}
	view := accountToView(updated)
	s.readRepo.CacheAccountView(context.Background(), view)
	if err := s.publisher.Publish(context.Background(), events.AccountEventsStream, events.AccountUpdated, events.AccountUpdatedEvent{
		AccountNumber: account.AccountNumber,
		UserID:        account.UserID,
		Name:          account.Name,
	}); err != nil {
		log.Printf("Failed to publish account.updated event: %v", err)
	}
	return view, nil
}

func (s *AccountCommandService) DeleteAccount(cmd cqrs.DeleteAccountCommand) error {
	account, err := s.writeRepo.GetByAccountNumber(cmd.AccountNumber)
	if err != nil {
		return err
	}
	if account.UserID != cmd.RequestingUserID {
		return fmt.Errorf("forbidden")
	}
	if err := s.writeRepo.Delete(cmd.AccountNumber); err != nil {
		return err
	}
	s.readRepo.InvalidateAccountView(context.Background(), cmd.AccountNumber)
	if err := s.publisher.Publish(context.Background(), events.AccountEventsStream, events.AccountDeleted, events.AccountDeletedEvent{
		AccountNumber: account.AccountNumber,
		UserID:        account.UserID,
	}); err != nil {
		log.Printf("Failed to publish account.deleted event: %v", err)
	}
	return nil
}

// HandleTransactionEvent reacts to transaction.created events by updating the
// account balance. Idempotent: duplicate delivery of the same transaction ID
// is detected via Redis and skipped without modifying the balance.
func (s *AccountCommandService) HandleTransactionEvent(ctx context.Context, event events.Event) error {
	log.Printf("Received transaction event: %s", event.Type)
	if event.Type != events.TransactionCreated {
		return nil
	}
	dataBytes, _ := json.Marshal(event.Data)
	var data events.TransactionCreatedEvent
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal transaction.created event: %w", err)
	}
	if s.readRepo.IsTransactionProcessed(ctx, data.TransactionID) {
		log.Printf("Transaction %s already processed, skipping duplicate event", data.TransactionID)
		return nil
	}
	account, err := s.writeRepo.GetByAccountNumber(data.AccountNumber)
	if err != nil {
		return fmt.Errorf("failed to get account for balance update: %w", err)
	}
	var newBalance float64
	if data.Type == "deposit" {
		newBalance = account.Balance + data.Amount
	} else {
		newBalance = account.Balance - data.Amount
	}
	if err := s.writeRepo.UpdateBalance(data.AccountNumber, newBalance); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	// Record the transaction ID before updating the cache, so that any
	// redelivery after this point is detected and skipped.
	s.readRepo.MarkTransactionProcessed(ctx, data.TransactionID)
	account.Balance = newBalance
	s.readRepo.CacheAccountView(ctx, accountToView(account))
	if err := s.publisher.Publish(ctx, events.AccountEventsStream, events.BalanceUpdated, events.BalanceUpdatedEvent{
		AccountNumber: data.AccountNumber,
		NewBalance:    newBalance,
		Change:        data.Amount,
	}); err != nil {
		log.Printf("Failed to publish balance.updated event: %v", err)
	}
	log.Printf("Balance updated for account %s: %.2f -> %.2f", data.AccountNumber, account.Balance, newBalance)
	return nil
}

// accountToView converts the PostgreSQL write model to the Redis read view model.
func accountToView(a *models.Account) *models.AccountView {
	return &models.AccountView{
		AccountNumber: a.AccountNumber,
		UserID:        a.UserID,
		SortCode:      a.SortCode,
		Name:          a.Name,
		AccountType:   a.AccountType,
		Balance:       a.Balance,
		Currency:      a.Currency,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}
