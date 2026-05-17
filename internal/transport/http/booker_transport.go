package handler

import (
	"context"
	"net/http"

	"ebooker/internal/config"
	"ebooker/internal/entity"
	"ebooker/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/logger"
)

const _maxRequestBodySize = 32 << 20

type (
	EventService interface {
		Create(ctx context.Context, req service.CreateEventRequest) (*entity.Event, error)
		GetWithStats(ctx context.Context, id uuid.UUID) (*entity.EventWithStats, error)
		List(ctx context.Context) ([]entity.EventWithStats, error)
	}

	BookingService interface {
		Book(ctx context.Context, eventID, userID uuid.UUID) (*entity.Booking, error)
		Confirm(ctx context.Context, bookingID uuid.UUID) error
		GetBooking(ctx context.Context, id uuid.UUID) (*entity.Booking, error)
		ListByEvent(ctx context.Context, eventID uuid.UUID) ([]entity.Booking, error)
	}

	UserService interface {
		RegisterUser(ctx context.Context, req service.RegisterUserRequest) (*entity.User, error)
		GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, error)
		LinkTelegramByToken(ctx context.Context, token string, chatID *int64) error
		GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
		GetUserByTelegramID(ctx context.Context, chatID *int64) (*entity.User, error)
		LoginByEmail(ctx context.Context, email string) (*entity.User, error)
		List(ctx context.Context) ([]entity.User, error)
	}
)

type BookingHandler struct {
	eventSvc   EventService
	bookingSvc BookingService
	userSvc    UserService
	log        logger.Logger
	router     *gin.Engine

	botCfg config.TG
}

func NewBookingHandler(
	eventSvc EventService,
	bookingSvc BookingService,
	userSvc UserService,
	log logger.Logger,
	botCfg config.TG,
) *BookingHandler {
	h := &BookingHandler{
		eventSvc:   eventSvc,
		bookingSvc: bookingSvc,
		userSvc:    userSvc,
		log:        log,
		botCfg:     botCfg,
	}

	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, _maxRequestBodySize)
	})

	router.Use(h.requestIDMiddleware())
	router.Use(h.loggingMiddleware())
	router.Use(h.baseCORSMiddleware())
	router.Use(gin.Recovery())

	h.router = router

	h.router.LoadHTMLGlob("web/*.html")
	h.router.Static("/static", "./web")

	h.setupRoutes()

	return h
}

func (h *BookingHandler) Engine() *gin.Engine {
	return h.router
}
