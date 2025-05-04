package service

import (
	"production_wallet_back/models"
	"production_wallet_back/pkg/repository"
)

type AuthService struct {
	repos repository.Authorization
}

func NewAuthService(repos repository.Authorization) *AuthService {
	return &AuthService{
		repos: repos,
	}
}

func (s *AuthService) CreateUser(user models.User) (int64, error) {
	return s.repos.CreateUser(user)
}
func (s *AuthService) GetUserByTelegramId(telegramId int64) (models.User, error) {
	return s.repos.GetUserByTelegramId(telegramId)
}
