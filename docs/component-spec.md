# コンポーネント設計仕様書

`web/frontend` のFigmaデザイン（`https://www.figma.com/design/I0n9kvKiHCnN8vlCalkq6H/RSS-Feeder`）とコード（`web/frontend/src/components/`）の対応関係を管理する。

Figma Code Connect は導入していない（チームライブラリへの公開・Organization/Enterpriseプランが前提のため、個人開発の本プロジェクトでは利用不可。`docs/steering/20260620_design_token_cleanup/requirements.md` 参照）。そのため、Figma側のComponent Propertiesとコードのpropsはこのドキュメントで手動管理する。

## デザイントークン

Figma Variable Collection `Design Tokens`（mode: `Value`）と `tailwind.config.js` の `theme.extend` が対応する。

| Figma Variable | 値 | Tailwind トークン |
|---|---|---|
| `border/subtle` | `rgba(148, 163, 184, 0.12)` | `border-slate-400/10`（標準カラーをそのまま使用） |
| `surface/raised` | `#1e2733` | `bg-surface-raised` |
| `text/primary` | `#e2e8f0` | `text-slate-200`（標準カラー） |
| `text/secondary` | `#64748b` | `text-slate-500`（標準カラー） |
| `radius/sm` | `3.75px` | `rounded`（Tailwindデフォルトの`rounded`に近似） |

## 再利用コンポーネント（Figma Component ↔ コード）

| Figma Component | Variant / Property | コードコンポーネント | props | 備考 |
|---|---|---|---|---|
| `IconButton`（Component Set） | `Icon`: Close / Trash | `ui/IconButton.tsx` | `icon: 'close' \| 'trash'`, `onClick`, `ariaLabel`, `className?` | 1:1対応 |
| `Button`（Component Set） | `Icon`: Refresh / List / Bookmark / Plus, `Style`: Outline / Filled（Plusのみ Filled） | `ui/Button.tsx` | `icon`, `variant?: 'outline' \| 'filled'`, `onClick?`, `type?`, `children`, `active?`, `spinning?`, `badge?` | Label（ボタン文言）・badge件数はFigma側では各Variantに固定テキストとして埋め込み、Component Propertyとしては公開していない（4箇所とも呼び出し元の文言が固定であり、汎用テキストpropにする実益がないため。Active状態の見た目はコード側のみで再現し、Figma側は既定状態のみ表現） |
| `TextField`（Component Set） | `Has Icon`: True / False | `ui/TextField.tsx` | `value`, `onChange`, `placeholder?`, `disabled?`, `hasIcon?`, `className?` | True=検索欄（虫眼鏡アイコン付き）、False=フィード追加URL欄 |
| `SelectField`（Component） | `Label`（TEXT property） | `ui/SelectField.tsx` | `value`, `onChange`, `children`（`<option>`） | コード側に対応する`label`propは無い。表示文字列はブラウザが選択中の`<option>`から自動描画するため |
| `PageButton`（Component Set） | `Variant`: Prev / Next / Number | `ui/PageButton.tsx` | `variant: 'nav' \| 'number'`, `onClick`, `children`, `disabled?`, `active?`, `ariaLabel?` | Figma側はPrev/Nextを別Variantとして分離（矢印の向きが異なるため）。コード側は`variant='nav'`で両方を表現し、子要素（アイコン）で向きを切り替える |
| `Table`（コードのみ。Figma側はComponent化していない） | - | `ui/Table.tsx` | `children` | 外枠の角丸・ボーダー・ヘッダ背景・行区切り線のみを共通化したスタイルシェル。列構成は`ArticleTable`と`FeedManagementModal`で全く異なるため、Figma側は各セクションの該当フレームの色をDesign Tokens変数にバインドするのみで、独立したComponentとしては作成していない |

## ページ・セクション（1回しか使われないフレーム）

再利用されないセクションはFigma Componentにせず、コードのコンポーネント名と同じ名前のFrameとして整理している。

| Figmaフレーム名 | コードコンポーネント / ファイル | 備考 |
|---|---|---|
| `ArticleListPage`（旧名: `Index`） | `App.tsx` | 記事一覧画面のページ全体 |
| `Header` | `components/Header.tsx` | タイトル（アイコン+ラベル）＋ `Button` インスタンス3つ |
| `SearchFilterBar`（旧名: `Container`） | `components/SearchFilterBar.tsx` | `TextField` インスタンス1つ＋ `SelectField` インスタンス2つ |
| `ArticleTable`（旧名: `Container (margin)`） | `components/ArticleTable.tsx` | ヘッダ行＋記事行（ソートボタン・ブックマークアイコン・カテゴリバッジ等は個別実装） |
| `Pagination`（旧名: `Container (margin)`） | `components/Pagination.tsx` | `PageButton` インスタンス（Prev/Number×3/Next） |
| `Footer`（旧名: `Container (margin)`） | `components/Footer.tsx` | コピーライト的なテキスト1行 |
| `Feed Management Modal` | `components/FeedManagementModal.tsx` | モーダルヘッダ（`IconButton` インスタンス）＋追加フォーム（`TextField`+`Button`）＋フィード一覧（`<table>`、行ごとに `IconButton` インスタンス） |

## 既知の差異・あえて対応していない箇所

- **Code Connect未導入**: 上記の対応関係はすべて本ドキュメントでの手動管理。Figmaコンポーネントの構造変更時はこのファイルも更新すること
- **`Button`のActive状態・`SelectField`のLabel**: Figma側は代表的な既定状態のみを表現し、インタラクティブな状態変化（ホバー・アクティブ・選択値の変化等）はコード側のみで実装する。静的なデザインファイルで全状態を機械的に再現することの労力に対して実益が小さいと判断した
- **Figma Variablesの適用範囲**: `Design Tokens` は今回作成したコンポーネント・関連フレームにのみ適用した。`ArticleTable`内の個々の記事行（55行）やカテゴリバッジの配色は対象外（`docs/web-ui-spec.md` に記載の通り、カテゴリ配色は元々「実装時に再定義してよい」参考例の位置づけ）
