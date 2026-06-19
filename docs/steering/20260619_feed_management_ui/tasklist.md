# タスクリスト：フィード管理 Web UI

## バックエンド

- [x] `AddFeedUsecase.Execute` の戻り値を `(*domain.Feed, error)` に変更（`feedRepo.FindByURL` で登録結果を取得）
  - `internal/usecase/add_feed.go`
  - `internal/usecase/add_feed_test.go` を戻り値変更に追随
  - `internal/adapter/handler/cli/add_feed.go` の呼び出しを修正（戻り値の feed は無視）
- [x] `docs/openapi.yaml` に `Feed` / `AddFeedRequest` スキーマ、`Conflict` レスポンス、3 パス（`GET /api/feeds`, `POST /api/feeds`, `DELETE /api/feeds/{id}`）を追加
- [x] `go generate ./internal/adapter/handler/web/openapi/...` で Go 型を再生成
- [x] `internal/adapter/handler/web/feed.go` 新規作成（`ListFeedsHandler` / `AddFeedHandler` / `RemoveFeedHandler`）
- [x] `cmd/web/main.go` にルート登録・DI 追加、`cors.Options.AllowedMethods` に `DELETE` を追加

## フロントエンド

- [ ] `cd web/frontend && npm run generate:api` で型を再生成
- [ ] `web/frontend/src/types.ts` に `Feed` 型を追加
- [ ] `web/frontend/src/api.ts` に `fetchFeeds` / `addFeed` / `removeFeed` を追加
- [ ] `web/frontend/src/components/icons.tsx` に `SettingsIcon` / `TrashIcon` を追加
- [ ] `web/frontend/src/components/FeedManagementModal.tsx` 新規作成
- [ ] `web/frontend/src/components/Header.tsx` に「フィード管理」ボタンを追加
- [ ] `web/frontend/src/App.tsx` にモーダル開閉状態を追加

## 確認

- [x] `go test ./...`
- [ ] `cd web/frontend && npx tsc --noEmit && npm run test && npm run build`
- [x] `curl` で `GET /api/feeds` / `POST /api/feeds` / `DELETE /api/feeds/{id}` の実エンドポイント確認（`CLAUDE.md` のフロントエンド確認方針により、ブラウザでの目視確認はユーザー自身が行う）
