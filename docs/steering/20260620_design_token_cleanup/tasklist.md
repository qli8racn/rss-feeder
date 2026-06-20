# タスクリスト：フロントエンド デザイントークン・コンポーネント整理

## 調査

- [x] 既存コンポーネントで使われている任意値カラー・フォントサイズを洗い出す
- [x] 重複しているスタイルパターンを洗い出す
- [x] Figma側の現状（コンポーネント化の有無）を確認する

## トークン定義

- [ ] `tailwind.config.js` の `theme.extend` に色（`surface.base` / `surface.raised`）・フォントサイズトークンを追加する
- [ ] `Header.tsx` / `FeedManagementModal.tsx` / `SearchFilterBar.tsx` の任意値カラー（`bg-[#0d1117]` / `bg-[#1e2733]`）をトークン参照に置き換える
- [ ] 任意値フォントサイズ（`text-[10px]`〜`text-[13px]`）をトークン化したクラスに置き換える

## 共通コンポーネント抽出

- [ ] `IconButton` 抽出の対象範囲を決め、`web/frontend/src/components/ui/IconButton.tsx` として切り出す（閉じる・削除・ヘッダーボタン群）
- [ ] `SelectField` / `TextField` 抽出の対象範囲を決め、必要なら切り出す（過剰な共通化は避け、2箇所以上で完全一致するスタイルのみ対象とする）

## Figma側のコンポーネント化・Code Connect導入

- [ ] `figma-generate-library` skill で、抽出した共通コンポーネント（`IconButton` 等）に対応する要素をFigma上でMain Component化し、Variantsを設定する
- [ ] `figma-code-connect` skill で、`web/frontend/src/components/ui/` の各コンポーネントに対応する `.figma.tsx` を作成する
- [ ] Code Connect Publish後、対応関係が反映されていることを確認する

## 確認

- [ ] `npx tsc --noEmit && npm run test && npm run build`
- [ ] 既存画面の見た目に変化がないことを目視確認（ユーザー自身、`CLAUDE.md` の方針通り）
