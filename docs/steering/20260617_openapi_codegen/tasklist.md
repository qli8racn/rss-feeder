# タスクリスト：OpenAPI による API 仕様管理とコード生成

## 実装タスク

- [x] `docs/openapi.yaml` を作成（既存 5 エンドポイントの仕様を記述）
- [x] `go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest` で Go tool として追加
- [x] `internal/adapter/handler/web/openapi/config.yaml`・`generate.go` を作成し `go generate` で型生成
- [x] `internal/adapter/handler/web/article.go`・`fetch.go`・`response.go` を生成型（`openapi.Article` 等）に置き換え
- [x] `web/frontend` に `openapi-typescript` を devDependency として追加
- [x] `web/frontend/package.json` に `generate:api` スクリプトを追加し `src/api/schema.gen.ts` を生成
- [x] `go mod tidy`

## テスト・検証

- [x] `go build ./internal/adapter/... ./cmd/web/... ./internal/usecase/...`
- [x] `go test $(go list ./... | grep -v internal/driver/anthropic)`
- [x] `cd web/frontend && npx tsc -b --noEmit`
- [x] `cd web/frontend && npm run build`
- [x] `cd web/frontend && npm run test`

## ドキュメント

- [x] `AGENTS.md` に再生成コマンドを追記
- [x] `docs/design.md` に OpenAPI・コード生成の節、依存ライブラリ、ディレクトリ構成を追記
- [x] `docs/product-requirements.md` の機能要求表にフェーズ 12 を追記
- [x] `docs/steering/20260617_openapi_codegen/`（本ディレクトリ）を作成

## フォローアップ（このフェーズ外）

- [x] フロントエンドの `api.ts`／`types.ts` を生成型（`schema.gen.ts`）に統合する
  - `types.ts` を撤廃し、`domain/article.ts`（`Article`/`ArticlesResponse`/`FetchLatestResult`/`Mode`/
    `SortField`/`SortOrder`とその定数）・`domain/feed.ts`（`Feed`）に分割。Go側の
    `internal/domain/article.go`・`feed.go` と同じエンティティ単位の構成に合わせた
  - 各型は `schema.gen.ts` の `components['schemas']` 型のエイリアスとし、手書き定義を撤廃
  - `api.ts` の手書き `FetchLatestResult` interface・`mode` のリテラル型を `domain/article.ts` 経由の生成型に置き換え
  - `domain/filter.ts`・`ArticleTable.tsx`・`ArticleListPage.tsx`・`ArticleListTemplate.tsx`・
    `SearchFilterBar.tsx`・`FeedManagementModal.tsx` のimport元を `domain/article`・`domain/feed` に変更
  - `npx tsc -b --noEmit` / `npm run test` / `npm run build` で確認
- [x] サーバー側のリクエスト・レスポンスのスキーマバリデーション（ミドルウェア等）の追加
  - リクエスト検証のみ実装（レスポンスは生成型 `openapi.Article` 等の使用で形状を保証する前提のため対象外、ユーザー判断）
  - `github.com/oapi-codegen/nethttp-middleware`（`kin-openapi` ベース）を追加し、
    `internal/adapter/handler/web/validator.go` に `NewRequestValidatorMiddleware(specPath string)` を実装
  - `cmd/web/main.go` に `--openapi-spec`（デフォルト `docs/openapi.yaml`）フラグを追加し、
    既存の `/api/...` ルートを `r.Route("/api", ...)` のサブルーターにまとめてミドルウェアを適用
    （仕様外パスは404を返すため、`/*` の静的ファイル配信を巻き込まないよう `/api` 配下に限定）
  - `go build ./...` / `go vet ./...` / `go test $(go list ./... | grep -v internal/driver/anthropic)` で確認
  - `curl` で実機確認：不正なクエリ（`sort=invalid_field`・`per_page=abc`）・ボディ不足（`POST /api/feeds`）が
    `400` で弾かれること、仕様外パスが `404`、正常リクエストと `/`（SPA）が影響を受けないことを確認
- [x] `oapi-codegen` の `chi-server` 生成によるルーティング自動生成への移行検討
  - 2026-06-20: 検討の結果、見送り（ユーザー判断）。理由は以下の通り
    - 現在は usecase ごとにクロージャを返す関数（`ListArticlesHandler(uc *usecase.ListUsecase) http.HandlerFunc` 等）
      というシンプルなDIパターンだが、`chi-server` は単一の `ServerInterface` を1つの構造体に実装させる方式のため、
      全8エンドポイント分のハンドラを1つの `Server` 構造体にまとめる大規模な書き換えが必要になる
    - `chi-server` 移行で主に削減できるのは `article.go` の `parseListQuery` 等によるクエリパラメータの手動パースだが、
      直前に追加した [[20260617_openapi_codegen/tasklist|リクエスト検証ミドルウェア]]（`internal/adapter/handler/web/validator.go`）で
      クエリ・ボディの妥当性チェックは既にカバーされており、追加で得られる削減効果は小さい
    - 現状のエンドポイント数（8件）では移行コストが効果を上回ると判断した
    - エンドポイント数が大きく増える、またはハンドラの手動パース・バリデーションが複雑化した場合は再検討する
