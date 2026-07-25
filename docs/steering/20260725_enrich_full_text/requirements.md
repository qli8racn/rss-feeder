# フェーズ：enrich 本文取得改善

## 概要

`bin/rss-agent enrich` が記事本文を要約する際、RSS フィードの snippet（`Content` フィールド、多くは冒頭のみ）
ではなく、記事 URL から取得した HTML のフルテキストを使って要約・カテゴリ精度を向上させる。

## 背景・目的

現在の enrich は `domain.Article.Content`（RSS が配信する本文断片）を Haiku に送信するが、
多くのフィードは冒頭 1〜2 段落しか配信しない。結果として:

- 要約が記事の全体像を捉えられない
- カテゴリが表面的な情報のみで判断される
- curate が趣向判定に使う summary の質が低い

記事 URL から HTML を取得して本文テキストを抽出することで、より正確な要約・カテゴリが得られる。

## 受け入れ条件

- `bin/rss-agent enrich` 実行時に、各記事の URL から HTML を取得して本文テキストを抽出する
- HTML 取得に失敗した場合（タイムアウト・4xx/5xx・非 HTML など）は既存の RSS Content にフォールバックする
- バッチ内の URL 取得は並列実行し、最大同時実行数（`maxFetchConcurrency = 5`）で制限する
- LLM に送信する本文は引き続き `maxContentRunes`（2000文字）で切り詰める
- 既存の `--limit`・`--force`・`--feed`・`--batch-size`・`--concurrency` オプションは変更しない

## スコープ外

- HTML から本文を高精度に抽出するためのルール整備（v1 は goquery で article/main/body を順に試す）
- JavaScript レンダリングが必要なページへの対応（静的 HTML のみ）
- URL フェッチのタイムアウトを enrich 専用に調整すること（既存 `htmlfetch` の 15秒をそのまま使う）
- `bin/rss-feeder fetch` 時の本文取得（enrich 時のみ）
