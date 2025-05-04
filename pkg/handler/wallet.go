package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	wallet2 "production_wallet_back/internal/wallet"
	"production_wallet_back/models"
)

func (h *Handler) WithdrawTest(c *gin.Context) {
	var req struct {
		PrivKey   string  `json:"priv_key"`
		ToAddress string  `json:"to_address"`
		Amount    float64 `json:"amount"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	txID, err := h.service.Wallet.Withdraw(req.PrivKey, req.ToAddress, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tx_id": txID})
}

func (h *Handler) CreateWallet(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}

	// Генерация кошелька
	wallet, err := wallet2.GenerateTRONWallet()
	if err != nil {
		logrus.Errorf("wallet generation failed: %s", err.Error())
		newErrorResponse(c, http.StatusInternalServerError, "wallet generation failed")
		return
	}

	// Сохраняем кошелек и получаем ID
	walletID, err := h.service.Wallet.CreateWallet(telegramId, wallet.PrivateKey, wallet.Address)
	if err != nil {
		logrus.Errorf("failed to save wallet: %s", err.Error())
		newErrorResponse(c, http.StatusInternalServerError, "failed to save wallet")
		return
	}

	logrus.Infof("wallet created with id: %d", walletID)
	// Инициализация баланса в 0 USDT
	err = h.service.Wallet.InitBalance(walletID, "USDT")
	if err != nil {
		logrus.Errorf("failed to initialize balance: %s", err.Error())
		newErrorResponse(c, http.StatusInternalServerError, "failed to initialize balance")
		return
	}

	wrapOkJSON(c, map[string]interface{}{
		"wallet_id":   walletID,
		"address":     wallet.Address,
		"private_key": wallet.PrivateKey,
	})
}

func (h *Handler) GetBalance(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	balance, err := h.service.Wallet.GetUSDTBalance("TDZVaZMrSuABymCsb2EgDkXjup6TNVxQ3w")
	if err != nil {
		logrus.Errorf("failed to get balance: %s", err.Error())
	}
	logrus.Infof("balance of TABpZBp8GeBZzDY5JcgLx4ApWJc4zcaBVQ - ", balance)
	balances, err := h.service.Wallet.GetBalance(telegramId)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, "failed to get balances")
		return
	}

	c.JSON(http.StatusOK, gin.H{"balances": balances})
}

func (h *Handler) Deposit(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	var input models.DepositInput
	if err := c.ShouldBindJSON(&input); err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid input")
		return
	}

	err = h.service.Wallet.Deposit(telegramId, input.TokenSymbol, input.Amount)

	if err != nil {
		logrus.Errorf("failed to deposit: %s", err.Error())
		newErrorResponse(c, http.StatusInternalServerError, "failed to deposit")
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"status": "deposite successful",
	})
}

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

func (h *Handler) Withdraw(c *gin.Context) {
	//var input models.WithdrawInput
	//telegramId, err := GetTelegramId(c)
	//if err != nil {
	//	newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
	//	return
	//}

	//err = h.service.Wallet.Withdraw(telegramId, input.ToAddress, input.TokenSymbol, input.Amount)
	//if err != nil {
	//	newErrorResponse(c, http.StatusBadRequest, err.Error())
	//}

	wrapOkJSON(c, map[string]interface{}{
		"status": "withdraw successful",
	})

}

func (h *Handler) GetTransactions(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	transactions, err := h.service.Wallet.GetTransactions(telegramId)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"data": transactions,
	})
}

const ownerAddress = "defd4194c48af65d285abc8"

func (h *Handler) Pay(c *gin.Context) {
	//telegramId, err := GetTelegramId(c)
	//if err != nil {
	//	newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
	//	return
	//}
	//var input models.DepositInput
	//err = h.service.Wallet.Withdraw(telegramId, ownerAddress, input.TokenSymbol, input.Amount)
	//if err != nil {
	//	newErrorResponse(c, http.StatusInternalServerError, err.Error())
	//	return
	//}

	// тут логика как пополняеться счет владельуа именно в кощельке нашем
	//
	//if err != nil {
	//	newErrorResponse(c, http.StatusInternalServerError, err.Error())
	//	return
	//}
	wrapOkJSON(c, map[string]interface{}{
		"status": "pay successful",
	})

}
