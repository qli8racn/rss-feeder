# フェーズ 5：お気に入り登録

## 概要

記事 ID を指定してお気に入り（ブックマーク）に登録・解除する

## 受け入れ条件

- 記事 ID を引数に取り、お気に入りの登録・解除をトグルできる（`bookmarked` を 0/1 で切り替え）
- 存在しない ID を指定した場合はエラーを返す
- Claude Code が `rss-feeder bookmark <id>` を Bash ツールで実行する前、**PreToolUse Hook** が Go バイナリ経由で ID の存在確認を実施する
- 実行後、**PostToolUse Hook** が Go バイナリ経由で操作を audit_log に記録する
