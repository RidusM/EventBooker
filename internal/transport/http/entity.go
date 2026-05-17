// nolint: revive, staticcheck
package handler

import (
	"time"

	"github.com/google/uuid"
)

const (
	msgRegisteredViaEmail = "Registered via Email"
	msgLinkTokenGenerated = "Click the link in Telegram to link your account"
	linkTokenExpiration   = "1 hour"
)

// swagger:model CreateEventRequest
type CreateEventRequest struct {
	Title          string    `json:"title"           binding:"required"        example:"Winter tour SABATON"`
	Description    string    `json:"description"     binding:"required"        example:"SABATON income with tour in Russia"`
	Date           time.Time `json:"date"            binding:"required"        example:"2025-10-29T19:00:00Z"`
	TotalSeats     int       `json:"total_seats"     binding:"required,gte=0"  example:"350"`
	BookingTTLMins int       `json:"booking_ttl_min" binding:"gte=3,lte=10080" example:"15"`
}

// swagger:model CreateEventResponse
type CreateEventResponse struct {
	ID             uuid.UUID `json:"id"              example:"550e8400-e29b-41d4-a716-446655440001"`
	Title          string    `json:"title"           example:"Winter tour SABATON"`
	Description    string    `json:"description"     example:"SABATON income with tour in Russia"`
	Date           time.Time `json:"date"            example:"2025-10-29T19:00:00Z"`
	TotalSeats     int       `json:"total_seats"     example:"350"`
	BookingTTLMins int       `json:"booking_ttl_min" example:"15"`
	CreatedAt      time.Time `json:"created_at"      example:"2023-10-26T10:00:00Z"`
	UpdatedAt      time.Time `json:"updated_at"      example:"2023-10-26T10:00:00Z"`
}

// swagger:model EventWithStatsResponse
type EventWithStatsResponse struct {
	ID             uuid.UUID `json:"id"              example:"550e8400-e29b-41d4-a716-446655440001"`
	Title          string    `json:"title"           example:"Winter tour SABATON"`
	Description    string    `json:"description"     example:"SABATON income with tour in Russia"`
	Date           time.Time `json:"date"            example:"2025-10-29T19:00:00Z"`
	TotalSeats     int       `json:"total_seats"     example:"350"`
	BookedSeats    int       `json:"booked_seats"    example:"127"`
	AvailableSeats int       `json:"available_seats" example:"223"`
	BookingTTLMins int       `json:"booking_ttl_min" example:"15"`
	CreatedAt      time.Time `json:"created_at"      example:"2023-10-26T10:00:00Z"`
	UpdatedAt      time.Time `json:"updated_at"      example:"2023-10-26T10:00:00Z"`
}

// swagger:model ListEventsResponse
type ListEventsResponse struct {
	Events []EventWithStatsResponse `json:"events"`
	Total  int                      `json:"total"  example:"42"`
}

// swagger:model BookEventRequest
type BookEventRequest struct {
	UserID string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440003" binding:"required,uuid"`
}

// swagger:model BookEventResponse
type BookEventResponse struct {
	ID        uuid.UUID `json:"id"         example:"550e8400-e29b-41d4-a716-446655440002"`
	EventID   uuid.UUID `json:"event_id"   example:"550e8400-e29b-41d4-a716-446655440001"`
	UserID    uuid.UUID `json:"user_id"    example:"550e8400-e29b-41d4-a716-446655440003"`
	Status    string    `json:"status"     example:"pending"`
	ExpiresAt time.Time `json:"expires_at" example:"2023-10-27T10:10:00Z"`
	CreatedAt time.Time `json:"created_at" example:"2023-10-27T10:00:00Z"`
}

// swagger:model ConfirmBookingRequest
type ConfirmBookingRequest struct {
	BookingID string `json:"booking_id" example:"550e8400-e29b-41d4-a716-446655440002" binding:"required,uuid"`
}

// swagger:model ConfirmBookingResponse
type ConfirmBookingResponse struct {
	Status string `json:"status" example:"confirmed"`
}

// swagger:model BookingResponse
type BookingResponse struct {
	ID        string    `json:"id"         example:"550e8400-e29b-41d4-a716-446655440002"`
	EventID   string    `json:"event_id"   example:"550e8400-e29b-41d4-a716-446655440001"`
	UserID    string    `json:"user_id"    example:"550e8400-e29b-41d4-a716-446655440003"`
	Status    string    `json:"status"     example:"pending"`
	ExpiresAt time.Time `json:"expires_at" example:"2023-10-27T10:10:00Z"`
	CreatedAt time.Time `json:"created_at" example:"2023-10-27T10:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-10-27T10:00:00Z"`
}

// swagger:model ListBookingsResponse
type ListBookingsResponse struct {
	Bookings []BookingResponse `json:"bookings"`
	Total    int               `json:"total"    example:"15"`
}

// swagger:model RegisterUserRequest
type RegisterUserRequest struct {
	Name  string `json:"name"  binding:"required,min=1,max=100" example:"John Doe"`
	Email string `json:"email" binding:"email"                  example:"john.doe@example.com"`
}

// swagger:model RegisterUserResponse
type RegisterUserResponse struct {
	ID         uuid.UUID `json:"id"                    example:"550e8400-e29b-41d4-a716-446655440003"`
	Name       string    `json:"name"                  example:"Ivan Petrov"`
	Email      string    `json:"email"                 example:"ivan@example.com"`
	TelegramID *int64    `json:"telegram_id,omitempty" example:"123456789"`
	CreatedAt  time.Time `json:"created_at"            example:"2023-10-26T10:00:00Z"`
	UpdatedAt  time.Time `json:"updated_at"            example:"2023-10-26T10:00:00Z"`
}

// swagger:model UserRegisteredResponse
type UserRegisteredResponse struct {
	UserID  uuid.UUID `json:"user_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440003"`
	Message string    `json:"message"                         example:"Registered via Email"`
}

// swagger:model LoginRequest
type LoginRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

// swagger:model LoginResponse
type LoginResponse struct {
	UserID  uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	Name    string    `json:"name"    example:"Ivan Petrov"`
	Email   string    `json:"email"   example:"ivan@example.com"`
	Message string    `json:"message" example:"Logged in successfully"`
}

// swagger:model UserResponse
type UserResponse struct {
	ID         uuid.UUID `json:"id"                    example:"550e8400-e29b-41d4-a716-446655440003"`
	Name       string    `json:"name"                  example:"Ivan Petrov"`
	Email      string    `json:"email"                 example:"ivan@example.com"`
	TelegramID *int64    `json:"telegram_id,omitempty" example:"123456789"`
	CreatedAt  time.Time `json:"created_at"            example:"2023-10-26T10:00:00Z"`
	UpdatedAt  time.Time `json:"updated_at"            example:"2023-10-26T10:00:00Z"`
}

// swagger:model LinkTokenResponse
type LinkTokenResponse struct {
	Token     string `json:"token"      binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Link      string `json:"link"       binding:"required" example:"https://t.me/mybot?start=abc123"`
	Message   string `json:"message"                       example:"Click the link in Telegram to link your account"`
	ExpiresIn string `json:"expires_in" binding:"required" example:"1 hour"`
}

// swagger:model ListUsersResponse
type ListUsersResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total" example:"128"`
}

// swagger:model ErrorResponse
type ErrorResponse struct {
	Error   string `json:"error"             example:"event not found"`
	Code    string `json:"code,omitempty"    example:"not_found"`
	Details string `json:"details,omitempty" example:"event with id 123 does not exist"`
}

// swagger:model SuccessResponse
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// swagger:model HealthResponse
type HealthResponse struct {
	Status string    `json:"status" example:"ok"`
	Time   time.Time `json:"time"   example:"2026-05-08T06:04:15Z"`
}
