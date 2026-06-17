# フェーズ 12：OpenAPI による API 仕様管理とコード生成

## 概要

`cmd/web` が提供する JSON API（`GET /api/articles` など）の仕様を OpenAPI（`docs/openapi.yaml`）で一元管理する。
また、OpenAPI 仕様書をもとに Go（サーバー）・TypeScript（フロントエンド）双方の型をコード生成し、
手書きの DTO・型定義との不整合を防ぐ。

## 背景・目的

- フェーズ 9（Web ブラウザでの記事閲覧）で `internal/adapter/handler/web/` にハンドラと DTO（`articleDTO` 等）を
  手書きで実装したが、API 仕様を記述したドキュメントがなく、フロントエンド側の型（`web/frontend/src/types.ts`）も
  別途手書きで維持していた
- 仕様変更時に Go の DTO・TypeScript の型・実際のレスポンス内容が食い違うリスクがある
- OpenAPI を正とすることで、仕様変更時の影響範囲（型）をコード生成で機械的に追従できるようにする

## 受け入れ条件

- `docs/openapi.yaml`（OpenAPI 3.0）に既存 5 エンドポイントの仕様を記述する
  - `GET /api/articles`・`GET /api/articles/search`・`POST /api/articles/{id}/bookmark`・
    `GET /api/categories`・`POST /api/articles/fetch`
  - リクエストパラメータ（クエリ・パス）・レスポンスボディ・エラーレスポンスを実装と一致させる
- Go 側：`oapi-codegen` を `go.mod` の `tool` ディレクティブで管理し、
  `internal/adapter/handler/web/openapi/` に型（`models`）のみを生成する
  - 既存の `internal/adapter/handler/web/` のハンドラ実装・ルーティング（chi）は変更せず、
    DTO 定義（`articleDTO` 等）を生成型（`openapi.Article` 等）に置き換える
- フロントエンド側：`openapi-typescript` を `web/frontend` の devDependency に追加し、
  `web/frontend/src/api/schema.gen.ts` に型のみを生成する
  - 既存の `web/frontend/src/api.ts`（fetch ラッパー）・`web/frontend/src/types.ts`（手書き型）は変更しない
    （型の発展的な統合は本フェーズのスコープ外）
- 生成コマンドを `package.json`（`generate:api`）・`go generate` から実行できるようにし、
  `AGENTS.md` / `docs/design.md` に手順を記載する
- 生成ファイル（`*.gen.go` / `*.gen.ts`）はリポジトリにコミットし、コード生成ツールが無い環境でも
  `go build` / `npm run build` がそのまま通る
- 既存のテスト（`go test $(go list ./... | grep -v internal/driver/anthropic)`）・
  フロントエンドの型チェック・テスト・ビルド（`tsc -b`・`npm run test`・`npm run build`）が通る

## スコープ外（このフェーズ外）

- サーバー側ルーティング・ハンドラ実装自体のコード生成（`oapi-codegen` の `chi-server` 等）への置き換え
- フロントエンドの実行時クライアント（`openapi-fetch` 等）の導入、`api.ts`/`types.ts` の生成型への統合
- リクエスト・レスポンスのスキーマバリデーション（ミドルウェアでの検証）の追加
