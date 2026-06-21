# 設計：フィード取得・要約の定期実行

## 全体構成

```
devcontainer起動
  └─ postStartCommand: "sudo service cron start"
       └─ /etc/cron.d/poll-feeds（1時間ごと、vscodeユーザー）
            └─ bash /workspaces/rss-feeder/scripts/poll-feeds.sh
                 ├─ bin/rss-feeder fetch   （新着記事をDBに保存）
                 └─ bin/rss-agent enrich   （未要約記事を要約・カテゴライズしてDBに保存）
                      └─ logs/poll-feeds.log に結果を記録
```

devcontainerを停止すればcronデーモンもコンテナ内プロセスとして終了するため、起動・停止の制御を
devcontainer側に追加する必要はない。

## `scripts/poll-feeds.sh`

- `reader.db`（`internal/driver/readerdb/client.go`のDSNが相対パス`reader.db`）と`internal/config/config.yml`
  （`viper`の`AddConfigPath("internal/config")`も相対パス）への参照が相対パスのため、
  スクリプト自身の場所からリポジトリルートを解決して`cd`する（`BASH_SOURCE`基準）
- `bin/rss-feeder` / `bin/rss-agent` が存在しない場合はエラーを記録して終了する（cron実行時に
  `go build`を自動実行するとビルド失敗時の挙動が読みづらくなるため、ビルド済みバイナリを前提とする）
- 実行結果（fetch/enrichの標準出力）はタイムスタンプ付きで`logs/poll-feeds.log`に追記する

## `.devcontainer/poll-feeds.cron`

```
0 * * * * vscode /bin/bash /workspaces/rss-feeder/scripts/poll-feeds.sh >> /workspaces/rss-feeder/logs/cron.log 2>&1
```

- `/etc/cron.d/`形式のため、通常のユーザーcrontabと異なりコマンドの前にユーザー名（`vscode`）を指定する
- `vscode`は`mcr.microsoft.com/devcontainers/go`イメージの既定リモートユーザー
  （`devcontainer.json`に`remoteUser`の指定が無いため既定値に従う）
- パスは`workspaceFolder`の既定値（`/workspaces/${localWorkspaceFolderBasename}`）がリポジトリ名
  `rss-feeder`から`/workspaces/rss-feeder`になることを前提に固定で書いている
- `>> .../logs/cron.log 2>&1`はcron自体の起動失敗（パーミッション等）を捕捉するためのもので、
  fetch/enrichの実行結果自体は`poll-feeds.sh`が`logs/poll-feeds.log`に別途記録する

## Dockerfile / devcontainer.json

- `Dockerfile`: `cron`パッケージを追加インストールし、`COPY poll-feeds.cron /etc/cron.d/poll-feeds`で
  ビルド時に配置（`/etc/cron.d/`配下のファイルは`0644`で十分・実行権限は不要）
- `devcontainer.json`: `postCreateCommand`（コンテナ作成時に1回のみ）とは別に`postStartCommand`
  （コンテナ起動ごとに実行）で`sudo service cron start`を実行する。`postCreateCommand`に置くと
  コンテナを再起動した際にcronデーモンが起動しない

## ホストmacOSのlaunchd（撤回した実装）

検討の過程で一時的に`~/Library/LaunchAgents/com.rss-feeder.poll.plist`（`StartInterval=3600`）と
`launchctl load -w`での登録まで行ったが、devcontainer中心の運用方針に合わないため`launchctl unload -w`で
停止し、plistファイルも削除した。ホストとdevcontainerが同じファイルをバインドマウントで共有しているため、
devcontainer内で`go build`し直すとホスト用バイナリ（Mach-O）がLinux用（ELF）に上書きされ、
ホスト側のジョブが`exec format error`で壊れる点が決定的な理由。
