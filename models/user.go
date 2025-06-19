package models

import "time"

type User struct {
	ID         int64     `db:"id" json:"id"`
	TelegramID int64     `db:"telegram_id" json:"telegram_id" binding:"required"`
	Username   string   `db:"username" json:"username"` // username может быть NULL
	FirstName  string    `db:"first_name" json:"first_name"`
	LastName   string    `db:"last_name" json:"last_name"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
