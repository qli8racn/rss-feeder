# 設計：記事の検索

## 対象コマンド

```bash
rss-feeder search <keyword> [flags]
```

## フラグ

| フラグ | 説明 |
|------|------|
| （なし） | 全記事を対象に検索 |
| `--bookmarked` | お気に入り記事のみを対象にする |

## 出力イメージ

```
ID   タイトル                          公開日時              既読  お気に入り
---  --------------------------------  --------------------  ----  ----------
 60  「そのpromptちょうだい」と…        2026-06-08 02:36      ✓     ★

該当: 1 件
```

```
該当記事が見つかりませんでした
```

## 実装方針

- SQL: `WHERE (title LIKE '%keyword%' OR content LIKE '%keyword%')`
- `--bookmarked` 指定時: `AND bookmarked = 1` を追加
- 結果は `published_at DESC` で並べる
- 表示形式は `list` コマンドと共通のテーブルフォーマットを使用

## 追加ファイル（新規実装）

| ファイル | 役割 |
|---------|------|
| `internal/usecase/search.go` | 検索ロジック |
| `internal/adapter/handler/search.go` | cobra コマンド |
| `internal/adapter/repository/article/article.go` | Search() メソッド追加 |
| `cmd/rss-feeder/main.go` | search コマンド登録 |
