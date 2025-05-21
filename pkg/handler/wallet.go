package handler

import (
	"context"
	"fmt"
	"net/http"
	wallet2 "production_wallet_back/internal/wallet"
	"production_wallet_back/models"
	"production_wallet_back/pkg/tronclient"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) WithdrawTest(c *gin.Context) {
	// Создаем контекст с таймаутом в 3 минуты
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()

	// Заменяем стандартный контекст на наш с таймаутом
	c.Request = c.Request.WithContext(ctx)

	var req struct {
		PrivKey      string  `json:"priv_key"`
		ToAddress    string  `json:"to_address"`
		Amount       float64 `json:"amount"`
		USDTContract string  `json:"usdt_contract"` // Адрес вашего контракта USDT
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.USDTContract == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usdt_contract address is required"})
		return
	}

	// Создаем временный клиент с правильным адресом контракта
	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), req.USDTContract)

	// Получаем адрес отправителя из приватного ключа
	fromAddr, _, _, err := wallet2.GetTronAddressFromPrivKey(req.PrivKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid private key: %v", err)})
		return
	}

	// Проверяем баланс USDT отправителя
	fromBalance, err := tempClient.GetUSDTBalance(fromAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check balance: %v", err)})
		return
	}

	fmt.Printf("Checking balance for address %s: %.6f USDT\n", fromAddr, fromBalance)

	if fromBalance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insufficient USDT balance: have %.6f, need %.6f", fromBalance, req.Amount)})
		return
	}

	// В случае с USDT мы делаем прямой transfer без approve
	// Так как это ваш собственный контракт и в нем не требуется approve
	txID, err := h.service.Wallet.WithdrawWithContract(req.PrivKey, req.ToAddress, req.Amount, req.USDTContract)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("transfer failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transfer_tx": txID,
	})
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
func (h *Handler) GetWallet(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	wallet, err := h.service.Wallet.GetWallet(telegramId)
	if err != nil {
		logrus.Errorf("failed to get wallet: %s", err.Error())
		newErrorResponse(c, http.StatusInternalServerError, "failed to get wallet")
		return
	}

	wrapOkJSON(c, map[string]interface{}{
		"data": wallet,
	})
}
func (h *Handler) GetBalance(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	balance, err := h.service.Wallet.GetUSDTBalance("THJW81cGM7QAkYu2dkN7LxxNK3cDWKK6ac")
	if err != nil {
		logrus.Errorf("failed to get balance: %s", err.Error())
	}
	logrus.Infof("balance of THJW81cGM7QAkYu2dkN7LxxNK3cDWKK6ac - ", balance)
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

// CheckUSDTBalance проверяет баланс USDT для указанного адреса
func (h *Handler) CheckUSDTBalance(c *gin.Context) {
	var req struct {
		Address      string `json:"address"`
		USDTContract string `json:"usdt_contract"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.USDTContract == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usdt_contract address is required"})
		return
	}

	if req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
		return
	}

	// Создаем временный клиент с нужным адресом контракта
	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), req.USDTContract)

	balance, err := tempClient.GetUSDTBalance(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check balance: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address":  req.Address,
		"balance":  balance,
		"contract": req.USDTContract,
	})
}

// CheckTransactionStatus проверяет статус транзакции
func (h *Handler) CheckTransactionStatus(c *gin.Context) {
	var req struct {
		TxID string `json:"tx_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.TxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tx_id is required"})
		return
	}

	// Создаем временный клиент
	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), "")

	status, err := tempClient.GetTransactionStatus(req.TxID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check transaction status: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tx_id":  req.TxID,
		"status": status,
	})
}

// CheckTRXBalance проверяет баланс TRX для адреса
func (h *Handler) CheckTRXBalance(c *gin.Context) {
	var req struct {
		Address string `json:"address"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
		return
	}

	// Создаем временный клиент
	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), "")

	balance, err := tempClient.GetTRXBalance(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check TRX balance: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address": req.Address,
		"balance": balance,
	})
}
