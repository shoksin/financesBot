package messages

//Разобраться с tracer и opentelemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/shoksin/financesBot/internal/helpers/timeutils"
	"github.com/shoksin/financesBot/internal/logger"
	"github.com/shoksin/financesBot/internal/models/bottypes"
	"go.opentelemetry.io/otel"
)

const (
	txtStart          = "Привет, *%v*. Я помогаю вести учет расходов. Выберите действие."
	txtUnknownCommand = "К сожалению, данная команда мне неизвестна. Для начала работы введите /start"
	txtRecSave        = "Запись успешно сохранена."
	txtRecOverLimit   = "Запись не сохранена: превышен бюджет раходов в текущем месяце."
	txtHelp           = "Я - бот, помогающий вести учет расходов. Для начала работы введите /start"
)

var btnStart = []bottypes.TgRowButtons{
	{bottypes.TgInlineButton{DisplayName: "Добавить категорию", Value: "/add_cat"}, bottypes.TgInlineButton{DisplayName: "Добавить расход", Value: "/add_rec"}},
	{bottypes.TgInlineButton{DisplayName: "Отчёт за неделю", Value: "/report_w"}, bottypes.TgInlineButton{DisplayName: "Отчёт за месяц", Value: "/report_m"}, bottypes.TgInlineButton{DisplayName: "Отчёт за год", Value: "/report_y"}},
	{bottypes.TgInlineButton{DisplayName: "Ввести данные за прошлый период", Value: "/add_tbl"}},
	{bottypes.TgInlineButton{DisplayName: "Выбрать валюту", Value: "/choice_currency"}, bottypes.TgInlineButton{DisplayName: "Установить лимит", Value: "/set_limit"}},
}

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
	ctx             context.Context
	tgClient        MessagesSender
	storage         UserDataStorage
	currencies      ExchangeRates
	reportCache     LRUCache
	kafkaProducer   kafkaProducer
	lastUserCat     map[int64]string
	lastUserCommand map[int64]string
}

func New(ctx context.Context, tgClient MessagesSender, storage UserDataStorage, currencies ExchangeRates, reportCache LRUCache, kafka kafkaProducer) *Model {
	return &Model{
		ctx:             ctx,
		tgClient:        tgClient,
		storage:         storage,
		currencies:      currencies,
		reportCache:     reportCache,
		kafkaProducer:   kafka,
		lastUserCat:     map[int64]string{},
		lastUserCommand: map[int64]string{},
	}
}

type Message struct {
	Text            string
	UserID          int64
	UserName        string
	UserDisplayname string
	IsCallback      bool
	CallbackMsgID   string
}

func (s *Model) GetCtx() context.Context {
	return s.ctx
}

func (s *Model) SetCtx(ctx context.Context) {
	s.ctx = ctx
}

func (s *Model) IncomingMessage(msg Message) error {
	tracer := otel.Tracer("messages")
	ctx, span := tracer.Start(s.ctx, "IncomingMessage")
	s.ctx = ctx
	defer span.End()

	lastUserCat := s.lastUserCat[msg.UserID]
	lastUserCommand := s.lastUserCommand[msg.UserID]

	s.lastUserCat[msg.UserID] = ""
	s.lastUserCommand[msg.UserID] = ""

	// Проверка ввода суммы расхода по выбранной категории и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterCategorySum(s, msg, lastUserCat); err != nil || isNeedReturn {
		return err
	}

	// Проверка ввода новой категории и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterNewCategory(s, msg, lastUserCommand); err != nil || isNeedReturn {
		return err
	}

	// Проверка ввода лимита и сохранение, если введено.

}

// Область "Внешний интерфейс": конец.

// Область "Служебные функции": начало.

// Область "Распознавание входящих команд": начало.

// Проверка ввода суммы расхода по выбранной категории.
func checkIfEnterCategorySum(s *Model, msg Message, lastUserCat string) (bool, error) {
	if lastUserCat != "" && msg.Text != "" {
		tracer := otel.Tracer("messages")
		ctx, span := tracer.Start(s.ctx, "checkIfEnterCategorySum")
		s.ctx = ctx
		defer span.End()

		// Парсинг и конвертация введенной суммы.
		catSum, err := parseAndConvertSumFromCurrency(s, msg.UserID, msg.Text)
		if err != nil {
			return true, err
		}

		newRec := bottypes.UserDataRecord{UserID: msg.UserID, Category: lastUserCat, Sum: catSum, Period: time.Now()}
		isOverLimit, err := s.storage.InsertUserDataRecord(s.ctx, msg.UserID, newRec, msg.UserName, timeutils.BeginOfMonth(newRec.Period))
		if err != nil {
			if isOverLimit {
				return true, s.tgClient.SendMessage(msg.UserID, txtRecOverLimit)
			} else {
				logger.Error("Error saving record", "err", err)
				return true, fmt.Errorf("Insert data record error: %v", err)
			}
		}
		// Ответ пользователю об успешном сохранении.
		return true, s.tgClient.SendMessage(msg.UserID, txtRecSave)

	}

	//Ввели не расход
	return false, nil
}

func checkIfEnterNewCategory(s *Model, msg Message, lastUserCommand string) (bool, error) {

}
