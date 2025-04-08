// Package dbutils Хелпер-обёртка для выполнения запросов на базе sqlx и для функций подключения к БД (pgx).
package dbutils

// Хелпер-обёртка для функций подключения к БД (pgx)

// pgxLogger Логгер для pgx, реализующий интерфейс Logger пакета pgx.
type pgxLogger struct{}

func NewDBConnect(connString string) {
	connString = ""
}
