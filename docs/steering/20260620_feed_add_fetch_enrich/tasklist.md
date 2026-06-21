# タスクリスト：フィード追加時の自動取得・要約

## enrich のフィード絞り込み対応

- [x] `internal/adapter/driver/anthropic/enrich.go` の `EnrichOptions` に `FeedURL string` フィールドを追加
- [x] `internal/driver/anthropic/enrich.go` の `Run` 内 `Force` 分岐を `a.repo.FetchLatest(ctx, opts.Limit, opts.FeedURL)` に変更
- [x] `internal/driver/anthropic/enrich_test.go` に `FeedURL` 指定時のテストケースを追加
- [x] `internal/adapter/handler/agent/enrich.go` の `NewEnrichCommand` に `--feed` フラグを追加し `EnrichOptions.FeedURL` に渡す

## enrich 用サブプロセス機構（新設）

- [x] `internal/adapter/driver/feedenrich/feedenrich.go` 新規作成（`Agent` interface、`ErrAgentUnavailable`）
- [x] `internal/driver/feedenrich/subprocess.go` 新規作成（`exec.CommandContext` で `bin/rss-agent enrich --feed <url> --force --limit <n>` 実行。同時実行数制限は無し）
- [x] `internal/driver/feedenrich/subprocess_test.go` 新規作成
- [x] `internal/usecase/trigger_enrich.go` 新規作成（`TriggerEnrichUsecase`。`feedenrich.Agent` への薄いラッパー。
      `enrichTimeout`=30秒（`resolve_feed_url.go` の `stepThreeTimeout` と同じ考え方）を内部で `context.WithTimeout` に設定）
- [x] `internal/usecase/trigger_enrich_test.go` 新規作成（ctxキャンセルが `agent.Enrich` に伝播することを含めてテスト）

## cmd/rss-feeder（CLI）: 自動 fetch + 自動 enrich（サブプロセス経由）

- [x] `internal/adapter/handler/cli/add_feed.go` の `NewAddFeedCommand` に `fetchUC`・`triggerEnrichUC` を追加し、
      登録成功後に fetch→（保存件数>0なら）enrichトリガーを呼ぶ（失敗は標準エラー出力に警告、処理は続行）
- [x] `cmd/rss-feeder/main.go` に `feedenrich.Agent` をサブプロセス実装で構築し `triggerEnrichUC` を `NewAddFeedCommand` に渡す
      （`--rss-agent-path` フラグ自体は `docs/steering/20260619_feed_url_discovery/` のアーキテクチャ変更で追加済みのため、
      ここでは既存フラグを再利用するのみ）

## cmd/web: 自動 fetch のみ

- [x] `internal/adapter/handler/web/feed.go` の `AddFeedHandler` に `fetchUC` を追加し、
      登録成功後に fetch を呼ぶ（失敗は `log.Printf` で記録、処理は続行。enrich は呼ばない）
- [x] `cmd/web/main.go` の `web.AddFeedHandler(...)` 呼び出しに既存の `fetchUC` を追加で渡す

## 確認

- [x] `go build ./...` / `go vet ./...` / `go test ./...`
- [x] `bin/rss-agent enrich --force --limit 5`（`--feed` 未指定）が従来通り全フィード対象で動作することを確認（後方互換）
  - 実機確認：全フィード横断で最新5件（結果的に1フィード由来だったが、`--feed`による絞り込みは無し）を
    要約・分類できることを確認
- [x] `bin/rss-feeder add-feed <url>` で、フィード登録→記事取得→要約・カテゴライズ（サブプロセス経由）が連続して実行され、
      `bin/rss-feeder list` で要約・カテゴリ付きの記事が確認できることを確認
  - 実機確認：`https://b.hatena.ne.jp/hotentry/it.rss` で登録→26件取得→要約・カテゴライズが連続実行され、
    `list --category Tech` で要約・カテゴリ付き記事が確認できた
- [x] `curl` で `POST /api/feeds` を実行し、レスポンスが既存と同じ（`Feed` DTO・`201`）であること、
      かつ記事が自動取得されること（要約・カテゴリは付与されないこと）を確認
  - 実機確認：`201`・`Feed` DTO（`id`/`feed_url`/`title`/`last_fetched`/`created_at`）で応答し、
    26件の記事が自動取得され、要約は0件（未付与）であることを確認
- [x] `bin/rss-agent` を一時的にリネームした状態で `add-feed` を実行し、フィード登録・記事取得は成功し、
      enrich失敗の警告のみ標準エラー出力に出ることを確認
  - 実機確認：`登録しました`→`警告: 要約・カテゴライズに失敗しました: rss-agent is not available`が
    標準エラー出力に出力され、記事26件は正常に取得され、コマンドの終了コードは0のまま処理が継続した

### 確認作業中に発見した不具合の修正

- [x] **`remove-feed`のカスケード削除が機能していなかった**：`articles.feed_id`に
  `FOREIGN KEY ... ON DELETE CASCADE`を定義していたが、DSNで`foreign_keys`プラグマを
  有効化していなかったため、SQLiteのデフォルト動作（制約OFF）によりカスケード削除が
  機能せず、フィード削除後も記事が孤立して残っていた
  - 修正方針はユーザーと相談し、`_foreign_keys=on`の有効化ではなく「明示的削除」を選択
    （理由：①このプロジェクトの追加方式マイグレーション（`CREATE TABLE IF NOT EXISTS`）では
    既存DBにスキーマ変更が反映されないため宣言的なFK制約に頼るのは脆い、②`foreign_keys`
    プラグマは`audit_log.article_id`等の他のFKにも影響し意図しない副作用のリスクがある）
  - `internal/migration/migration.go`の`articles.feed_id`FK定義から`ON DELETE CASCADE`を削除
  - `internal/driver/readerdb/feed/feed.go`の`Remove`をトランザクション化し、
    `DELETE FROM articles WHERE feed_id = ?`→`DELETE FROM feeds WHERE id = ?`の順で明示的に削除
  - `audit_log`は削除対象に含めない（履歴として削除後も記事への参照を残す設計判断）
  - テスト追加：`TestFeedRepository_Remove_DeletesAssociatedArticles`
  - 実機の`reader.db`で発覚した孤立記事182件（過去のフィード削除由来79件＋原因未特定の
    `feed_id=0`が103件）はユーザー判断で全件削除して整理した

> 上記4項目は外部アクセス・APIコストを伴う手動確認のため保留した（ユーザー判断、2026-06-20）。
> 本番運用前に必要であれば手動で確認すること。
