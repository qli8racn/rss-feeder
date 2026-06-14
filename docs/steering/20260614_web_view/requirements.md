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

## スコープ外（このフェーズ）

- フロントエンドの実装（figma-mcp で別途生成）
- フィード管理（add-feed 等）の Web UI 化
- リセット・要約などのその他コマンドの Web 化
