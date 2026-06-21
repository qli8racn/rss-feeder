# タスクリスト：GitHub Actions 上での Claude Code 連携

## ワークフロー

- [x] `.github/workflows/claude.yml` 新規作成（トリガー: `issue_comment` / `pull_request_review_comment` / `issues` / `pull_request_review`、`@claude` メンション + `author_association` による起動制限）
- [x] `permissions`（`contents`/`pull-requests`/`issues`/`id-token`: write, `actions`: read）設定
- [x] `anthropics/claude-code-action@v1` + `additional_permissions: actions: read` 設定

## 運用上の手動作業（ユーザー側）

- [x] ローカルで `/install-github-app` を実行し、Claude GitHub App をリポジトリにインストール
- [x] `ANTHROPIC_API_KEY` をリポジトリの Settings → Secrets and variables → Actions に登録

## 確認

- [x] `.github/workflows/claude.yml` をコミット（push は未実施）
- [x] Issueに `@claude` メンションでコメントし、ワークフローが起動してClaudeが応答することを確認
- [x] Claudeが作成したPRが既存の `auto-pr.yml` と競合しないことを確認

## auto-pr.yml の拡張（PR description自動生成 + Claudeレビュー）

- [x] `Create PR if not exists` ステップでコミット一覧 + `git diff --stat` からPR descriptionを生成
- [x] `Claude summary & review` ステップ追加（PR description先頭への概要追記、`gh pr comment` でのコードレビュー投稿）
- [x] レビュー時のバグ修正（`title` 未定義のシェル構文エラー、`pr_number` 取得方法、`issues: write` 権限追加、プロンプトでの差分再計算の削減）
- [x] `claude.yml` の `if` 条件に `github.actor != 'github-actions[bot]'` を追加（auto-pr.ymlのレビューコメントによる再起動防止）
- [x] `auto-pr.yml` の `on.push.branches-ignore` に `claude/**` を追加（Issueメンション経由でClaudeが作成するPRとのレース回避）

## 確認（追加分）

- [ ] 通常のブランチpushで `auto-pr.yml` がPR description生成 + Claudeレビューを実施することを確認
- [ ] Issueで `@claude` に実装を依頼し、Claudeが `claude/` ブランチでPRを作成した際に `auto-pr.yml` が起動しないことを確認
