package repository

import (
	"context"
	"errors"
	"fmt"

	"ebooker/internal/entity"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
)

const _eventColumnts = "id, title, description, date, total_seats, booking_ttl_min, created_at, updated_at"

type EventRepository struct {
	db *pgxdriver.Postgres
}

func NewEventRepository(db *pgxdriver.Postgres) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	e entity.Event,
) error {
	const op = "repository.event.Create"

	sql, args, err := r.db.Insert("events").
		Columns(_eventColumnts).
		Values(e.ID, e.Title, e.Description, e.Date, e.TotalSeats, e.BookEventTTL, e.CreatedAt, e.UpdatedAt).
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

func (r *EventRepository) GetByID(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	id uuid.UUID,
	forUpdate bool,
) (*entity.Event, error) {
	const op = "repository.event.GetByID"

	q := r.db.Select(_eventColumnts).
		From("events").
		Where(squirrel.Eq{"id": id})

	if forUpdate {
		q = q.Suffix("FOR UPDATE")
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var e entity.Event
	row := execOrDB(qe, r.db).QueryRow(ctx, sql, args...)
	err = row.Scan(
		&e.ID,
		&e.Title,
		&e.Description,
		&e.Date,
		&e.TotalSeats,
		&e.BookEventTTL,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrEventNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &e, nil
}

func (r *EventRepository) List(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
) ([]entity.Event, error) {
	const op = "repository.event.List"

	sql, args, err := r.db.Select(_eventColumnts).
		From("events").
		OrderBy("date ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := execOrDB(qe, r.db).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var events []entity.Event
	for rows.Next() {
		var e entity.Event
		if err = rows.Scan(
			&e.ID,
			&e.Title,
			&e.Description,
			&e.Date,
			&e.TotalSeats,
			&e.BookEventTTL,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		events = append(events, e)
	}

	return events, nil
}

func (r *EventRepository) CountBookings(
	ctx context.Context,
	qe pgxdriver.QueryExecuter,
	eventID uuid.UUID,
	status entity.Status,
) (int, error) {
	const op = "repository.event.CountBookings"

	sql, args, err := r.db.Select("COUNT(*)").
		From("bookings").
		Where("event_id = ? AND status = ?", eventID, status).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	var count int
	if err = execOrDB(qe, r.db).QueryRow(ctx, sql, args...).Scan(
		&count,
	); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return count, nil
}
