package entity

import "github.com/google/uuid"

type CancelledNotification struct {
	BookingID  uuid.UUID
	UserID     uuid.UUID
	UserName   string
	UserEmail  string
	TelegramID *int64
	EventID    uuid.UUID
	EventTitle string
	EventDate  string
}
