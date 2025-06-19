package models

type OrderQR struct {
	Id         int64   `json:"id" db:"id"`
	TelegramId int64   `json:"telegram_id" db:"telegram_id"`
	QRCode     string  `json:"qr_code" db:"qrcode"`
	Summa      float64 `json:"summa" db:"summa"`
	Crypto     float64 `json:"crypto" db:"crypto"`
	IsPaid     bool    `json:"is_paid" db:"ispaid"`
}
