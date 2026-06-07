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

---

## 機能要求（フェーズ別）

### フェーズ 1：RSS リンクの読み込み

**概要**: テキストファイルから RSS フィードの URL 一覧を取得する

**受け入れ条件**:
- `feeds.txt` に改行区切りで記載された URL を読み込める
- 空行・コメント行（`#` 始まり）はスキップする
- URL が 0 件の場合は警告メッセージを出力して終了する

---

### フェーズ 2：記事の取得と標準出力

**概要**: フィード URL にアクセスし、取得した記事情報を標準出力に表示する

**受け入れ条件**:
- RSS 2.0 および Atom 形式のフィードを解析できる
- 各記事について以下を出力する：タイトル・URL・公開日時
- フィードへのアクセスに失敗した場合はエラーを表示し、次のフィードに続行する

---

### フェーズ 3：記事の SQLite 保存

**概要**: 取得した記事を SQLite データベースに保存する

**受け入れ条件**:
- 初回起動時にデータベースとテーブルを自動作成する
- 同一 URL の記事は重複して保存しない（URL を一意キーとする）
- 保存成功時に件数をログ出力する
- Claude Code が `rss-feeder fetch` を Bash ツールで実行した後、**PostToolUse Hook** が Go バイナリ経由で保存操作を audit_log に記録する

---

### フェーズ 4：取得済み記事の一覧表示

**概要**: データベースに保存済みの記事を一覧表示する

**受け入れ条件**:
- 未読のみ・お気に入りのみ・全件 の切り替えオプションを持つ
- デフォルトは未読記事のみ表示
- 表示した記事は既読（`read = 1`）に自動マークする
- 表示項目：ID・タイトル・URL・公開日時・既読フラグ・お気に入りフラグ

---

### フェーズ 5：お気に入り登録

**概要**: 記事 ID を指定してお気に入り（ブックマーク）に登録・解除する

**受け入れ条件**:
- 記事 ID を引数に取り、お気に入りの登録・解除をトグルできる（`bookmarked` を 0/1 で切り替え）
- 存在しない ID を指定した場合はエラーを返す
- Claude Code が `rss-feeder bookmark <id>` を Bash ツールで実行する前、**PreToolUse Hook** が Go バイナリ経由で ID の存在確認を実施する
- 実行後、**PostToolUse Hook** が Go バイナリ経由で操作を audit_log に記録する

---

### フェーズ 6：記事のリセット

**概要**: お気に入り登録済みを除いた記事をデータベースから削除する

**受け入れ条件**:
- `bookmarked = 0` の記事のみを削除する
- 実行前に削除件数を表示し、確認を求める（`-y` フラグで確認をスキップ可能）
- Claude Code が `rss-feeder reset` を Bash ツールで実行する前、**PreToolUse Hook** が Go バイナリ経由でお気に入り記事が削除対象に含まれないことを保証する
- 削除後に残件数を表示する

---

## 非機能要求

| 項目 | 要求 |
|------|------|
| 言語 | Go 1.21 以上 |
| データストア | SQLite（`reader.db`） |
| 開発環境 | VSCode + devcontainer |
| ビルド | `go build` 単一バイナリ |
| テスト | 各フェーズに対応するユニットテストを用意 |
| ログ | 操作ログは SQLite の `audit_log` テーブルに記録 |

---

## スコープ外

- Web UI・API サーバー
- プッシュ通知・定期自動取得（cron 等）
- 複数ユーザーの管理
- RSS 以外のフォーマット（JSON Feed 等）

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
| PostToolUse | フェーズ 3・5 の書き込み後 | Go バイナリ経由で audit_log に自動記録 |
| PreToolUse | フェーズ 5・6 の書き込み前 | Go バイナリ経由で ID 存在確認・お気に入り保護 |
| Stop | セッション終了時 | Go バイナリ経由で DB の VACUUM・整合性チェック |
