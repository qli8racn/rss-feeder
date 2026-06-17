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
  "fetched_at": "2026-06-14T01:00:00Z",
  "publisher": "Example Times",
  "thumbnail_url": "https://example.com/thumb.jpg",
  "summary": "記事の要約",
  "category": "Tech"
}
```

`publisher` / `thumbnail_url` / `summary` / `category` はフェーズ10（`docs/steering/20260614_article_metadata/`）で追加されたフィールド。未設定の記事では空文字になる。

`domain.Article` に JSON タグを付与せず、`adapter/handler` 層で DTO に変換して返す（domain を外側の表現に依存させない）。

### 静的ファイル配信

- `--static-dir` で指定したディレクトリを `/` 以下にそのまま配信する（`http.FileServer`）
- フロントエンドは React + TypeScript（Vite）で実装し、ビルド成果物をこのディレクトリに配置する（詳細は下記「フロントエンド構成」参照）

### フロントエンド構成（React + TypeScript）

```
web/
├── frontend/   # 開発で使うソースコード（Vite + React + TypeScript + Tailwind CSS、Git管理対象）
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
└── static/     # ビルド成果物（npm run build の出力先。.gitignore 対象に変更予定）
```

- ソース: `web/frontend/`。コンポーネント初期コードは figma-mcp（`get_design_context`）で生成し、API連携・状態管理を手動で配線する（詳細は `docs/web-ui-spec.md` 参照）
- ビルド: `npm run build`（Vite の `build.outDir` を `../static` に向け、`web/static/` に直接出力する）
- 配信側（`cmd/web` / `--static-dir`）の変更は不要。ビルド済み静的アセットを配信するのみ
- ルーティング不要（単一画面のため React Router 等は導入しない）。状態管理は `useState` / `useEffect` で十分とする
- `web/static/` はビルド生成物のため `.gitignore` 対象とし、現行のプレースホルダー `index.html` は削除する
- Go バイナリの成果物（`bin/`）とは別管理とする。`bin/` はルート直下のまま変更しない（既存の `.gitignore` で除外済み、`AGENTS.md` / `README.md` の既存ビルド手順とも整合させるため統合しない）

## API 仕様（追加分・Figma デザイン反映）

詳細な画面仕様は `docs/web-ui-spec.md` を参照。

### `GET /api/articles` / `GET /api/articles/search`（拡張）

| クエリパラメータ | 説明 |
|------|------|
| `category` | カテゴリで絞り込み |
| `sort` | `title` / `publisher` / `category` / `published_at` |
| `order` | `asc`（デフォルト） / `desc` |
| `page` | ページ番号（デフォルト 1） |
| `per_page` | 1ページあたり件数（デフォルト 25） |

レスポンス形式を配列から以下のオブジェクトに変更する：

```json
{
  "articles": [ ... ],
  "total": 55,
  "page": 1,
  "per_page": 25
}
```

### `GET /api/categories`（新規）

- 記事に設定済みの `category` を DISTINCT で取得し、JSON 配列で返す（カテゴリドロップダウンの選択肢生成用）
- レスポンス例: `["AI", "Design", "Finance", "Science", "Tech", "Work"]`

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
