# フェーズ 13：エージェント系コマンド（rss-agent）の再確認とリファクタリング

## 概要

`rss-agent`（`summarize`/`preference`/`enrich`）の実装を、コストとパフォーマンス（実行時間）の
観点で再確認し、必要なリファクタリングを行う。

## 背景・目的

- `internal/driver/anthropic` 配下の各エージェントは導入時の実装のままモデル選定や
  `Thinking`/`effort` 設定を見直していなかった
- `ANTHROPIC_API_KEY` は `claude.ai`（コンシューマー向けチャットサービス）のアカウントとは別物で、
  Claude Console（`console.anthropic.com`、開発者向け API プラットフォーム）で発行する API キーを
  使う必要がある。現状ドキュメント上この区別が明記されておらず、`claude.ai` の認証情報を
  探してしまう恐れがある
- 1 回の実行あたりにかかる費用（トークン使用量 × モデル単価）が見えず、コスト感を把握しづらい
- 1 回の実行あたりの実行時間が見えず、パフォーマンスチューニングの基準がない

## 受け入れ条件

- `summarize`/`preference`/`enrich` の各エージェントについて、モデル選択
  （Opus/Sonnet/Haiku）・`Thinking`（adaptive 有無）・件数上限が、タスクの難易度に対して
  過剰でないか再確認し、必要に応じて変更する
- README.md / AGENTS.md に、`ANTHROPIC_API_KEY` は `claude.ai` のアカウントではなく
  Claude Console（`https://console.anthropic.com`）で発行する API キーであることを明記する
- 各エージェント実行時に、レスポンスの `Usage`（input/output/cache トークン数）から
  概算費用（USD）を算出し、標準出力に表示する
- 各エージェント実行時に、実行時間（開始〜終了の経過時間）を計測し、標準出力に表示する
- 上記の費用・実行時間の表示方法（常時表示か、フラグで切り替えるか）は設計時に決定し
  `design.md` に記載する

## スコープ外（このフェーズでは扱わない）

- レート制限やリトライ処理の実装
- プロンプトキャッシュ（`cache_control`）の導入（現状の各システムプロンプトはモデルごとの
  キャッシュ最小トークン数に届かないため、導入しても効果が薄いと判断）
