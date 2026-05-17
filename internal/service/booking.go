package service

import (
	"context"
	"fmt"
	"time"

	"ebooker/internal/entity"
	"ebooker/internal/transport/notifier"

	"github.com/google/uuid"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/dbpg/pgx-driver/transaction"
	"github.com/wb-go/wbf/logger"
)

const (
	_queryLimit = 100
)

type BookingService struct {
	bookingRepo BookingRepository
	eventRepo   EventRepository
	userRepo    UserRepository
	tm          transaction.Manager
	notifier    notifier.Notifier
	eventSvc    *EventService
	log         logger.Logger
}

func NewBookingService(
	bookingRepo BookingRepository,
	eventRepo EventRepository,
	userRepo UserRepository,
	tm transaction.Manager,
	notifier notifier.Notifier,
	eventSvc *EventService,
	log logger.Logger,
) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
		userRepo:    userRepo,
		tm:          tm,
		notifier:    notifier,
		eventSvc:    eventSvc,
		log:         log,
	}
}

func (s *BookingService) Book(ctx context.Context, eventID, userID uuid.UUID) (*entity.Booking, error) {
	const op = "service.booking.Book"

	log := s.log.With("op", op)
	startTime := time.Now().UTC()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.Any("event_id", eventID),
		logger.Any("user_id", userID),
	)

	log.LogAttrs(ctx, logger.InfoLevel, "book seat requested",
		logger.Any("event_id", eventID),
		logger.Any("user_id", userID),
	)

	var booking entity.Booking
	err := s.tm.ExecuteInTransaction(ctx, op, func(tx pgxdriver.QueryExecuter) error {
		event, err := s.eventRepo.GetByID(ctx, tx, eventID, true)
		if err != nil {
			return transaction.HandleError(err)
		}

		pending, err := s.eventRepo.CountBookings(ctx, tx, eventID, entity.StatusPending)
		if err != nil {
			return transaction.HandleError(err)
		}
		confirmed, err := s.eventRepo.CountBookings(ctx, tx, eventID, entity.StatusConfirmed)
		if err != nil {
			return transaction.HandleError(err)
		}

		log.LogAttrs(ctx, logger.DebugLevel, "seat availability checked",
			logger.Int("total", event.TotalSeats),
			logger.Int("pending", pending),
			logger.Int("confirmed", confirmed),
		)

		if pending+confirmed >= event.TotalSeats {
			return entity.ErrNoSeatsLeft
		}

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}

		now := time.Now().UTC()
		booking = entity.Booking{
			ID:        id,
			EventID:   eventID,
			UserID:    userID,
			Status:    entity.StatusPending,
			ExpiresAt: now.Add(time.Duration(event.BookingTTLMin) * time.Minute),
			CreatedAt: now,
			UpdatedAt: &now,
		}

		return s.bookingRepo.Create(ctx, tx, booking)
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "booking failed", logger.Any("error", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "booking created",
		logger.Any("booking_id", booking.ID),
		logger.Duration("duration", time.Since(startTime)),
	)
	return &booking, nil
}

func (s *BookingService) Confirm(ctx context.Context, bookingID uuid.UUID) error {
	const op = "service.booking.Confirm"

	log := s.log.With("op", op, "booking_id", bookingID)
	startTime := time.Now().UTC()
	defer logSlowOperation(ctx, s.log, op, startTime,
		logger.Any("booking_id", bookingID),
	)

	log.LogAttrs(ctx, logger.InfoLevel, "confirm booking requested",
		logger.Any("booking_id", bookingID),
	)

	err := s.tm.ExecuteInTransaction(ctx, op, func(tx pgxdriver.QueryExecuter) error {
		b, err := s.bookingRepo.GetByID(ctx, tx, bookingID, true)
		if err != nil {
			return transaction.HandleError(err)
		}

		log.LogAttrs(ctx, logger.DebugLevel, "booking found",
			logger.String("status", b.Status.String()),
		)

		if b.Status == entity.StatusConfirmed {
			return entity.ErrAlreadyPaid
		}
		if b.Status == entity.StatusCancelled {
			return entity.ErrBookingExpired
		}
		if b.IsExpired() {
			if err = s.bookingRepo.UpdateStatus(ctx, tx, bookingID, entity.StatusCancelled); err != nil {
				log.LogAttrs(ctx, logger.WarnLevel, "failed to cancel expired booking",
					logger.Any("error", err),
				)
			}
			return entity.ErrBookingExpired
		}

		if err = s.bookingRepo.UpdateStatus(ctx, tx, bookingID, entity.StatusConfirmed); err != nil {
			return transaction.HandleError(err)
		}
		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "confirm failed", logger.Any("error", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "booking confirmed",
		logger.Any("booking_id", bookingID),
		logger.Duration("duration", time.Since(startTime)),
	)
	return nil
}

func (s *BookingService) GetBooking(ctx context.Context, id uuid.UUID) (*entity.Booking, error) {
	const op = "service.booking.GetBooking"

	booking, err := s.bookingRepo.GetByID(ctx, nil, id, false)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return booking, nil
}

func (s *BookingService) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]entity.Booking, error) {
	const op = "service.booking.ListByEvent"

	bookings, err := s.bookingRepo.ListByEvent(ctx, nil, eventID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return bookings, nil
}

func (s *BookingService) RunExpiryWorker(ctx context.Context, interval time.Duration) {
	s.log.LogAttrs(ctx, logger.InfoLevel, "expiry worker started",
		logger.Duration("interval", interval),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.LogAttrs(ctx, logger.InfoLevel, "expiry worker stopped")
			return
		case <-ticker.C:
			s.processExpired(ctx)
		}
	}
}

func (s *BookingService) processExpired(ctx context.Context) {
	const op = "service.booking.processExpired"

	log := s.log.With("op", op)
	startTime := time.Now().UTC()

	var expired []entity.Booking
	err := s.tm.ExecuteInTransaction(ctx, op, func(tx pgxdriver.QueryExecuter) error {
		var err error
		expired, err = s.bookingRepo.GetExpiredPending(ctx, tx, _queryLimit)
		if err != nil {
			return fmt.Errorf("get expired: %w", err)
		}

		log.LogAttrs(ctx, logger.DebugLevel, "expired bookings found",
			logger.Int("count", len(expired)),
		)

		for _, b := range expired {
			if err = s.bookingRepo.UpdateStatus(ctx, tx, b.ID, entity.StatusCancelled); err != nil {
				log.LogAttrs(ctx, logger.ErrorLevel, "cancel expired booking failed",
					logger.Any("booking_id", b.ID),
					logger.Any("error", err),
				)
			}
		}
		return nil
	})
	if err != nil {
		log.LogAttrs(ctx, logger.ErrorLevel, "expiry batch failed", logger.Any("error", err))
		return
	}

	if len(expired) > 0 {
		log.LogAttrs(ctx, logger.InfoLevel, "expired bookings cancelled",
			logger.Int("count", len(expired)),
			logger.Duration("duration", time.Since(startTime)),
		)
	}

	for _, b := range expired {
		s.notifyExpired(ctx, b)
	}
}

func (s *BookingService) notifyExpired(ctx context.Context, b entity.Booking) {
	user, err := s.userRepo.GetByID(ctx, nil, b.UserID)
	if err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "notify expired: get user failed",
			logger.Any("booking_id", b.ID),
			logger.Any("error", err),
		)
		return
	}

	event, err := s.eventRepo.GetByID(ctx, nil, b.EventID, false)
	if err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "notify expired: get event failed",
			logger.Any("booking_id", b.ID),
			logger.Any("error", err),
		)
		return
	}

	req := entity.CancelledNotification{
		BookingID:  b.ID,
		UserID:     b.UserID,
		UserName:   user.Name,
		UserEmail:  user.Email,
		TelegramID: user.TelegramID,
		EventID:    b.EventID,
		EventTitle: event.Title,
		EventDate:  event.Date.Format("02.01.2006 15:04"),
	}

	if err = s.notifier.NotifyBookingCancelled(ctx, req); err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "notify expired: send failed",
			logger.Any("booking_id", b.ID),
			logger.Any("error", err),
		)
	} else {
		s.log.LogAttrs(ctx, logger.InfoLevel, "expiry notification sent",
			logger.Any("booking_id", b.ID),
		)
	}
}
