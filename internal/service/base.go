package service

import (
	"context"
	"time"

	"ebooker/internal/entity"

	"github.com/google/uuid"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/logger"
)

const (
	_slowOperationThreshold = 200 * time.Millisecond
	_attrCount              = 2
)

type (
	EventRepository interface {
		Create(ctx context.Context, qe pgxdriver.QueryExecuter, e entity.Event) error
		GetByID(ctx context.Context, qe pgxdriver.QueryExecuter, id uuid.UUID, forUpdate bool) (*entity.Event, error)
		List(ctx context.Context, qe pgxdriver.QueryExecuter) ([]entity.Event, error)
		CountBookings(
			ctx context.Context,
			qe pgxdriver.QueryExecuter,
			eventID uuid.UUID,
			status entity.Status,
		) (int, error)
	}

	BookingRepository interface {
		Create(ctx context.Context, qe pgxdriver.QueryExecuter, b entity.Booking) error
		GetByID(ctx context.Context, qe pgxdriver.QueryExecuter, id uuid.UUID, forUpdate bool) (*entity.Booking, error)
		UpdateStatus(ctx context.Context, qe pgxdriver.QueryExecuter, id uuid.UUID, status entity.Status) error
		GetExpiredPending(ctx context.Context, qe pgxdriver.QueryExecuter, limit uint64) ([]entity.Booking, error)
		ListByEvent(ctx context.Context, qe pgxdriver.QueryExecuter, eventID uuid.UUID) ([]entity.Booking, error)
	}

	UserRepository interface {
		Create(ctx context.Context, qe pgxdriver.QueryExecuter, u entity.User) error
		GetByID(ctx context.Context, qe pgxdriver.QueryExecuter, id uuid.UUID) (*entity.User, error)
		GetByTelegramID(ctx context.Context, qe pgxdriver.QueryExecuter, chatID *int64) (*entity.User, error)
		UpdateTelegramID(ctx context.Context, qe pgxdriver.QueryExecuter, userID uuid.UUID, chatID *int64) error
		CreateLinkToken(
			ctx context.Context,
			qe pgxdriver.QueryExecuter,
			userID uuid.UUID,
			token string,
			expiresAt time.Time,
		) error
		GetUserByLinkToken(ctx context.Context, qe pgxdriver.QueryExecuter, token string) (uuid.UUID, error)
		DeleteLinkToken(ctx context.Context, qe pgxdriver.QueryExecuter, token string) error
		List(ctx context.Context, qe pgxdriver.QueryExecuter) ([]entity.User, error)
	}
)

func logSlowOperation(
	ctx context.Context,
	log logger.Logger,
	op string,
	startTime time.Time,
	attrs ...logger.Attr,
) {
	duration := time.Since(startTime)
	if duration > _slowOperationThreshold {
		all := make([]logger.Attr, 0, _attrCount+len(attrs))
		all = append(all,
			logger.String("op", op),
			logger.Duration("duration", duration),
		)
		all = append(all, attrs...)
		log.Ctx(ctx).LogAttrs(ctx, logger.WarnLevel, "slow operation detected", all...)
	}
}
