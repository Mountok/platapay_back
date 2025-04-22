package service

import (
	"production_wallet_back/models"
	"production_wallet_back/pkg/repository"
)

type Authorization interface{}
type Wallet interface {
	Convert(models.ConvertRequest) (error, models.ConvertResponse)
}

type Service struct {
	Authorization
	Wallet
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Wallet: NewWalletService(repos.Wallet),
	}
}
