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

- [ ] README.md に `ANTHROPIC_API_KEY` は Claude Console（console.anthropic.com）で発行する
  キーであり、claude.ai のアカウントとは別物である旨を明記
- [ ] AGENTS.md の `rss-agent` セットアップ手順に同様の注記を追加
- [ ] `internal/config/config.example.yml` のコメントに同様の注記を追加
- [ ] `docs/design.md` の `rss-agent` セクションに本フェーズの内容（モデル選定の指針・
  コスト計測の仕組み）を追記

## 費用・実行時間の可視化（未実装）

- [ ] `internal/driver/anthropic/usage.go` を新規作成し、モデル別料金テーブルと
  `estimateCostUSD` を実装
- [ ] `loop.go` の `runAgentLoop` を `Usage` を加算して返すように変更
- [ ] `summarize.go`/`preference.go`/`enrich.go` の `Run()` で、実行時間（`time.Since`）と
  概算費用を標準エラー出力に1行で表示する

## テスト

- [x] `truncateRunes` の単体テスト（`enrich_test.go`：境界値・マルチバイト文字・空文字列）
- [x] `parseLimitInput` の単体テスト（`loop_test.go`：デフォルト値・上限クランプ・不正 JSON）
- [ ] `estimateCostUSD` の単体テスト（既知の `Usage` 値に対する計算結果を検証）
- [ ] `runAgentLoop` の `Usage` 加算ロジックのテスト（複数ターンでの合算を検証）

## フォローアップ（このフェーズ外）

- [ ] モデル価格改定時に `modelPricing` を更新する運用の整理（変更検知の仕組みは設けない）
