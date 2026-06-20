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

## 受け入れ条件

- `tailwind.config.js` の `theme.extend` に、現在使われている値から抽出したデザイントークンを定義する（色・フォントサイズ・ボーダー等。命名・分類の方針は `design.md` で定める）
- 既存コンポーネント（`Header` / `FeedManagementModal` / `SearchFilterBar` / `ArticleTable` / `Pagination` / `Footer` / `icons`）のハードコードされた任意値クラスを、定義したトークンを参照するユーティリティクラスに置き換える
- 重複しているスタイルパターンを共通コンポーネントに抽出する: `IconButton`（アイコンのみのボタン）・`Button`（アイコン+ラベルボタン）・`TextField`（入力欄）・`SelectField`（セレクト）・`PageButton`（ページネーションボタン）・`Table`（テーブルの外枠・ヘッダ背景・行区切り線のシェル）
- `FeedManagementModal` のフィード一覧をdivによる擬似テーブルから実際の `<table>` マークアップに修正する
- Figma側で、上記で抽出した共通コンポーネント（`Table`を除く）をMain Component化し、Design Tokens変数にバインドした上で、既存の重複箇所をインスタンスに置き換える
- 再利用されない単位（`Header`/`SearchFilterBar`/`ArticleTable`/`Pagination`/`Footer`、ページ全体）は、コードのコンポーネント名と対応する分かりやすい名前のFigma Frameに整理する（`Index`→`ArticleListPage`等）
- Figma Component ↔ コードコンポーネントの対応関係（命名・props・既知の差異）を `docs/component-spec.md` に記録する
- 整理後も `npx tsc --noEmit` / `npm run test` / `npm run build` が通り、画面の見た目（DOM構造・スタイルの見え方）が変化しないこと（フォントサイズについては0.5px未満の近似値統合は許容する。`design.md` 参照）

## スコープ外

- 新規機能・画面の追加
- レイアウト・配色そのものの変更（既存の見た目を保ったままトークン化するのみ）
- `ArticleTable`・`FeedManagementModal`の列構成そのものの統合（外枠・ヘッダ背景・行区切り線のスタイルのみ`Table`シェルとして共通化し、列の内容は個別実装を維持する）
- **Figma Code Connect**（チームライブラリへの公開・Organization/Enterpriseプランが前提のため、個人開発である本プロジェクトでは対象外）
- React コンポーネントテストの新規導入（既存方針通り未導入を維持する。`docs/web-ui-spec.md` 参照）
