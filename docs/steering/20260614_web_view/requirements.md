# フェーズ 9：Web ブラウザでの記事閲覧

## 概要

`rss-feeder serve` コマンドで HTTP サーバーを起動し、ブラウザから保存済み記事を閲覧できるようにする。

## 背景・目的

- GitHub Copilot の活用（ローカルでのコードレビュー、リモートでの PR レビュー）の練習対象を作る
- figma-mcp によるフロントエンド自動生成の練習対象を作る

バックエンドは JSON API として最小実装にとどめ、フロントエンド（HTML/CSS/JS）は別途 figma-mcp で生成したものを静的ファイルとして配置する想定。

## 受け入れ条件

- CLI（`rss-feeder`）と Web API はエントリーポイント（バイナリ）を分離する
  - CLI: `cmd/rss-feeder`（既存）
  - Web API: `cmd/web`（新規、`rss-feeder-web` としてビルド）
- `rss-feeder-web [--port <port>]` でローカル HTTP サーバーが起動する（デフォルトポート 8080）
- `GET /api/articles` で記事一覧を JSON で取得できる
  - `?mode=all|unread|bookmarked` で既存 `list` コマンドと同じフィルタを指定できる（デフォルト: `all`）
- `GET /api/articles/search?q=<keyword>` でキーワード検索結果を JSON で取得できる
  - `&bookmarked=true` でお気に入りのみに絞り込める
- `POST /api/articles/{id}/bookmark` で記事のお気に入りをトグルし、結果を JSON で返す
  - 既存 CLI の `bookmark` コマンドと同様に `audit_log` に記録する
- 静的ファイル（フロントエンド）は `web/static/` 配下から配信される
- 認証・認可は行わない（ローカル利用のみを想定）

## 追加要件（Figma デザイン反映、2026-06-17）

フロントエンドの記事一覧画面を Figma デザインに合わせて実装するため、API に以下を追加する。
詳細な画面仕様は `docs/web-ui-spec.md` を参照。

- `GET /api/articles` / `GET /api/articles/search` に以下のクエリパラメータを追加する
  - `category`: カテゴリで絞り込み
  - `sort`: `title` / `publisher` / `category` / `published_at`
  - `order`: `asc` / `desc`
  - `page`, `per_page`: ページネーション（デフォルト `per_page=25`）
- 上記2エンドポイントのレスポンスを `{"articles": [...], "total": <int>, "page": <int>, "per_page": <int>}` の形式に変更する
- カテゴリドロップダウンの選択肢生成用に `GET /api/categories`（DISTINCT な `category` 一覧を返す）を追加する

## スコープ外（このフェーズ）

- フロントエンドの実装（React + TypeScript（Vite + Tailwind CSS）。figma-mcp でコンポーネント初期コードを生成し別途実装する。詳細は `docs/web-ui-spec.md` 参照）
- フィード管理（add-feed 等）の Web UI 化
- リセット・要約などのその他コマンドの Web 化
