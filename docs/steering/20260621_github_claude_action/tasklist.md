# タスクリスト：GitHub Actions 上での Claude Code 連携

## ワークフロー

- [x] `.github/workflows/claude.yml` 新規作成（トリガー: `issue_comment` / `pull_request_review_comment` / `issues` / `pull_request_review`、`@claude` メンション + `author_association` による起動制限）
- [x] `permissions`（`contents`/`pull-requests`/`issues`/`id-token`: write, `actions`: read）設定
- [x] `anthropics/claude-code-action@v1` + `additional_permissions: actions: read` 設定

## 運用上の手動作業（ユーザー側）

- [x] ローカルで `/install-github-app` を実行し、Claude GitHub App をリポジトリにインストール
- [ ] `ANTHROPIC_API_KEY` をリポジトリの Settings → Secrets and variables → Actions に登録

## 確認

- [ ] `.github/workflows/claude.yml` をコミット・push
- [ ] Issueに `@claude` メンションでコメントし、ワークフローが起動してClaudeが応答することを確認
- [ ] Claudeが作成したPRが既存の `auto-pr.yml` と競合しないことを確認
