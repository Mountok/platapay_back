package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Error struct {
	Message string `json:"message"`
}

func newErrorResponse(c *gin.Context, statusCode int, message string) {
	logrus.Error(message)
	c.AbortWithStatusJSON(statusCode, Error{Message: message})
}

func wrapOkJSON(c *gin.Context, response map[string]interface{}) {
	c.JSON(http.StatusOK, response)
}

func GetTelegramId(c *gin.Context) (int64, error) {
	telegramIDStr := c.GetHeader("X-Telegram-ID")
	if telegramIDStr == "" {
		return 0, nil
	}
	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	return telegramID, err
}
