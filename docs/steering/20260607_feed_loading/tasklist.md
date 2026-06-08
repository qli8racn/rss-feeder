# タスクリスト：RSS リンクの読み込み

## 実装タスク

- [x] `feeds.txt` のパース実装（`internal/driver/file/feeds_reader.go`）
- [x] コメント行（`#` 始まり）・空行スキップ処理
- [x] URL 0件時の警告出力と終了処理
- [x] FeedsReader interface 定義（`internal/adapter/driver/file/feeds_reader.go`）

## テスト

- [~] ユニットテスト作成（`internal/driver/file/feeds_reader_test.go`）
  - フェーズ 8 で file ドライバごと削除済みのため不要
