# 設計：フィード管理 Web UI

## 処理フロー

```
GET /api/feeds
  └─ web.ListFeedsHandler(listFeedsUC)
       └─ usecase/list_feeds.go（既存・変更なし）
            └─ feedrepo.Repository.ListAll()

POST /api/feeds  body: { "feed_url": string }
  └─ web.AddFeedHandler(addFeedUC)
       └─ usecase/add_feed.go（戻り値を *domain.Feed, error に変更）
            ├─ feedrepo.Repository.Register()
            │    └─ RowsAffected() == 0 → ErrAlreadyExists
            └─ feedrepo.Repository.FindByURL()  # 登録結果を取得して返す

DELETE /api/feeds/{id}
  └─ web.RemoveFeedHandler(removeFeedUC)
       └─ usecase/remove_feed.go（既存・変更なし）
            └─ feedrepo.Repository.Remove()
                 └─ RowsAffected() == 0 → ErrNotFound
```

`feedrepo.Repository.FindByURL` は既存メソッドであり、driver 実装の追加は不要。

## AddFeedUsecase の変更

CLI 版（`add-feed` コマンド）は登録結果を表示しないため、現在は `Execute(ctx, url) error` で十分だった。
Web 版は作成されたフィード（ID・タイトル等）をレスポンスとして返す必要があるため、戻り値を `(*domain.Feed, error)` に変更する。

```go
// internal/usecase/add_feed.go
func (uc *AddFeedUsecase) Execute(ctx context.Context, url string) (*domain.Feed, error) {
	if err := uc.feedRepo.Register(ctx, url); err != nil {
		if errors.Is(err, feedrepo.ErrAlreadyExists) {
			return nil, fmt.Errorf("すでに登録済みです %s: %w", url, err)
		}
		return nil, err
	}
	feed, err := uc.feedRepo.FindByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	if feed == nil {
		// Register 成功直後なので通常到達しないが、FindByURL は未存在時に (nil, nil) を返す実装
		// (internal/driver/readerdb/feed/feed.go) のため、nil ポインタを Web ハンドラに渡さないよう防御する。
		return nil, fmt.Errorf("フィード登録直後の取得に失敗しました %s", url)
	}
	return feed, nil
}
```

`internal/adapter/handler/cli/add_feed.go` は戻り値の `*domain.Feed` を無視する（既存の出力フォーマットは変更しない）。

## Web ハンドラ

新規ファイル `internal/adapter/handler/web/feed.go`。既存の `web/article.go` の構成（`writeJSON` / `writeJSONError` の利用、`chi.URLParam` での id 取得）に倣う。

```go
func ListFeedsHandler(uc *usecase.ListFeedsUsecase) http.HandlerFunc
func AddFeedHandler(uc *usecase.AddFeedUsecase) http.HandlerFunc
func RemoveFeedHandler(uc *usecase.RemoveFeedUsecase) http.HandlerFunc
```

- `AddFeedHandler`: リクエストボディを `openapi.AddFeedRequest` にデコード。`FeedURL` が空なら 400。
  `usecase.AddFeedUsecase.Execute` が `feedrepo.ErrAlreadyExists` を返したら 409、それ以外のエラーは 500。成功時は 201 + `openapi.Feed`。
- `RemoveFeedHandler`: `chi.URLParam(r, "id")` を `strconv.ParseInt` で変換できなければ 400。
  `usecase.RemoveFeedUsecase.Execute` が `feedrepo.ErrNotFound` を返したら 404、それ以外のエラーは 500。成功時は 204（ボディなし）。
- `ListFeedsHandler`: エラー時は 500。成功時は 200 + `openapi.Feed` の配列（`/api/categories` と同様、ページネーションなし）。

既存の `POST /api/articles/{id}/bookmark` ・ `POST /api/articles/fetch` はいずれも 200 を返しているが、
それらは既存リソースの更新（トグル・再取得）であり、`POST /api/feeds` は新規リソース作成にあたるため、
本フェーズでは意図的に RESTful な `201 Created` を採用する（既存 POST 系との不統一は意図した差分）。

## ルーティング・DI（`cmd/web/main.go`）

```go
addFeedUC    := usecase.NewAddFeedUsecase(do.MustInvoke[feedrepo.Repository](i))
listFeedsUC  := usecase.NewListFeedsUsecase(do.MustInvoke[feedrepo.Repository](i))
removeFeedUC := usecase.NewRemoveFeedUsecase(do.MustInvoke[feedrepo.Repository](i))

r.Get("/api/feeds", web.ListFeedsHandler(listFeedsUC))
r.Post("/api/feeds", web.AddFeedHandler(addFeedUC))
r.Delete("/api/feeds/{id}", web.RemoveFeedHandler(removeFeedUC))
```

`cors.Options.AllowedMethods` に `"DELETE"` を追加する（現状 `GET`, `POST` のみ）。

## OpenAPI（`docs/openapi.yaml`）

`Feed` スキーマ・`AddFeedRequest` スキーマ・`Conflict` レスポンスを追加し、3 パスを追加する。

```yaml
  /api/feeds:
    get:
      operationId: listFeeds
      responses:
        "200": # array of Feed
    post:
      operationId: addFeed
      requestBody: # AddFeedRequest
      responses:
        "201": # Feed
        "400": BadRequest
        "409": Conflict

  /api/feeds/{id}:
    delete:
      operationId: removeFeed
      parameters: [id]
      responses:
        "204": # no content
        "400": BadRequest
        "404": NotFound

components:
  schemas:
    Feed:
      properties: [id, feed_url, title, last_fetched, created_at]
    AddFeedRequest:
      required: [feed_url]
      properties: [feed_url]
  responses:
    Conflict:
      schema: Error
```

追加後、既存の生成コマンドで型を再生成する。

```bash
go generate ./internal/adapter/handler/web/openapi/...
cd web/frontend && npm run generate:api
```

## フロントエンド

### ディレクトリ構成への追加

```
web/frontend/src/
  api.ts                          fetchFeeds / addFeed / removeFeed を追加
  types.ts                        Feed 型を追加
  components/
    FeedManagementModal.tsx       新規：一覧・追加フォーム・削除ボタンを持つモーダル
    Header.tsx                   「フィード管理」ボタンを追加
    icons.tsx                     SettingsIcon・TrashIcon を追加
  App.tsx                         モーダルの開閉状態を管理
```

`docs/web-ui-spec.md` のアーキテクチャ規約（`components → usecase → domain`、ルーティングなし）を維持する。
モーダルの開閉は `App.tsx` の `useState` で管理し、`FeedManagementModal` は受け取った props のみに依存する（`SearchFilterBar` 等と同じ構成）。

### `api.ts` 追加分

```ts
export interface Feed {
  id: number
  feed_url: string
  title: string
  last_fetched: string | null
  created_at: string
}

export async function fetchFeeds(): Promise<Feed[]>
export async function addFeed(feedUrl: string): Promise<Feed>   // 409 は呼び出し元でハンドリングできるようエラーメッセージに状態を含める
export async function removeFeed(id: number): Promise<void>
```

エラーレスポンスは既存の `toggleBookmark` 等と同様、`res.ok` チェック + `Error` throw のパターンに揃える。
409（重複）・404（未存在）はステータスコードに応じたメッセージ文言を出し分ける。

### `FeedManagementModal.tsx`

- props: `open: boolean`, `onClose: () => void`
- 内部状態: `feeds`, `loading`, `error`, `newUrl`, `submitting`
- `open` が `true` になったタイミングで `fetchFeeds()` を呼ぶ（モーダルを開くたびに最新化）
- 追加フォーム: テキスト入力 + 「追加」ボタン。送信中は disabled。成功したら入力をクリアして一覧を再取得
- 各行に削除ボタン。クリックで `window.confirm` による確認後 `removeFeed(id)` を呼び、成功したら一覧から除去
- 0 件の場合はガイドメッセージ（`docs/steering/20260608_feed_management/requirements.md` の CLI 版ガイドメッセージに合わせた文言）を表示
- スタイリングは既存のダークテーマ（`#0d1117` 系背景・`slate` 系テキスト）に合わせる

### `Header.tsx` 変更

既存の「最新フィードを取得」「ブックマーク」ボタンと並べて「フィード管理」ボタンを追加し、`onOpenFeedManagement` props を呼ぶ。

### `App.tsx` 変更

```ts
const [feedModalOpen, setFeedModalOpen] = useState(false)
```

を追加し、`<Header onOpenFeedManagement={() => setFeedModalOpen(true)} ... />` と
`<FeedManagementModal open={feedModalOpen} onClose={() => setFeedModalOpen(false)} />` を配置する。
記事一覧の再取得（`reloadToken` 等）とは連動させない（要件どおりスコープ外）。

### テスト

`docs/design.md` のテスト戦略に倣い、`components/` 単体（`FeedManagementModal` 含む）は新規テスト対象としない
（既存の `ArticleTable` 等と同様、コンポーネントテストは未導入の方針を維持）。
ロジックを `domain/` / `usecase/` に切り出す箇所が生まれた場合のみ、その関数にユニットテストを追加する
（本フェーズでは API 呼び出しと状態管理のみで純粋なロジック切り出しは発生しない想定）。

バックエンド側は既存方針通り、`usecase/add_feed_test.go` を戻り値変更（`*domain.Feed, error`）に追随させる。
`internal/adapter/handler/web/` はテスト対象外（既存の `article.go` 等と同様、handler 層は対象外という既存方針を維持）。
