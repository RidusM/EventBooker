package handler

import (
	"errors"
	"net/http"

	"ebooker/internal/entity"

	"github.com/gin-gonic/gin"
)

func (h *BookingHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrInvalidData):
		h.respondError(c, http.StatusBadRequest, "invalid_data",
			"Invalid input data", err)
	case errors.Is(err, entity.ErrConflictingData):
		h.respondError(c, http.StatusConflict, "conflict",
			"Data conflict occurred", err)
	case errors.Is(err, entity.ErrEventNotFound):
		h.respondError(c, http.StatusNotFound, "not_found",
			"Event not found", err)
	case errors.Is(err, entity.ErrBookingNotFound):
		h.respondError(c, http.StatusNotFound, "not_found",
			"Booking not found", err)
	case errors.Is(err, entity.ErrUserNotFound):
		h.respondError(c, http.StatusNotFound, "not_found",
			"User not found", err)
	case errors.Is(err, entity.ErrNoSeatsLeft):
		h.respondError(c, http.StatusConflict, "no_seats_left",
			"No seats left on this event", err)
	case errors.Is(err, entity.ErrAlreadyPaid):
		h.respondError(c, http.StatusConflict, "already_paid",
			"This booking already paid", err)
	case errors.Is(err, entity.ErrBookingExpired):
		h.respondError(c, http.StatusConflict, "booking_expired",
			"This booking is expired", err)
	default:
		h.respondError(c, http.StatusInternalServerError, "internal_error",
			"Internal server error occurred", err)
	}
}
