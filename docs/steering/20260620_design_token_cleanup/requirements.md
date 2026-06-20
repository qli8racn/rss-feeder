# フェーズ：フロントエンド デザイントークン・コンポーネント整理

## 概要

`web/frontend` の初期画面は、Figma Make で作成した1ページ分のデザインのみをコピペし、figma-mcp（`get_design_context`）で読み取って React コンポーネントの初期コードを生成したもの（`docs/web-ui-spec.md` 参照）。
その後の機能追加（フィード管理モーダル等）も、Figma上で既存ノードを目視コピーし、「新規デザイントークンは追加せず既存画面に目視で合わせる」方針で進めてきた（`docs/steering/20260619_feed_management_ui/tasklist.md` 参照）。

この結果、以下の負債が確認できる。

- `tailwind.config.js` の `theme.extend` が空で、色・フォントサイズ等のデザイントークンが一切定義されていない
- `bg-[#0d1117]`（3箇所）・`bg-[#1e2733]`（7箇所）のような任意値カラーが、トークンを介さずコンポーネントに直接ハードコードされている
- `text-slate-500`（29箇所）・`border-slate-400/10`（22箇所）など、Tailwind標準カラーの利用も含め配色がファイルごとに分散しており、一括変更や一貫性チェックの手段がない
- `text-[10px]` 〜 `text-[13px]` のような任意値フォントサイズがコンポーネントごとに微妙に異なる値で散在している
- アイコンボタン（`rounded border border-slate-400/10 p-1.5` 等）のような繰り返しスタイルが、共通コンポーネント化されずコンポーネントごとに重複している
- Figma側もFigma Makeで作った1ページのみで、コンポーネント化（Variants）・Code Connectのマッピングが未整備。新規UI追加の都度、既存ノードの目視複製で対応している

機能追加を続けるほど、見た目の微妙なズレ（同じ意味の色・サイズが画面ごとに違う値になる）とFigma↔コードの対応関係の追跡不可能性が拡大していく。

本フェーズでは、見た目を変えないリファクタリングとして、フロントエンドのデザイントークンとコンポーネント構成を整理する。Figma側も対応する要素をコンポーネント化する。

> **2026-06-20 改訂**: 当初は最終ステップとして Figma Code Connect 導入を予定していたが、Code Connect はFigmaの**チームライブラリへの公開（Publish）**と**Organization/Enterpriseプランを持つチーム**が前提となる機能であり、本プロジェクトは個人開発（所属組織・チームライブラリなし）であるため対象外と判断し、スコープから外した。Figma側のコンポーネント化（Main Component + Variants、Design Tokens変数）はそのまま実施する。

> **2026-06-20 再改訂**: `IconButton`/`SelectField`の抽出だけでは「`Index`画面・`Feed Management Modal`という1セクション単位の構造」が残っており、再利用可能な部品への分解が不十分との指摘を受けた。`Button`（アイコン+ラベルボタン）・`TextField`（入力欄）・`PageButton`（ページネーションボタン）を追加で抽出し、テーブル部分は外枠・ヘッダ背景・行区切り線のみを共通の`Table`シェルとして括る（列構成は`ArticleTable`と`FeedManagementModal`で異なるため完全な汎用化はしない）。再利用されない単位（`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`、ページ全体の`Index`）はFigma Componentにはせず、コードのコンポーネント名に合わせた分かりやすいFrame名に統一する。Figma↔コードの対応関係は新規 `docs/component-spec.md` で管理する。

> **2026-06-20 三次改訂**: 上記対応後、以下の追加指摘を受けて対応範囲を拡張した。
> 1. **Atomic Design再構成**: `Button`をatom（`icon`の閉じたenumを持つ専用ボタン）として実装したのは誤りで、`Button`/`Icon`をatom（汎用的な最小単位）、`IconButton`/`PageButton`をmolecule（atomを組み合わせた部品）とすべきとの指摘を受けた。`Button`から`icon`enumを廃し、`children`でアイコン・ラベルを自由合成する汎用atomに再設計した。コードのディレクトリも`components/{atoms,molecules,organisms,templates,pages}/`の5階層に再編した
> 2. **TextFieldの汎用性不足**: `hasIcon`という真偽値で検索アイコン固定の表示切り替えをしていたのは汎用性が低いとの指摘を受け、`icon?: ReactNode`に変更し任意のアイコンを受け取れるようにした
> 3. **Categoryバッジのコンポーネント化**: `ArticleTable`内で2箇所重複していたカテゴリバッジの配色ロジックを`molecules/CategoryBadge.tsx`として切り出した
> 4. **デザイントークンの追加**: `text.primary`/`text.secondary`/`accent.default`をセマンティックトークンとして`tailwind.config.js`に追加した
> 5. **Figma側のボタン背景色統一**: Header3ボタン（最新フィードを取得/フィード管理/ブックマーク）が枠線のみ（背景transparent）だったのを、追加ボタンと同じfilled（背景色あり）に統一した
> 6. **Figma側の全セクションComponent化**: 再利用されない単位（`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`/`Feed Management Modal`）も含め、Frame名で整理する方針からMain Component化する方針に転換した
> 7. **Figmaページ構成の再編**: 単一ページ構成から、`General`（atoms/molecules）と`Frontend`（organisms以上・ページフレーム）の2ページ構成に分割した
>
> 今回の対応については、ユーザーが差分を明確に確認したいとの意向のため、コードの自動コミットを控えて差分提示のみ行う運用とした。

## 受け入れ条件

- `tailwind.config.js` の `theme.extend` に、現在使われている値から抽出したデザイントークンを定義する（色・フォントサイズ・ボーダー等。命名・分類の方針は `design.md` で定める）
- 既存コンポーネント（`Header` / `FeedManagementModal` / `SearchFilterBar` / `ArticleTable` / `Pagination` / `Footer` / `icons`）のハードコードされた任意値クラスを、定義したトークンを参照するユーティリティクラスに置き換える
- 重複しているスタイルパターンを共通コンポーネントに抽出する: `IconButton`（アイコンのみのボタン）・`Button`（アイコン+ラベルボタン）・`TextField`（入力欄）・`SelectField`（セレクト）・`PageButton`（ページネーションボタン）・`Table`（テーブルの外枠・ヘッダ背景・行区切り線のシェル）
- `FeedManagementModal` のフィード一覧をdivによる擬似テーブルから実際の `<table>` マークアップに修正する
- Figma側で、上記で抽出した共通コンポーネント（`Table`を除く）をMain Component化し、Design Tokens変数にバインドした上で、既存の重複箇所をインスタンスに置き換える
- Figma Component ↔ コードコンポーネントの対応関係（命名・props・既知の差異）を `docs/component-spec.md` に記録する
- 整理後も `npx tsc --noEmit` / `npm run test` / `npm run build` が通り、画面の見た目（DOM構造・スタイルの見え方）が変化しないこと（フォントサイズについては0.5px未満の近似値統合は許容する。`design.md` 参照）

### 三次改訂分の追加受け入れ条件（2026-06-20）

- コード側を `components/{atoms,molecules,organisms,templates,pages}/` のAtomic Design階層に再編する（`Button`/`atoms/icons.tsx`をatom、`IconButton`/`TextField`/`SelectField`/`PageButton`/`Table`/`CategoryBadge`をmolecule、`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`/`FeedManagementModal`をorganism、`ArticleListTemplate`をtemplate、`ArticleListPage`をpageとする）
- `Button`を`icon`の閉じたenumを持つ専用コンポーネントから、`children`でアイコン・ラベルを自由合成する汎用atomに再設計する
- `TextField`の`hasIcon: boolean`を`icon?: ReactNode`に変更し、任意のアイコンを受け取れるようにする
- カテゴリバッジの配色ロジックを`molecules/CategoryBadge.tsx`として切り出す
- `text.primary`/`text.secondary`/`accent.default`をセマンティックトークンとして`tailwind.config.js`に追加する
- Figma側で`Icon` atom（Component Set）を新規作成し、既存コンポーネントに埋め込まれていたベクターを共通化する
- Figma側で`Button`を単一Component（`Icon`をINSTANCE_SWAP、`Label`をTEXTプロパティ）に再構築し、Header/Feed Management Modalの実インスタンスをfilled背景に統一する
- Figma側で`TextField`に`Icon`のINSTANCE_SWAPプロパティを追加する
- Figma側で再利用されない単位（`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`/`Feed Management Modal`）もMain Component化する（従来の「Frame名で整理するのみ」から方針転換）
- Figmaのページ構成を`General`（atoms/molecules）と`Frontend`（organisms以上・ページフレーム）に分割する
- 今回の対応はコードの自動コミットを行わず、差分をユーザーが確認した上で明示的な承認を得てからコミットする

## スコープ外

- 新規機能・画面の追加
- レイアウト・配色そのものの変更（既存の見た目を保ったままトークン化するのみ）
- `ArticleTable`・`FeedManagementModal`の列構成そのものの統合（外枠・ヘッダ背景・行区切り線のスタイルのみ`Table`シェルとして共通化し、列の内容は個別実装を維持する）
- **Figma Code Connect**（チームライブラリへの公開・Organization/Enterpriseプランが前提のため、個人開発である本プロジェクトでは対象外）
- React コンポーネントテストの新規導入（既存方針通り未導入を維持する。`docs/web-ui-spec.md` 参照）

## 今後の検討事項（TODO、本フェーズのスコープ外）

- **カテゴリ配色のDB管理化**: 現状`web/frontend/src/domain/category.ts`にカテゴリ名→配色のマッピングをハードコードしている（2026-06-20時点で15カテゴリ）。`bin/rss-agent enrich`がカテゴリを自由に分類するため、カテゴリの種類は今後も増減しうる。カテゴリごとの配色をDBで管理し、APIから配色（カラーコード）を取得して表示する設計に切り替えることで、データソースを単一に保てる案が提案された。現時点ではカテゴリ数が少なく、ハードコードでの追記コストが低いため見送ったが、カテゴリ種類が増加・流動化した場合は本格的に検討する（新規`categories`テーブル・管理用CLI/APIエンドポイントの設計が必要になる規模の変更のため、別フェーズの`docs/steering`で扱う）
