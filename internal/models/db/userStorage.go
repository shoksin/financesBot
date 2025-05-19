package db

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/shoksin/financesBot/internal/helpers/dbutils"
)

type UserDataReportRecordDB struct {
	Category string  `db:"name"`
	Sum      float64 `db:"sum"`
}

type UserStorage struct {
	db              *sqlx.DB
	defaultCurrency string
	defaultLimits   float64
}

func NewUserStorage(db *sqlx.DB, defaultCurrecny string, defaultLimits float64) *UserStorage {
	return &UserStorage{db: db, defaultCurrency: defaultCurrecny, defaultLimits: defaultLimits}
}

func (storage *UserStorage) InsertUser(ctx context.Context, userID int64, userName string) error {
	const sqlString = `
		INSER INTO users (tg_id, name, currency, limits)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tg_id) DO NOTHING;`

	if _, err := dbutils.Exec(ctx, storage.db, sqlString, userID, userName, storage.defaultCurrency, storage.defaultLimits); err != nil {
		return err
	}
	return nil
}

func (storage *UserStorage) CheckIfUserExist(ctx context.Context, userID int64) (bool, error) {
	const sqlString = `SELECT COUNT(id) as countusers FROM users WHERE tg_id = $1;` // TODO: potential not id, but tg_id

	cnt, err := dbutils.GetMap(ctx, storage.db, sqlString, userID)
	if err != nil {
		return false, err
	}

	countusers, ok := cnt["countusers"].(int64)
	if !ok {
		return false, errors.New("error in type conversion of the query result")
	}
	if countusers == 0 {
		return false, nil
	}
	return true, nil
}

func (storage *UserStorage) CheckIfUserExistAndAdd(ctx context.Context, userID int64, userName string) (bool, error) {
	exist, err := storage.CheckIfUserExist(ctx, userID)
	if err != nil {
		return false, err
	}

	if !exist {
		err = storage.InsertUser(ctx, userID, userName)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}
