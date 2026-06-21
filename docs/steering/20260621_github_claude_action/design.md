# 設計：GitHub Actions 上での Claude Code 連携

## ワークフロー構成（`.github/workflows/claude.yml`）

```
trigger: issue_comment / pull_request_review_comment / issues / pull_request_review
  └─ if: 本文・コメントに "@claude" を含む
       かつ author_association が OWNER/MEMBER/COLLABORATOR
       └─ anthropics/claude-code-action@v1 を実行
            ├─ permissions: contents/pull-requests/issues/id-token: write, actions: read
            └─ additional_permissions: actions: read（CI失敗ログの分析に使用）
```

### トリガー制限の理由

`@claude` メンションだけを条件にすると、フォークからのIssueコメントや外部ユーザーのコメントでもワークフローが起動し、
APIコストの発生や意図しないコード変更PRの作成につながる。`author_association` でリポジトリのOWNER/MEMBER/COLLABORATORに
限定することで、書き込み権限を持つ信頼できるユーザーのみが起動できるようにする。

イベントごとに `author_association` の参照元が異なるため、`if` 条件内で `github.event.comment.author_association` /
`github.event.review.author_association` / `github.event.issue.author_association` の順に `||` で繋いで判定する。

> **注意**: `issue_comment` イベントのペイロードには `issue`（Issue/PRの起票者の関連性）と `comment`
> （実際のコメント投稿者の関連性）が**両方**独立して存在する。`issue.author_association` を先頭に置くと、
> 常に起票者側の値が truthy として優先され、実際のコメント投稿者の権限チェックをすり抜けてしまう
> （OWNERが起票したIssueに、権限のない第三者が `@claude` とコメントしても通過してしまう）。
> そのため `comment`/`review` を優先し、コメント・レビュー自体が存在しない `issues` イベント
> （Issue本文がトリガー）でのみ `issue.author_association` にフォールバックする順序にしている。

### permissions と `actions: read`

`anthropics/claude-code-action` の公式ドキュメント（`docs/setup.md`）が示す基本構成は
`contents: write` / `pull-requests: write` / `issues: write` / `id-token: write`。
これに加えて `actions: read` を付与し、`with.additional_permissions: actions: read` を設定することで、
Claudeが `mcp__github_ci__get_ci_status` / `get_workflow_run_details` / `download_job_log` の3つのMCPツールを使えるようになり、
「このCI失敗を直して」のような依頼にも対応できる。`actions: read` を付けない場合、Claudeはこの機能が必要な場面で警告を出す。

### バージョン指定

`@beta` ではなく `@v1`（現行の正式リリース）を指定する。

## 認証・GitHub App（リポジトリ外の手動作業）

このワークフローファイル自体はリポジトリにコミットすれば動作するが、以下はGitHub UI / ローカルCLIでの作業が別途必要
（Claude Codeからは実行できない）。

1. ローカルで `claude` を起動し `/install-github-app` を実行 → このリポジトリへの Claude GitHub App インストールをガイドされる
2. リポジトリの Settings → Secrets and variables → Actions → New repository secret で `ANTHROPIC_API_KEY` を設定
   （`AGENTS.md` に記載の `internal/config/config.yml` 用キーとは別に、Actions用Secretとして登録する）

## 既存ドキュメントとの関係

`CLAUDE.md` の「フロントエンドの動作確認方針」（ブラウザ目視確認をしない）や `AGENTS.md` のビルド・テスト手順
（`internal/driver/anthropic` 関連のOOM対策フラグ等）は、Claude Codeがリポジトリルートの `CLAUDE.md` / `AGENTS.md` を
自動的に読み込むため、GitHub Actions経由の実行でも追加設定なしにそのまま引き継がれる。
