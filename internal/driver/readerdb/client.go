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
	// sqlite3の:memory:はコネクションごとに別DBになるため、プールが新規コネクションを
	// 開くと別の空DBに繋がってしまう（SetMaxOpenConns(1)だけでは、破棄されたコネクションの
	// 代わりに新規コネクションが開かれた場合に防げない）。cache=sharedで全コネクションが
	// 同一のインメモリDBを参照するようにし、SetMaxOpenConns(1)で同時アクセスによる
	// SQLITE_BUSYも避ける。
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
