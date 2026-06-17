# Web UI 仕様書（記事一覧画面）

Figma デザイン（RSS-Feeder）を元にした、ブラウザでの記事一覧画面の UI 仕様。
バックエンド API 仕様は `docs/design.md`、フェーズ要件は `docs/steering/20260614_web_view/` を参照。
このドキュメントは画面構成・インタラクション・既存 API との差分をまとめる。

## 参照

- Figma: https://www.figma.com/design/I0n9kvKiHCnN8vlCalkq6H/RSS-Feeder?node-id=0-1
- 実装配置先: `web/static/`（ビルド成果物）。ソースは `web/frontend/`

## 技術スタック

- React + TypeScript（Vite）
- スタイリングは Tailwind CSS（figma-mcp の `get_design_context` が Tailwind ユーティリティクラス付きの JSX を返すため、生成コードをそのまま活かせる）
- ルーティングなし（単一画面）。状態管理は `useState` / `useEffect` で十分とする
- ビルド成果物は `web/static/` に出力し、既存の `cmd/web --static-dir` 配信をそのまま利用する（バックエンド変更不要）

## アーキテクチャ・コード規約

### ディレクトリ構成

```
web/frontend/src/
  api.ts              バックエンド API への HTTP 呼び出し（fetchArticles, searchArticles, fetchCategories, toggleBookmark）
  types.ts            ドメインの型定義（Article, ArticlesResponse, SortField, SortOrder, SORT_FIELDS）
  domain/             副作用を持たない純粋なビジネスロジック・型
    date.ts             formatDate（日時の相対/絶対表示切り替え）
    category.ts          categoryStyle（カテゴリ → 配色マッピング）
    filter.ts             FilterState 型、parseFilterState / buildFilterQuery（クエリ文字列とのパース/構築。window には依存しない）
  usecase/            domain + ブラウザ API 等の外部依存を組み合わせた処理
    syncFilterStateToURL.ts   FilterState を URL へ反映する（window.history.replaceState を呼ぶ唯一の場所）
  components/         UI コンポーネント（React のみに依存し、domain/usecase の関数を呼び出す）
  App.tsx             状態管理・データ取得のオーケストレーション
```

### 依存方向

バックエンド（`docs/design.md` の `adapter(handler) → usecase → domain`）に倣い、フロントエンドも以下の方向を守る。

```
components / App.tsx → usecase → domain
components / App.tsx → domain（直接呼んでもよい。例: ArticleTable が formatDate / categoryStyle を直接呼ぶ）
```

- `domain/` は副作用を持たない（`window` / `fetch` 等のブラウザ API に依存しない）。テストしやすい純粋関数・型のみを置く
- `usecase/` は `domain/` の関数を使い、ブラウザ API 等の外部依存を伴う処理を担う（例: `syncFilterStateToURL` は URL 書き込みという副作用を持つ）
- API 呼び出し（`api.ts`）は現状 `domain` / `usecase` に分類せず独立したモジュールとして扱う。バックエンドの `driver` 相当に切り出すかは将来的な検討課題（`tasklist.md` 参照）

### 状態管理

- ルーティングライブラリは導入しない（単一画面のため）。検索条件は `App.tsx` の `useState` で保持し、`usecase/syncFilterStateToURL` で副作用として URL に同期する
- `history.replaceState` を使い、ブラウザ履歴は積まない（フィルター操作ごとに戻るボタンの履歴が増えるのを避けるため）。`popstate` 相当の URL 変更検知は実装しない。URL 同期はリロード/共有時の状態復元のみを目的とする
- 検索キーワードの生入力・デバウンス処理は `SearchFilterBar` に閉じ込め、確定値のみを親（`App.tsx`）へコールバックで伝播する。`sort` / `order` のように複数コンポーネント（`SearchFilterBar` のドロップダウンと `ArticleTable` のカラムヘッダクリック）から変更される状態は、共通の親（`App.tsx`）が単一の真実の源として保持する

### テスト

- Vitest + jsdom を使用する（`npm run test`）
- `domain/` ・`usecase/` の関数は新規追加時にユニットテストを必須とする
- React コンポーネントの結合テストは現時点では導入しない（必要になった場合に検討）

## 実装の進め方（figma-mcp 活用）

1. `web/frontend/`（Vite + React + TypeScript + Tailwind）の雛形を作成する
2. figma-mcp（`get_design_context` / `figma-generate-design` スキル）で Figma デザインから React コンポーネントの初期コードを生成し、`web/frontend/src/` 配下に配置する
3. 生成されたコードはモックデータ・ハードコードされたテキストを含むため、以下を手作業で配線する
   - API 呼び出し（`GET /api/articles` 等、`category`/`sort`/`order`/`page`/`per_page` 対応含む）
   - ブックマークトグル（`POST /api/articles/{id}/bookmark`）
   - 検索・カテゴリフィルター・ソート・ページネーションの状態管理
   - 日時の相対/絶対表示切り替えロジック
   - レスポンシブ対応・空状態（本書の該当セクション参照。Figma デザインに含まれないため生成コードでは対応されない）

## 画面構成

### 1. ヘッダー

- 左: RSS アイコン + "RSS Feed Viewer" タイトル
- 右: ブックマークボタン（アイコン + "ブックマーク" ラベル + 件数バッジ）
  - クリックでブックマーク済み記事のみの絞り込み表示に切り替える（`mode=bookmarked` 相当のトグル）
  - 件数バッジはブックマーク済み記事の総数

### 2. 検索・フィルターバー

| 要素 | 説明 |
|------|------|
| 検索入力 | 虫眼鏡アイコン付き。placeholder: `タイトル・メディア・サマリーで検索...`。既存 `GET /api/articles/search?q=` に対応 |
| カテゴリドロップダウン | デフォルト表示 "すべてのカテゴリ"。記事に設定された `category` の一覧から選択肢を動的生成 |
| 並び順ドロップダウン | ソート順の切り替え（新しい順 / 古い順 等。具体的な選択肢ラベルはデザイン上未確定のため実装時に決定） |

### 3. 結果件数 / ページ位置表示

- `NN 件`：現在の絞り込み条件にマッチする記事の総数
- `X / Y ページ`：現在のページ番号 / 全ページ数

### 4. 記事テーブル

列構成: `#` | タイトル（ソート可） | メディア（ソート可） | カテゴリ（ソート可） | 日時（ソート可）

各行の要素:

- `#`: ページ内の表示順の連番（記事 ID ではない）
- ブックマークアイコンボタン: クリックで `POST /api/articles/{id}/bookmark` を呼び出してトグル。ブックマーク済みは塗りアイコン（オレンジ系）で強調表示
- タイトル: リンク。クリックで記事 URL を新規タブで開く
- サマリー: タイトル下に1行省略表示（`summary` フィールド）。未設定の記事は非表示
- メディア: `publisher`
- カテゴリバッジ: 色分けされたピル表示。配色は Figma のスクリーンショット観察に基づく参考例であり固定仕様ではない（実装時にデザイントークンとして再定義してよい）

  | カテゴリ例 | 配色傾向 |
  |---|---|
  | Tech | 青系 |
  | AI | 紫系 |
  | Work | ピンク系 |
  | Science | 緑系 |
  | Finance | 黄系 |
  | Design | マゼンタ系 |

  未知のカテゴリ文字列が来た場合は既定色にフォールバックする（`category` は固定マスタを持たず Claude が自由分類するため、新しいカテゴリ文字列が随時増える前提 — `docs/steering/20260614_article_metadata/requirements.md` 参照）

- 日時: 7日以内は相対表示（`2h前` / `18h前` / `1d前` など）、7日超は絶対日付（`2026/06/07`）表示に切り替える

### 5. ページネーション

- 前へ / 次へアイコンボタン + ページ番号ボタン（現在ページをハイライト表示）
- 1ページあたり件数: デザイン上は 25件/ページ（55件で 3ページ: 25, 25, 5）

### 6. フッター

- `NN feeds indexed — last sync YYYY-MM-DD HH:mm JST` 形式のステータステキスト
- 注意: デザイン上の件数（55）は記事件数（"55 件"）と一致しており、文言の "feeds" は「記事」の誤記、または意図的に「フィード経由で取得した記事の総数」を指している可能性がある。実装時に意図を確認し、`総記事件数` を表示する想定で実装する（フィード数を表示する場合は別途要件化が必要）

## ビジュアルスタイル

- ダークテーマ（背景はほぼ黒、GitHub Dark 風の配色）
- ヘッダーは半透明の背景 + 下端に薄いボーダー
- 罫線・区切りは低彩度のスレートカラーを使用

## 必要な API 拡張（`docs/design.md` からの差分）

この画面を実現するには、既存の `GET /api/articles` 系 API に以下の拡張が必要。

| 追加クエリパラメータ | 対象 | 説明 |
|---|---|---|
| `category` | `/api/articles`, `/api/articles/search` | カテゴリで絞り込み |
| `sort` | 同上 | `title` / `publisher` / `category` / `published_at` |
| `order` | 同上 | `asc` / `desc` |
| `page`, `per_page` | 同上 | ページネーション（デフォルト `per_page=25`） |

レスポンスは記事配列に加え、総件数・ページ情報を含むメタ情報が必要。

```json
{
  "articles": [ ... ],
  "total": 55,
  "page": 1,
  "per_page": 25
}
```

カテゴリドロップダウンの選択肢生成のため、新規エンドポイント `GET /api/categories`（DISTINCT な `category` 一覧を返す）が必要。

ブックマーク件数表示は既存 `GET /api/articles?mode=bookmarked` のレスポンス件数（`total`）を利用すれば追加エンドポイントは不要。

## レスポンシブ対応（モバイル幅）

Figma デザインはデスクトップ幅のみのため、以下はデザイン非対応領域の推定方針（実装時に再確認すること）。

- ブレークポイント目安: 768px 未満をモバイル幅として扱う
- ヘッダー: タイトルとブックマークボタンは維持。ブックマークボタンはラベルを省略しアイコン+件数のみに縮小してよい
- 検索・フィルターバー: 検索入力を全幅で1段目、カテゴリ・並び順ドロップダウンを2段目に折り返して縦積みにする
- テーブル: 横スクロールさせず、1記事1カードのリスト表示に切り替える（カード内にタイトル・サマリー・メディア・カテゴリバッジ・日時・ブックマークボタンを縦積み）
- ページネーション: 前へ/次へボタンと現在ページ表示（例: `1 / 3`）のみとし、ページ番号ボタンの個別表示は省略してよい

## 空状態（該当記事が0件の場合）

- 検索・カテゴリ絞り込みの結果が0件の場合、テーブル領域をスケルトン（行の形を模した低彩度のプレースホルダーブロック）で構成し、その中央にメッセージ（例: `該当する記事がありません`）とアイコンを重ねて表示する
- ヘッダー・検索バー・フィルターバーは通常表示を維持する
- ページネーション・結果件数表示（`0 件`）は表示するが、ページ番号ボタンは表示しない
- 初回読み込み中のローディング状態は本仕様の対象外（別途検討）

## スコープ外（このドキュメント）

- フィード管理・記事削除などの UI
- 初回読み込み中のローディング状態の UI
- サムネイル画像の表示（本画面のデザインには含まれていない）
