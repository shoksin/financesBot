package messages

//Разобраться с tracer и opentelemetry

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shoksin/financesBot/internal/helpers/timeutils"
	"github.com/shoksin/financesBot/internal/logger"
	"github.com/shoksin/financesBot/internal/models/bottypes"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	txtStart            = "Привет, *%v*. Я помогаю вести учет расходов. Выберите действие."
	txtUnknownCommand   = "К сожалению, данная команда мне неизвестна. Для начала работы введите /start"
	txtReportError      = "Не удалось получить данные."
	txtReportEmpty      = "За указанный период данные отсутствуют."
	txtReportWait       = "Формирование отчета. Пожалуйста, подождите..."
	txtCatAdd           = "Введите название категории (не более 30 символов). Для отмены введите 0."
	txtCatView          = "Выберите категорию, а затем введите сумму."
	txtCatChoice        = "Выбрана категория *%v*. Введите сумму (только число). Для отмены введите 0. Используемая валюта: *%v*"
	txtCatSave          = "Категория успешно сохранена."
	txtCatEmpty         = "Пока нет категорий, сначала добавьте хотя бы одну категорию."
	txtRecSave          = "Запись успешно сохранена."
	txtRecOverLimit     = "Запись не сохранена: превышен бюджет раходов в текущем месяце."
	txtRecTbl           = "Для загрузки истории расходов введите таблицу в следующем формате (дата сумма категория):\n`YYYY-MM-DD 0.00 XXX`\nНапример: \n`2022-09-20 1500 Кино`\n`2022-07-12 350.50 Продукты, еда`\n`2022-08-30 8000 Одежда и обувь`\n`2022-09-01 60 Бензин`\n`2022-09-27 425 Такси`\n`2022-09-26 1500 Бензин`\n`2022-09-26 950 Кошка`\n`2022-09-25 50 Бензин`\nИспользуемая валюта: *%v*"
	txtReportQP         = "За какой период будем смотреть отчет? Команды периодов: /report_w - неделя, /report_m - месяц, /report_y - год"
	txtHelp             = "Я - бот, помогающий вести учет расходов. Для начала работы введите /start"
	txtCurrencyChoice   = "В качестве основной задана валюта: *%v*. Для изменения выберите другую валюту."
	txtCurrencySet      = "Валюта изменена на *%v*."
	txtCurrencySetError = "Ошибка сохранения валюты."
	txtLimitInfo        = "Текущий ежемесячный бюджет: *%v*. Для изменения введите число, например, 80000."
	txtLimitSet         = "Бюджет изменен на *%v*."
)

var btnStart = []bottypes.TgRowButtons{
	{bottypes.TgInlineButton{DisplayName: "Добавить категорию", Value: "/add_cat"}, bottypes.TgInlineButton{DisplayName: "Добавить расход", Value: "/add_rec"}},
	{bottypes.TgInlineButton{DisplayName: "Отчёт за неделю", Value: "/report_w"}, bottypes.TgInlineButton{DisplayName: "Отчёт за месяц", Value: "/report_m"}, bottypes.TgInlineButton{DisplayName: "Отчёт за год", Value: "/report_y"}},
	{bottypes.TgInlineButton{DisplayName: "Ввести данные за прошлый период", Value: "/add_tbl"}},
	{bottypes.TgInlineButton{DisplayName: "Выбрать валюту", Value: "/choice_currency"}, bottypes.TgInlineButton{DisplayName: "Установить лимит", Value: "/set_limit"}},
}

var lineRegexp = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (\d+.?\d{0,2}) (.+)$`)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("models/messages")
}

// Область "Внешний интерфейс": начало.
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
	GetUserLimit(ctx context.Context, userID int64) (float64, error)
	SetUserLimit(ctx context.Context, userID int64, limit float64, userName string) error
}

// LRUCache Интерфейс для работы с кэшем отчетов.
type LRUCache interface {
	Add(key string, value any)
	Get(key string) any
}

// ExchangeRates Интерфейс для работы с курсами валют.
type ExchangeRates interface {
	ConvertSumFromBaseToCurrency(currencyName string, sum float64) (float64, error)
	ConvertSumFromCurrencyToBase(currencyName string, sum float64) (float64, error)
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
	UserDisplayName string
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
	if isNeedReturn, err := checkIfEnterNewLimit(s, msg, lastUserCommand); err != nil || isNeedReturn {
		return err
	}

	// Проверка ввода данных в виде таблицы и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterTableData(s, msg, lastUserCommand); err != nil || isNeedReturn {
		return err
	}

	// Проверка выбора категории для ввода расхода.
	if isNeedReturn, err := checkIfChoiceCategory(s, msg); err != nil || isNeedReturn {
		return err
	}

	// Проверка выбора валюты.
	if isNeedReturn, err := checkIfChoiceCurrency(s, msg); err != nil || isNeedReturn {
		return err
	}

	// Распознавание стандартных команд.
	if isNeedReturn, err := checkBotCommands(s, msg); err != nil || isNeedReturn {
		return err
	}

	return s.tgClient.SendMessage(msg.UserID, txtUnknownCommand)

}

// SendReportToUser Отправка отчета за период.
func (s *Model) SendReportToUser(dt []bottypes.UserDataReportRecord, userID int64, reportKey string) error {
	ctx, span := tracer.Start(s.ctx, "SendReportToUser")
	s.ctx = ctx
	defer span.End()

	strReportTitle := "Отчёт за "
	switch reportKey {
	case "w":
		strReportTitle += "*последнюю неделю"
	case "m":
		strReportTitle += "*последний месяц"
	case "y":
		strReportTitle += "*последний год"
	}

	// Получение данных из БД.
	userCurrency := getUserCurrency(s, userID)
	answerText := formatReport(s, dt, userCurrency)
	if len(answerText) == 0 {
		answerText = txtReportEmpty
	} else {
		answerText = fmt.Sprintln(strReportTitle+" ("+userCurrency+")") + answerText
	}

	//Save in cache

	reportCacheKey := strconv.Itoa(int(userID)) + reportKey
	s.reportCache.Add(reportCacheKey, answerText)
	err := s.tgClient.SendMessage(userID, answerText)
	if err != nil {
		logger.Error("Error sending message to Telegram", "err", err)
	}
	return nil
}

// Область "Внешний интерфейс": конец.

// Область "Служебные функции": начало.

// Область "Распознавание входящих команд": начало.

// Проверка ввода суммы расхода по выбранной категории.
func checkIfEnterCategorySum(s *Model, msg Message, lastUserCat string) (bool, error) {
	if lastUserCat != "" && msg.Text != "" {
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
				return true, fmt.Errorf("insert data record error: %w", err)
			}
		}
		// Ответ пользователю об успешном сохранении.
		return true, s.tgClient.SendMessage(msg.UserID, txtRecSave)

	}

	//Ввели не расход
	return false, nil
}

func checkIfEnterNewCategory(s *Model, msg Message, lastUserCommand string) (bool, error) {
	if lastUserCommand == "/add_cat" {
		ctx, span := tracer.Start(s.ctx, "checkIfEnterNewCategory")
		s.ctx = ctx
		defer span.End()

		if msg.Text == "." {
			//Отмена ввода категории
			return true, nil
		} else {
			err := s.storage.InsertCategory(ctx, msg.UserID, msg.Text, msg.UserName)
			if err != nil {
				logger.Error("Error saving category", "err", err)
				return true, fmt.Errorf("insert category error: %w", err)
			}
			//Ответ пользователю об успещном сохранении
			return true, s.tgClient.SendMessage(msg.UserID, txtCatSave)
		}
	}
	return false, nil
}

func checkIfEnterNewLimit(s *Model, msg Message, lastUserCommand string) (bool, error) {
	if lastUserCommand == "/set_limit" {
		ctx, span := tracer.Start(s.ctx, "checkIfEnterNewLimit")
		s.ctx = ctx
		defer span.End()

		limit, err := parseAndConvertSumFromCurrency(s, msg.UserID, msg.Text)
		if err != nil {
			return true, err
		}

		if limit >= 0 {
			err := s.storage.SetUserLimit(ctx, msg.UserID, limit, msg.UserName)
			if err != nil {
				logger.Error("Error set currency", "err", err)
				return true, fmt.Errorf("set currency error: %w", err)
			}
			// Ответ пользователю об успешном сохранении.
			return true, s.tgClient.SendMessage(msg.UserID, fmt.Sprintf(txtLimitSet, msg.Text))
		}
	}
	// Это не ввод бюджета.
	return false, nil
}

func checkIfEnterTableData(s *Model, msg Message, lastUserCommand string) (bool, error) {
	if lastUserCommand == "/add_tbl" {
		ctx, span := tracer.Start(s.ctx, "checkIfEnterTableData")
		s.ctx = ctx
		defer span.End()

		answerText := ""
		if msg.Text == "." {
			//Ввод отменён
			return true, nil
		} else {

			lines := strings.Split(msg.Text, "\n")

			for i, line := range lines {
				isError := false
				txtError := ""

				rec, err := parseLineRec(line)

				if err != nil {
					isError = true
					txtError = "Ошибка распонавания формата строки."
				} else {
					// Сохранение данных.
					if err := s.storage.InsertCategory(s.ctx, msg.UserID, rec.Category, msg.UserName); err != nil {
						isError = true
						txtError = "Ошибка добавления категории."

					} else {
						rec.UserID = msg.UserID

						//Конвертация из валюты пользователя в базовую.
						if sum, err := convertSumFromCurrency(s, msg.UserID, rec.Sum); err != nil {
							isError = true
							txtError = "Ошибка конвертации валюты."

						} else {
							rec.Sum = sum
							//Сохранение записи
							if isOverLimit, err := s.storage.InsertUserDataRecord(s.ctx, msg.UserID, rec, msg.UserName, timeutils.BeginOfMonth(rec.Period)); err != nil {
								isError = true
								if isOverLimit {
									txtError = "Превышение бюджета."
								} else {
									txtError = "Ошибка сохранения записи."
								}
							}
						}
					}
				}
				if isError {
					answerText += fmt.Sprintf("%v. Ошибка. %v", i+1, txtError)
				} else {

				}
			}
			// Ответ пользователю об сохранении.
			answerText = txtRecSave + "\n" + answerText
			return true, s.tgClient.SendMessage(msg.UserID, answerText)
		}

	}
	//Это не ввод прошлых данных
	return false, nil
}

func checkIfChoiceCategory(s *Model, msg Message) (bool, error) {
	if msg.IsCallback {
		if strings.Contains(msg.Text, "/cat") {
			ctx, span := tracer.Start(s.ctx, "checkIfChoiceCategory")
			s.ctx = ctx
			defer span.End()

			cat := strings.Replace(msg.Text, "/cat ", "", -1)
			answerText := fmt.Sprintf(txtCatChoice, cat, getUserCurrency(s, msg.UserID))
			s.lastUserCat[msg.UserID] = cat
			return true, s.tgClient.SendMessage(msg.UserID, answerText)
		}
	}
	return false, nil
}

func checkIfChoiceCurrency(s *Model, msg Message) (bool, error) {
	if msg.IsCallback {
		if strings.Contains(msg.Text, "/curr ") {
			ctx, span := tracer.Start(s.ctx, "checkIfChoiceCurrency")
			s.ctx = ctx
			defer span.End()

			choice := strings.Replace(msg.Text, "/curr ", "", -1)
			answerText := fmt.Sprintf(txtCurrencySet, choice)

			if err := s.storage.SetUserCurrency(s.ctx, msg.UserID, choice, msg.UserName); err != nil {
				return true, s.tgClient.SendMessage(msg.UserID, txtCurrencySetError)
			} else {
				return true, s.tgClient.SendMessage(msg.UserID, answerText)
			}
		}
	}

	return false, nil
}

func checkBotCommands(s *Model, msg Message) (bool, error) {
	ctx, span := tracer.Start(s.ctx, "checkBotCommands")
	s.ctx = ctx
	defer span.End()

	switch msg.Text {
	case "/start":
		displayName := msg.UserDisplayName
		if len(displayName) == 0 {
			displayName = msg.UserName
		}
		return true, s.tgClient.ShowInlineButtons(fmt.Sprintf(txtStart, displayName), btnStart, msg.UserID)

	case "/report":
		return true, s.tgClient.SendMessage(msg.UserID, txtReportQP)
	case "/help":
		return true, s.tgClient.SendMessage(msg.UserID, txtHelp)
	case "/add_tbl":
		s.lastUserCommand[msg.UserID] = "/add_tbl"
		userCurrency := getUserCurrency(s, msg.UserID)
		return true, s.tgClient.SendMessage(msg.UserID, fmt.Sprintf(txtRecTbl, userCurrency))

	case "/report_w", "/report_m", "/report_y":
		return true, s.tgClient.SendMessage(msg.UserID, getReportByPeriod(s, msg))
	case "/add_cat":
		s.lastUserCommand[msg.UserID] = "/add_cat"
		return true, s.tgClient.SendMessage(msg.UserID, txtCatAdd)
	case "/add_rec":
		s.lastUserCommand[msg.UserID] = "/add_rec"
		// Отображение кнопок с существующими категориями для выбора.
		if btnCat, err := getCategoryButtons(s, msg.UserID); err != nil {
			return true, err
		} else {
			return true, s.tgClient.ShowInlineButtons(txtCatView, btnCat, msg.UserID)
		}
	case "/choice_currency":
		userCurrency := getUserCurrency(s, msg.UserID)
		if btnCurr, err := getCurrencyButtons(s, userCurrency); err != nil {
			return true, err
		} else {
			return true, s.tgClient.ShowInlineButtons(fmt.Sprintf(txtCurrencyChoice, userCurrency), btnCurr, msg.UserID)
		}
	case "/set_limit":
		s.lastUserCommand[msg.UserID] = "/set_limit"
		answerText := fmt.Sprintf(txtLimitInfo, "без ограничений")
		userLimit, _ := getUserLimit(s, msg.UserID)
		if userLimit > 0 {
			answerText = fmt.Sprintf(txtLimitInfo, userLimit)
		}
		return true, s.tgClient.SendMessage(msg.UserID, answerText)
	}
	// Команда не распознана.
	return false, nil
}

// Область "Распознавание входящих команд": конец.

// Область "Формирование отчета": начало.

func getReportByPeriod(s *Model, msg Message) string {
	ctx, span := tracer.Start(s.ctx, "getReportByPeriod")
	s.ctx = ctx
	defer span.End()

	answerText := ""
	reportKey := strings.Replace(msg.Text, "report_", "", -1)

	// Ключ для поиска в кэше.
	reportCacheKey := strconv.Itoa(int(msg.UserID)) + reportKey
	// Попытка получить значение из кэша.
	cacheValue := s.reportCache.Get(reportCacheKey)
	if cacheValue != nil {
		answerText, ok := cacheValue.(string)
		if ok {
			return answerText
		} else {
			logger.Error("Error converting the cache value to a string.")
		}
	}

	//Отправка запроса на формирование отчета в кафку.
	p, o, err := s.kafkaProducer.SendMessage(strconv.Itoa(int(msg.UserID)), reportKey)
	if err != nil {
		logger.Error("Error send message to Kafka", "err", err)
		answerText = txtReportError
	} else {
		logger.Debug(fmt.Sprintf("[KAFKA] Successful to write message, topic %s, offset: %d, partition: %d\n", s.kafkaProducer.GetTopic(), o, p))
		answerText = txtReportWait
	}

	return answerText
}

func formatReport(s *Model, recs []bottypes.UserDataReportRecord, userCurrency string) string {
	var res strings.Builder
	totalSum := 0.0
	for i, rec := range recs {
		sumCurrency, err := s.currencies.ConvertSumFromBaseToCurrency(userCurrency, rec.Sum)
		if err != nil {
			logger.Error("Error currency convertation", "err", err)
			return "ошибка конвертации валюты"
		}
		recs[i].Sum = sumCurrency
		totalSum += sumCurrency
	}
	maxSumStr := fmt.Sprintf("%.2f", totalSum)

	res.WriteString(fmt.Sprintf("`%*s | %v`", len(maxSumStr)+1, "Сумма", "Категория") + "\n")
	res.WriteString(fmt.Sprintf("`%v`", strings.Repeat("-", len(maxSumStr)+15)) + "\n")

	for _, rec := range recs {
		// Форматирование категории и числа до нужной ширины.
		res.WriteString(fmt.Sprintf("`%*.2f | %v`", len(maxSumStr)+1, rec.Sum, rec.Category) + "\n")
	}

	if len(recs) > 0 {
		res.WriteString(fmt.Sprintf("`%v`", strings.Repeat("-", len(maxSumStr)+15)) + "\n")
		res.WriteString(fmt.Sprintf("`%*.2f | %v`", len(maxSumStr)+1, totalSum, "ИТОГО") + "\n")
	}
	return res.String()
}

// Область "Получение данных пользователя": начало.
func getCategoryButtons(s *Model, userID int64) ([]bottypes.TgRowButtons, error) {
	userCategories, err := s.storage.GetUserCategories(s.ctx, userID)
	if err != nil {
		logger.Error("Error getting user categories", "err", err)
		return nil, fmt.Errorf("get user categories error: %w", err)
	}

	if len(userCategories) == 0 {
		return nil, s.tgClient.SendMessage(userID, txtCatEmpty)
	}

	var catButtons = []bottypes.TgRowButtons{}
	rowCounter := 0
	catButtons = append(catButtons, bottypes.TgRowButtons{})
	for i, cat := range userCategories {
		if i%3 == 0 && i > 0 {
			rowCounter++
			catButtons = append(catButtons, bottypes.TgRowButtons{})
		}
		catButtons[rowCounter] = append(catButtons[rowCounter], bottypes.TgInlineButton{DisplayName: cat, Value: "/cat " + cat})
	}

	return catButtons, nil
}

func getCurrencyButtons(s *Model, userCurrency string) ([]bottypes.TgRowButtons, error) {
	userCurrencies := s.currencies.GetCurrenciesList()
	rowCounter := 0
	var curButtons = []bottypes.TgRowButtons{}

	for i, cur := range userCurrencies {
		if cur == userCurrency {
			userCurrencies[0], userCurrencies[i] = userCurrencies[i], userCurrencies[0]
		}
	}

	for i, cur := range userCurrencies {
		if (i%3 == 0 || i > 0) && cur != userCurrency {
			rowCounter++
			curButtons = append(curButtons, bottypes.TgRowButtons{})
		}
		curButtons[rowCounter] = append(curButtons[rowCounter], bottypes.TgInlineButton{DisplayName: cur, Value: "/curr"})
	}
	return curButtons, nil
}

func getUserCurrency(s *Model, userID int64) string {
	userCurrency, _ := s.storage.GetUserCurrency(s.ctx, userID)
	if userCurrency == "" {
		userCurrency = s.currencies.GetMainCurrency()
	}
	return userCurrency
}

func getUserLimit(s *Model, userID int64) (float64, error) {
	userLimit, err := s.storage.GetUserLimit(s.ctx, userID)
	if err != nil {
		logger.Error("Error getting limit", "err", err)
		return 0, err
	}
	return userLimit, nil
}

// Область "Получение данных пользователя": конец.

// Область "Другие функции": начало.

func parseLineRec(line string) (bottypes.UserDataRecord, error) {
	matches := lineRegexp.FindStringSubmatch(line)
	// [всё регулярное выражение], [Дата], [Цена], [Категория]
	if len(matches) < 4 {
		return bottypes.UserDataRecord{}, errors.New("неверный формат строки")
	}

	dateStr := matches[1]
	priceStr := matches[2]
	category := matches[3]

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return bottypes.UserDataRecord{}, fmt.Errorf("incorrect price: %w", err)
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return bottypes.UserDataRecord{}, fmt.Errorf("incorrect date: %w", err)
	}

	return bottypes.UserDataRecord{
		Category: category,
		Sum:      price,
		Period:   date,
	}, nil

}

func convertSumFromCurrency(s *Model, userID int64, sum float64) (float64, error) {
	userCurrency := getUserCurrency(s, userID)
	sumBase, err := s.currencies.ConvertSumFromCurrencyToBase(userCurrency, sum)
	if err != nil {
		logger.Error("Error convertation currency", "err", err)
		return 0, err
	}
	return sumBase, nil

}

// Парсинг вводимого пользователем числа и конвертация суммы в базовую валюту.
func parseAndConvertSumFromCurrency(s *Model, userID int64, sumString string) (float64, error) {
	sum, err := strconv.ParseFloat(sumString, 64)
	if err != nil {
		return 0, fmt.Errorf("error parse sum: %w", err)
	}

	if sum, err = convertSumFromCurrency(s, userID, sum); err != nil {
		return 0, fmt.Errorf("error currency convertation: %w", err)
	}
	return sum, nil
}
