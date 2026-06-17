# 設計：OpenAPI による API 仕様管理とコード生成

## 仕様書

`docs/openapi.yaml`（OpenAPI 3.0.3）に既存 5 エンドポイントを記述する。

```yaml
paths:
  /api/articles:        { get: listArticles }
  /api/articles/search:  { get: searchArticles }
  /api/articles/{id}/bookmark: { post: toggleBookmark }
  /api/articles/fetch:   { post: fetchLatestArticles }
  /api/categories:       { get: listCategories }
components:
  schemas: [Mode, Sort, Order, Article, PagedArticles, FetchResult, Error]
  parameters: [Mode, Category, Sort, Order, Page, PerPage]   # mode/category/sort/order/page/per_page クエリの共通定義
```

`Article` 等のフィールドは `internal/adapter/handler/web/article.go` の `articleDTO`（変更前）と
`web/frontend/src/types.ts` の `Article` 型を基に、両者が一致する形で定義する。

`Mode`/`Sort`/`Order` は enum 値を持つため `components.schemas` に定義し、
`components.parameters.Mode/Sort/Order` はその `schema` を `$ref` で参照する
（`default` 等は `$ref` と同階層に置けないため schemas 側にまとめる）。
こうすることで `GET /api/articles` と `GET /api/articles/search` の双方から同じ型を参照させ、
`oapi-codegen` が操作ごとに別名の enum 型（`ListArticlesParamsMode` 等）を重複生成しないようにする
（最初の実装では `parameters` にのみ inline enum を書いていたため、この重複が発生していた）。

## Go 側：oapi-codegen

### ツールの追加

`go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest` で
`go.mod` に `tool` ディレクティブとして追加する（ランタイム依存にはしない。Go 1.24 の機能）。

```go
// go.mod
tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
```

### 生成設定

`internal/adapter/handler/web/openapi/` に生成専用パッケージを置く。

```yaml
# internal/adapter/handler/web/openapi/config.yaml
package: openapi
generate:
  models: true
output-options:
  name-normalizer: ToCamelCaseWithInitialisms   # id→ID・url→URL 等の頭字語を Go 命名規則に正規化
compatibility:
  old-enum-conflicts: true   # Mode/Sort/Order の定数名に ModeAll 等のプレフィックスを残す（無効だと All/Asc 等の汎用名になり衝突しやすい）
output: types.gen.go
```

```go
// internal/adapter/handler/web/openapi/generate.go
package openapi

//go:generate go tool oapi-codegen -config config.yaml ../../../../../docs/openapi.yaml
```

`generate: { models: true }` のみを指定し、`chi-server`（ルーティング生成）は使わない。
既存の `cmd/web/main.go`（chi ルーティング）・`internal/adapter/handler/web/` のハンドラ実装は変更しない。

### ハンドラ側の変更

`internal/adapter/handler/web/article.go` の `articleDTO`・`pagedArticlesDTO`、
`fetch.go` の `fetchResultDTO`、`response.go` の `writeJSONError` が使うエラー表現を、
生成された `openapi.Article` / `openapi.PagedArticles` / `openapi.FetchResult` / `openapi.Error` に置き換える。
`toArticleDTO` / `toArticleDTOs`（`domain.Article` → DTO 変換）はそのまま残し、戻り値の型のみ変更する。

```go
// internal/adapter/handler/web/article.go（変更後）
func toArticleDTO(a domain.Article) openapi.Article {
    return openapi.Article{
        ID: a.ID, FeedID: a.FeedID, FeedURL: a.FeedURL, /* ... */
    }
}
```

`mode`/`order` クエリの検証も、ハンドラ内の生文字列の switch 文ではなく生成された
`openapi.Mode` / `openapi.Order` の `Valid()` メソッドを使う（`ListArticlesHandler`・`parseListQuery`）。
`sort` のみ `articlerepo.ValidSortFields`（SQL カラム解決と共有する既存の正定義）を引き続き使う。

## フロントエンド側：openapi-typescript

`web/frontend` に `openapi-typescript` を devDependency として追加し、型のみを生成する。

```json
// web/frontend/package.json
"scripts": {
  "generate:api": "openapi-typescript ../../docs/openapi.yaml -o src/api/schema.gen.ts"
}
```

`src/api/schema.gen.ts` は `paths` / `components["schemas"]` 型を提供するのみで、
既存の `web/frontend/src/api.ts`（fetch ラッパー）・`web/frontend/src/types.ts`（手書き型）は
本フェーズでは変更しない。生成型と手書き型の統合（`api.ts` を `openapi-fetch` ベースに置き換える等）は
フォローアップとする。

## 生成ファイルの扱い

`internal/adapter/handler/web/openapi/types.gen.go`・`web/frontend/src/api/schema.gen.ts` は
リポジトリにコミットする。`oapi-codegen`／`openapi-typescript` が無い環境でも
`go build`・`npm run build` がそのまま通る状態を維持するため。
ファイル先頭に `DO NOT EDIT` コメントが自動付与される（生成ツール側の規約）。

## ドキュメント更新

- `AGENTS.md`：`cmd/web` セクションに再生成コマンドを追記
- `docs/design.md`：「API 仕様（OpenAPI）とコード生成」節を追加、依存ライブラリ表に `oapi-codegen` を追記、
  ディレクトリ構成に `internal/adapter/handler/web/openapi/` を追記
- `docs/product-requirements.md`：機能要求表にフェーズ 12 を追記
