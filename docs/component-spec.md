# コンポーネント設計仕様書

`web/frontend` のFigmaデザイン（`https://www.figma.com/design/I0n9kvKiHCnN8vlCalkq6H/RSS-Feeder`）とコード（`web/frontend/src/components/`）の対応関係を管理する。

Figma Code Connect は導入していない（チームライブラリへの公開・Organization/Enterpriseプランが前提のため、個人開発の本プロジェクトでは利用不可。`docs/steering/20260620_design_token_cleanup/requirements.md` 参照）。そのため、Figma側のComponent Propertiesとコードのpropsはこのドキュメントで手動管理する。

## ディレクトリ構成（Atomic Design）

コード側は `web/frontend/src/components/` 配下を Atomic Design の階層で整理している。Figma側も同じ階層で2つのページに分けて管理する。

| 階層 | コード | Figmaページ |
|---|---|---|
| atoms | `components/atoms/` | `General` |
| molecules | `components/molecules/` | `General` |
| organisms | `components/organisms/` | `Frontend` |
| templates | `components/templates/` | `Frontend`（テンプレート自体はFigma Componentにせず、ページフレーム内の構成として表現） |
| pages | `components/pages/` | `Frontend`（`ArticleListPage` / `Feed Management Modal` フレーム） |

Figma `General` ページ＝atoms・molecules（再利用される最小単位）、`Frontend` ページ＝organisms以上（画面構成・ページ単位）という分け方にしている。

## デザイントークン

Figma Variable Collection `Design Tokens`（mode: `Value`）と `tailwind.config.js` の `theme.extend` が対応する。

| Figma Variable | 値 | Tailwind トークン |
|---|---|---|
| `border/subtle` | `rgba(148, 163, 184, 0.12)` | `border-slate-400/10`（標準カラーをそのまま使用） |
| `surface/raised` | `#1e2733` | `bg-surface-raised` |
| `text/primary` | `#e2e8f0` | `text-text-primary` |
| `text/secondary` | `#64748b` | `text-text-secondary` |
| `radius/sm` | `3.75px` | `rounded`（Tailwindデフォルトの`rounded`に近似） |

`accent.default`（`#fbbf24`、`amber-400`相当）はFigma Variablesには未登録（ボタンのActive状態等、Figma側で機械的に再現していない箇所のみで使用するため）。

## atoms

| Figma Component | Property | コードコンポーネント | props | 備考 |
|---|---|---|---|---|
| `Icon`（Component Set） | `Name`: Rss / Refresh / Bookmark / Search / ChevronDown / ChevronLeft / ChevronRight / List / Plus / Trash / Close | `atoms/icons.tsx` の各関数（`RssIcon`等） | `className?`（`BookmarkIcon`のみ`filled?`も） | 既存のButton/IconButton/TextField/PageButton/SelectFieldに埋め込まれていたベクターを複製して作成。コード側は元から個別関数として分離済みのためファイル分割はしていない（`tasklist.md`参照） |
| `Button`（Component） | `Icon`（INSTANCE_SWAP, `Icon`atomを参照）、`Label`（TEXT） | `atoms/Button.tsx` | `onClick?`, `type?`, `children`, `disabled?`, `active?`, `ariaLabel?` | 常時filled背景（`bg-surface-raised`）。アイコン・ラベルはコード側では`children`として呼び出し元が自由合成、Figma側は`Icon`/`Label`プロパティで対応する具体的な値に差し替える。`active`状態の配色変化（amber）はコード側のみで再現し、Figma側は既定（非アクティブ）状態のみを表現する |

## molecules

| Figma Component | Variant / Property | コードコンポーネント | props | 備考 |
|---|---|---|---|---|
| `IconButton`（Component Set） | `Icon`: Close / Trash | `molecules/IconButton.tsx` | `icon: 'close' \| 'trash'`, `onClick`, `ariaLabel`, `className?` | 1:1対応 |
| `TextField`（Component Set） | `Has Icon`: True / False、`Icon`（INSTANCE_SWAP, `Has Icon=True`側のみ参照） | `molecules/TextField.tsx` | `value`, `onChange`, `placeholder?`, `disabled?`, `icon?: ReactNode`, `className?` | コード側は`hasIcon`真偽値から`icon?: ReactNode`に変更し、検索アイコン固定ではなく任意のアイコンを受け取れるよう汎用化した。Figma側も同様に`Icon`をINSTANCE_SWAPにして、`Has Icon=True`のデフォルト値（Search）を任意のIcon atomに差し替え可能にしている |
| `SelectField`（Component） | `Label`（TEXT property） | `molecules/SelectField.tsx` | `value`, `onChange`, `children`（`<option>`） | コード側に対応する`label`propは無い。表示文字列はブラウザが選択中の`<option>`から自動描画するため |
| `PageButton`（Component Set） | `Variant`: Prev / Next / Number | `molecules/PageButton.tsx` | `variant: 'nav' \| 'number'`, `onClick`, `children`, `disabled?`, `active?`, `ariaLabel?` | Figma側はPrev/Nextを別Variantとして分離（矢印の向きが異なるため）。コード側は`variant='nav'`で両方を表現し、子要素（アイコン）で向きを切り替える |
| `Table`（コードのみ。Figma側はComponent化していない） | - | `molecules/Table.tsx` | `children` | 外枠の角丸・ボーダー・ヘッダ背景・行区切り線のみを共通化したスタイルシェル。列構成は`ArticleTable`と`FeedManagementModal`で全く異なるため、Figma側は各セクションの該当フレームの色をDesign Tokens変数にバインドするのみで、独立したComponentとしては作成していない |
| `CategoryBadge`（コードのみ。Figma側はComponent化していない） | - | `molecules/CategoryBadge.tsx` | `category: string`, `className?` | カテゴリごとの配色（`domain/category.ts`の`categoryStyle`）を当てたバッジ。Figma側は元々`ArticleTable`内の各行に直接配色されたテキストとして存在し、独立したコンポーネントにはなっていない（`docs/web-ui-spec.md`参照） |

## organisms

すべてFigma Main Component化済み（`Frontend`ページ）。再利用はされていないが、画面の構成単位を明示するためComponent化した。

| Figma Component | コードコンポーネント / ファイル | 備考 |
|---|---|---|
| `Header` | `components/organisms/Header.tsx` | タイトル（アイコン+ラベル）＋ `Button` インスタンス3つ（最新フィードを取得・フィード管理・ブックマーク。いずれもfilled背景に統一） |
| `SearchFilterBar` | `components/organisms/SearchFilterBar.tsx` | `TextField` インスタンス1つ＋ `SelectField` インスタンス2つ |
| `ArticleTable` | `components/organisms/ArticleTable.tsx` | ヘッダ行＋記事行（ソートボタン・ブックマークアイコン・カテゴリバッジ等は個別実装） |
| `Pagination` | `components/organisms/Pagination.tsx` | `PageButton` インスタンス（Prev/Number×3/Next） |
| `Footer` | `components/organisms/Footer.tsx` | コピーライト的なテキスト1行 |
| `FeedManagementModal` | `components/organisms/FeedManagementModal.tsx` | モーダルヘッダ（`IconButton` インスタンス）＋追加フォーム（`TextField`+`Button`）＋フィード一覧（`<table>`、行ごとに `IconButton` インスタンス） |

## templates / pages

| Figmaフレーム名 | コードコンポーネント / ファイル | 備考 |
|---|---|---|
| `ArticleListPage`（旧名: `Index`） | `components/templates/ArticleListTemplate.tsx` + `components/pages/ArticleListPage.tsx` | Figma側はレイアウトと状態管理の分離まではフレーム単位で表現していない（1フレームのまま）。コード側はレイアウトのみの`ArticleListTemplate`と、状態管理を持つ`ArticleListPage`に分割している |

## 既知の差異・あえて対応していない箇所

- **Code Connect未導入**: 上記の対応関係はすべて本ドキュメントでの手動管理。Figmaコンポーネントの構造変更時はこのファイルも更新すること
- **`Button`のActive状態・`SelectField`のLabel**: Figma側は代表的な既定状態のみを表現し、インタラクティブな状態変化（ホバー・アクティブ・選択値の変化等）はコード側のみで実装する。静的なデザインファイルで全状態を機械的に再現することの労力に対して実益が小さいと判断した
- **Figma Variablesの適用範囲**: `Design Tokens` は今回作成したコンポーネント・関連フレームにのみ適用した。`ArticleTable`内の個々の記事行（25行）やカテゴリバッジの配色は対象外（`docs/web-ui-spec.md` に記載の通り、カテゴリ配色は元々「実装時に再定義してよい」参考例の位置づけ）
- **templates/pagesの分離はコード側のみ**: Figma側は`ArticleListPage`フレーム1枚のままで、コードのようなレイアウト（Template）と状態管理（Page）の分離は表現していない（Figmaは静的なデザインファイルであり、状態管理という概念自体が存在しないため）
