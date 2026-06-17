---
applyTo: "**/*.go"
---

# Go コード向けルール

- レイヤー構成: `internal/domain` → `internal/usecase` → `internal/driver` の依存方向を守る（逆方向の import は禁止）
- HTTP ハンドラ (`cmd/web`) はビジネスロジックを持たず、usecase 層を呼ぶだけにする
- 新規ロジックは既存の usecase / domain に重複する処理がないか確認する
- 変更後は対象パッケージのテストを実行する:
  ```bash
  go test ./internal/domain/...
  go test ./internal/usecase/...
  go test ./internal/driver/...
  ```
- ビルド確認:
  ```bash
  go build -o bin/web ./cmd/web
  ```
