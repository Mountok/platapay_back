package models

import "time"

type Transaction struct {
	ID           int64     `db:"id"`
	FromWalletID int64     `db:"from_wallet_id"`
	ToAddress    string    `db:"to_address"`
	TokenSymbol  string    `db:"token_symbol"`
	Amount       float64   `db:"amount"`  // как и в балансе — по желанию можно заменить на decimal.Decimal
	TxHash       *string   `db:"tx_hash"` // может быть NULL
	Status       string    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
}

type WithdrawInput struct {
	Amount      float64 `json:"amount" binding:"required"`
	ToAddress   string  `json:"to_address" binding:"required"`
	TokenSymbol string  `json:"token_symbol" binding:"required"`
}
