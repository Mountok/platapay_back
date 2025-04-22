package repository

import (
	"github.com/jmoiron/sqlx"
	"production_wallet_back/models"
)

type WalletPostgres struct {
	db *sqlx.DB
}

func NewWalletPostgres(db *sqlx.DB) *WalletPostgres {
	return &WalletPostgres{db: db}
}

func (r *WalletPostgres) Convert(req models.ConvertRequest) (error, models.ConvertResponse) {
	return nil, models.ConvertResponse{}
}
