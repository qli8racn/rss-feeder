# add-feed

指定したURLのRSSフィードを探し出して `feeds.txt` に追加する。

## 手順

1. ユーザーからURLを受け取る（引数がなければ入力を求める）

2. RSSフィードのURLを探す
   以下の候補URLに対して `curl -sI` でレスポンスを確認し、
   `Content-Type` が `application/rss+xml` / `application/atom+xml` / `text/xml` を含むか確認する:
   ```
   {url}/feed
   {url}/feed.xml
   {url}/rss
   {url}/rss.xml
   {url}/atom.xml
   {url}/index.xml
   {url}/feed/atom
   ```
   サイトのHTMLから `<link rel="alternate" type="application/rss+xml">` を探す場合は:
   ```bash
   curl -s {url} | grep -i 'application/rss\|application/atom'
   ```

3. 候補が見つかった場合はユーザーに提示して確認する
   ```
   🔍 RSSフィードが見つかりました:
   https://example.com/feed.xml

   feeds.txt に追加しますか？ [y/n]
   ```

4. 見つからなかった場合はユーザーに手動入力を求める
   ```
   ⚠️ RSSフィードを自動検出できませんでした。
   フィードのURLを直接入力してください:
   ```

5. ユーザーが承認したら `feeds.txt` に追記する
   - 重複チェック: すでに同じURLが登録されていないか確認する
   - 末尾に改行を保ったまま追記する
   - 追記後に「追加しました: {url}」と報告する

## feeds.txt のフォーマット

```
# RSS フィード URL リスト（1行1URL、# はコメント、空行はスキップ）
https://example.com/feed
```

## 注意

- `feeds.txt` への書き込みはユーザー確認後にのみ行う
- 重複するURLは追加しない
