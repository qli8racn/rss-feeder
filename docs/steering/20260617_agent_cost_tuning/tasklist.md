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

## フォローアップ（このフェーズ外）

- [ ] モデル価格改定時に `modelPricing` を更新する運用の整理（変更検知の仕組みは設けない）
