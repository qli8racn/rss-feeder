---
applyTo: "web/**/*.{ts,tsx}"
---

# TypeScript (Web フロントエンド) 向けルール

- 配置場所: `web/` 配下
- バックエンド API (`cmd/web`) が提供するエンドポイントを介してデータ取得・更新を行い、ロジックを重複させない
- API 呼び出しは共通の client/fetch ラッパーに集約し、コンポーネントごとに重複実装しない
- 型は API レスポンスの shape に合わせて定義し、Go 側 (`internal/domain`) のモデルと命名・構造を揃える
- ビルド確認はプロジェクトの package manager に従う（`npm run build` 等、`web/package.json` 参照）
