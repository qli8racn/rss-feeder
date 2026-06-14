# 設計：Web ブラウザでの記事閲覧

## エントリーポイント

CLI（`rss-feeder`）と Web API（`rss-feeder-web`）はバイナリを分離する。
`cmd/rss-feeder` は CLI 専用のままとし、HTTP サーバーは `cmd/web` を新しいエントリーポイントとする。

```bash
go build -o bin/rss-feeder-web ./cmd/web
./bin/rss-feeder-web [--port <port>] [--static-dir <dir>]
```

| フラグ | デフォルト | 説明 |
|------|----------|------|
| `--port` | `8080` | HTTP サーバーのリスニングポート |
| `--static-dir` | `web/static` | フロントエンド静的ファイルの配置ディレクトリ |

`cmd/agent`（rss-agent）と同様に、`cmd/web` 内で独自に `do.Injector` を構築し、必要な usecase
（`ListUsecase` / `SearchUsecase` / `BookmarkUsecase` / `AuditUsecase`）のみを組み立てる。
ルーティング自体は `internal/adapter/handler.NewMux(...)` として共通化し、`http.Handler`（chi ルーター）を返す。

## API 仕様

### `GET /api/articles`

| クエリパラメータ | 説明 |
|------|------|
| `mode` | `all`（デフォルト）/ `unread` / `bookmarked`。既存 `ListUsecase` の `ListMode` に対応 |

レスポンス: `Article` の JSON 配列。

### `GET /api/articles/search`

| クエリパラメータ | 説明 |
|------|------|
| `q` | 検索キーワード（必須。空の場合は 400） |
| `bookmarked` | `true` の場合お気に入りのみに絞り込み |

レスポンス: `Article` の JSON 配列。

### `POST /api/articles/{id}/bookmark`

- パスパラメータ `id` の記事のお気に入りをトグルする（既存 `BookmarkUsecase` を再利用）
- 成功時、既存 CLI と同様に `AuditUsecase` で `audit_log` に記録する（`action=bookmark`, `article_id=<id>`）
- レスポンス: 更新後の `Article` を JSON で返す
- 記事が存在しない場合は 404

### Article の JSON 表現

```json
{
  "id": 1,
  "feed_id": 1,
  "feed_url": "https://example.com/feed",
  "url": "https://example.com/articles/1",
  "title": "記事タイトル",
  "content": "本文",
  "published_at": "2026-06-14T00:00:00Z",
  "read": true,
  "bookmarked": false,
  "fetched_at": "2026-06-14T01:00:00Z"
}
```

`domain.Article` に JSON タグを付与せず、`adapter/handler` 層で DTO に変換して返す（domain を外側の表現に依存させない）。

### 静的ファイル配信

- `--static-dir` で指定したディレクトリを `/` 以下にそのまま配信する（`http.FileServer`）
- フロントエンド（HTML/CSS/JS）は figma-mcp で別途生成し、このディレクトリに配置する

## 実装方針

- ルーティングには `github.com/go-chi/chi/v5` を使用する
  - ミドルウェア（`middleware.Logger` / `middleware.Recoverer`）と `github.com/go-chi/cors` を適用し、figma-mcp 製フロントエンドを別ポートで開発する際の CORS に対応する
- 既存の `ListUsecase` / `SearchUsecase` / `BookmarkUsecase` / `AuditUsecase` をそのまま再利用し、ビジネスロジックの重複を避ける

## 追加ファイル（新規実装）

| ファイル | 役割 |
|---------|------|
| `internal/adapter/handler/serve.go` | `NewMux()`：HTTP ルーティング・DTO 変換 |
| `cmd/web/main.go` | Web API エントリーポイント（DI構築・HTTPサーバー起動） |
| `web/static/index.html` | プレースホルダー（figma-mcp 生成物に置き換え予定） |
