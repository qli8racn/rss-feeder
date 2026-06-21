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

### `auto-pr.yml` との競合防止

`auto-pr.yml` は PR作成時に `anthropics/claude-code-action` を使ってPRへのコメント（`gh pr comment`）を投稿する。
このコメント作成は `issue_comment` イベントを発火させるため、コメント本文に `@claude` という文字列が
含まれていると `claude.yml` が再度起動してしまう可能性がある。

通常は `auto-pr.yml` が `GH_TOKEN: ${{ github.token }}`（`github-actions[bot]`）でコメントを投稿するため、
そのコメントの `author_association` は `NONE` となり上記の権限チェックで素通りはしない。しかし、
将来 `.github/workflows` へのpush制限を回避する目的などでPAT（コラボレーター権限）に切り替えた場合、
`author_association` が `COLLABORATOR` 等になり権限チェックを通過してしまう可能性がある。

この依存を排除するため、`if` 条件の先頭に `github.actor != 'github-actions[bot]'` を明示的に追加し、
bot自身が起こしたイベントでは `author_association` の値に関わらず起動しないようにしている。

### permissions と `actions: read`

`anthropics/claude-code-action` の公式ドキュメント（`docs/setup.md`）が示す基本構成は
`contents: write` / `pull-requests: write` / `issues: write` / `id-token: write`。
これに加えて `actions: read` を付与し、`with.additional_permissions: actions: read` を設定することで、
Claudeが `mcp__github_ci__get_ci_status` / `get_workflow_run_details` / `download_job_log` の3つのMCPツールを使えるようになり、
「このCI失敗を直して」のような依頼にも対応できる。`actions: read` を付けない場合、Claudeはこの機能が必要な場面で警告を出す。

### バージョン指定

`@beta` ではなく `@v1`（現行の正式リリース）を指定する。

## ワークフロー構成（`.github/workflows/auto-pr.yml`）

```
trigger: push（main以外のブランチ。claude/** は対象外）
  └─ 同じブランチのPRが存在しない場合
       ├─ コミット一覧 + `git diff --stat` からPR descriptionを生成してdraft PRを作成
       └─ anthropics/claude-code-action@v1 を実行
            ├─ PR descriptionの先頭に「## Claude による変更概要」セクションを追記
            └─ コードレビュー結果をPRコメントとして投稿
```

### `claude.yml` が作成するブランチとのスコープ分離

`anthropics/claude-code-action` はIssueメンション経由で実装を依頼された場合、ブランチ作成からPR作成までを
自身で完結させる（デフォルトの `branch_prefix: claude/` でブランチを作成し、最後に `gh pr create` する）。

このブランチのpushも `auto-pr.yml` の `push` トリガーに引っかかるため、対策をしないと以下のレースが発生する。

- `claude.yml` のClaudeが先にPRを作成 → `auto-pr.yml` は「既存PRあり」で早期終了する（実害は小さい）
- `auto-pr.yml` が先にPRを作成 → 汎用的な説明文（コミット一覧＋diff統計）のPRが先に出来てしまい、
  `auto-pr.yml` 独自のレビューも走る。その後 `claude.yml` 側のClaudeが `gh pr create` すると
  「既にPRが存在する」エラーになり、Issueへの完了報告が失敗扱いになったり、レビューが二重に走ったりする

タイミング依存のレースを個別にハンドリングするのではなく、根本的にスコープを分離して解決する。
`auto-pr.yml` の `on.push.branches-ignore` に `claude/**` を追加し、Issueメンション経由でClaudeが
作成したブランチは最初から `auto-pr.yml` の対象から除外する。これにより、そのブランチのPR作成・
レビューは `claude.yml` 側のClaudeセッションが一貫して担当する。

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
