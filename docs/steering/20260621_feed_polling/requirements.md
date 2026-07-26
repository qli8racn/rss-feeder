# フェーズ：フィード取得・要約の定期実行（イベント駆動の検討と実装）

> **【廃止】** 2026-07-26 に devcontainer cron によるポーリング実装を撤去した。
> `.devcontainer/poll-feeds.cron` / `scripts/poll-feeds.sh` を削除し、Dockerfileの`cron`パッケージ
> インストール・`devcontainer.json`の`postStartCommand`も削除済み。本ドキュメントは当時の検討経緯の記録として残す。

## 概要

「DBにフィードが追加されたことをトリガーにrss-orchestratorエージェントを実行する」というイベント駆動の仕組みが
可能か検討した。最終的に、真のDBトリガーではなく**devcontainer内でのcronによる1時間ごとのポーリング**で
`bin/rss-feeder fetch` → `bin/rss-agent enrich` を実行する方式に決定した。

## 背景・目的

- `bin/rss-feeder add-feed` はフィード新規登録時にfetch・enrichを自動実行するため、Claude Code経由でフィードを
  追加する通常フローでは追加のトリガーは不要（`.claude/agents/rss-orchestrator.md` に既述）
- 一方で「新着記事の定期取得」自体は、フィード追加とは独立した別の継続的なニーズとしてある
- `bin/rss-agent enrich` はClaude Codeのエージェントループを介さず、Goバイナリ自身がAnthropic APIを直接呼んで
  要約するため（`internal/driver/anthropic/enrich.go`）、定期実行にClaude Codeのエージェント起動は不要

## 検討した方式と採用しなかった理由

1. **クラウドのRoutine（`RemoteTrigger`/`/schedule`）** → 不採用
   クラウド側は独立したサンドボックス環境で動作するため、ローカルの`reader.db`にアクセスできない。
2. **ホストmacOSのlaunchd** → 一時的に実装したが撤回
   本プロジェクトは`.devcontainer`での開発を前提としており、コンテナ内で`go build`し直すと
   `bin/rss-feeder`/`bin/rss-agent`がLinux用バイナリに上書きされ（バインドマウントのため実体は同一ファイル）、
   ホスト側launchdジョブが`exec format error`で壊れる。VSCode/devcontainerのライフサイクルと無関係に
   動き続けてしまう点も、devcontainer中心の運用方針と合わなかった。
3. **devcontainer内のcron** → 採用
   コンテナ起動時に`postStartCommand`でcronデーモンを起動し、コンテナ停止時はプロセスごと終了するため
   ライフサイクルがdevcontainerと自然に一致する。

## 受け入れ条件

- `scripts/poll-feeds.sh` が `bin/rss-feeder fetch` → `bin/rss-agent enrich` を順に実行し、結果を
  `logs/poll-feeds.log` に記録する
- devcontainer起動時（`postStartCommand`）にcronデーモンが自動起動し、コンテナ停止時は自動的に停止する
  （cronデーモンはコンテナ内プロセスのため、特別な停止処理は不要）
- 1時間ごとに実行する（`/etc/cron.d/poll-feeds`）
- ホストmacOSのlaunchdジョブは使用しない（撤去済み）

## スコープ外

- 真のDBレベルのイベント駆動（INSERTを即時検知する仕組み）。今回はポーリング（1時間間隔）にとどめる
- フィード追加経路の統一（Web API等、`add-feed`コマンド以外からのフィード追加には未対応）
