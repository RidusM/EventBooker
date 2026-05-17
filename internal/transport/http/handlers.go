// nolint: revive,staticcheck
package handler

import (
	"fmt"
	"net/http"
	"time"

	"ebooker/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @Summary Create a new event
// @Description Creates a new event with specified parameters. Returns the created event with generated ID.
// @Tags Events
// @Accept json
// @Produce json
// @Param request body CreateEventRequest true "Event creation data"
// @Success 201 {object} CreateEventResponse "Event created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or validation error"
// @Failure 409 {object} ErrorResponse "Conflict - event with same data exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /events [post]
func (h *BookingHandler) CreateEvent(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", "Invalid JSON format", err)
		return
	}

	serviceReq := service.CreateEventRequest{
		Title:         req.Title,
		Description:   req.Description,
		Date:          req.Date,
		TotalSeats:    req.TotalSeats,
		BookingTTLMin: req.BookingTTLMins,
	}

	event, err := h.eventSvc.Create(ctx, serviceReq)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusCreated, event)
}

// @Summary List all events with statistics
// @Description Returns a list of all events with booking statistics (booked, confirmed, available seats)
// @Tags Events
// @Accept json
// @Produce json
// @Success 200 {object} ListEventsResponse "List of events with stats"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /events [get]
func (h *BookingHandler) ListEvents(c *gin.Context) {
	ctx := c.Request.Context()

	events, err := h.eventSvc.List(ctx)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, events)
}

// @Summary Get event details with statistics
// @Description Returns detailed information about a specific event including booking statistics
// @Tags Events
// @Accept json
// @Produce json
// @Param id path string true "Event UUID" format(uuid)
// @Success 200 {object} EventWithStatsResponse "Event details with stats"
// @Failure 400 {object} ErrorResponse "Invalid event ID format"
// @Failure 404 {object} ErrorResponse "Event not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /events/{id} [get]
func (h *BookingHandler) GetEvent(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_id", "Invalid event ID format", err)
		return
	}

	event, err := h.eventSvc.GetWithStats(ctx, id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, event)
}

// @Summary Book an event seat
// @Description Reserve a seat for a specific event. Creates a pending booking that requires confirmation before TTL expires.
// @Tags Bookings
// @Accept json
// @Produce json
// @Param id path string true "Event UUID" format(uuid)
// @Param request body BookEventRequest true "Booking details"
// @Success 200 {object} BookEventResponse "Booking created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or UUID format"
// @Failure 404 {object} ErrorResponse "Event or user not found"
// @Failure 409 {object} ErrorResponse "No seats available or booking conflict"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /events/{id}/book [post]
func (h *BookingHandler) BookEvent(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	eventID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_event_id", "Invalid eventID format", err)
		return
	}

	var req BookEventRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", "Invalid JSON format", err)
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_user_id", "Invalid userID format", err)
		return
	}

	booking, err := h.bookingSvc.Book(ctx, eventID, userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, booking)
}

// @Summary Confirm a pending booking
// @Description Finalize a reservation by confirming a pending booking. Must be called before TTL expires.
// @Tags Bookings
// @Accept json
// @Produce json
// @Param request body ConfirmBookingRequest true "Confirmation details"
// @Success 201 {object} ConfirmBookingResponse "Booking confirmed successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or booking ID format"
// @Failure 404 {object} ErrorResponse "Booking not found"
// @Failure 409 {object} ErrorResponse "Booking already confirmed, cancelled, or expired"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /bookings/confirm [post]
func (h *BookingHandler) ConfirmBooking(c *gin.Context) {
	ctx := c.Request.Context()

	var req ConfirmBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", "Invalid JSON format", err)
		return
	}

	bookingID, err := uuid.Parse(req.BookingID)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_booking_id", "Invalid bookingID format", err)
		return
	}

	if err = h.bookingSvc.Confirm(ctx, bookingID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	response := SuccessResponse{
		Message: "confirmed",
	}

	h.respondJSON(c, http.StatusCreated, response)
}

// @Summary Register a new user
// @Description Registers a user to receive notifications via Email or Telegram. At least one contact method is required.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body RegisterUserRequest true "User registration data"
// @Success 201 {object} RegisterUserResponse "User registered successfully"
// @Failure 400 {object} ErrorResponse "Invalid input data - email or telegram_id required"
// @Failure 409 {object} ErrorResponse "User with this email or telegram_id already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/sign-up [post]
func (h *BookingHandler) RegisterUser(c *gin.Context) {
	ctx := c.Request.Context()

	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_input", "Validation failed", err)
		return
	}

	serviceReq := service.RegisterUserRequest{
		Name:  req.Name,
		Email: req.Email,
	}

	user, err := h.userSvc.RegisterUser(ctx, serviceReq)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	response := UserRegisteredResponse{
		UserID:  user.ID,
		Message: msgRegisteredViaEmail,
	}

	h.respondJSON(c, http.StatusCreated, response)
}

// @Summary User login by email
// @Description Authenticate user by email. Returns user data on success.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Login successful"
// @Failure 400 {object} ErrorResponse "Invalid email format"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *BookingHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", "Invalid email format", err)
		return
	}

	user, err := h.userSvc.LoginByEmail(ctx, req.Email)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	response := LoginResponse{
		UserID:  user.ID,
		Name:    user.Name,
		Email:   user.Email,
		Message: "Logged in successfully",
	}

	h.respondJSON(c, http.StatusOK, response)
}

// @Summary Get user by ID
// @Description Returns user data by ID. Used for checking Telegram link status.
// @Tags Users
// @Produce json
// @Param id path string true "User UUID" format(uuid)
// @Success 200 {object} UserResponse "User data"
// @Failure 400 {object} ErrorResponse "Invalid ID format"
// @Failure 404 {object} ErrorResponse "User not found"
// @Router /users/{id} [get]
func (h *BookingHandler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID format", err)
		return
	}

	user, err := h.userSvc.GetUserByID(ctx, id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	response := UserResponse{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		TelegramID: user.TelegramID,
		CreatedAt:  user.CreatedAt,
	}

	h.respondJSON(c, http.StatusOK, response)
}

// @Summary Generate Telegram Link Token
// @Description Generates a one-time token to link the user's account with Telegram bot. Token expires in 1 hour.
// @Tags Users
// @Accept json
// @Produce json
// @Param user_id path string true "User UUID" format(uuid)
// @Success 200 {object} LinkTokenResponse "Link token and instruction"
// @Failure 400 {object} ErrorResponse "Invalid user ID format"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /users/{user_id}/link-token [post]
func (h *BookingHandler) GenerateLinkToken(c *gin.Context) {
	ctx := c.Request.Context()

	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_id", "Invalid User ID", err)
		return
	}

	token, err := h.userSvc.GenerateLinkToken(ctx, userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	linkURL := fmt.Sprintf("https://t.me/%s?start=%s", h.botCfg.Alias, token)

	response := LinkTokenResponse{
		Token:     token,
		Link:      linkURL,
		Message:   msgLinkTokenGenerated,
		ExpiresIn: linkTokenExpiration,
	}

	h.respondJSON(c, http.StatusOK, response)
}

// @Summary List all users
// @Description Retrieve a list of all registered users. For administrative purposes.
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} ListUsersResponse "List of users"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /users [get]
func (h *BookingHandler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()

	events, err := h.userSvc.List(ctx)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, events)
}

// @Summary Health check endpoint
// @Description Return service status and current timestamp. No authentication required.
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Router /health [get]
func (h *BookingHandler) Health(c *gin.Context) {
	response := HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC(),
	}
	h.respondJSON(c, http.StatusOK, response)
}

func (h *BookingHandler) respondJSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

func (h *BookingHandler) respondError(c *gin.Context, status int, code, message string, err error) {
	response := ErrorResponse{
		Error: message,
		Code:  code,
	}
	if err != nil {
		response.Details = err.Error()
	}
	h.respondJSON(c, status, response)
}
