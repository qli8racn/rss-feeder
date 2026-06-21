# タスクリスト：エージェント系コマンドの再確認とリファクタリング

## 再確認・リファクタリング（実装済み）

- [x] `enrich.go`：モデルを `claude-opus-4-8` → `claude-haiku-4-5` に変更
- [x] `enrich.go`：記事本文を 2000 文字に切り詰める `truncateRunes` を追加
- [x] `summarize.go`：モデルを `claude-opus-4-8` → `claude-haiku-4-5` に変更
- [x] `summarize.go`：`Thinking: adaptive` を削除（単純な要約タスクには不要と判断）
- [x] `preference.go`：`fetch_bookmarked` に `limit` 引数を追加し、上限 50 件にキャップ
- [x] `preference.go`：モデル（`claude-opus-4-8`）・`Thinking: adaptive` は分析タスクの
  特性上維持（変更なしと判断した記録）
- [x] `loop.go`：`preference.go`/`summarize.go` で重複していた `limit` 入力のパース・
  デフォルト値・上限クランプ処理を `parseLimitInput` に共通化（コードレビューでの DRY 指摘対応）

## ドキュメント

- [x] README.md に `ANTHROPIC_API_KEY` は Claude Console（console.anthropic.com）で発行する
  キーであり、claude.ai のアカウントとは別物である旨を明記
- [x] AGENTS.md の `rss-agent` セットアップ手順に同様の注記を追加
- [x] `internal/config/config.example.yml` のコメントに同様の注記を追加
- [x] `docs/design.md` の `rss-agent` セクションに本フェーズの内容（モデル選定の指針・
  コスト計測の仕組み）を追記

## 費用・実行時間の可視化（実装済み）

- [x] `internal/driver/anthropic/usage.go` を新規作成し、モデル別料金テーブルと
  `estimateCostUSD`・`addUsage`・`logUsage` を実装
- [x] `loop.go` の `runAgentLoop` を `Usage` を加算して返すように変更（戻り値を
  `(string, anthropic.Usage, error)` に拡張）
- [x] `summarize.go`/`preference.go`/`enrich.go` の `Run()`（`enrich.go` は
  `summarizeAndCategorize`）で、実行時間（`time.Since`）と概算費用を `defer` で
  標準エラー出力に1行表示する
- [x] `usage.go`：`preference.go`/`summarize.go` で重複していた計測用ボイラープレート
  （計測開始・Usage集計・defer ログ出力）を `runAgentLoopWithUsageLog` に共通化
  （コードレビューでの DRY 指摘対応）

## テスト

- [x] `truncateRunes` の単体テスト（`enrich_test.go`：境界値・マルチバイト文字・空文字列）
- [x] `parseLimitInput` の単体テスト（`loop_test.go`：デフォルト値・上限クランプ・不正 JSON）
- [x] `estimateCostUSD` の単体テスト（`usage_test.go`：既知モデル・未知モデル・
  cache トークン込みの計算結果を検証）
- [x] `runAgentLoop` の合算ロジックのテスト（`usage_test.go`：合算処理を `addUsage` として
  切り出し、複数ターン分の加算をユニットテストで検証。`runAgentLoop` 自体は実際の HTTP
  呼び出しを伴うため、ここでは加算ロジック単体をテスト対象とした）

## enrich の並列バッチ処理（追加スコープ）

- [x] `enrich.go` に `enrichBatchSize`（40）・`enrichConcurrency`（4）の定数を追加
- [x] 記事一覧をバッチサイズ単位に分割する `chunkArticles` を実装
- [x] `summarizeAndCategorize` の戻り値を `(results, usage, error)` に変更し、内部の
  `defer logUsage` を削除
- [x] `sync.WaitGroup`+セマフォ（バッファ付きチャネル）で同時実行数を制限し、チャンクごとに
  ゴルーチンで `summarizeAndCategorize` を呼ぶ
- [x] 全チャンク完了後、Usage を合算し1回だけ `logUsage` を呼ぶ
- [x] 一部チャンクが失敗しても他チャンクの結果は保存し、エラーは `errors.Join` でまとめて返す
- [x] 実機確認：90件（3チャンク）・171件（5チャンク、並列度4を超える）で `--force` 実行し、
  `database is locked` 等のエラーなく完了することを確認（90件は約31秒、171件は約37秒）

### 1回目コードレビュー対応

- [x] 既にキャンセル済みの `ctx` で `Run()` が呼ばれた場合、APIを1件も呼ばずに即座に
  エラーを返す（`ctx.Err()` を冒頭でチェック）
- [x] CLIハンドラ：`n==0`かつ`err==nil`（処理対象なし）でも「0 件の記事を要約・分類しました」
  を表示する
- [x] `summarizeAndCategorize`：JSON解析失敗時に `resp.StopReason == anthropic.StopReasonMaxTokens`
  を確認し、MaxTokens切り詰めが原因と分かる専用エラーメッセージを返す
- [x] `enrichAgent.client` を具象の `anthropic.Client` から `messageCreator` インターフェース
  （`Messages.New`のみを抽象化）に変更し、実APIを呼ばずに`Run()`をテスト可能にした
- [x] DB への書き込みを `Repository.UpdateEnrichmentBatch`（1トランザクション）に変更
  （後に2回目レビューでチャンク単位に分割し直した。下記参照）

### 2回目コードレビュー対応

- [x] **DB書き込みをチャンク単位のトランザクションに分割**：1回目対応で全結果を1トランザクション
  にまとめたところ、1チャンクのDB書き込み失敗が他チャンクの保存済み結果まで巻き込んで
  ロールバックしてしまい、「1バッチが失敗しても他バッチの処理・DB保存は継続する」という
  当初方針に反することが判明したため、`UpdateEnrichmentBatch` をチャンクごとに呼び出す
  ように変更（DB層でもチャンク単位の部分成功を維持する）
- [x] **実行中キャンセルへの対応**：ディスパッチループで同時実行数の上限に達してブロックする
  「後」に `ctx.Err()` を確認するように変更（ブロックする「前」に確認しても、ブロック中に
  他チャンクがキャンセルする可能性があるため意味がなかった）。これにより、実行中に
  キャンセルされた場合、まだディスパッチしていないチャンクの分のAPIコストを節約できる
- [x] **DB更新失敗時に具体的な記事IDを含める**：`UpdateEnrichmentBatch` の実装側
  （`internal/driver/readerdb/article/article.go`）でエラーに `記事 %d の更新に失敗しました`
  と対象IDを含めるように変更し、呼び出し元（`enrich.go`）はそのまま伝播するだけにした
- [x] **`cmd.SilenceUsage` をルートコマンドに移動**：`enrich` コマンドだけに設定していたのを
  `cmd/agent/main.go` のルートコマンドに移し、`summarize`/`preference` にも同じ挙動が
  適用されるようにした
- [x] **`messageCreator` を共有化**：`enrich.go` 専用だったインターフェースを `usage.go` に
  移動し、`loop.go`（`runAgentLoop`/`runAgentLoopWithUsageLog`）・`summarize.go`・
  `preference.go` も同じインターフェースに依存するように統一（テスト容易性の改善を
  3エージェントに揃えた）
- [x] **`errgroup` を手動セマフォに戻した**：1回目対応で `errgroup.Group.SetLimit` を採用したが、
  各 `g.Go` が常に `nil` を返してエラー伝播機能を使っていなかったため、`errgroup`
  （`golang.org/x/sync` の直接依存化）を導入する理由がなかった。元の `sync.WaitGroup`+
  セマフォに戻し、依存を追加せずに済むようにした
- [x] **使われていなかった単発`UpdateEnrichment`を削除**：チャンク単位の `UpdateEnrichmentBatch`
  に置き換えたことで本番コードからの呼び出しがなくなっていたため、`Repository` インターフェース・
  実装・専用テスト・5つのusecaseモックから完全に削除した
- [x] **同一IDの重複結果を排除**：モデルが同一記事IDを複数回返した場合に重複した
  `UpdateEnrichmentBatch` 呼び出し・処理件数の過大表示が起きないよう、チャンクごとに
  `buildEnrichmentUpdates` でID重複を除去するようにした

### テスト（追加スコープ）

- [x] `chunkArticles` の単体テスト（境界値：ちょうど割り切れる件数・余りが出る件数・
  空リスト）
- [x] 複数チャンクの Usage 合算結果のテスト（既存の `addUsage` テストと同様の手法）
- [x] 一部チャンクが失敗した場合に、成功分のみ結果が返り、エラーも返ることのテスト
- [x] `Repository.UpdateEnrichmentBatch` の単体テスト（複数件更新・空リスト・
  ctxキャンセルによる2件目の失敗で1件目もロールバックされることの確認）
- [x] `enrichAgent.Run()` のfakeクライアント・fakeリポジトリを使ったテスト（複数チャンク成功・
  部分失敗時の保存範囲・DB書き込み失敗時のエラー合成・ctx既キャンセル時にAPI未呼び出し・
  1チャンクのDB書き込み失敗が他チャンクの保存済み結果を巻き込まないこと・実行中キャンセルで
  未ディスパッチのチャンクがAPIを呼ばないこと、の7パターン）
- [x] `buildEnrichmentUpdates` の単体テスト（未要求ID除外・重複ID排除）

## フォローアップ（このフェーズ外）

- [x] モデル価格改定時に `modelPricing` を更新する運用の整理（変更検知の仕組みは設けない）
  - `docs/design.md` に運用手順（Anthropicの価格ページを確認し `usage.go` の `modelPricing` を
    手動更新するだけでよいこと）を追記。コード変更は無し
- [ ] バッチサイズ・並列度の CLI フラグ化
- [ ] レート制限（429）検知時のバックオフ・リトライ
- [x] SQLite の WAL モード化・`busy_timeout` 設定（今回はDB書き込みのチャンク単位トランザクション
  化で軽減。チャンクサイズが大きい場合のロック保持時間は依然残る）
  - `internal/driver/readerdb/client.go` の DSN に `_journal_mode=WAL&_busy_timeout=5000` を追加
    （`cmd/web`・`cmd/rss-feeder`・`cmd/agent`が共通で使う`NewClient`のみ。テスト用の
    `NewInMemoryDB`・`article_test.go`等のファイルベース一時DBは対象外）
  - 実機確認：`PRAGMA journal_mode`/`PRAGMA busy_timeout` で `wal`/`5000` が反映されることを確認
- [ ] `summarize`/`preference` への並列処理の適用（`messageCreator` の共有化で土台はできたが、
  バッチ分割自体は未実装）
- [ ] MaxTokens切り詰め検知後の自動分割リトライ（現状は専用エラーメッセージのみ）
