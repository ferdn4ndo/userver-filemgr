package lib

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

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
	sslMode := "disable"
	if !env.IsLocal() {
		sslMode = "require"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPass, env.DBHost, env.DBPort, env.DBName, sslMode)
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		logger.Panic(err)
	}
	db.SetMaxOpenConns(25)
	db.SetConnMaxIdleTime(90 * time.Second)
	db.SetMaxIdleConns(5)

	lc.Append(fx.StopHook(func(ctx context.Context) error {
		return db.Close()
	}))

	logger.Info("Database connection established")
	return Database{DB: db}
}
