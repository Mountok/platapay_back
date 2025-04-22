package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"production_wallet_back/models"
)

// Конвертация валюты (RUB --> USDT ). Надо передать в теле запроса {amount:int,from:int,to:int}
func (h *Handler) Convert(c *gin.Context) {
	var req models.ConvertRequest
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err, res := h.service.Wallet.Convert(req)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	wrapOkJSON(c, map[string]interface{}{
		"data": res,
	})
}
