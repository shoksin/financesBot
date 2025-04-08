package main

import (
	"context"
	"time"

	"github.com/shoksin/financesBot/internal/metrics"
	"github.com/shoksin/financesBot/internal/tracing"

	"github.com/shoksin/financesBot/internal/clients/tg"
	"github.com/shoksin/financesBot/internal/config"
	"github.com/shoksin/financesBot/internal/logger"
)

// default settings
var (
	mainCurrency                = "BYN"
	currenciesName              = []string{"USD", "EUR", "RUB", "BYN"}
	currenciesUpdatePeriod      = 30 * time.Minute //Периодичность обновления курсов валют (раз в 30 минут).
	currenciesUpdateCachePeriod = 30 * time.Minute //Периодичность кэширования курсов валют из базы данных (раз в 30 минут).
	connectionStringDB          = ""
	kafkaTopic                  = "tgbot"
	brokersList                 = []string{"localhost:9092"} //Список адресов брокеров сообщений (адрес Kafka)
)

func main() {
	logger.Info("Application start")

	ctx := context.Background()

	config, err := config.New()
	if err != nil {
		logger.Debug("Error to get config", "err", err)
	}

	setConfig(config.GetConfig())

	/* TODO
	Оборачивание в Middleware функции обработки сообщения для метрик и трейсинга.*/
	tgProcessingFuncHandler := tg.HandlerFunc(tg.ProcessingMessages)
	tgProcessingFuncHandler = metrics.MetricsMiddleware(tgProcessingFuncHandler)
	tgProcessingFuncHandler = tracing.TracingMiddleware(tgProcessingFuncHandler)

	tgClient, err := tg.New(config, tgProcessingFuncHandler)
	if err != nil {
		logger.Fatal("Error initialazing tg-client:", "err", err)
	}

	//Инициализация хранилищ (подключение к базе данных)
	

	logger.Info("Application stop")
}

func setConfig(config config.Config) {
	if config.MainCurrency != "" {
		mainCurrency = config.MainCurrency
	}

	if len(config.CurrenciesName) > 0 {
		currenciesName = config.CurrenciesName
	}

	if config.CurrenciesUpdatePeriod > 0 {
		currenciesUpdatePeriod = time.Duration(config.CurrenciesUpdatePeriod)
	}

	if config.CurrenciesUpdateCachePeriod > 0 {
		currenciesUpdateCachePeriod = time.Duration(config.CurrenciesUpdateCachePeriod)
	}

	if config.ConnectionStringDB != "" {
		connectionStringDB = config.ConnectionStringDB
	}

	if config.KafkaTopic != "" {
		kafkaTopic = config.KafkaTopic
	}

	if len(config.BrokersList) > 0 {
		brokersList = config.BrokersList
	}
}
