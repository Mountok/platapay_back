package repository

import (
	"github.com/jmoiron/sqlx"
	"production_wallet_back/models"
)

type AuthPostgres struct {
	db *sqlx.DB
}

func NewAuthPostgres(db *sqlx.DB) *AuthPostgres {
	return &AuthPostgres{db: db}
}

func (r *AuthPostgres) GetUserByTelegramId(telegramID int64) (models.User, error) {
	var user models.User
	query := `SELECT id, telegram_id, username, first_name, last_name, created_at FROM users WHERE telegram_id = $1`
	err := r.db.Get(&user, query, telegramID)
	return user, err
}

func (r *AuthPostgres) CreateUser(user models.User) (int64, error) {
	var id int64
	query := `
        INSERT INTO users (telegram_id, username, first_name, last_name)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	err := r.db.QueryRow(
		query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
	).Scan(&id)
	return id, err
}
