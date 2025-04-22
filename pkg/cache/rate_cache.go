package cache

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type CachedRate struct {
	Rate      float64
	Timestamp time.Time
}

var (
	cachedRates   = make(map[string]CachedRate)
	cacheDuration = 10 * time.Minute
	mu            sync.Mutex
)

// GetCachedRate возвращает курс из кэша или false, если его нет или он устарел
func GetCachedRate(key string) (float64, bool) {
	mu.Lock()
	defer mu.Unlock()

	rateData, ok := cachedRates[key]
	if !ok {
		return 0, false
	}

	if time.Since(rateData.Timestamp) > cacheDuration {
		return 0, false
	}

	logrus.Infof("Курс взят из кэша для", key)
	return rateData.Rate, true
}

// SetCachedRate сохраняет курс в кэш
func SetCachedRate(key string, rate float64) {
	mu.Lock()
	defer mu.Unlock()

	cachedRates[key] = CachedRate{
		Rate:      rate,
		Timestamp: time.Now(),
	}

	logrus.Infof("Курс сохранён в кэш для", key)
}
