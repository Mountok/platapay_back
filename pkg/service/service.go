package service

import (
	"production_wallet_back/models"
	"production_wallet_back/pkg/repository"
	"production_wallet_back/pkg/tronclient"
)

type Authorization interface {
	GetUserByTelegramId(telegramId int64) (models.User, error)
	CreateUser(models.User) (int64, error)
}
type Wallet interface {
	CreateWallet(userID int64, privKey, address string) (int64, error)
	InitBalance(walletID int64, tokenSymbol string) error
	GetBalance(telegramId int64) ([]models.Balance, error)
	GetUSDTBalance(address string) (float64, error)
	Deposit(telegramId int64, tokenSymbol string, amount float64) error
	Withdraw(privKey string, toAddress string, amount float64) (string, error)
	//Withdraw(telegramId int64, ToAddress string, tokenSymbol string, amount float64) error
	GetTransactions(telegramId int64) ([]models.Transaction, error)
	Pay(telegramId int64, tokenSymbol string, amount float64) error
	Convert(models.ConvertRequest) (error, models.ConvertResponse)
}

type Service struct {
	Authorization
	Wallet
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Authorization: NewAuthService(repos.Authorization),
		Wallet:        NewWalletService(repos.Wallet, tronclient.NewTronHTTPClient("dbd1331e-96a5-493e-9fa5-1f37bc008b1f", "TDZVaZMrSuABymCsb2EgDkXjup6TNVxQ3w")),
	}
}
