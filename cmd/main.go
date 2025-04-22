package main

import (
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	production "production_wallet_back"

	"production_wallet_back/pkg/handler"
	"production_wallet_back/pkg/repository"
	"production_wallet_back/pkg/service"
)

func main() {
	logrus.SetFormatter(new(logrus.JSONFormatter))
	logrus.Infoln("Запус сервера")
	if err := godotenv.Load(); err != nil {
		logrus.Infof("Ошибка инициализации переменных окружения .env: %s \n", err)
	}

	if err := InitConfig(); err != nil {
		logrus.Fatalf("Ошибка (viper) при инициализации конгфига .yaml: %s \n", err.Error())
	}
	logrus.Infoln("Конфиг YAML инициализирован")

	db, err := repository.NewPostgresDB(repository.Config{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		Password: os.Getenv("DB_PASS_LOCAL"),
		DBName:   viper.GetString("db.dbname"),
		SSLMode:  viper.GetString("db.sslmode"),
	})
	if err != nil {
		logrus.Fatalf("Ошибка при инициализации базы данных: %s \n", err.Error())
	}
	logrus.Info("База данных подключена")

	repos := repository.NewRepository(db)
	service := service.NewService(repos)
	handler := handler.NewHandler(service)

	srv := new(production.Server)
	if err := srv.Run(os.Getenv("PORT"), handler.InitRoute()); err != nil {
		logrus.Fatalf("Ошибка при запуске сервера: %s \n", err)
	}

}

func InitConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
