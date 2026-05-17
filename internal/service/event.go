package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ebooker/internal/entity"

	"github.com/google/uuid"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/dbpg/pgx-driver/transaction"
	"github.com/wb-go/wbf/logger"
)

const _defaultTTL = 15

type (
	CreateEventRequest struct {
		Title         string
		Description   string
		Date          time.Time
		TotalSeats    int
		BookingTTLMin int
	}

	EventService struct {
		eventRepo EventRepository
		tm        transaction.Manager
		log       logger.Logger

		bookingTTLMin int
	}
)

func NewEventService(
	eventRepo EventRepository,
	tm transaction.Manager,
	log logger.Logger,
	opts ...Option,
) *EventService {
	s := &EventService{
		eventRepo:     eventRepo,
		tm:            tm,
		bookingTTLMin: _defaultTTL,
		log:           log,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *EventService) Create(ctx context.Context, req CreateEventRequest) (*entity.Event, error) {
	const op = "service.event.Create"

	log := s.log.With("op", op)
	startTime := time.Now().UTC()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.String("title", req.Title),
	)

	log.LogAttrs(ctx, logger.InfoLevel, "create event started",
		logger.String("title", req.Title),
		logger.Int("total_seats", req.TotalSeats),
	)

	if req.Date.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("%s: %w: date must be in future", op, entity.ErrInvalidData)
	}
	if req.TotalSeats <= 0 {
		return nil, fmt.Errorf("%s: %w: total seats must be > 0", op, entity.ErrInvalidData)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("%s: generate id: %w", op, err)
	}

	ttl := req.BookingTTLMin
	now := time.Now().UTC()
	event := entity.Event{
		ID:            id,
		Title:         req.Title,
		Description:   req.Description,
		Date:          req.Date,
		TotalSeats:    req.TotalSeats,
		BookingTTLMin: ttl,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err = s.tm.ExecuteInTransaction(ctx, op, func(tx pgxdriver.QueryExecuter) error {
		if err = s.eventRepo.Create(ctx, tx, event); err != nil {
			return transaction.HandleError(err)
		}
		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "create event failed", logger.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "event created",
		logger.Any("event_id", event.ID),
		logger.Duration("duration", time.Since(startTime)),
	)
	return &event, nil
}

func (s *EventService) GetWithStats(ctx context.Context, id uuid.UUID) (*entity.EventWithStats, error) {
	const op = "service.event.GetWithStats"

	log := s.log.With("op", op)
	startTime := time.Now().UTC()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.Any("event_id", id),
	)

	log.LogAttrs(ctx, logger.DebugLevel, "get event stats requested",
		logger.Any("event_id", id),
	)

	var result *entity.EventWithStats
	err := s.tm.ExecuteInTransaction(ctx, op, func(tx pgxdriver.QueryExecuter) error {
		event, err := s.eventRepo.GetByID(ctx, tx, id, false)
		if err != nil {
			if errors.Is(err, entity.ErrEventNotFound) {
				return entity.ErrEventNotFound
			}
			return transaction.HandleError(err)
		}

		booked, err := s.eventRepo.CountBookings(ctx, tx, id, entity.StatusPending)
		if err != nil {
			return fmt.Errorf("count pending: %w", err)
		}
		confirmed, err := s.eventRepo.CountBookings(ctx, tx, id, entity.StatusConfirmed)
		if err != nil {
			return fmt.Errorf("count confirmed: %w", err)
		}

		occupied := booked + confirmed
		free := event.TotalSeats - occupied
		if free < 0 {
			free = 0
		}

		result = &entity.EventWithStats{
			Event:          *event,
			BookedSeats:    booked,
			ConfirmedSeats: confirmed,
			FreeSeats:      free,
		}
		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "get event stats failed", logger.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "event stats ready",
		logger.Any("event_id", id),
		logger.Int("free_seats", result.FreeSeats),
		logger.Duration("duration", time.Since(startTime)),
	)
	return result, nil
}

func (s *EventService) List(ctx context.Context) ([]entity.EventWithStats, error) {
	const op = "service.event.List"

	log := s.log.With("op", op)
	startTime := time.Now().UTC()
	defer logSlowOperation(ctx, s.log, op, startTime)

	log.LogAttrs(ctx, logger.DebugLevel, "list events requested")

	events, err := s.eventRepo.List(ctx, nil)
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "list events failed", logger.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]entity.EventWithStats, 0, len(events))
	for _, e := range events {
		booked, _ := s.eventRepo.CountBookings(ctx, nil, e.ID, entity.StatusPending)
		confirmed, _ := s.eventRepo.CountBookings(ctx, nil, e.ID, entity.StatusConfirmed)
		occupied := booked + confirmed
		free := e.TotalSeats - occupied
		if free < 0 {
			free = 0
		}
		result = append(result, entity.EventWithStats{
			Event:          e,
			BookedSeats:    booked,
			ConfirmedSeats: confirmed,
			FreeSeats:      free,
		})
	}

	log.LogAttrs(ctx, logger.InfoLevel, "events listed",
		logger.Int("count", len(result)),
		logger.Duration("duration", time.Since(startTime)),
	)
	return result, nil
}
