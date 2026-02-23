package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/eaglebank/shared/cqrs"
	"github.com/eaglebank/shared/events"
	"github.com/eaglebank/shared/models"
	"github.com/eaglebank/shared/utils"
	"github.com/eaglebank/user-service/internal/repository"
)

// UserCommandService writes user state to PostgreSQL and keeps the Redis
// read model up to date.
type UserCommandService struct {
	writeRepo *repository.UserWriteRepository
	readRepo  *repository.UserReadRepository
	publisher *events.Publisher
}

func NewUserCommandService(
	writeRepo *repository.UserWriteRepository,
	readRepo *repository.UserReadRepository,
	publisher *events.Publisher,
) *UserCommandService {
	return &UserCommandService{
		writeRepo: writeRepo,
		readRepo:  readRepo,
		publisher: publisher,
	}
}

func (s *UserCommandService) CreateUser(cmd cqrs.CreateUserCommand) (*models.User, error) {
	passwordHash, err := utils.HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user := &models.User{
		ID:           utils.GenerateID("usr"),
		Name:         cmd.Name,
		Email:        cmd.Email,
		PasswordHash: passwordHash,
		PhoneNumber:  cmd.PhoneNumber,
		Address:      cmd.Address,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.writeRepo.Create(user); err != nil {
		return nil, err
	}
	ctx := context.Background()
	s.readRepo.CacheUserView(ctx, userToView(user))
	if err := s.publisher.Publish(ctx, events.UserEventsStream, events.UserCreated, events.UserCreatedEvent{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}); err != nil {
		log.Printf("Failed to publish user.created event: %v", err)
	}
	return user, nil
}

func (s *UserCommandService) UpdateUser(cmd cqrs.UpdateUserCommand) (*models.UserView, error) {
	user, err := s.writeRepo.GetByID(cmd.UserID)
	if err != nil {
		return nil, err
	}
	user.Name = cmd.Name
	user.Email = cmd.Email
	user.PhoneNumber = cmd.PhoneNumber
	user.Address = cmd.Address
	user.UpdatedAt = time.Now().UTC()
	if err := s.writeRepo.Update(user); err != nil {
		return nil, err
	}
	view := userToView(user)
	s.readRepo.CacheUserView(context.Background(), view)
	if err := s.publisher.Publish(context.Background(), events.UserEventsStream, events.UserUpdated, events.UserUpdatedEvent{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}); err != nil {
		log.Printf("Failed to publish user.updated event: %v", err)
	}
	return view, nil
}

// DeleteUser rejects the operation if the user still has open accounts.
func (s *UserCommandService) DeleteUser(cmd cqrs.DeleteUserCommand) error {
	if s.readRepo.HasActiveAccounts(context.Background(), cmd.UserID) {
		return fmt.Errorf("user has active accounts")
	}
	if err := s.writeRepo.Delete(cmd.UserID); err != nil {
		return err
	}
	s.readRepo.InvalidateUserView(context.Background(), cmd.UserID)
	if err := s.publisher.Publish(context.Background(), events.UserEventsStream, events.UserDeleted, events.UserDeletedEvent{
		UserID: cmd.UserID,
	}); err != nil {
		log.Printf("Failed to publish user.deleted event: %v", err)
	}
	return nil
}

// HandleAccountEvent is the Redis stream subscriber handler.
// It reacts to account.created / account.deleted events to keep user-side
// metadata and logs current.
func (s *UserCommandService) HandleAccountEvent(ctx context.Context, event events.Event) error {
	log.Printf("Received account event: %s", event.Type)
	switch event.Type {
	case events.AccountCreated:
		dataBytes, _ := json.Marshal(event.Data)
		var data events.AccountCreatedEvent
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			return fmt.Errorf("failed to unmarshal account.created event: %w", err)
		}
		log.Printf("User %s created account %s", data.UserID, data.AccountNumber)
		s.readRepo.IncrAccountCount(ctx, data.UserID)
	case events.AccountDeleted:
		dataBytes, _ := json.Marshal(event.Data)
		var data events.AccountDeletedEvent
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			return fmt.Errorf("failed to unmarshal account.deleted event: %w", err)
		}
		log.Printf("User %s deleted account %s", data.UserID, data.AccountNumber)
		s.readRepo.DecrAccountCount(ctx, data.UserID)
	}
	return nil
}

func userToView(u *models.User) *models.UserView {
	return &models.UserView{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		Address:     u.Address,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
