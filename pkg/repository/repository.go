package repository

import (
	"github.com/jmoiron/sqlx"
	"production_wallet_back/models"
)

type Authorization interface {
	GetUserByTelegramId(int64) (models.User, error)
	CreateUser(models.User) (int64, error)
}
type Wallet interface {
	CreateWallet(userID int64, privKey, address string) (int64, error)
	GetWallet(telegramId int64) (models.WalletResponce, error)
	InitBalance(walletID int64, tokenSymbol string) error
	GetBalances(telegramID int64) ([]models.Balance, error)
	Deposit(telegramId int64, tokenSymbol string, amount float64) error
	WithdrawBalance(walletID int64, amount float64, tokenSymbol string) error
	CreateTransaction(walletID int64, toId string, token string, amount float64, status string, tx_hash string) error
	GetTransactions(telegramId int64) ([]models.Transaction, error)
	Pay(telegramId int64, tokenSymbol string, amount float64) error
	Convert(models.ConvertRequest) (error, models.ConvertResponse)

	CreateOrder(qr models.OrderQR) (int, error)
	OrdersHistory(telegramId int64) ([]models.OrderQR, error)
	GetOrderState(orderId int) (bool, error)
	GetOrders() ([]models.OrderQR, error)
	PayQR(orderId int) (bool, error)
}

type Repository struct {
	Authorization
	Wallet
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		Authorization: NewAuthPostgres(db),
		Wallet:        NewWalletPostgres(db),
	}
}
