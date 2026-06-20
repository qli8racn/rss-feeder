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
