# タスクリスト：Web ブラウザでの記事閲覧

## 実装タスク

- [x] HTTP ハンドラ実装（`internal/adapter/handler/serve.go` の `NewMux()`）
  - `GET /api/articles`（`mode` クエリ対応）
  - `GET /api/articles/search`（`q` / `bookmarked` クエリ対応）
  - `POST /api/articles/{id}/bookmark`（audit_log 記録含む）
  - 静的ファイル配信（`--static-dir`）
- [x] `web/static/index.html` プレースホルダー作成
- [x] Web API エントリーポイント `cmd/web/main.go` を新規作成（CLIとは別バイナリ）

## テスト

- [ ] serve ハンドラの統合テスト（任意・フェーズ外でも可）

## フォローアップ（このフェーズ外）

- [ ] figma-mcp でフロントエンド（HTML/CSS/JS）を生成し `web/static/` に配置
- [ ] GitHub Copilot によるローカル / PR レビューの実践
