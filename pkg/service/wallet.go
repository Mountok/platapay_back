package service

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"production_wallet_back/models"
	"production_wallet_back/pkg/cache"
	"production_wallet_back/pkg/repository"
	"strings"
)

type WalletService struct {
	repos repository.Wallet
}

func NewWalletService(repos repository.Wallet) *WalletService {
	return &WalletService{
		repos: repos,
	}
}

func (s *WalletService) Convert(convertReq models.ConvertRequest) (error, models.ConvertResponse) {
	var response models.ConvertResponse
	from := strings.ToLower(convertReq.From)
	to := strings.ToLower(convertReq.To)
	if from == "" || to == "" || convertReq.Amount <= 0 {
		return errors.New("Неверно переданы данные в тело запроса для конвертации"), response
	}

	key := currencyID(convertReq.To) + "_" + convertReq.From

	// Попробуем получить курс из кэша
	if rate, found := cache.GetCachedRate(key); found {
		converted := convertReq.Amount / rate
		response = models.ConvertResponse{
			ConvertedAmount: converted,
			Currency:        strings.ToUpper(convertReq.To),
			Message:         fmt.Sprintf("Переведите %.2f на адрес  Tx..", converted),
			Wallet:          "Tx...",
		}
		return nil, response
	}
	// Если в кэше нет — запрос к CoinGecko
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + currencyID(to) + "&vs_currencies=" + from
	client := resty.New()

	log.Println("Запрос к API CoinGecko:", url)

	resp, err := client.R().
		SetHeader("x-cg-pro-api-key", "CG-wmi7LpR5B84uad7kPFE1knYa").
		SetHeader("Accept", "application/json").
		SetResult(map[string]map[string]float64{}).
		Get(url)

	if err != nil || resp.IsError() {
		log.Println("Ошибка при получении курса:", err)
		log.Println("Ответ от API:", resp)
		return errors.New("Не удалось получить курс"), response
	}

	data := *resp.Result().(*map[string]map[string]float64)
	rate := data[currencyID(to)][from]

	if rate == 0 {
		return errors.New("Некорректный курс"), response
	}

	cache.SetCachedRate(key, rate)

	converted := convertReq.Amount / rate

	response = models.ConvertResponse{
		ConvertedAmount: converted,
		Currency:        strings.ToUpper(convertReq.To),
		Message:         fmt.Sprintf("Переведите %.2f на адрес  Tx..", converted),
		Wallet:          "Tx...",
	}

	return nil, response
}

func currencyID(symbol string) string {
	switch strings.ToLower(symbol) {
	case "usdt":
		return "tether"
	case "btc":
		return "bitcoin"
	case "eth":
		return "ethereum"
	default:
		return strings.ToLower(symbol)
	}
}
