package models

type Wallet struct {
	Id         int64  `json:"id" db:"id"`
	UserId     int64  `json:"user_id" db:"user_id"`
	PrivateKey string `json:"private_key" db:"private_key"`
	PublicKey  string `json:"public_key" db:"public_key"`
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
