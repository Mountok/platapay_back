package repository

import (
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"production_wallet_back/models"
)

type WalletPostgres struct {
	db *sqlx.DB
}

func NewWalletPostgres(db *sqlx.DB) *WalletPostgres {
	return &WalletPostgres{db: db}
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
