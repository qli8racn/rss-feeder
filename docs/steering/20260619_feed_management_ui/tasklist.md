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

## Figmaデザイン

- [x] 既存ヘッダー（node `6:2` Header Actions）に「フィード管理」ボタンを追加（`6:3` Fetch Button と同じスタイル：ボーダーのみ・JetBrains Mono Medium・アイコン+ラベル）
- [x] 新規フレーム「Feed Management Modal」（node `15:2`、fileKey `I0n9kvKiHCnN8vlCalkq6H`）を作成
  - ヘッダー行（タイトル「フィード管理」+ 閉じる×ボタン）
  - フィード追加行（URL入力欄 + 「追加」ボタン）
  - 一覧ヘッダー行（URL / タイトル / 最終取得 の列ラベル）
  - フィード一覧3行（Qiita・Zenn・未取得サンプル）+ 各行の削除（ゴミ箱）ボタン
- [x] 色・フォント・ボーダー・角丸は既存画面から実測した値を再利用（背景 `#0d1117`、コントロール背景 `#1e2733`、ボーダー `rgba(148,163,184,0.12)`、角丸 `3.75px`、JetBrains Mono）。新規デザイントークンは追加せず「既存画面に目視で合わせる」方針を維持
- [x] `get_screenshot` で各セクション・ヘッダーボタン・モーダル全体を視覚確認

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
