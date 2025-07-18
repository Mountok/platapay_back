package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	wallet2 "production_wallet_back/internal/wallet"
	"production_wallet_back/models"
	"production_wallet_back/pkg/tronclient"
	"production_wallet_back/pkg/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) WithdrawTest(c *gin.Context) {
	// Создаем контекст с таймаутом в 5 минут
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
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
		// Проверяем, связана ли ошибка с недостатком TRX
		if err.Error() == "insufficient TRX balance for energy" {
			// Получаем дополнительную информацию
			trxBalance, trxErr := tempClient.GetTRXBalance(fromAddr)
			if trxErr == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":       fmt.Sprintf("insufficient TRX for gas fees: %v. Current TRX balance: %.6f. You need to add some TRX to your wallet for transaction fees.", err, trxBalance),
					"trx_balance": trxBalance,
					"need_trx":    true,
				})
				return
			}
		}
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
	_, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
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

func (h *Handler) CreateOrder(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	var req models.OrderCreateRequest
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	data := models.OrderQR{
		TelegramId: telegramId,
		QRCode:     req.QRLink,
		Summa:      req.Amount,
		Crypto:     req.Crypto,
		IsPaid:     false,
	}

	orderId, err := h.service.Wallet.CreateOrder(data)

	go func(input models.OrderQR) {
		logrus.Println("Отправка уведомления на почту")
		utils.SendMailMailjet(input.TelegramId, input.QRCode, input.Summa, input.Crypto)
		utils.SendMail(input.TelegramId, input.QRCode, input.Summa, input.Crypto)
	}(data)

	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"data":     "order created successfully",
		"order_id": orderId,
	})

}

func (h *Handler) PrivatKey(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	logrus.Infof("Get privat key by telegram_id: %s", telegramId)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}

	key, err := h.service.Wallet.GetPrivatKey(telegramId)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	logrus.Infof("private key get successfully")
	wrapOkJSON(c, map[string]interface{}{
		"key": key,
	})
}

func (h *Handler) PayQR(c *gin.Context) {
	logrus.Infof("PayQR")
	idString := c.Param("id")
	orderId, err := strconv.Atoi(idString)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid order_id")
		return
	}
	logrus.Infof("order id: %d", orderId)

	status, err := h.service.Wallet.PayQR(orderId)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"id":     orderId,
		"status": status,
	})
}

func (h *Handler) Orders(c *gin.Context) {

	orders, err := h.service.Wallet.GetOrders()
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"data": orders,
	})
}
func (h *Handler) OrdersHistory(c *gin.Context) {
	telegramId, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}

	logrus.Infof("Getting orders history for user with id: %d.", telegramId)

	orders, err := h.service.Wallet.OrdersHistory(telegramId)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	wrapOkJSON(c, map[string]interface{}{
		"data": orders,
	})

}

func (h *Handler) StateOrder(c *gin.Context) {
	_, err := GetTelegramId(c)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid telegram_id")
		return
	}
	idString := c.Param("id")
	OrderId, err := strconv.Atoi(idString)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, "invalid order_id")
		return
	}

	state, err := h.service.Wallet.GetOrderState(OrderId)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	wrapOkJSON(c, map[string]interface{}{
		"data": state,
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

// CheckUSDTBalance проверяет баланс USDT для указанного адреса с учетом виртуальных списаний
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

	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), req.USDTContract)

	// Получаем реальный баланс с блокчейна
	balance, err := tempClient.GetUSDTBalance(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check balance: %v", err)})
		return
	}

	// Получаем wallet_id по адресу
	wallet, err := h.service.Wallet.GetWalletByAddress(req.Address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet not found"})
		return
	}

	// Обновляем баланс в БД
	err = h.service.Wallet.UpdateBalance(wallet.WalletID, "USDT", balance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update balance in DB"})
		return
	}

	// Получаем сумму виртуальных списаний
	pendingSum, err := h.service.Wallet.SumPendingVirtualTransfers(wallet.WalletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pending sum"})
		return
	}

	available := balance - pendingSum

	// Округляем available_balance до 4 знаков после запятой
	available = math.Floor(available*10000) / 10000

	c.JSON(http.StatusOK, gin.H{
		"address":           req.Address,
		"real_balance":      balance,
		"pending_virtual":   pendingSum,
		"available_balance": available,
	})
}

// Endpoint для виртуального списания USDT (виртуальная оплата)
func (h *Handler) VirtualWithdraw(c *gin.Context) {
	var req struct {
		Address string  `json:"address"`
		Amount  float64 `json:"amount"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if req.Address == "" || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address and positive amount required"})
		return
	}
	// Получаем wallet_id по адресу
	wallet, err := h.service.Wallet.GetWalletByAddress(req.Address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet not found"})
		return
	}
	// Проверяем доступный баланс
	realBalance, err := h.service.Wallet.GetUSDTBalance(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get real balance"})
		return
	}
	pendingSum, err := h.service.Wallet.SumPendingVirtualTransfers(wallet.WalletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pending sum"})
		return
	}
	available := realBalance - pendingSum
	if req.Amount > available {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient available balance"})
		return
	}
	// Добавляем виртуальное списание
	err = h.service.Wallet.AddVirtualTransfer(wallet.WalletID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add virtual transfer"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "virtual withdraw successful"})
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

// EstimateRequiredTRX проверяет необходимое количество TRX для транзакции
func (h *Handler) EstimateRequiredTRX(c *gin.Context) {
	var req struct {
		FromAddress  string  `json:"from_address"`
		ToAddress    string  `json:"to_address"`
		Amount       float64 `json:"amount"`
		USDTContract string  `json:"usdt_contract"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.USDTContract == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usdt_contract address is required"})
		return
	}

	if req.FromAddress == "" || req.ToAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from_address and to_address are required"})
		return
	}

	// Создаем временный клиент с нужным адресом контракта
	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), req.USDTContract)

	// Получаем текущий баланс TRX
	currentTRX, err := tempClient.GetTRXBalance(req.FromAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get TRX balance: %v", err)})
		return
	}

	// Оцениваем необходимый TRX
	requiredTRX, err := tempClient.EstimateRequiredTRX(req.FromAddress, req.ToAddress, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to estimate required TRX: %v", err)})
		return
	}

	// Проверяем баланс USDT
	usdtBalance, err := tempClient.GetUSDTBalance(req.FromAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get USDT balance: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"from_address":    req.FromAddress,
		"to_address":      req.ToAddress,
		"amount":          req.Amount,
		"current_trx":     currentTRX,
		"required_trx":    requiredTRX,
		"current_usdt":    usdtBalance,
		"sufficient_trx":  currentTRX >= requiredTRX,
		"sufficient_usdt": usdtBalance >= req.Amount,
		"missing_trx":     requiredTRX - currentTRX,
	})
}

// WithdrawWithAutoTRX автоматически добавляет TRX для газа и выполняет перевод USDT
func (h *Handler) WithdrawWithAutoTRX(c *gin.Context) {
	// Создаем контекст с таймаутом в 5 минут
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Заменяем стандартный контекст на наш с таймаутом
	c.Request = c.Request.WithContext(ctx)

	var req struct {
		PrivKey       string  `json:"priv_key"`
		ToAddress     string  `json:"to_address"`
		Amount        float64 `json:"amount"`
		USDTContract  string  `json:"usdt_contract"`
		SystemPrivKey string  `json:"system_priv_key"` // Приватный ключ владельца для отправки TRX
		AutoAddTRX    bool    `json:"auto_add_trx"`    // Автоматически добавлять TRX
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

	fmt.Printf("=== WithdrawWithAutoTRX DEBUG ===\n")
	fmt.Printf("From Address: %s\n", fromAddr)
	fmt.Printf("To Address: %s\n", req.ToAddress)
	fmt.Printf("Amount: %.6f USDT\n", req.Amount)

	// 1. Проверяем баланс USDT отправителя
	fromBalance, err := tempClient.GetUSDTBalance(fromAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check USDT balance: %v", err)})
		return
	}

	fmt.Printf("Current USDT balance: %.6f\n", fromBalance)
	fmt.Printf("USDT Contract used: %s\n", req.USDTContract)
	fmt.Printf("From Address: %s\n", fromAddr)

	// Дополнительная проверка через другой endpoint
	fmt.Printf("=== Additional USDT Balance Check ===\n")
	checkReq := struct {
		Address      string `json:"address"`
		USDTContract string `json:"usdt_contract"`
	}{
		Address:      fromAddr,
		USDTContract: req.USDTContract,
	}

	checkJSON, _ := json.Marshal(checkReq)
	fmt.Printf("Check request: %s\n", string(checkJSON))

	if fromBalance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insufficient USDT balance: have %.6f, need %.6f", fromBalance, req.Amount)})
		return
	}

	// 2. Проверяем текущий баланс TRX
	trxBalance, err := tempClient.GetTRXBalance(fromAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check TRX balance: %v", err)})
		return
	}

	fmt.Printf("Current TRX balance: %.6f\n", trxBalance)

	// 3. Оцениваем необходимый TRX для транзакции
	requiredTRX, err := tempClient.EstimateRequiredTRX(fromAddr, req.ToAddress, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to estimate required TRX: %v", err)})
		return
	}

	fmt.Printf("Estimated required TRX: %.6f\n", requiredTRX)

	// 4. Если TRX недостаточно и включена автоматическая добавка
	if trxBalance < requiredTRX && req.AutoAddTRX && req.SystemPrivKey != "" {
		// Вычисляем сколько TRX нужно добавить (с запасом в 1 TRX)
		trxToAdd := requiredTRX - trxBalance + 1.0

		fmt.Printf("TRX insufficient. Adding %.6f TRX to address %s\n", trxToAdd, fromAddr)

		// Проверяем баланс системного кошелька
		systemAddr, _, _, err := wallet2.GetTronAddressFromPrivKey(req.SystemPrivKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid system private key: %v", err)})
			return
		}

		systemTrxBalance, err := tempClient.GetTRXBalance(systemAddr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check system TRX balance: %v", err)})
			return
		}

		fmt.Printf("System TRX balance: %.6f\n", systemTrxBalance)

		if systemTrxBalance < trxToAdd {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insufficient TRX in system wallet: have %.6f, need %.6f", systemTrxBalance, trxToAdd)})
			return
		}

		// 5. Отправляем TRX с системного кошелька на адрес пользователя
		fmt.Printf("Sending %.6f TRX from system wallet to user wallet...\n", trxToAdd)

		trxTxID, err := h.service.Wallet.SendTRXForGas(req.SystemPrivKey, fromAddr, trxToAdd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to send TRX for gas: %v", err)})
			return
		}

		fmt.Printf("TRX sent successfully: %s\n", trxTxID)

		// 6. Ждем обработки транзакции TRX
		fmt.Printf("Waiting for TRX transaction to be processed...\n")
		time.Sleep(time.Second * 10) // Ждем 10 секунд

		// 7. Проверяем новый баланс TRX
		newTrxBalance, err := tempClient.GetTRXBalance(fromAddr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check new TRX balance: %v", err)})
			return
		}

		fmt.Printf("New TRX balance: %.6f\n", newTrxBalance)

		if newTrxBalance < requiredTRX {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("TRX transfer completed but balance still insufficient: have %.6f, need %.6f", newTrxBalance, requiredTRX)})
			return
		}
	} else if trxBalance < requiredTRX {
		// Если TRX недостаточно и автоматическое добавление отключено
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        fmt.Sprintf("insufficient TRX for gas fees: have %.6f, need %.6f. Enable auto_add_trx to automatically add TRX.", trxBalance, requiredTRX),
			"trx_balance":  trxBalance,
			"required_trx": requiredTRX,
			"need_trx":     true,
		})
		return
	}

	// 8. Выполняем перевод USDT
	fmt.Printf("Executing USDT transfer...\n")
	txID, err := h.service.Wallet.WithdrawWithContract(req.PrivKey, req.ToAddress, req.Amount, req.USDTContract)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("USDT transfer failed: %v", err)})
		return
	}

	fmt.Printf("USDT transfer completed successfully: %s\n", txID)
	fmt.Printf("=== WithdrawWithAutoTRX COMPLETED ===\n")

	c.JSON(http.StatusOK, gin.H{
		"transfer_tx":      txID,
		"auto_trx_added":   req.AutoAddTRX && trxBalance < requiredTRX,
		"trx_added_amount": requiredTRX - trxBalance + 1.0,
		"message":          "Transfer completed successfully",
	})
}

// SendTRXForGasEndpoint отправляет TRX для оплаты газа
func (h *Handler) SendTRXForGasEndpoint(c *gin.Context) {
	var req struct {
		SystemPrivKey string  `json:"system_priv_key"` // Приватный ключ владельца
		ToAddress     string  `json:"to_address"`      // Адрес получателя
		Amount        float64 `json:"amount"`          // Количество TRX
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.SystemPrivKey == "" || req.ToAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system_priv_key and to_address are required"})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
		return
	}

	fmt.Printf("=== SendTRXForGas DEBUG ===\n")
	fmt.Printf("To Address: %s\n", req.ToAddress)
	fmt.Printf("Amount: %.6f TRX\n", req.Amount)

	// Проверяем баланс системного кошелька
	systemAddr, _, _, err := wallet2.GetTronAddressFromPrivKey(req.SystemPrivKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid system private key: %v", err)})
		return
	}

	tempClient := tronclient.NewTronHTTPClient(h.service.Wallet.GetAPIKey(), "")
	systemTrxBalance, err := tempClient.GetTRXBalance(systemAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check system TRX balance: %v", err)})
		return
	}

	fmt.Printf("System TRX balance: %.6f\n", systemTrxBalance)

	if systemTrxBalance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insufficient TRX in system wallet: have %.6f, need %.6f", systemTrxBalance, req.Amount)})
		return
	}

	// Отправляем TRX
	fmt.Printf("Sending %.6f TRX from system wallet to %s...\n", req.Amount, req.ToAddress)

	txID, err := h.service.Wallet.SendTRXForGas(req.SystemPrivKey, req.ToAddress, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to send TRX: %v", err)})
		return
	}

	fmt.Printf("TRX sent successfully: %s\n", txID)
	fmt.Printf("=== SendTRXForGas COMPLETED ===\n")

	c.JSON(http.StatusOK, gin.H{
		"tx_id":          txID,
		"amount":         req.Amount,
		"to_address":     req.ToAddress,
		"system_address": systemAddr,
		"message":        "TRX sent successfully for gas fees",
	})
}

// GetAddressFromPrivateKey получает адрес из приватного ключа
func (h *Handler) GetAddressFromPrivateKey(c *gin.Context) {
	var req struct {
		PrivKey string `json:"priv_key"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.PrivKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priv_key is required"})
		return
	}

	// Получаем адрес из приватного ключа
	fromAddr, fromAddrHex, _, err := wallet2.GetTronAddressFromPrivKey(req.PrivKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid private key: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"private_key": req.PrivKey,
		"address":     fromAddr,
		"address_hex": fromAddrHex,
	})
}

// AdminWalletsWithHistory возвращает список всех кошельков с историей списаний
func (h *Handler) AdminWalletsWithHistory(c *gin.Context) {
	// Проверка секретного ключа
	const adminSecret = "@$KC@f~vms|IXP#" // Задай свой ключ!
	secret := c.GetHeader("X-Admin-Secret")
	if secret != adminSecret {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	wallets, err := h.service.Wallet.GetAllWallets()
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, "failed to get wallets")
		return
	}

	type HistoryItem struct {
		ID          int64   `json:"id"`
		Amount      float64 `json:"amount"`
		Type        string  `json:"type"` // "virtual" или "real"
		Status      string  `json:"status"`
		ToAddress   string  `json:"to_address,omitempty"`
		TxHash      *string `json:"tx_hash,omitempty"`
		CreatedAt   string  `json:"created_at"`
		ProcessedAt *string `json:"processed_at,omitempty"`
	}

	var result []map[string]interface{}

	for _, w := range wallets {
		// Виртуальные списания
		virtuals, _ := h.service.Wallet.GetVirtualTransfersByWalletID(w.ID)
		// Реальные списания (только USDT)
		reals, _ := h.service.Wallet.GetTransactionsByWalletID(w.ID, "USDT")

		history := []HistoryItem{}
		for _, v := range virtuals {
			var processedAt *string
			if v.ProcessedAt != nil {
				str := v.ProcessedAt.Format("2006-01-02T15:04:05Z07:00")
				processedAt = &str
			}
			history = append(history, HistoryItem{
				ID:          v.ID,
				Amount:      v.Amount,
				Type:        "virtual",
				Status:      v.Status,
				CreatedAt:   v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				ProcessedAt: processedAt,
			})
		}
		for _, r := range reals {
			history = append(history, HistoryItem{
				ID:        r.ID,
				Amount:    r.Amount,
				Type:      "real",
				Status:    r.Status,
				ToAddress: r.ToAddress,
				TxHash:    r.TxHash,
				CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
		result = append(result, map[string]interface{}{
			"wallet_id": w.ID,
			"user_id":   w.UserID,
			"address":   w.Address,
			"history":   history,
		})
	}
	c.JSON(http.StatusOK, result)
}
