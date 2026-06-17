# RSS Feeder

Claude Code のエージェント・Hooks・SQLite 連携を学習するプロジェクト。

共通の開発ルール（セットアップ、CLI コマンド、テスト、トラブルシューティング）は [AGENTS.md](./AGENTS.md) を参照。

---

## Hooks

Hook スクリプトは `.claude/hooks/` に配置する。
設定は `.claude/settings.json` の `hooks` セクションで行う（詳細は `docs/design.md` 参照）。

Hook が動かない場合: Ctrl+O で verbose mode を有効化。

---

## フロントエンドの動作確認方針

Claude Code は `web/frontend` の変更確認において、ブラウザ（chromium-cli・Playwright 等）を使った目視確認は実施しない。
確認は以下の範囲で行う。

- `tsc --noEmit` による型チェック
- `npm run test`（Vitest）によるユニットテスト
- `npm run build` によるビルド確認
- バックエンド API は `curl` 等での実エンドポイント確認

ブラウザでの目視確認が必要な場合はユーザー自身が行う。
