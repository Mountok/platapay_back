package middleware

import (
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		telegramID := c.GetHeader("X-Telegram-ID")
		if telegramID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "telegram_id is required in 'X-Telegram-ID' header"})
			c.Abort()
			return
		}
		logrus.Infof("AuthMiddleware: telegram_id: %s", telegramID)
		c.Next()
	}
}
