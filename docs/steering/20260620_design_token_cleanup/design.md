# 設計：フロントエンド デザイントークン・コンポーネント整理

## 現状調査

### 任意値カラーの使用箇所

```
bg-[#0d1117]   3箇所（Header / FeedManagementModal）  … ダーク背景（base）
bg-[#1e2733]   7箇所（Header / SearchFilterBar / FeedManagementModal）  … カード・入力欄背景（raised）
```

### Tailwind標準カラーの使用頻度（上位）

```
text-slate-500    29箇所
border-slate-400   22箇所（多くが `/10` 透過指定で「枠線」として使用）
text-slate-200     10箇所
text-amber-400      3箇所
border-amber-500    2箇所
text-slate-400      2箇所
bg-amber-500        1箇所
text-amber-500      1箇所
text-slate-300      1箇所
```

`slate-500` がテキストの「補助色」、`slate-200` が「主要テキスト」、`amber-*` が強調（フェッチ中表示・バッジ等）として使われているが、いずれもコンポーネント側で個別にクラス名を書いており、意味づけ（semantic name）が存在しない。

### 任意値フォントサイズ

`text-[10px]` / `text-[10.5px]` / `text-[11px]` / `text-[11.25px]` / `text-[12px]` / `text-[13px]` など、似た意図（ラベル・本文・キャプション）で微妙に異なる値が複数存在する（例: `FeedManagementModal.tsx` 内だけで `10.5px` と `11.25px` が混在）。

### 重複しているスタイルパターン

- アイコンボタン: `rounded border border-slate-400/10 p-1.5`（`FeedManagementModal.tsx` の閉じるボタン・削除ボタンで重複、`Header.tsx` のボタンも同系統）
- セレクト風入力: `appearance-none rounded border border-slate-400/10 bg-[#1e2733] py-2 pl-3 pr-8 font-mono text-[11px] text-slate-200 focus:outline-none`（`SearchFilterBar.tsx` 内で2回ほぼ同一）
- テキスト入力欄: `rounded border border-slate-400/10 bg-[#1e2733] ... text-slate-200 placeholder:text-slate-500 focus:outline-none`（`SearchFilterBar.tsx` の検索欄と `FeedManagementModal.tsx` のURL欄で同系統）

### Figma側の状況

- ファイル: `https://www.figma.com/design/I0n9kvKiHCnN8vlCalkq6H/RSS-Feeder?node-id=0-1`
- Figma Makeで生成した1ページ（記事一覧画面）のみが存在する
- 後続の機能追加（フィード管理モーダル等）は、既存ノードの色・フォント・ボーダー・角丸を目視で実測し、新規フレームに複製する方式で進めてきた（`docs/steering/20260619_feed_management_ui/tasklist.md` 参照）
- コンポーネント化（Main Component / Variants）・Figma Variables・Code Connectのいずれも未導入

## 方針

### デザイントークン定義（`tailwind.config.js`）

実測値に意味づけした名前で `theme.extend` に追加する案。

```js
theme: {
  extend: {
    colors: {
      surface: {
        base: '#0d1117',    // 画面・モーダルの背景
        raised: '#1e2733',  // 入力欄・カード・テーブルヘッダの背景
      },
    },
    fontSize: {
      micro: ['10px', { lineHeight: '1.4' }],     // 列ラベル等（10px・10.5pxを統合）
      caption: ['11px', { lineHeight: '1.4' }],   // 補助テキスト（11px・11.25pxを統合）
      small: ['12px', { lineHeight: '1.4' }],     // バッジ等
      body: ['13px', { lineHeight: '1.5' }],      // 本文
    },
  },
},
```

実測値は6種（10px/10.5px/11px/11.25px/12px/13px）あったが、`10.5px→micro`・`11.25px→caption`の2件は0.5px未満の差であり、Figma上の実測誤差と判断して統合する。`12px`は他の値と十分な差があるため独立したトークン（`small`）として残し、見た目への実質的な影響を避ける。

`border-slate-400/10` のような既存Tailwind標準カラー+透過指定は、Tailwind標準のまま使い続けて問題ない（独自トークン化の優先度は低い）。今回は **任意値（`bg-[#...]` 等）をトークンに置き換えること** を主目的とし、標準カラーの再命名は対象外とする。

### 共通コンポーネント抽出の候補

- `IconButton`: アイコン1つ+ボーダー枠のボタン（閉じる・削除・既存ヘッダーボタン群）
- `SelectField` / `TextField`: `SearchFilterBar` / `FeedManagementModal` のセレクト・入力欄共通スタイル

実際に抽出するかどうか・範囲は `tasklist.md` で個別に判断する（無理に共通化すると却って複雑になるケースもあるため、2箇所以上で完全一致するスタイルのみを対象にする）。

### Figma側の進め方

1. コード側の共通コンポーネント抽出（`IconButton` / `SelectField`）を先に完了させる（トークン定義より後、Figma作業より前）
2. `figma-generate-library` skill を使い、対象範囲（アイコンボタン、セレクト）をFigma上でMain Component化する。色・角丸はFigma Variables（`Design Tokens` コレクション）として定義し、コード側のTailwindトークンと値を一致させてバインドする
3. 既存の重複箇所（Closeボタン・3つのDeleteボタン・カテゴリ/件数セレクト）を、作成したコンポーネントのインスタンスに置き換える
4. `get_screenshot` で見た目に変化がないことを確認する

対象範囲は画面全体（`ArticleTable` 等の複雑なレイアウト）ではなく、本フェーズで抽出する再利用可能な小コンポーネントに限定する（`requirements.md` のスコープ外参照）。

> **2026-06-20 改訂**: 当初はこの後の手順としてFigma Code Connect導入（`.figma.tsx`作成）を計画していたが、Code Connectはチームライブラリへの公開・Organization/Enterpriseプランが前提の機能であり、個人開発である本プロジェクトには適用できないと判明したため、対象外とした（`requirements.md` 参照）。

### 実施結果（Figma側）

- Variable Collection `Design Tokens`（mode: `Value`）: `border/subtle`・`surface/raised`・`text/primary`・`text/secondary`（いずれもCOLOR、既存画面の実測値と同一）・`radius/sm`（FLOAT、3.75）
- Component Set `IconButton`（Variant: `Icon=Close` / `Icon=Trash`）: 24×24、`border/subtle` にバインドした枠線、アイコンの線色は `text/secondary` にバインド
- Component `SelectField`: `surface/raised` 背景・`border/subtle` 枠線・`radius/sm` 角丸、ラベルは `text/primary`、chevronは `text/secondary`
- 既存のCloseボタン（26×26）・3つのDeleteボタン（24×24）を `IconButton` のインスタンスに置き換え（Closeボタンは24×24に統一。コード側も既に同サイズで実装済みのため見た目への実質的な影響はない）
- カテゴリセレクト（`Button` 1:31）・件数セレクト（壊れていた `Container` 1:35。テキストが欠落していたため `25件`＝コードの`DEFAULT_PER_PAGE`を補って表示）を `SelectField` のインスタンスに置き換え

### Component Properties の整備（コードとの対応付け）

Code Connect は導入しないが、Figma側のComponent Propertiesとコード側のpropsの対応関係を明示的に揃える。

- **IconButton**: `combineAsVariants` により自動生成されたVariantプロパティ `Icon`（`Close` / `Trash`）を、コード側 `IconButton` の `icon: 'close' | 'trash'` propとして1:1で対応させた。従来は `children` でアイコン要素を呼び出し側から渡す設計だったが、Figma側が「アイコンの種類をVariantで選ぶ」構造になっているため、コード側もそれに合わせて閉じた選択肢のpropに変更した（`web/frontend/src/components/ui/IconButton.tsx`）
- **SelectField**: ラベルの文字列を生のテキストレイヤー直接編集ではなく、TEXTタイプのComponent Property `Label` として公開した（デフォルト値 `すべてのカテゴリ`）。
  - コード側に対応する `label` propは追加していない。コード側の `<select>` は選択中の `<option>` の表示テキストをブラウザが自動描画するため、明示的な `label` propを持たせても実際には使われない（呼び出し側が `children`（`<option>` 群）と `value` を渡せば表示文字列は自動的に決まる）。Figmaの `Label` は「静的なモックアップ上で見た目を作るための値」であり、コードでは `value` に対応する `<option>` の文字列が実質的な対応物となる

### 追加の分解（2026-06-20 再改訂）

`IconButton`/`SelectField`だけでは `Index`（記事一覧ページ全体）・`Feed Management Modal` という1セクション単位の構造が残っていたため、以下を追加で抽出した。

- **`Button`**（アイコン+ラベルボタン）: Header の「最新フィードを取得」「ブックマーク」「フィード管理」、Feed Management Modal の「追加」の4箇所が対象。Variant `Icon`（Refresh/List/Bookmark/Plus）と `Style`（Outline/Filled。Plusのみ Filled）の2軸。ブックマークの件数バッジ・アクティブ状態の配色切り替えはコード側のみで表現し、Figmaは既定状態（非アクティブ）のみを表現する
- **`TextField`**: 検索欄（虫眼鏡アイコン付き）とフィード追加URL欄を、Variant `Has Icon`（True/False）で統合
- **`PageButton`**: ページネーションの矢印（Prev/Next）・数字ボタンを統合。Prev/Nextは矢印の向きが異なるため別Variantとして分離した
- **`Table`シェル**: `ArticleTable` と `Feed Management Modal` のフィード一覧で、外枠の角丸・ボーダー・ヘッダ背景・行区切り線のみを共通化。列構成は完全に異なるため、列の内容（セル）は共通化していない。Feed Management Modal側は元々divによる擬似テーブルだったため、実際の `<table>` マークアップに修正した上で適用した

Figma側は、上記4コンポーネント（`Table`を除く）をMain Component化し、既存の重複箇所をインスタンスに置き換えた。`Table`はFigma側では新規Componentを作らず、`ArticleTable`・Feed Management Modalの該当フレームの色を `Design Tokens` 変数にバインドするだけに留めた（列構成が異なる2つのセクションを1つのComponentインスタンスとして表現する手段がないため）。

再利用されない単位（`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`、ページ全体）はFigma Componentにせず、コードのコンポーネント名に合わせてFrame名を整理した（`Index`→`ArticleListPage` 等）。Figma↔コードの対応関係は `docs/component-spec.md` に記録する。

#### 作業中に発生した不具合と対処

- `Button` の `Label` をComponent Setの共有Component Propertyとして公開したところ、4つのVariant全てのテキストが1つ目のVariantの文言に統一されてしまった（Component Set内でプロパティ名が衝突し共有されたため）。プロパティを削除し、各Variantに固定テキストを直接設定する方式に戻した
- `TextField` の `Has Icon=False` バリアントが、クローン元（モーダルの入力欄。元は親のautolayoutに対して`FILL`サイジングだった）の設定を引き継いだ結果、1px幅に潰れた。`layoutSizingHorizontal/Vertical` を `FIXED` に変更し、元のサイズに `resize()` して修正した
- ページネーションの「次へ」ボタンに、誤って「前へ」と同じ左向き矢印のVariantを適用してしまった。`Variant=Next` を新たに作成し、矢印アイコンのベクターを `relativeTransform` で水平反転して修正した
- Component Set内で一部のVariantにのみ追加のVariantプロパティ（例: `Direction`）を持たせようとしたところ、「Component set has existing errors」で操作が失敗した。Component Set内の全Variantは同じプロパティキーの組み合わせを持つ必要があるため、単一の `Variant` プロパティの値を増やす（`Nav`→`Prev`/`Next`に分割）方式に変更した

## ディレクトリ構成（Atomic Design、2026-06-20 三次改訂）

`ui/` というフラットな置き場ではなく、Atomic Designの階層で再編した。

```
web/frontend/src/components/
  atoms/        Button.tsx, icons.tsx
  molecules/    IconButton.tsx, PageButton.tsx, SelectField.tsx, TextField.tsx, Table.tsx, CategoryBadge.tsx
  organisms/    Header.tsx, SearchFilterBar.tsx, ArticleTable.tsx, Pagination.tsx, Footer.tsx, FeedManagementModal.tsx
  templates/    ArticleListTemplate.tsx（App.tsxからレイアウトJSXのみを抽出）
  pages/        ArticleListPage.tsx（App.tsxから状態管理・データ取得ロジックを抽出）
```

`App.tsx`は削除し、`main.tsx`は`pages/ArticleListPage`を読み込む。

### Button・TextFieldの再設計（atomic design層の誤りの修正）

最初に作成した`ui/Button.tsx`は`icon: 'refresh' | 'list' | 'plus' | 'bookmark'`という閉じたenumを持ち、特定の組み合わせごとに見た目が変わる設計だった。これは実質的に「特定の意味を持った組み合わせ」であり、atomではなくmoleculeの性質を持っていた。本来atomにすべきは「枠線・角丸・背景色だけを持つ最小単位」であり、アイコン・ラベルは呼び出し側が`children`として自由に合成すべきという指摘を受け、以下のように再設計した。

- **`Button`（atom）**: `icon`/`variant`/`spinning`/`badge`propを廃し、`children`でアイコン・ラベルを自由合成する。背景色も「常時filled」に簡略化した（後述のFigma側デザイン変更に合わせたもの）。`active`（amberの配色）は枠の見た目を切り替えるだけの軸であり、コンテンツ合成とは独立した関心事のためatomのpropとして残した
- **`atoms/icons.tsx`**: 元から個別の関数コンポーネント（`RssIcon`等）として分離されていたため、実質的に元からatomの粒度だった。SVGファイルへの分割は、現状11種・107行という規模では本格的な恩恵（ビルド設定の複雑化に対して）が薄いため見送った（11種を大きく超える、またはFigmaからのSVG書き出しを頻繁に取り込む運用になったら再検討する）
- **`IconButton`・`PageButton`（molecule）**: 元からアイコン（atom）をpropまたはchildrenとして受け取る構造だったため、Button atomの再設計に伴う変更は不要だった
- **`TextField`（molecule）**: `hasIcon: boolean`で検索アイコンの表示を固定的に切り替えていたのは汎用性が低いという指摘を受け、`icon?: ReactNode`に変更し任意のアイコンを受け取れるようにした
- **`CategoryBadge`（molecule、新規）**: `ArticleTable.tsx`内で2箇所重複していたカテゴリ配色ロジック（`domain/category.ts`の`categoryStyle`を呼ぶ部分）を切り出した

### セマンティックカラートークンの追加

`text-slate-200`/`text-slate-500`/`text-amber-400`という生のTailwindカラー名には意味づけがなく、用途（主要テキスト/補助テキスト/アクセント）が名前から分からない問題があった。CSS変数ではなくTailwindの`theme.extend`で管理する方針を継続し、以下を追加した。

```js
colors: {
  text: {
    primary: '#e2e8f0',    // 主要テキスト（旧: slate-200）
    secondary: '#64748b',  // 補助テキスト（旧: slate-500）
  },
  accent: {
    default: '#fbbf24',    // アクセント（旧: amber-400）
  },
},
```

ランタイムでのテーマ切り替え要件がない（ダーク1色のみ）ため、CSS変数より素のTailwind theme値の方がIDE補完・utility classとしての一貫性の面で優位と判断した。

### Figma側: Icon atom・Button再構築・全セクションComponent化（2026-06-20 三次改訂）

- **`Icon`（Component Set、`General`ページ）**: 既存のButton/IconButton/TextField/PageButton/SelectFieldに埋め込まれていたベクター（icon/refresh, icon/close等）をそれぞれクローンし、`Name`プロパティ（Rss/Refresh/Bookmark/Search/ChevronDown/ChevronLeft/ChevronRight/List/Plus/Trash/Close）を持つ単一のComponent Setとして再構成した。ゼロから描画するのではなく既存ベクターを複製したため、見た目の再現性が高い
- **`Button`の再構築**: 旧来の4 Variant（`Icon`×`Style`）構成を、単一Componentに統合した。`Icon`をINSTANCE_SWAPプロパティ（`Icon`atomを参照）、`Label`をTEXTプロパティとして公開し、背景は常時`surface/raised`にバインドしたfilledスタイルに統一した。Header3インスタンス・Feed Management Modalの追加ボタンインスタンスを新Componentにスワップ（`instance.swapComponent()`）した上で、`setProperties()`で各ボタンのIcon/Labelを設定し、旧Component Setは削除した
- **`TextField`へのINSTANCE_SWAP追加**: `Has Icon=True`側に直接描画されていた検索アイコンのベクターを、`Icon`atomのインスタンス（デフォルト値Search）に置き換え、INSTANCE_SWAPプロパティとして公開した。検索アイコン固定ではなく任意のアイコンに差し替え可能であることが、Figma上のプロパティとしても明示されるようにした
- **全セクションのMain Component化**: 従来「再利用されないため命名のみ整理する」としていた`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`/`Feed Management Modal`を、その場で`figma.createComponentFromNode()`によりMain Componentに変換した（位置・親子関係は維持したまま、ノードタイプのみFRAME→COMPONENTに変更）。再利用の有無に関わらずコンポーネントとして明示する方針に転換した
- **Figmaページ構成の分割**: 単一の`Page 1`を`General`（atoms/molecules）にリネームし、新規ページ`Frontend`を作成して`ArticleListPage`・`Feed Management Modal`フレーム（および新たにComponent化した6つのorganism）を移動した

#### 作業中に発生した不具合と対処（三次改訂分）

- `figma.createComponentFromNode()`を、すでに`COMPONENT`型であるノード（既存Component Setのバリアントをクローンしたもの）に対して呼び出すと「Cannot create component from node」で失敗した。クローンした時点で既に`COMPONENT`型だったため、変換不要だったことに気付き、そのまま`Button`として使用するよう修正した
- `addComponentProperty('Icon', 'INSTANCE_SWAP', ...)`のデフォルト値に、Iconインスタンスの`id`（インスタンス自身のID）を渡すと「Default value for instance swap component property is invalid」で失敗した。INSTANCE_SWAPのデフォルト値には参照先の**メインComponentのID**を渡す必要があり、修正した
- `instance.componentPropertyReferences = { instanceSwapContent: propKey }`は「Unrecognized key(s) in object: 'instanceSwapContent'」で失敗した。Plugin APIの型定義を確認したところ、INSTANCE_SWAPプロパティの参照キーは`mainComponent`であり、修正した

## 確認方法

- `npx tsc --noEmit && npm run test && npm run build` がすべて成功すること
- 既存画面の見た目（DOM構造・スタイルの見え方）に変化がないことの目視確認は、`CLAUDE.md` の方針通りユーザー自身が行う（Figma側の見た目変化は`get_screenshot`で都度確認した）
