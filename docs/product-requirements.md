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

---

## 非機能要求

| 項目 | 要求 |
|------|------|
| 言語 | Go 1.24 以上 |
| データストア | SQLite（`reader.db`） |
| 開発環境 | VSCode + devcontainer |
| ビルド | `go build` 単一バイナリ |
| テスト | 各フェーズに対応するユニットテストを用意 |
| ログ | 操作ログは SQLite の `audit_log` テーブルに記録 |

---

## スコープ外

- プッシュ通知・定期自動取得（cron 等）
- 複数ユーザーの管理
- RSS 以外のフォーマット（JSON Feed 等）
- 認証・認可（Web ビューはローカル利用のみを想定）

---

## データベーススキーマ

```sql
CREATE TABLE feeds (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  feed_url     TEXT UNIQUE NOT NULL,
  title        TEXT,
  last_fetched DATETIME,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE articles (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  feed_id      INTEGER NOT NULL,
  url          TEXT UNIQUE NOT NULL,
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
  FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
);

CREATE TABLE audit_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  action     TEXT NOT NULL,
  article_id INTEGER,
  old_state  TEXT,
  new_state  TEXT,
  timestamp  DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(article_id) REFERENCES articles(id)
);
```

---

## Hooks 設計方針

| Hook | タイミング | 用途 |
|------|-----------|------|
| PostToolUse | フェーズ 3・5・6 の書き込み後 | Go バイナリ経由で audit_log に自動記録 |
| PreToolUse | フェーズ 5・6 の書き込み前 | Go バイナリ経由で ID 存在確認・お気に入り保護 |
| Stop | セッション終了時 | Go バイナリ経由で DB の VACUUM・整合性チェック |
