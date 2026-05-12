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
		events.POST("/events", h.CreateEvent)
		events.GET("/events", h.ListEvents)
		events.GET("/events/:id", h.GetEvent)
		events.POST("/events/:id/book", h.BookEvent)
		events.POST("/events/:id/confirm", h.ConfirmBooking)
	}

	users := h.router.Group("/users")
	{
		users.POST("", h.RegisterUser)
		users.POST("/:user_id/link-token", h.GenerateLinkToken)
		users.GET("/users", h.ListUsers)
	}

	h.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
