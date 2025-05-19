package dbutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Форматирование текстов ошибок.
func sqlErr(err error, query string, args ...any) error {
	return fmt.Errorf(`run query "%s" with args %+v: %w`, query, args, err)
}

// Заполнение запросов именованными параметрами.
func namedQuery(query string, arg any) (nq string, args []any, err error) {
	nq, args, err = sqlx.Named(query, arg)
	if err != nil {
		return "", nil, sqlErr(err, query, args...)
	}
	return nq, args, nil
}

// Exec Выполнение запросов с параметрами (неименованные, в виде $1...$n).
func Exec(ctx context.Context, db sqlx.ExecerContext, query string, args ...any) (sql.Result, error) {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return res, sqlErr(err, query, args...)
	}
	return res, nil
}

// Выполнение запросов с именованными параметрами.
func NamedExec(ctx context.Context, db sqlx.ExtContext, query string, arg any) (sql.Result, error) {
	nq, args, err := namedQuery(query, arg)
	if err != nil {
		return nil, err
	}
	return Exec(ctx, db, db.Rebind(nq), args...)
}

// Выборка по запросу с параметрами (неименованные, в виде $1...$n).
func Select(ctx context.Context, db sqlx.QueryerContext, dest any, query string, args ...any) error {
	if err := sqlx.SelectContext(ctx, db, dest, query, args...); err != nil {
		return sqlErr(err, query, args...)
	}
	return nil
}

// GetMap Выборка по запросу с параметрами (неименованные, в виде $1...$n).
func GetMap(ctx context.Context, db sqlx.QueryerContext, query string, args ...any) (ret map[string]any, err error) {
	row := db.QueryRowxContext(ctx, query, args...)
	if row.Err() != nil {
		return nil, sqlErr(row.Err(), query, args...)
	}

	ret = map[string]any{}
	if err = row.MapScan(ret); err != nil {
		return nil, sqlErr(err, query, args...)
	}
	return ret, nil
}

// TxFunc Описание типа вложенной функции для выполнения в транзакции.
type TxFunc func(tx *sqlx.Tx) error

type TxRunner interface {
	BeginTxx(context.Context, *sql.TxOptions) (*sqlx.Tx, error)
}

func RunTx(ctx context.Context, db TxRunner, f TxFunc) (err error) {
	var tx *sqlx.Tx

	opts := &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	}

	// Запуск транзакции.
	tx, err = db.BeginTxx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// Откат или коммит транзакции при завершении функции.
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		} else {
			err = tx.Commit()
		}
	}()
	// Выполнение вложенной функции и возврат результата.
	return f(tx)
}
