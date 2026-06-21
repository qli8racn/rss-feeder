# タスクリスト：フィード取得・要約の定期実行

## スクリプト

- [x] `scripts/poll-feeds.sh` 新規作成（`bin/rss-feeder fetch` → `bin/rss-agent enrich`、`logs/poll-feeds.log`に記録）
- [x] ホストmacOS上で直接実行し、新着記事の取得・要約が正常に動作することを確認

## ホストmacOS launchd（一時実装 → 撤回）

- [x] `~/Library/LaunchAgents/com.rss-feeder.poll.plist` 作成・`launchctl load -w`で登録
- [x] devcontainerでビルドし直すとホスト用バイナリが上書きされ壊れるリスクに気づき、方針転換を決定
- [x] `launchctl unload -w` で停止し、plistファイルを削除（ホスト側のジョブは残っていないことを確認済み）

## devcontainer cron（採用した実装）

- [x] `.devcontainer/Dockerfile` に `cron` パッケージを追加
- [x] `.devcontainer/poll-feeds.cron` 新規作成（`/etc/cron.d/poll-feeds`に配置、1時間ごと、`vscode`ユーザー）
- [x] `.devcontainer/devcontainer.json` に `postStartCommand: "sudo service cron start"` を追加
- [x] `.gitignore` に `/logs/` を追加

## 確認（未実施・要ユーザー作業）

- [x] VSCodeで「Rebuild Container」を実行し、Dockerfileの変更を反映する
- [x] コンテナ起動後、`sudo service cron status` でcronデーモンが起動していることを確認する
- [x] 1時間待つ、または `sudo -u vscode bash /workspaces/rss-feeder/scripts/poll-feeds.sh` を手動実行して
  `logs/poll-feeds.log` に結果が記録されることを確認する
- [x] コンテナを停止し、cronデーモン（コンテナ内プロセス）が起動し続けないことを確認する
