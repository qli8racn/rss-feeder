# プロダクト要求定義書

## プロダクトビジョンと目的

Go 製 CLI ツールと SQLite を活用した RSS リーダーシステム。
Claude Code のエージェント・Hooks・SQLite の連携を段階的な機能開発（steering）で実践的に学習することを目的とする。

---

## ターゲットユーザーと課題・ニーズ

### ペルソナ

- Claude Code の理解を深めたいエンジニア
- RSS を使って情報を効率よく収集したい方

### 課題

- Feedly のような専門アプリは設定が煩雑で、カスタマイズに限界がある
- AI エージェントが使える現代において、専門アプリに依存せずとも自分のワークフローに合ったリーダーを構築できるはずである

### ニーズ

- 自分が決めた RSS フィードから記事を取得したい
- 取得済み記事を後から参照できるようにしたい
- 気になった記事をお気に入りとして保存したい
- 不要になった記事をリセットして管理を整理したい
- キーワードで過去の記事を検索したい

---

## 機能要求（フェーズ別）

各フェーズの要件・設計・タスクは `docs/steering/` 以下に格納する。

| フェーズ | 概要 | ディレクトリ |
|---------|------|-------------|
| 1 | RSS リンクの読み込み（`feeds.txt` ベース、フェーズ 8 で SQLite 移行済み） | [20260607_feed_loading](steering/20260607_feed_loading/) |
| 2 | 記事の取得と標準出力 | [20260607_article_fetch](steering/20260607_article_fetch/) |
| 3 | 記事の SQLite 保存 | [20260607_sqlite_save](steering/20260607_sqlite_save/) |
| 4 | 取得済み記事の一覧表示 | [20260607_article_list](steering/20260607_article_list/) |
| 5 | お気に入り登録 | [20260607_bookmark](steering/20260607_bookmark/) |
| 6 | 記事のリセット | [20260607_reset](steering/20260607_reset/) |
| 7 | 記事の検索 | [20260608_search](steering/20260608_search/) |
| 8 | フィード管理の SQLite 移行（add-feed / list-feeds / remove-feed） | [20260608_feed_management](steering/20260608_feed_management/) |
| 9 | Web ブラウザでの記事閲覧（JSON API + 静的フロントエンド配信） | [20260614_web_view](steering/20260614_web_view/) |
| 10 | 記事メタデータ拡充（出版元・サムネイル・要約・カテゴリ） | [20260614_article_metadata](steering/20260614_article_metadata/) |
| 11 | 設定ファイルによる ANTHROPIC_API_KEY 管理 | [20260615_config_apikey](steering/20260615_config_apikey/) |
| 12 | OpenAPI による API 仕様管理とコード生成 | [20260617_openapi_codegen](steering/20260617_openapi_codegen/) |
| — | Supabase（Postgres）DBドライバ対応（`db.driver`設定でSQLite/Supabaseを切替） | [20260726_supabase_db_driver](steering/20260726_supabase_db_driver/) |
| — | MCPサーバー利用者（クライアント）単位でのフィード管理（マルチユーザー対応） | [20260726_mcp_user_management](steering/20260726_mcp_user_management/) |

> **Note:** 上記フェーズ番号は `20260617_openapi_codegen`（フェーズ12）以降、付番を追従できておらず
> 欠番（`—`）のまま追加している。フェーズ12以降に追加された `docs/steering/` 配下の全ディレクトリ
> （フィード管理UI・フィードURL自動探索・curate/discover・構造化ログ・MCPサーバー等）を網羅した
> 一覧ではないため、最新の全体像は `ls docs/steering/` で確認すること。

---

## 非機能要求

| 項目 | 要求 |
|------|------|
| 言語 | Go 1.25 以上 |
| データストア | SQLite（`rss-feeder-db/reader.db`、デフォルト）または Supabase（Postgres）。`internal/config/config.yml` の `db.driver` で選択 |
| 開発環境 | VSCode + devcontainer |
| ビルド | `go build` 単一バイナリ |
| テスト | 各フェーズに対応するユニットテストを用意 |
| ログ | 操作ログは SQLite の `audit_log` テーブルに記録 |

---

## スコープ外

- プッシュ通知・定期自動取得（cron 等）
- RSS 以外のフォーマット（JSON Feed 等）
- 認証・認可（`cmd/mcp --user-id` はユーザー識別のみで、なりすまし対策のない任意の文字列。
  Web ビューはローカル利用のみを想定し引き続き認証を持たない）

> 「複数ユーザーの管理」は [20260726_mcp_user_management](steering/20260726_mcp_user_management/) で
> 対応済みのためスコープ外から除外（`cmd/mcp` のみ `--user-id` でユーザーを切替可能。
> `cmd/rss-feeder`・`cmd/web`・`cmd/agent` は引き続き単一の `default` ユーザー固定）。

---

## データベーススキーマ

SQLite版（`rss-feeder-db/schema.sql`）を正とする。Supabase（Postgres）版は方言のみ異なる
同等のスキーマ（詳細は `docs/steering/20260726_supabase_db_driver/design.md`）。

```sql
CREATE TABLE users (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT UNIQUE NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- name = 'default' は cmd/rss-feeder・cmd/web・cmd/agent が常に暗黙的に利用するユーザー。
-- cmd/mcp のみ --user-id で任意の識別子に切替可能（マルチユーザー対応）。

CREATE TABLE feeds (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id      INTEGER NOT NULL REFERENCES users(id),
  feed_url     TEXT NOT NULL,
  title        TEXT,
  last_fetched DATETIME,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, feed_url)
);

CREATE TABLE articles (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  feed_id      INTEGER NOT NULL,
  url          TEXT NOT NULL,
  title        TEXT NOT NULL,
  content      TEXT,
  published_at DATETIME,
  read         BOOLEAN DEFAULT 0,
  bookmarked   BOOLEAN DEFAULT 0,
  fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  publisher    TEXT,
  thumbnail_url TEXT,
  summary      TEXT,
  category     TEXT,
  FOREIGN KEY(feed_id) REFERENCES feeds(id),
  UNIQUE(feed_id, url)
);
-- articles には user_id を持たせず、feed_id → feeds.user_id の関連で所有ユーザーを判定する
-- （feeds が「1ユーザー1購読=1行」の設計のため）。url は旧グローバルUNIQUEからfeed_id複合UNIQUEに
-- 変更されており、同一ユーザーが内容の重なるフィードを複数購読していると記事が複数行になりうる。

CREATE TABLE audit_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  action     TEXT NOT NULL,
  article_id INTEGER,
  old_state  TEXT,
  new_state  TEXT,
  timestamp  DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(article_id) REFERENCES articles(id)
);
-- audit_log は user_id でスコープしていない（cmd/web・cmd/rss-feederからのみ書き込まれ、
-- 常にdefaultユーザーのため現状実害なし）。
```

---

## Hooks 設計方針

| Hook | タイミング | 用途 |
|------|-----------|------|
| PostToolUse | フェーズ 3・5・6 の書き込み後 | Go バイナリ経由で audit_log に自動記録 |
| PreToolUse | フェーズ 5・6 の書き込み前 | Go バイナリ経由で ID 存在確認・お気に入り保護 |
| Stop | セッション終了時 | Go バイナリ経由で DB の VACUUM・整合性チェック |
