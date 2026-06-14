# フェーズ 10：記事メタデータ拡充（出版元・サムネイル・要約・カテゴリ）

## 概要

記事（`Article`）に以下のメタデータを追加し、保存・取得できるようにする。

- 出版元（Publisher）
- サムネイル画像URL（ThumbnailURL）
- 要約（Summary）
- カテゴリ（Category）

## 背景・目的

- 一覧表示時に記事のソースやサムネイルが分かると視認性が上がる
- 大量の記事を要約・カテゴリ分類できると、後から効率的に振り返れる
- 既存の `rss-agent summarize` は実行時にその場で要約するのみで、結果が DB に残らない。
  要約・カテゴリを記事に永続化することで、一覧・検索・Web API から再利用できるようにする

## 受け入れ条件

### 出版元・サムネイル（フィード取得時に自動設定）

- `articles` テーブルに `publisher`（TEXT）、`thumbnail_url`（TEXT）カラムを追加する
- `fetch` コマンド実行時、RSS フィードから以下を取得し保存する
  - `publisher`: フィードの `<title>`（`Feed.Title`）
  - `thumbnail_url`: 記事の `<media:thumbnail>` / `<enclosure type="image/*">` / `<itunes:image>` のいずれかから取得（優先順位あり、いずれも無ければ空文字）

### 要約・カテゴリ（AIエージェントによる後付け）

- `articles` テーブルに `summary`（TEXT）、`category`（TEXT）カラムを追加する
- `rss-agent enrich` コマンドを追加し、要約・カテゴリが未設定の記事に対して Claude に要約・カテゴリ分類させ、結果を DB に保存する
  - カテゴリは固定リストではなく Claude が記事内容から自由に分類する（例: "Tech", "Business", "Sports" など）
- 既に `summary` が設定済みの記事は再処理しない（`--force` オプションで再実行可能）

### 表示

- `list` / `search` コマンドの出力に `Publisher` を表示する（`FeedURL` の代わり、または併記）
- Web API（`GET /api/articles` など）のレスポンスに `publisher` / `thumbnail_url` / `summary` / `category` を含める

## スコープ外（このフェーズ）

- フロントエンド（`web/static/`）でのサムネイル表示・カテゴリフィルタ UI
- カテゴリの固定マスタ管理・カテゴリ別一覧コマンド
- 要約の多言語対応・要約フォーマットのカスタマイズ
- 既存記事に対する一括 `publisher` / `thumbnail_url` のバックフィル（再 `fetch` で対応）
