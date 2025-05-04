package service

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"production_wallet_back/models"
	"production_wallet_back/pkg/cache"
	"production_wallet_back/pkg/repository"
	"production_wallet_back/pkg/tronclient"
	"strings"
)

type WalletService struct {
	repos      repository.Wallet
	tronClient *tronclient.TronHTTPClient
}

func NewWalletService(repos repository.Wallet, tronclient *tronclient.TronHTTPClient) *WalletService {
	return &WalletService{
		repos:      repos,
		tronClient: tronclient,
	}
}

func (s *WalletService) CreateWallet(userID int64, privKey, address string) (int64, error) {
	return s.repos.CreateWallet(userID, privKey, address)
}

func (s *WalletService) InitBalance(walletID int64, tokenSymbol string) error {
	return s.repos.InitBalance(walletID, tokenSymbol)
}

func (s *WalletService) GetBalance(telegramId int64) ([]models.Balance, error) {
	return s.repos.GetBalances(telegramId)
}
func (s *WalletService) GetUSDTBalance(address string) (float64, error) {
	return s.tronClient.GetUSDTBalance(address)
}
func (s *WalletService) Deposit(telegramId int64, tokenSymbol string, amount float64) error {
	return s.repos.Deposit(telegramId, tokenSymbol, amount)
}

func (s *WalletService) Withdraw(privKey string, toAddress string, amount float64) (string, error) {
	txID, err := s.tronClient.SendUSDT(privKey, toAddress, amount)
	if err != nil {
		return "", err
	}
	return txID, nil
}

//func (s *WalletService) Withdraw(telegramId int64, ToAddress string, tokenSymbol string, amount float64) error {
//	balance, err := s.repos.GetBalances(telegramId)
//	logrus.Infof(fmt.Sprintf("balances %+v", balance))
//	if err != nil {
//		return err
//	}
//	if balance[0].Amount < amount {
//		return errors.New("insufficient funds")
//	}
//	err = s.repos.WithdrawBalance(balance[0].WalletID, amount, tokenSymbol)
//	if err != nil {
//		return err
//	}
//	err = s.repos.CreateTransaction(balance[0].WalletID, ToAddress, tokenSymbol, amount, "pending", "")
//	if err != nil {
//		return errors.New(fmt.Sprintf("Failed to save transaction: %s", err))
//	}
//	return nil
//}

func (s *WalletService) GetTransactions(telegramId int64) ([]models.Transaction, error) {
	return s.repos.GetTransactions(telegramId)
}

func (s *WalletService) Pay(telegramId int64, tokenSymbol string, amount float64) error {

	err := s.repos.Pay(telegramId, tokenSymbol, amount)
	if err != nil {
		return err
	}
	// Далле логтка для оплаты qr и транзакции в сети
	return err
}

func (s *WalletService) Convert(convertReq models.ConvertRequest) (error, models.ConvertResponse) {
	var response models.ConvertResponse
	from := strings.ToLower(convertReq.From)
	to := strings.ToLower(convertReq.To)
	if from == "" || to == "" || convertReq.Amount <= 0 {
		return errors.New("Неверно переданы данные в тело запроса для конвертации"), response
	}

	key := currencyID(convertReq.To) + "_" + convertReq.From

	// Попробуем получить курс из кэша
	if rate, found := cache.GetCachedRate(key); found {
		converted := convertReq.Amount / rate
		response = models.ConvertResponse{
			ConvertedAmount: converted,
			Currency:        strings.ToUpper(convertReq.To),
			Message:         fmt.Sprintf("Переведите %.2f на адрес  Tx..", converted),
			Wallet:          "Tx...",
		}
		return nil, response
	}
	// Если в кэше нет — запрос к CoinGecko
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + currencyID(to) + "&vs_currencies=" + from
	client := resty.New()

	log.Println("Запрос к API CoinGecko:", url)

	resp, err := client.R().
		SetHeader("x-cg-demo-api-key", "CG-wmi7LpR5B84uad7kPFE1knYa").
		SetHeader("Accept", "application/json").
		SetResult(map[string]map[string]float64{}).
		Get(url)

	if err != nil || resp.IsError() {
		log.Println("Ошибка при получении курса:", err)
		log.Println("Ответ от API:", resp)
		return errors.New("Не удалось получить курс"), response
	}

	data := *resp.Result().(*map[string]map[string]float64)
	rate := data[currencyID(to)][from]

	if rate == 0 {
		return errors.New("Некорректный курс"), response
	}

	cache.SetCachedRate(key, rate)

	converted := convertReq.Amount / rate

	response = models.ConvertResponse{
		ConvertedAmount: converted,
		Currency:        strings.ToUpper(convertReq.To),
		Message:         fmt.Sprintf("Переведите %.2f на адрес  Tx..", converted),
		Wallet:          "Tx...",
	}

	return nil, response
}

func currencyID(symbol string) string {
	switch strings.ToLower(symbol) {
	case "usdt":
		return "tether"
	case "btc":
		return "bitcoin"
	case "eth":
		return "ethereum"
	default:
		return strings.ToLower(symbol)
	}
}
