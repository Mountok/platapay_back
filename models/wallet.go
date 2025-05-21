package models

import "time"

type Wallet struct {
	ID         int64     `db:"id" json:"id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	PrivateKey string    `db:"private_key" json:"private_key"`
	Address    string    `db:"address" json:"address"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
type WalletResponce struct {
	WalletID int64  `json:"id" db:"id"`
	Address  string `json:"address" db:"address"`
}
type ConvertRequest struct {
	Amount float64 `json:"amount"` // сумма
	From   string  `json:"from"`   // исходная валюта, например: "RUB"
	To     string  `json:"to"`     // целевая валюта, например: "USDT"
}

type ConvertResponse struct {
	ConvertedAmount float64 `json:"convertedAmount"`
	Currency        string  `json:"currency"`
	Wallet          string  `json:"wallet,omitempty"`
	Message         string  `json:"message"`
}
