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
	GetWallet(telegramId int64) (models.WalletResponce, error)
	InitBalance(walletID int64, tokenSymbol string) error
	GetBalance(telegramId int64) ([]models.Balance, error)
	GetUSDTBalance(address string) (float64, error)
	Deposit(telegramId int64, tokenSymbol string, amount float64) error
	Withdraw(privKey string, toAddress string, amount float64) (string, error)
	WithdrawWithContract(privKey string, toAddress string, amount float64, contractAddress string) (string, error)
	ApproveUSDT(privKey string, spenderAddress string, amount float64) (string, error)
	GetTransactions(telegramId int64) ([]models.Transaction, error)
	Pay(telegramId int64, tokenSymbol string, amount float64) error
	Convert(models.ConvertRequest) (error, models.ConvertResponse)
	GetAPIKey() string
}

type Service struct {
	Authorization
	Wallet
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Authorization: NewAuthService(repos.Authorization),
		Wallet:        NewWalletService(repos.Wallet, tronclient.NewTronHTTPClient("dbd1331e-96a5-493e-9fa5-1f37bc008b1f", "TTpC8a19eUj9LbQmZLrX7bZyHCyCWhrv2C")),
	}
}
