package messages

import (
	"context"
	"time"

	"github.com/shoksin/financesBot/internal/models/bottypes"
)

const (
	txtStart          = "Привет, *%v*. Я помогаю вести учет расходов. Выберите действие."
	txtUnknownCommand = "К сожалению, данная команда мне неизвестна. Для начала работы введите /start"
	txtHelp           = "Я - бот, помогающий вести учет расходов. Для начала работы введите /start"
)

// MessageSender Интерфейс для работы с сообщениями.
type MessagesSender interface {
	SendMessage(userID int64, text string) error
	ShowInlineButtons(text string, buttons []bottypes.TgRowButtons, userID int64) error
}

type UserDataStorage interface {
	InsertUserDataRecord(ctx context.Context, userID int64, rec bottypes.UserDataRecord, userName string, limitPeriod time.Time) (bool, error)
	GetUserDataRecord(ctx context.Context, userID int64, period time.Time) ([]bottypes.UserDataReportRecord, error)
	InsertCategory(ctx context.Context, userID int64, catName string, userName string) error
	GetUserCategories(ctx context.Context, userID int64) ([]string, error)
	GetUserCurrency(ctx context.Context, userID int64) (string, error)
	SetUserCurrency(ctx context.Context, userID int64, currencyName string, userName string) error
	GetUserLimit(ctx context.Context, userID int64) (int64, error)
	SetUserLimit(ctx context.Context, userID int64, limit int64, userName string) error
}

// LRUCache Интерфейс для работы с кэшем отчетов.
type LRUCache interface {
	Add(key string, value any)
	Get(key string) any
}

// ExchangeRates Интерфейс для работы с курсами валют.
type ExchangeRates interface {
	ConvertSumFromBaseToCurrency(currencyName string, sum int64) (int64, error)
	ConvertSumFromCurrencyToBase(currencyName string, sum int64) (int64, error)
	GetExchangeRate(currencyName string) (float64, error)
	GetMainCurrency() string
	GetCurrenciesList() []string
}

// kafkaProducer Интерфейс для отправки сообщений в кафку.
type kafkaProducer interface {
	SendMessage(key string, value string) (partition int32, offset int64, err error)
	GetTopic() string
}

// Model Модель бота (клиент, хранилище, последние команды пользователя)
type Model struct {
	ctx      context.Context
	tgClient MessagesSender
}
