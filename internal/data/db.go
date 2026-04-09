package data

import (
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

var ErrNotFound = errors.New("not found")

// DB wraps sqlx for data access (legacy table names).
type DB struct {
	db *sqlx.DB
}

func NewDB(database lib.Database) *DB {
	return &DB{db: database.DB}
}
