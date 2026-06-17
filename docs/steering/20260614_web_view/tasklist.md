# タスクリスト：Web ブラウザでの記事閲覧

## 実装タスク

- [x] HTTP ハンドラ実装（`internal/adapter/handler/serve.go` の `NewMux()`）
  - `GET /api/articles`（`mode` クエリ対応）
  - `GET /api/articles/search`（`q` / `bookmarked` クエリ対応）
  - `POST /api/articles/{id}/bookmark`（audit_log 記録含む）
  - 静的ファイル配信（`--static-dir`）
- [x] `web/static/index.html` プレースホルダー作成
- [x] Web API エントリーポイント `cmd/web/main.go` を新規作成（CLIとは別バイナリ）

## 実装タスク（追加・Figma デザイン反映、2026-06-17）

- [x] `GET /api/articles` / `GET /api/articles/search` に `category` / `sort` / `order` / `page` / `per_page` 対応を追加し、レスポンスを `{articles, total, page, per_page}` 形式に変更する
- [x] `GET /api/categories`（DISTINCT カテゴリ一覧）を新規実装する

## テスト

- [ ] serve ハンドラの統合テスト（任意・フェーズ外でも可）

## フォローアップ（このフェーズ外）

- [ ] `web/frontend/`（Vite + React + TypeScript + Tailwind CSS）の雛形を作成する
- [ ] figma-mcp で Figma デザインから React コンポーネントの初期コードを生成し、API 連携・状態管理を実装する（詳細は `docs/web-ui-spec.md` の「実装の進め方」参照）
- [ ] `npm run build` の出力を `web/static/` に配置する
- [ ] GitHub Copilot によるローカル / PR レビューの実践
