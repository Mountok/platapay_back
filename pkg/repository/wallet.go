package repository

import (
	"errors"
	"fmt"
	"log"
	"production_wallet_back/models"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type WalletPostgres struct {
	db *sqlx.DB
}

func NewWalletPostgres(db *sqlx.DB) *WalletPostgres {
	return &WalletPostgres{db: db}
}

func (r *WalletPostgres) GetPrivatKey(telegramId int64) (string, error) {
	var privKey string
	query := `SELECT private_key FROM wallets WHERE user_id = $1`
	err := r.db.QueryRow(query, telegramId).Scan(&privKey)
	if err != nil {
		logrus.Errorf("get private key error: %v", err)
		return "", err
	}
	return privKey, nil
}

func (r *WalletPostgres) OrdersHistory(telegramId int64) ([]models.OrderQR, error) {
	var orders []models.OrderQR
	query := "SELECT * FROM orderqr WHERE telegram_id = $1"
	err := r.db.Select(&orders, query, telegramId)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return orders, nil
}

func (r *WalletPostgres) PayQR(orderId int) (bool, error) {
	logrus.Infof("Pay QR for order %d", orderId)
	query := "UPDATE orderQR SET ispaid = true WHERE id = $1"
	_, err := r.db.Exec(query, orderId)
	if err != nil {
		logrus.Error(err)
		return false, err
	}
	logrus.Infof("ispaid: %v", true)
	return true, nil
}

func (r *WalletPostgres) GetOrders() ([]models.OrderQR, error) {
	logrus.Info("Geting new orders")
	var orders []models.OrderQR
	query := "SELECT * FROM orderqr WHERE ispaid is false;"
	err := r.db.Select(&orders, query)
	if err != nil {
		return []models.OrderQR{}, err
	}
	if len(orders) == 0 {
		return []models.OrderQR{}, errors.New("no orders found")
	}
	logrus.Infof("GetOrders: %v", orders)
	return orders, nil
}

func (r *WalletPostgres) GetOrderState(orderId int) (bool, error) {
	logrus.Infof("Getting order state for order %d", orderId)
	query := "SELECT ispaid FROM orderqr WHERE id = $1"
	var ispaid bool
	err := r.db.QueryRow(query, orderId).Scan(&ispaid)
	if err != nil {
		logrus.Error(err)
		return false, err
	}
	logrus.Infof("Getting order state for order %d", orderId)
	return ispaid, nil
}

func (r *WalletPostgres) CreateOrder(qr models.OrderQR) (int, error) {
	logrus.Infof("Creating order with %v", qr)
	var orderid int
	query := "insert into orderqr (telegram_id, qrcode, summa, crypto) values ($1,$2,$3, $4) RETURNING id;"
	err := r.db.QueryRow(query, qr.TelegramId, qr.QRCode, qr.Summa, qr.Crypto).Scan(&orderid)
	if err != nil {
		logrus.Error(err)
		return 0, err
	}
	logrus.Infof("Created order with %v", qr)
	return orderid, nil
}

func (r *WalletPostgres) CreateWallet(userID int64, privKey, address string) (int64, error) {
	log.Printf("Creating wallet: userID=%d, address=%s", userID, address)
	var walletId int64
	query := `INSERT INTO wallets (user_id, private_key, address) VALUES ($1, $2, $3) returning id;`
	err := r.db.QueryRow(query, userID, privKey, address).Scan(&walletId)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Error creating wallet: %s", err))
	}
	return walletId, err
}

func (r *WalletPostgres) GetWallet(telegramId int64) (models.WalletResponce, error) {
	log.Printf("Getting wallet: telegramId=%d", telegramId)
	var wallet models.WalletResponce
	query := `SELECT id, address FROM wallets WHERE user_id = $1`
	err := r.db.Get(&wallet, query, telegramId)
	if err != nil {
		return wallet, errors.New(fmt.Sprintf("Error getting wallet: %s", err))
	}
	return wallet, nil
}

func (r *WalletPostgres) GetWalletByAddress(address string) (models.WalletResponce, error) {
	var wallet models.WalletResponce
	query := `SELECT id, address FROM wallets WHERE address = $1`
	err := r.db.Get(&wallet, query, address)
	if err != nil {
		return wallet, err
	}
	return wallet, nil
}

func (r *WalletPostgres) InitBalance(walletID int64, tokenSymbol string) error {
	query := `INSERT INTO balances (wallet_id, token_symbol) VALUES ($1, $2)`
	_, err := r.db.Exec(query, walletID, tokenSymbol)
	return err
}

func (r *WalletPostgres) GetBalances(telegramID int64) ([]models.Balance, error) {
	var balances []models.Balance

	query := `
		SELECT b.wallet_id, b.token_symbol, b.amount, b.updated_at
		FROM balances b
		JOIN wallets w ON b.wallet_id = w.id
		WHERE w.user_id = $1
	`
	err := r.db.Select(&balances, query, telegramID)
	return balances, err
}

func (r *WalletPostgres) Deposit(telegramId int64, tokenSymbol string, amount float64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var walletId int64
	queryWallet := "SELECT w.id FROM wallets w JOIN users u on u.telegram_id = w.user_id WHERE u.telegram_id = $1;"
	err = tx.Get(&walletId, queryWallet, telegramId)
	if err != nil {
		return err
	}
	queryBalance := "UPDATE balances SET amount = amount + $1 WHERE WALLET_id = $2;"
	_, err = tx.Exec(queryBalance, amount, walletId)
	if err != nil {
		return err
	}

	return nil
}

func (r *WalletPostgres) Convert(req models.ConvertRequest) (error, models.ConvertResponse) {
	return nil, models.ConvertResponse{}
}

func (r *WalletPostgres) WithdrawBalance(walletID int64, amount float64, tokenSymbol string) error {
	query := `
	UPDATE balances
	SET amount = amount - $1, updated_at = NOW()
	WHERE wallet_id = $2 and token_symbol = $3 and amount >= $1
	`
	_, err := r.db.Exec(query, amount, walletID, tokenSymbol)
	if err != nil {
		return errors.New(fmt.Sprintf("Error executing withdrawal: %s", err))
	}
	return nil
}

func (r *WalletPostgres) CreateTransaction(walletID int64, to_address string, token string, amount float64, status string, tx_hash string) error {
	query := `
		INSERT INTO transactions (from_wallet_id, to_address, token_symbol, amount, tx_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6);`
	_, err := r.db.Exec(query, walletID, to_address, token, amount, tx_hash, status)
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating transaction: %s", err))
	}
	return nil
}

func (r *WalletPostgres) GetTransactions(telegramID int64) ([]models.Transaction, error) {
	query := `
	SELECT t.id, t.from_wallet_id, t.to_address, t.token_symbol, t.amount, t.tx_hash, t.status, t.created_at
	FROM transactions t
	JOIN wallets w ON w.id = t.from_wallet_id
	WHERE w.user_id = $1
	ORDER BY t.created_at DESC;
	`

	rows, err := r.db.Query(query, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.FromWalletID,
			&tx.ToAddress,
			&tx.TokenSymbol,
			&tx.Amount,
			&tx.TxHash,
			&tx.Status,
			&tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (r *WalletPostgres) Pay(telegramId int64, tokenSymbol string, amount float64) error {
	return nil
}

// Добавить виртуальное списание
func (r *WalletPostgres) AddVirtualTransfer(walletID int64, amount float64) error {
	query := `INSERT INTO usdt_virtual_transfers (wallet_id, amount) VALUES ($1, $2)`
	_, err := r.db.Exec(query, walletID, amount)
	return err
}

// Получить сумму всех pending виртуальных списаний
func (r *WalletPostgres) SumPendingVirtualTransfers(walletID int64) (float64, error) {
	var sum float64
	query := `SELECT COALESCE(SUM(amount), 0) FROM usdt_virtual_transfers WHERE wallet_id = $1 AND status = 'pending'`
	err := r.db.Get(&sum, query, walletID)
	return sum, err
}

// Получить все pending виртуальные списания
func (r *WalletPostgres) GetPendingVirtualTransfers(walletID int64) ([]models.VirtualTransfer, error) {
	var transfers []models.VirtualTransfer
	query := `SELECT * FROM usdt_virtual_transfers WHERE wallet_id = $1 AND status = 'pending'`
	err := r.db.Select(&transfers, query, walletID)
	return transfers, err
}

// Обновить статус виртуальных списаний на processed
func (r *WalletPostgres) MarkVirtualTransfersProcessed(ids []int64) error {
	query := `UPDATE usdt_virtual_transfers SET status = 'processed', processed_at = NOW() WHERE id = ANY($1)`
	_, err := r.db.Exec(query, ids)
	return err
}

// Обновить баланс в таблице balances
func (r *WalletPostgres) UpdateBalance(walletID int64, tokenSymbol string, amount float64) error {
	query := `UPDATE balances SET amount = $1, updated_at = NOW() WHERE wallet_id = $2 AND token_symbol = $3`
	_, err := r.db.Exec(query, amount, walletID, tokenSymbol)
	return err
}

// GetAllWallets возвращает все кошельки
func (r *WalletPostgres) GetAllWallets() ([]models.Wallet, error) {
	var wallets []models.Wallet
	query := `SELECT id, user_id, private_key, address, created_at FROM wallets`
	err := r.db.Select(&wallets, query)
	return wallets, err
}

// GetVirtualTransfersByWalletID возвращает все виртуальные списания по wallet_id
func (r *WalletPostgres) GetVirtualTransfersByWalletID(walletID int64) ([]models.VirtualTransfer, error) {
	var transfers []models.VirtualTransfer
	query := `SELECT id, wallet_id, amount, status, created_at, processed_at FROM usdt_virtual_transfers WHERE wallet_id = $1`
	err := r.db.Select(&transfers, query, walletID)
	return transfers, err
}

// GetTransactionsByWalletID возвращает все реальные списания по wallet_id и токену
func (r *WalletPostgres) GetTransactionsByWalletID(walletID int64, tokenSymbol string) ([]models.Transaction, error) {
	var txs []models.Transaction
	query := `SELECT id, from_wallet_id, to_address, token_symbol, amount, tx_hash, status, created_at FROM transactions WHERE from_wallet_id = $1 AND token_symbol = $2`
	err := r.db.Select(&txs, query, walletID, tokenSymbol)
	return txs, err
}
