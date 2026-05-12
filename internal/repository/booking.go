package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ebooker/internal/entity"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
)

const (
	_bookingColumns = "id, event_id, user_id, status, expires_at, created_at, updated_at"
)

type BookingRepository struct {
	db *pgxdriver.Postgres
}

func NewBookingRepository(db *pgxdriver.Postgres) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) Create(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	b entity.Booking,
) error {
	const op = "repository.booking.Create"

	sql, args, err := r.db.Insert("bookings").
		Columns(_bookingColumns).
		Values(b.ID, b.EventID, b.UserID, b.Status, b.ExpiresAt, b.CreatedAt, b.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = execOrDB(qe, r.db).Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, entity.ErrConflictingData)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *BookingRepository) GetByID(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	id uuid.UUID,
	forUpdate bool,
) (*entity.Booking, error) {
	const op = "repository.booking.GetByID"

	query := r.db.Select(_bookingColumns).
		From("bookings").
		Where(squirrel.Eq{"id": id})

	if forUpdate {
		query = query.Suffix("FOR UPDATE")
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	b := &entity.Booking{}
	err = execOrDB(qe, r.db).QueryRow(ctx, sql, args...).Scan(
		&b.ID,
		&b.EventID,
		&b.UserID,
		&b.Status,
		&b.ExpiresAt,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrBookingNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return b, nil
}

func (r *BookingRepository) UpdateStatus(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	id uuid.UUID,
	status entity.Status,
) error {
	const op = "repository.booking.UpdateStatus"

	sql, args, err := r.db.Update("bookings").
		Set("status", status).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	res, err := execOrDB(qe, r.db).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, entity.ErrBookingNotFound)
	}

	return nil
}

func (r *BookingRepository) GetExpiredPending(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	limit uint64,
) ([]entity.Booking, error) {
	const op = "repository.booking.GetExpiredPending"

	sql, args, err := r.db.Select(_bookingColumns).
		From("bookings").
		Where("status = ? AND expires_at < ?", entity.StatusPending, time.Now().UTC()).
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := execOrDB(qe, r.db).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var bookings []entity.Booking
	for rows.Next() {
		var b entity.Booking
		if err = rows.Scan(
			&b.ID,
			&b.EventID,
			&b.UserID,
			&b.Status,
			&b.ExpiresAt,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		bookings = append(bookings, b)
	}

	return bookings, nil
}

func (r *BookingRepository) ListByEvent(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	eventID uuid.UUID,
) ([]entity.Booking, error) {
	const op = "repository.booking.ListByEvent"

	sql, args, err := r.db.Select(_bookingColumns).
		From("bookings").
		Where("event_id = ?", eventID).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := execOrDB(qe, r.db).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var bookings []entity.Booking
	for rows.Next() {
		var b entity.Booking
		if err = rows.Scan(
			&b.ID,
			&b.EventID,
			&b.UserID,
			&b.Status,
			&b.ExpiresAt,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		bookings = append(bookings, b)
	}

	return bookings, nil
}
