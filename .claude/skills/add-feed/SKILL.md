# add-feed

指定したサイトの RSS フィードを探し出して `bin/rss-feeder add-feed <url>` で DB に登録する。

## 手順

1. ユーザーからサイト URL を受け取る（引数がなければ入力を求める）

2. RSS フィードの URL を探す
   以下の候補 URL に対して `curl -sI` でレスポンスを確認し、
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
   サイトの HTML から `<link rel="alternate" type="application/rss+xml">` を探す場合:
   ```bash
   curl -s {url} | grep -i 'application/rss\|application/atom'
   ```

3. 候補が見つかった場合はユーザーに提示して確認する
   ```
   RSS フィードが見つかりました:
   https://example.com/feed.xml

   DB に登録しますか？ [y/n]
   ```

4. 見つからなかった場合はユーザーに手動入力を求める
   ```
   RSS フィードを自動検出できませんでした。
   フィードの URL を直接入力してください:
   ```

5. ユーザーが承認したら `bin/rss-feeder add-feed <rss-url>` を実行する
   ```bash
   bin/rss-feeder add-feed https://example.com/feed.xml
   ```
   - 「登録しました」と表示されれば成功
   - 「すでに登録済みです」と表示された場合はその旨を伝える

## 注意

- `bin/rss-feeder add-feed` への実行はユーザー確認後にのみ行う
- 重複登録は rss-feeder 側で自動検出される
- 登録後に `bin/rss-feeder list-feeds` で確認できることを案内する
