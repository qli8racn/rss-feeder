package domain

import "time"

// User は cmd/mcp・cmd/rss-feeder・cmd/web・cmd/agent のプロセスが動作する際に
// 固定されるMCPサーバー利用者（クライアント）を表す。
type User struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// DefaultUserName は cmd/rss-feeder・cmd/web・cmd/agent が常に暗黙的に利用する
// デフォルトユーザーの識別子。マイグレーション（既存データの紐付け先）と
// 各エントリポイントの main.go の両方から参照し、文字列のずれを防ぐ。
const DefaultUserName = "default"
