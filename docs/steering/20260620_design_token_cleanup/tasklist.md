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

## 追加の共通コンポーネント抽出（2026-06-20 再改訂）

- [x] `ui/Button.tsx` を作成し、Header の3ボタン（Fetch/Bookmark/FeedManagement）・Feed Management Modalの追加ボタンを統合（`icon`/`variant`/`active`/`spinning`/`badge` props）
- [x] `ui/TextField.tsx` を作成し、検索欄・URL入力欄を統合（`hasIcon` prop）
- [x] `ui/Table.tsx`（外枠・ヘッダ背景・行区切り線の共通シェル）を作成し、`ArticleTable.tsx` に適用（`bg-[rgba(...)]` 任意値も `bg-surface-raised/*` に修正）
- [x] `FeedManagementModal.tsx` のフィード一覧をdivによる擬似テーブルから実際の `<table>` に修正し、`Table` シェルを適用
- [x] `ui/PageButton.tsx` を作成し `Pagination.tsx` を更新（`variant: 'nav' | 'number'`）
- [x] `npx tsc --noEmit && npm run test && npm run build && npm run lint`（すべて成功。CSSサイズの微増は`FeedManagementModal`のテーブル化に伴う想定内の変化）

## Figma側: 追加コンポーネント化・フレーム整理（2026-06-20 再改訂）

- [x] `Button`（Component Set: `Icon`×`Style`）を既存4ボタンから作成し、インスタンスに置き換え（作成時にLabelプロパティ共有による文言統一バグが発生→プロパティを削除し固定テキストに修正）
- [x] `TextField`（Component Set: `Has Icon`）を作成し、検索欄・URL欄をインスタンスに置き換え（`Has Icon=False`が1px幅に潰れるバグ→`layoutSizingHorizontal/Vertical`を`FIXED`に修正）
- [x] `PageButton`（Component Set: `Variant`=Prev/Number/Next）を作成し、ページネーションをインスタンスに置き換え（Next用に左矢印を流用してしまうバグ→ベクターを水平反転して`Variant=Next`を新規作成）
- [x] `ArticleTable`・Feed Management Modalのテーブル部分の外枠・ヘッダ背景を`Design Tokens`変数にバインド（Componentとしては作成せず、色のみ統一）
- [x] `Index`→`ArticleListPage`、`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`のフレーム名をコードのコンポーネント名に整理
- [x] `get_screenshot` で各箇所の見た目に変化がないことを確認

## デザイン仕様書

- [x] `docs/component-spec.md` を新規作成し、Figma Component ↔ コードコンポーネントの対応表（props・既知の差異）を記録

## 確認

- [x] `npx tsc --noEmit && npm run test && npm run build`（すべて成功。`npm run lint` も確認）
- [ ] 既存画面の見た目に変化がないことを目視確認（ユーザー自身、`CLAUDE.md` の方針通り）

## Atomic Design再編・追加コンポーネント化（2026-06-20 三次改訂）

- [x] `components/` を `atoms/molecules/organisms/templates/pages/` の5階層に再編し、各ファイルのimportパスを修正
- [x] `App.tsx` を `templates/ArticleListTemplate.tsx`（レイアウトのみ）と `pages/ArticleListPage.tsx`（状態管理）に分割し、`main.tsx` の参照先を変更
- [x] `Button` を `icon` の閉じたenumから `children` でアイコン・ラベルを自由合成する汎用atomに再設計（`variant`propも不要になったため削除し常時filledスタイルに統一）。Header3ボタン・追加ボタンの呼び出し側をchildren合成に更新
- [x] `TextField` の `hasIcon: boolean` を `icon?: ReactNode` に変更し、`SearchFilterBar.tsx` の呼び出しを更新
- [x] `molecules/CategoryBadge.tsx` を新規作成し、`ArticleTable.tsx` 内の2箇所のカテゴリバッジ配色ロジックを置き換え
- [x] `tailwind.config.js` に `text.primary`/`text.secondary`/`accent.default` セマンティックトークンを追加し、既存の `text-slate-200`/`text-slate-500`/`text-amber-400` 使用箇所を置き換え
- [x] `npx tsc --noEmit && npm run test && npm run build && npm run lint`（すべて成功。CSSサイズの微増はトークン名変更に伴う想定内の変化）
- [x] Figma: `Page 1` を `General` にリネームし、新規ページ `Frontend` を作成。`ArticleListPage`・`Feed Management Modal` フレームを `Frontend` に移動
- [x] Figma: `Icon`（Component Set、11種）を `General` ページに作成。既存のButton/IconButton/TextField/PageButton/SelectFieldに埋め込まれていたベクターを複製して再利用
- [x] Figma: `Button`（旧: Icon×Style 4 Variant）を単一Component（`Icon`をINSTANCE_SWAP、`Label`をTEXTプロパティ、常時filled背景）に再構築。Header3インスタンス・追加ボタンインスタンスを新Componentにスワップし、旧Component Setを削除
- [x] Figma: `TextField` の `Has Icon=True` 側に埋め込まれていた検索アイコンのベクターを、`Icon`（INSTANCE_SWAP, デフォルト値Search）に変更
- [x] Figma: `Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`/`FeedManagementModal` をその場でMain Component化（`Frontend`ページ）
- [x] `get_screenshot` で各箇所（Header全体・Add Feed Row・ArticleListPage全体・Feed Management Modal）の見た目に変化がないことを確認
- [x] `docs/component-spec.md` をAtomic Design階層・Figmaページ構成に合わせて全面更新

## 開発サーバーでの動作確認中に発覚した不具合・追加対応（2026-06-20）

- [x] `bin/rss-agent enrich` で未分類だった記事48件（全227件中）に要約・カテゴリを付与
- [x] `FeedManagementModal.tsx` の削除ボタン（`IconButton`、実測幅28px）に対し `ACTION_COL_CLASS` が `w-6`（24px）で狭く、`table-fixed` レイアウトで見切れていたバグを `w-8`（32px）に修正
- [x] `domain/category.ts` に、enrichで新たに付与された9カテゴリ（Business/Career/Security/QA/Entertainment/Education/Sports/Project/Personal）の配色を追加
- [ ] コードの差分をユーザーが確認し、コミットの承認を得る（自動コミットは行わない方針のため保留）
