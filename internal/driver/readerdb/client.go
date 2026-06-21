package readerdb

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/samber/do/v2"
)

// dsn は WAL モード（複数コネクションからの同時読み書きで読み取りをブロックしない）と
// busy_timeout（書き込みロック競合時に即座に SQLITE_BUSY を返さず、指定msだけ待ってリトライする）
// を設定する。cmd/web・cmd/rss-feeder・cmd/agent の enrich（並列バッチ）等から同時に
// DB へアクセスされる可能性があるため。
const dsn = "reader.db?_journal_mode=WAL&_busy_timeout=5000"

func NewClient(_ do.Injector) (*sql.DB, error) {
	return sql.Open("sqlite3", dsn)
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
