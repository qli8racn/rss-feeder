# フェーズ：GitHub Actions 上での Claude Code 連携

## 概要

GitHub の Issue / PR 上で `@claude` をメンションすると、Claude Code が自動的に応答・実装・PR作成まで行う仕組みを導入する。

## 背景・目的

- 既存の開発フローは「`docs/steering` に設計ドキュメントを作成 → 実装 → PRがマージされれば対応完了」という形を、ローカルのClaude Codeとのやりとりで回している
- これに加えて、GitHub上でIssueを起票し `@claude` メンションで対応を依頼する経路を追加し、Issue起票からPR作成までをGitHub上で完結できるようにする
- 公式の `anthropics/claude-code-action`（GitHub Action）を利用する

## コードレビューの方式

コードレビューは `@claude このPRをレビューして` のような**メンション起点**でのみ行う（既存の `issue_comment` /
`pull_request_review_comment` トリガーでカバーされるため追加実装は不要）。PRオープン・更新時に
メンション無しで自動的にレビューを走らせる方式（`on: pull_request: types: [opened, synchronize]` +
`prompt:` で無条件にレビュー指示を送る構成）は採用しない。

## 受け入れ条件

- `.github/workflows/claude.yml` を新規作成し、以下のイベントで `@claude` を含むコメント・本文をトリガーとする
  - `issue_comment`（Issue/PRへのコメント）
  - `pull_request_review_comment`（PRのコードレビューコメント）
  - `issues`（Issue本文。新規作成・assigned時）
  - `pull_request_review`（PRレビュー本文）
- トリガーは `author_association` が `OWNER`/`MEMBER`/`COLLABORATOR` のいずれかのユーザーに限定する（第三者による不要な起動・APIコスト発生・意図しないコード変更を防ぐ）
- ワークフローの `permissions` は `contents: write` / `pull-requests: write` / `issues: write` / `id-token: write` / `actions: read` を付与する
- `anthropics/claude-code-action@v1` を使用し、CI失敗のログ分析もできるよう `additional_permissions: actions: read` を設定する
- 認証は `ANTHROPIC_API_KEY`（リポジトリSecrets）を使用する
- Claude Codeが応答する際、リポジトリ内の `CLAUDE.md` / `AGENTS.md` の方針（フロントエンドのブラウザ目視確認をしない、`ANTHROPIC_API_KEY` の発行元の区別、等）をそのまま引き継ぐ（Claude Codeは標準でこれらのファイルを読み込むため追加設定は不要）

## 運用上の前提（リポジトリ外の手動作業）

- Claude GitHub App をこのリポジトリにインストールする（ローカルで `claude` を起動し `/install-github-app` を実行）
- `ANTHROPIC_API_KEY` をリポジトリの Settings → Secrets and variables → Actions に登録する

## スコープ外

- ワークフロー実行時のコスト監視・アラート
- `@claude` メンション専用の Issue テンプレート整備
- セルフホストランナーでの実行
- `CLAUDE_CODE_OAUTH_TOKEN`（Pro/Max契約者向けOAuth認証）の導入（まずは既存の `ANTHROPIC_API_KEY` 運用に統一する）
- `pull_request` イベント（`opened`/`synchronize`）を起点とした、メンション不要の自動コードレビュー
  （PRごとに無条件でAPIコストが発生するため見送り。レビューは `@claude` メンションで都度依頼する運用とする）
