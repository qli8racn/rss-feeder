# morning

朝のルーティンを1コマンドで実行する。
fetch → enrich → curate の順に実行し、推薦記事を提示してブックマークを促す。

## 手順

1. 新着記事を取得する
   ```bash
   bin/rss-feeder fetch
   ```
   取得件数をユーザーに報告する。

2. 新着記事を enrich する
   ```bash
   bin/rss-agent enrich --limit 20
   ```
   要約・カテゴリの付与が完了したらユーザーに報告する。
   API コストがかかるため、失敗した場合はエラー内容を伝えて手順3に進む。

3. 今日読む記事を AI に推薦してもらう
   ```bash
   bin/rss-agent curate
   ```
   curate の出力をそのままユーザーに表示する。

4. ブックマーク登録を促す
   以下のメッセージを表示する:
   ```
   記事をブックマークする場合は ID を教えてください（例: 123 456）。
   スキップする場合は Enter を押してください。
   ```
   - ID が入力された場合 → 各 ID に対して `bin/rss-feeder bookmark <id>` を実行する
   - 入力がなければ終了する

## 注意

- `bin/rss-feeder fetch` はリポジトリルートから実行する（相対パスで DB を参照するため）
- enrich はバックグラウンドではなく直列で実行し、完了を待ってから curate に進む
- curate の出力は Markdown 形式のため、そのまま表示してよい
- 週に1度 `bin/rss-agent discover` でフィード発掘を勧めるとよい（このスキルの対象外）
