package handler

import (
	"net/http"

	_ "ebooker/docs" // required for Swagger

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Event Booker Service API
// @version         1.0
// @description     API for work with image processor
// @termsOfService  http://swagger.io/terms/
// @contact.name    RidusM
// @contact.email   stormkillpeople@gmail.com
// @license.name    MIT-0
// @license.url     https://github.com/aws/mit-0
// @host            localhost:8080
// @BasePath        /
func (h *BookingHandler) setupRoutes() {
	h.router.GET("/health", h.Health)

	events := h.router.Group("/events")
	{
		events.POST("", h.CreateEvent)
		events.GET("", h.ListEvents)
		events.GET("/:id", h.GetEvent)
		events.POST("/:id/book", h.BookEvent)
	}

	bookings := h.router.Group("/bookings")
	{
		bookings.POST("/confirm", h.ConfirmBooking)
	}

	users := h.router.Group("/users")
	{
		users.POST("/:user_id/link-token", h.GenerateLinkToken)
		users.GET("/:id", h.GetUser)
		users.GET("", h.ListUsers)
	}

	auth := h.router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/sign-up", h.RegisterUser)
	}

	h.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
