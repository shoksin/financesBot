package bottypes

import "time"

type Empty struct{}

// Множество уникальных категорий покупок пользователя
type UserCategorySet map[string]Empty

type UserDataRecord struct {
	UserID   int64
	Category string
	Sum      float64
	Period   time.Time
}

type UserDataReportRecord struct {
	Category string
	Sum      float64
}

// Типы для описания состава кнопок телеграм сообщения.
// Кнопка сообщения.
type TgInlineButton struct {
	DisplayName string
	Value       string
}

// Строка с кнопками сообщения.
type TgRowButton []TgInlineButton

// Тип для хранения курса валюты в формате "USD" = 0.01659657
type ExchangeRate map[string]float64
