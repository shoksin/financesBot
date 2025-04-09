package dbutils

//Хелпер-обёртка для функций подключения к БД (pgx).

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jmoiron/sqlx"
	"github.com/shoksin/financesBot/internal/logger"
)

// pgxLogger Логгер для pgx, реализующий интерфейс Logger пакета pgx.
type pgxLogger struct{}

// Log Функция реализации интерфейса Logger пакета pgx.
func (pl *pgxLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	var buffer bytes.Buffer
	buffer.WriteString(msg)
	for k, v := range data {
		buffer.WriteString(fmt.Sprintf(" %s=%+v", k, v))
	}
	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelNone, tracelog.LogLevelDebug:
		logger.Debug(buffer.String())
	case tracelog.LogLevelInfo:
		logger.Info(buffer.String())
	case tracelog.LogLevelWarn:
		logger.Warning(buffer.String())
	case tracelog.LogLevelError:
		logger.Error(buffer.String())
	default:
		logger.Debug(buffer.String())
	}
}

func NewDBConnect(connString string) (*sqlx.DB, error) {
	connConfig, err := pgx.ParseConfig(connString)
	if err != nil {
		logger.Error("Error parsing connection string", "err", err)
		return nil, err
	}

	connConfig.RuntimeParams["application_name"] = "tg-bot"
	connConfig.Tracer = &tracelog.TraceLog{
		Logger:   &pgxLogger{},
		LogLevel: tracelog.LogLevelDebug,
	}

	connStr := stdlib.RegisterConnConfig(connConfig)

	db, err := sqlx.Connect("pgx", connStr)
	if err != nil {
		logger.Error("Error connecing to DB", "err", err)
		return nil, err
	}
	logger.Info("Successful connection to DB")
	return db, nil
}
