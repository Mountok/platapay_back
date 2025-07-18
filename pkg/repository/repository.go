package repository

import (
	"production_wallet_back/models"

	"github.com/jmoiron/sqlx"
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
	GetPrivatKey(telegramId int64) (string, error)
	AddVirtualTransfer(walletID int64, amount float64) error
	SumPendingVirtualTransfers(walletID int64) (float64, error)
	GetPendingVirtualTransfers(walletID int64) ([]models.VirtualTransfer, error)
	MarkVirtualTransfersProcessed(ids []int64) error
	GetWalletByAddress(address string) (models.WalletResponce, error)
	UpdateBalance(walletID int64, tokenSymbol string, amount float64) error

	// Новый метод для админки
	GetAllWallets() ([]models.Wallet, error)
	GetVirtualTransfersByWalletID(walletID int64) ([]models.VirtualTransfer, error)
	GetTransactionsByWalletID(walletID int64, tokenSymbol string) ([]models.Transaction, error)
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
