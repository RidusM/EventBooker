package entity

import (
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	UserID    uuid.UUID
	Status    Status
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (b *Booking) IsExpired() bool {
	return b.Status == StatusPending && time.Now().UTC().After(b.ExpiresAt)
}
