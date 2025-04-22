package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"production_wallet_back/pkg/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (h *Handler) InitRoute() *gin.Engine {
	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://meek-medovik-c85f42.netlify.app", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Ping)
		auth.GET("/me")
	}

	api := router.Group("/api")
	{
		wallet := api.Group("/wallet")
		{
			wallet.GET("/balance")
			wallet.POST("/deposit")
			wallet.POST("/withdraw")
			wallet.GET("/transactions")
			wallet.POST("/convert", h.Convert)
			wallet.POST("/pay")
		}

	}
	return router
}
