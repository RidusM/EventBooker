package entity

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Date          time.Time
	TotalSeats    int
	BookingTTLMin int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type EventWithStats struct {
	Event          Event
	BookedSeats    int
	ConfirmedSeats int
	FreeSeats      int
}
