package readerdb

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/do/v2"
)

func NewClient(_ do.Injector) (*sql.DB, error) {
	return sql.Open("sqlite3", "reader.db")
}

func NewInMemoryDB() (*sql.DB, error) {
	return sql.Open("sqlite3", ":memory:")
}
