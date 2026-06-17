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

- [ ] フロントエンドの `api.ts`／`types.ts` を生成型（`schema.gen.ts`）に統合する
- [ ] サーバー側のリクエスト・レスポンスのスキーマバリデーション（ミドルウェア等）の追加
- [ ] `oapi-codegen` の `chi-server` 生成によるルーティング自動生成への移行検討
