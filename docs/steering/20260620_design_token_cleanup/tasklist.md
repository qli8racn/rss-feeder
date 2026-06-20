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

## Figma側のコンポーネント化

- [x] Variable Collection `Design Tokens`（`border/subtle` / `surface/raised` / `text/primary` / `text/secondary` / `radius/sm`）を作成（既存画面の実測値と同一）
- [x] `figma-generate-library` skill で `IconButton`（Variant: `Icon=Close` / `Icon=Trash`）・`SelectField` をMain Component化し、上記Variablesにバインド
- [x] 既存のCloseボタン・3つのDeleteボタン・カテゴリ/件数セレクトを、作成したコンポーネントのインスタンスに置き換える（件数セレクトはテキストが欠落していたため `25件` を補完）
- [x] `get_screenshot` で見た目に変化がないことを確認

## Code Connect（対象外と判断）

- [x] Code Connectの前提条件（チームライブラリへの公開・Organization/Enterpriseプラン）を確認した結果、個人開発である本プロジェクトでは適用できないと判断し、スコープから除外した（`requirements.md` 参照）

## Component Properties の整備（コードとの対応付け）

- [x] `IconButton`: Figma側のVariantプロパティ `Icon`（`Close`/`Trash`）に合わせ、コード側を `children` 受け渡しから `icon: 'close' | 'trash'` propに変更する（`web/frontend/src/components/ui/IconButton.tsx`、`FeedManagementModal.tsx` の呼び出し2箇所を追随）
- [x] `SelectField`: ラベルをTEXTタイプのComponent Property `Label` として公開する（生のテキストレイヤー直接編集から変更）。コード側に対応する `label` propは追加しない（`<select>` の表示文字列はブラウザが選択中の `<option>` から自動描画するため、明示的なpropが実際には使われない。`design.md` 参照）

## 確認

- [x] `npx tsc --noEmit && npm run test && npm run build`（すべて成功。`npm run lint` も確認。ビルド後のCSSサイズは変化なし）
- [ ] 既存画面の見た目に変化がないことを目視確認（ユーザー自身、`CLAUDE.md` の方針通り）
