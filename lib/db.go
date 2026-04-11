package lib

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/fx"
)

type QueryAble interface {
	sqlx.Ext
	NamedExec(query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	Select(dest interface{}, query string, args ...interface{}) error
	Get(dest interface{}, query string, args ...interface{}) error
}

type Database struct {
	*sqlx.DB
}

func (d Database) StdDB() *sql.DB {
	return d.DB.DB
}

func NewDatabase(env Env, logger Logger, lc fx.Lifecycle) Database {
	dbUser := url.QueryEscape(env.DBUser)
	dbPass := url.QueryEscape(env.DBPassword)
	sslMode := env.DBSSLMode
	if sslMode == "" {
		if env.IsProduction() {
			sslMode = "require"
		} else {
			sslMode = "disable"
		}
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPass, env.DBHost, env.DBPort, env.DBName, sslMode)
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		logger.Panic(err)
	}
	db.SetMaxOpenConns(env.DBMaxOpenConns)
	db.SetConnMaxIdleTime(env.DBConnMaxIdleTime)
	db.SetMaxIdleConns(env.DBMaxIdleConns)
	if env.DBConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(env.DBConnMaxLifetime)
	}

	lc.Append(fx.StopHook(func(ctx context.Context) error {
		return db.Close()
	}))

	logger.Info("Database connection established (sslmode=" + sslMode + ")")
	return Database{DB: db}
}
