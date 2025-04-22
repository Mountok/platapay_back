package repository

import (
	"github.com/jmoiron/sqlx"
	"production_wallet_back/models"
)

type Authorization interface{}
type Wallet interface {
	Convert(models.ConvertRequest) (error, models.ConvertResponse)
}

type Repository struct {
	Authorization
	Wallet
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		Wallet: NewWalletPostgres(db),
	}
}
