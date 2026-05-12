package entity

import "errors"

var (
	ErrConflictingData = errors.New("conflicting data")
	ErrInvalidData     = errors.New("invalid data")
	ErrNoSeatsLeft     = errors.New("no seats left")
	ErrAlreadyPaid     = errors.New("booking already paid")
	ErrBookingExpired  = errors.New("booking expired")
	ErrBookingNotFound = errors.New("booking not found")
	ErrEventNotFound   = errors.New("event not found")
	ErrUserNotFound    = errors.New("user not found")
)
