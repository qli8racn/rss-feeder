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

## auto-pr.yml の拡張（PR description自動生成）

- [x] `Create PR if not exists` ステップでコミット一覧 + `git diff --stat` からPR descriptionを生成
- [x] バグ修正（`title` 未定義のシェル構文エラー）
- [x] `claude.yml` の `if` 条件に `github.actor != 'github-actions[bot]'` を追加（bot自身のイベントによる再起動防止。defense-in-depth）
- [x] `auto-pr.yml` の `on.push.branches-ignore` に `claude/**` を追加（Issueメンション経由でClaudeが作成するPRとのレース回避）
- [x] PR作成時の自動Claudeレビューは見送り（「PRを作ったら必ずレビューしてほしいわけではない」という判断。
  レビューしたい場合は人間のPRと同様に `@claude` メンションで都度依頼する。検証時に一度実装したが、
  `claude-code-action` が `push` イベント未対応で `Unsupported event type: push` になることが分かり、
  別ワークフロー（`pull_request: opened`）への分離も検討した上で機能自体を撤回した）

## 確認（追加分）

- [x] 通常のブランチpushで `auto-pr.yml` がPR description付きのdraft PRを作成することを確認
- [x] 作成されたPRに `@claude このPRをレビューして` とコメントし、`claude.yml` が起動してレビューが投稿されることを確認
- [x] Issueで `@claude` に実装を依頼し、Claudeが `claude/` ブランチでPRを作成した際に `auto-pr.yml` が起動しないことを確認
