package handler

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"production_wallet_back/models"
	"strconv"
)

func (h *Handler) Login(c *gin.Context) {
	var input models.User

	if err := c.BindJSON(&input); err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Authorization.GetUserByTelegramId(input.TelegramID)
	if err != nil {
		// Если пользователь не найден — создаём
		if errors.Is(err, sql.ErrNoRows) {
			id, err := h.service.Authorization.CreateUser(input)
			if err != nil {
				newErrorResponse(c, http.StatusInternalServerError, "cannot create user")
				return
			}

			input.ID = id
			c.JSON(http.StatusOK, input)
			return
		}

		newErrorResponse(c, http.StatusInternalServerError, "something went wrong")
		return
	}

	// Пользователь уже есть
	wrapOkJSON(c, map[string]interface{}{
		"user": user,
	})
}

func (h *Handler) GetMe(c *gin.Context) {
	telegramIdStr := c.Query("telegram_id")
	if telegramIdStr == "" {
		newErrorResponse(c, http.StatusBadRequest, "telegram_id is required")
		return
	}
	telegramId, err := strconv.ParseInt(telegramIdStr, 10, 64)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "telegram_id is required")
		return
	}
	user, err := h.service.Authorization.GetUserByTelegramId(telegramId)
	if err != nil {
		newErrorResponse(c, http.StatusUnauthorized, "something went wrong")
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"user": user,
	})
}
