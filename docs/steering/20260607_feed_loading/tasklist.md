# タスクリスト：RSS リンクの読み込み

## 実装タスク

- [x] `feeds.txt` のパース実装（`internal/driver/file/feeds_reader.go`）
- [x] コメント行（`#` 始まり）・空行スキップ処理
- [x] URL 0件時の警告出力と終了処理
- [x] FeedsReader interface 定義（`internal/adapter/driver/file/feeds_reader.go`）

## テスト

- [ ] ユニットテスト作成（`internal/driver/file/feeds_reader_test.go`）
  - 正常系: 複数 URL の読み込み
  - コメント行・空行のスキップ確認
  - 0件時の挙動確認
