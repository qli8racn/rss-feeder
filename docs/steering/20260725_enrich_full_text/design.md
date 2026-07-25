# 設計：enrich 本文取得改善

## 全体フロー（変更後）

```
enrichAgent.Run
  └─ チャンクに分割（既存）
       └─ goroutine（並列、既存）
            ├─ [NEW] fetchFullContent(ctx, chunk)   ← URL 並列フェッチ・HTML テキスト抽出
            └─ summarizeAndCategorizeWithSplitRetry  ← 入力が RSS snippet → フルテキストに変わる
```

## 変更ファイル

```
internal/driver/anthropic/enrich.go  … enrichAgent に Fetcher 追加、fetchFullContent 実装
go.mod                                … goquery を indirect → direct に昇格
```

## enrichAgent の変更

```go
type enrichAgent struct {
    client  messageCreator
    repo    articlerepo.Repository
    fetcher adapterhtmlfetch.Fetcher  // NEW: HTML 取得用
}
```

`NewEnrichAgent` で `do.MustInvoke[adapterhtmlfetch.Fetcher](i)` を追加。
`cmd/agent/main.go` の DI 登録はすでに `driverhtmlfetch.NewFetcher` が存在するため変更不要。

## fetchFullContent

```
func (a *enrichAgent) fetchFullContent(ctx, articles) []domain.Article
```

- 記事ごとに goroutine でフェッチ（semaphore で `maxFetchConcurrency = 5` に制限）
- フェッチ成功 → `extractTextFromHTML(html)` でテキスト化 → `article.Content` を上書き
- フェッチ失敗・テキスト抽出結果が空 → 元の `Content` を保持（フォールバック）

goroutine ごとに `result[i]` に書き込む（インデックス重複なし、`result` 自体は mutex 不要）。
進捗ログ用の `successCount` カウントには `sync.Mutex` を使用する。

## extractTextFromHTML

`github.com/PuerkitoBio/goquery` を使用（go.sum に既存、direct 昇格のみ）。

```
1. script / style / nav / header / footer / aside を削除
2. article → main → [role=main] → .content → .post-content → body の順で最初に見つかった要素のテキストを返す
3. 空白を正規化して返す
```

## 定数追加

```go
const maxFetchConcurrency = 5
```

## goroutine 内の変更（Run メソッド）

```go
go func() {
    defer wg.Done()
    defer func() { <-sem }()
    enrichedChunk := a.fetchFullContent(ctx, chunk)           // NEW
    results, usage, err := a.summarizeAndCategorizeWithSplitRetry(ctx, enrichedChunk)
    outcomes[i] = chunkOutcome{results: results, usage: usage, err: err}
}()
```

## コスト・パフォーマンス考慮

- URL フェッチは無料だがレイテンシを増加させる（15秒タイムアウト × ceil(n/5) ラウンド）
  - 40件バッチ: 最大 8 ラウンド × 15秒 = 最大 120秒（ほぼフェッチ成功なら数秒）
- フルテキストを 2000文字に切り詰めるため LLM 入力トークンはほぼ変わらない
  - ただし RSS snippet < 2000文字の記事では入力トークンが増加する可能性がある
- Haiku モデルのままでコスト増加は軽微
