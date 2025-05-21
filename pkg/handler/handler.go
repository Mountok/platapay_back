package handler

import (
	"production_wallet_back/pkg/middleware"
	"production_wallet_back/pkg/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) InitRoute() *gin.Engine {
	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://platapay.ru"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Telegram-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.GET("/me", h.GetMe)
	}

	api := router.Group("/api")
	{
		wallet := api.Group("/wallet", middleware.AuthMiddleware())
		{
			wallet.GET("/", h.GetWallet)
			wallet.POST("/create", h.CreateWallet)
			wallet.GET("/balance", h.GetBalance)
			wallet.POST("/deposit", h.Deposit)
			wallet.POST("/withdraw", h.Withdraw)
			wallet.POST("/withdraw/test", h.WithdrawTest)
			wallet.GET("/transactions", h.GetTransactions)
			wallet.POST("/convert", h.Convert)
			wallet.POST("/pay", h.Pay)
			wallet.POST("/check-balance", h.CheckUSDTBalance)
			wallet.POST("/check-tx", h.CheckTransactionStatus)
			wallet.POST("/check-trx-balance", h.CheckTRXBalance)

		}

	}
	return router
}
