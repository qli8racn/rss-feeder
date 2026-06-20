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
      // 既存の任意値をいくつかの段階に統合する（完全な1:1対応は目指さない）
      'micro': ['10px', { lineHeight: '1.4' }],   // 列ラベル等
      'caption': ['11px', { lineHeight: '1.4' }], // 補助テキスト
      'body': ['13px', { lineHeight: '1.5' }],    // 本文
    },
  },
},
```

`border-slate-400/10` のような既存Tailwind標準カラー+透過指定は、Tailwind標準のまま使い続けて問題ない（独自トークン化の優先度は低い）。今回は **任意値（`bg-[#...]` 等）をトークンに置き換えること** を主目的とし、標準カラーの再命名は対象外とする。

### 共通コンポーネント抽出の候補

- `IconButton`: アイコン1つ+ボーダー枠のボタン（閉じる・削除・既存ヘッダーボタン群）
- `SelectField` / `TextField`: `SearchFilterBar` / `FeedManagementModal` のセレクト・入力欄共通スタイル

実際に抽出するかどうか・範囲は `tasklist.md` で個別に判断する（無理に共通化すると却って複雑になるケースもあるため、2箇所以上で完全一致するスタイルのみを対象にする）。

### Figma側の進め方・Code Connect導入

本フェーズの最終ステップとして、コード側で抽出した共通コンポーネントをFigma側にも反映し、Code Connectで対応づける。

1. コード側の共通コンポーネント抽出（`IconButton` 等）を先に完了させる（トークン定義より後、Figma作業より前）
2. `figma-generate-library` skill を使い、対象範囲（ヘッダーボタン、アイコンボタン、入力欄）をFigma上でMain Component化し、状態違い（disabled等）をVariantsとして設定する。色・フォントサイズは本フェーズで定義したTailwindトークンの値とFigma Variablesを一致させる
3. `figma-code-connect` skill を使い、`web/frontend/src/components/ui/` の各コンポーネントに対応する `.figma.tsx` を作成する。Figmaのprops（Variant名）とReact側のprops（`variant` / `disabled` 等）を対応づける
4. Code Connect Publish後、対応関係が反映されていることを確認する

対象範囲は画面全体（`ArticleTable` 等の複雑なレイアウト）ではなく、本フェーズで抽出する再利用可能な小コンポーネントに限定する（`requirements.md` のスコープ外参照）。

## ディレクトリ構成への変更（案）

```
web/frontend/
  tailwind.config.js     theme.extend にトークン追加
  src/components/
    ui/                   新設（共通コンポーネントの置き場。抽出対象が決まった場合のみ作成）
```

## 確認方法

- `npx tsc --noEmit && npm run test && npm run build` がすべて成功すること
- 既存画面の見た目（DOM構造・スタイルの見え方）に変化がないことの目視確認は、`CLAUDE.md` の方針通りユーザー自身が行う
