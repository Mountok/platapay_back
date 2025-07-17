package models

import "time"

type VirtualTransfer struct {
	ID          int64      `db:"id"`
	WalletID    int64      `db:"wallet_id"`
	Amount      float64    `db:"amount"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
	ProcessedAt *time.Time `db:"processed_at"`
}
