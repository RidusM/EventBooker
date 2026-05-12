package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"ebooker/internal/entity"

	"github.com/google/uuid"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/dbpg/pgx-driver/transaction"
	"github.com/wb-go/wbf/logger"
)

const _serviceTokenByteLength = 16

type (
	RegisterUserRequest struct {
		Name       string
		Email      string
		TelegramID *int64
	}

	UserService struct {
		userRepo UserRepository
		tm       transaction.Manager
		log      logger.Logger
	}
)

func NewUserService(
	userRepo UserRepository,
	tm transaction.Manager,
	log logger.Logger,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		tm:       tm,
		log:      log,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, req RegisterUserRequest) (*entity.User, error) {
	const op = "service.RegisterUser"

	log := s.log.With("op", op)
	startTime := time.Now()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.String("name", req.Name),
		logger.String("email", req.Email),
	)

	log.LogAttrs(ctx, logger.InfoLevel, "register user requested",
		logger.String("name", req.Name),
		logger.String("email", req.Email),
	)

	if req.Email == "" && (req.TelegramID == nil || *req.TelegramID == 0) {
		return nil, fmt.Errorf("%s: email or telegram_id is required: %w", op, entity.ErrInvalidData)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("%s: generate id: %w", op, err)
	}

	var telegramID *int64
	if req.TelegramID != nil {
		telegramID = req.TelegramID
	}

	user := entity.User{
		ID:         id,
		Name:       req.Name,
		Email:      req.Email,
		TelegramID: telegramID,
		CreatedAt:  time.Now(),
	}

	err = s.tm.ExecuteInTransaction(ctx, "register_user", func(tx pgxdriver.QueryExecuter) error {
		if err = s.userRepo.Create(ctx, tx, user); err != nil {
			return transaction.HandleError(err)
		}
		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "register failed", logger.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "user registered",
		logger.String("user_id", id.String()),
		logger.Duration("duration", time.Since(startTime)),
	)
	return &user, nil
}

func (s *UserService) GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, error) {
	const op = "service.GenerateLinkToken"

	log := s.log.With("op", op)
	startTime := time.Now()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.String("user_id", userID.String()),
	)

	log.LogAttrs(ctx, logger.InfoLevel, "generate link token requested",
		logger.String("user_id", userID.String()),
	)

	bytes := make([]byte, _serviceTokenByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("%s: generate random: %w", op, err)
	}
	token := hex.EncodeToString(bytes)

	expiresAt := time.Now().Add(1 * time.Hour)

	err := s.tm.ExecuteInTransaction(ctx, "create_link_token", func(tx pgxdriver.QueryExecuter) error {
		if err := s.userRepo.CreateLinkToken(ctx, tx, userID, token, expiresAt); err != nil {
			return transaction.HandleError(err)
		}
		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "create link token failed", logger.Any("error", err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "link token generated successfully",
		logger.String("user_id", userID.String()),
		logger.Duration("duration", time.Since(startTime)),
	)
	return token, nil
}

func (s *UserService) LinkTelegramByToken(ctx context.Context, token string, chatID *int64) error {
	const op = "service.LinkTelegramByToken"

	log := s.log.With("op", op)
	startTime := time.Now()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.Int64("chat_id", *chatID),
	)

	log.LogAttrs(ctx, logger.InfoLevel, "link telegram by token requested",
		logger.Int64("chat_id", *chatID),
	)

	err := s.tm.ExecuteInTransaction(ctx, "link_telegram_by_token", func(tx pgxdriver.QueryExecuter) error {
		userID, err := s.userRepo.GetUserByLinkToken(ctx, tx, token)
		if err != nil {
			if errors.Is(err, entity.ErrUserNotFound) || errors.Is(err, entity.ErrInvalidData) {
				return fmt.Errorf("%s: invalid or expired token: %w", op, entity.ErrInvalidData)
			}
			return fmt.Errorf("%s: get user id by token: %w", op, err)
		}

		if err = s.userRepo.DeleteLinkToken(ctx, tx, token); err != nil {
			return fmt.Errorf("%s: delete token: %w", op, err)
		}

		user, err := s.userRepo.GetByID(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("%s: get user details: %w", op, err)
		}

		if err = s.userRepo.UpdateTelegramID(ctx, tx, user.ID, chatID); err != nil {
			return transaction.HandleError(err)
		}

		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "link telegram by token failed", logger.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "telegram linked successfully",
		logger.String("user_id", "hidden"),
		logger.Int64("chat_id", *chatID),
		logger.Duration("duration", time.Since(startTime)),
	)
	return nil
}

func (s *UserService) GetUserByTelegramID(ctx context.Context, chatID *int64) (*entity.User, error) {
	const op = "service.GetUserByTelegramID"

	log := s.log.With("op", op)
	startTime := time.Now()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.Int64("chat_id", *chatID),
	)

	log.LogAttrs(ctx, logger.DebugLevel, "get user by telegram id requested",
		logger.Int64("chat_id", *chatID),
	)

	user, err := s.userRepo.GetByTelegramID(ctx, nil, chatID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			log.LogAttrs(ctx, logger.DebugLevel, "user not found by telegram id")
		} else {
			log.LogAttrs(ctx, logger.ErrorLevel, "get user by telegram id failed", logger.Any("error", err))
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.DebugLevel, "user found by telegram id",
		logger.String("user_id", user.ID.String()),
		logger.Duration("duration", time.Since(startTime)),
	)
	return user, nil
}

func (s *UserService) GetTelegramStartHandler() func(ctx context.Context, username string, chatID *int64, startPayload string) error {
	return func(ctx context.Context, username string, chatID *int64, startPayload string) error {
		const op = "service.UserService.TelegramStartHandler"
		log := s.log.With("op", op, "username", username)

		if startPayload != "" {
			err := s.LinkTelegramByToken(ctx, startPayload, chatID)
			if err == nil {
				log.LogAttrs(ctx, logger.InfoLevel, "account linked via token")
				return nil
			}
			log.LogAttrs(ctx, logger.DebugLevel, "token link failed, falling back to lookup/register",
				logger.Any("error", err),
			)
		}

		user, err := s.userRepo.GetByTelegramID(ctx, nil, chatID)
		switch {
		case errors.Is(err, entity.ErrUserNotFound):
			regReq := RegisterUserRequest{
				Name:       username,
				Email:      "",
				TelegramID: chatID,
			}
			if _, err = s.RegisterUser(ctx, regReq); err != nil {
				return fmt.Errorf("%s: register new user: %w", op, err)
			}
			log.LogAttrs(ctx, logger.InfoLevel, "new user registered via telegram",
				logger.Int64("chat_id", *chatID),
			)

		case err != nil:
			return fmt.Errorf("%s: lookup user by telegram: %w", op, err)

		default:
			log.LogAttrs(ctx, logger.InfoLevel, "existing user found",
				logger.String("user_id", user.ID.String()),
			)
		}

		return nil
	}
}

func (s *UserService) List(ctx context.Context) ([]entity.User, error) {
	const op = "service.user.List"

	users, err := s.userRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return users, nil
}
