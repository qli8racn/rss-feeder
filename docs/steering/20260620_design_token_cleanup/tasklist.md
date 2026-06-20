# タスクリスト：フロントエンド デザイントークン・コンポーネント整理

## 調査

- [x] 既存コンポーネントで使われている任意値カラー・フォントサイズを洗い出す
- [x] 重複しているスタイルパターンを洗い出す
- [x] Figma側の現状（コンポーネント化の有無）を確認する

## トークン定義

- [x] `tailwind.config.js` の `theme.extend` に色（`surface.base` / `surface.raised`）・フォントサイズトークン（`micro`/`caption`/`small`/`body`）を追加する
- [x] `Header.tsx` / `FeedManagementModal.tsx` / `SearchFilterBar.tsx` / `App.tsx` の任意値カラー（`bg-[#0d1117]` / `bg-[#1e2733]`）をトークン参照に置き換える
- [x] 任意値フォントサイズ（`text-[10px]`〜`text-[13px]`、`10.5px`/`11.25px`は近似統合）をトークン化したクラスに置き換える

## 共通コンポーネント抽出

- [x] `IconButton` を `web/frontend/src/components/ui/IconButton.tsx` として切り出す（`FeedManagementModal.tsx` の閉じる・削除ボタンの完全一致スタイルが対象。`Header.tsx` のボタンはテキスト付きで構造が異なるため対象外）
- [x] `SelectField` を `web/frontend/src/components/ui/SelectField.tsx` として切り出す（`SearchFilterBar.tsx` 内のカテゴリ・件数セレクトが完全一致スタイルのため対象）。`TextField` は検索欄（`pl-9 pr-4`、検索アイコン付き）とURL入力欄（`px-3`、アイコンなし）でパディング・構造が異なり完全一致しないため抽出対象外と判断した

## Figma側のコンポーネント化・Code Connect導入

- [ ] `figma-generate-library` skill で、抽出した共通コンポーネント（`IconButton` 等）に対応する要素をFigma上でMain Component化し、Variantsを設定する
- [ ] `figma-code-connect` skill で、`web/frontend/src/components/ui/` の各コンポーネントに対応する `.figma.tsx` を作成する
- [ ] Code Connect Publish後、対応関係が反映されていることを確認する

## 確認（トークン定義・コンポーネント抽出分）

- [x] `npx tsc --noEmit && npm run test && npm run build`（すべて成功。`npm run lint` も確認。ビルド後のCSSサイズは変化なし）
- [ ] 既存画面の見た目に変化がないことを目視確認（ユーザー自身、`CLAUDE.md` の方針通り）
