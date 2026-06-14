# RSS Feeder

Claude Code のエージェント・Hooks・SQLite 連携を学習するプロジェクト。

共通の開発ルール（セットアップ、CLI コマンド、テスト、トラブルシューティング）は [AGENTS.md](./AGENTS.md) を参照。

---

## Hooks

Hook スクリプトは `.claude/hooks/` に配置する。
設定は `.claude/settings.json` の `hooks` セクションで行う（詳細は `docs/design.md` 参照）。

Hook が動かない場合: Ctrl+O で verbose mode を有効化。
