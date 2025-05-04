package models

import "time"

type Balance struct {
	ID          int64     `db:"id" json:"id"`
	WalletID    int64     `db:"wallet_id" json:"wallet_id"`
	TokenSymbol string    `db:"token_symbol" json:"token_symbol"`
	Amount      float64   `db:"amount" json:"amount"` // можно использовать decimal.Decimal для высокой точности
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type DepositInput struct {
	TokenSymbol string  `json:"token_symbol" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
}
