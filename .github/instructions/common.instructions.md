---
applyTo: "**"
---

# RSS Feeder 共通ルール

詳細は [AGENTS.md](../../AGENTS.md) を参照。

- 機能要求 → `docs/product-requirements.md`
- 機能設計・技術仕様・アーキテクチャ → `docs/design.md`
- Web UI は Go バックエンド (`cmd/web`, `internal`) と TypeScript フロントエンド (`web/`) で構成される
- 既存のドメイン/usecase 層のロジックを再利用し、フロントエンド側に重複させない
