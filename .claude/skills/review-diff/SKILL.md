# review-diff

ステージングの差分をレビューし、アーキテクチャ違反・Typo・DRY 違反を報告する。

## 手順

1. ステージングが空でないか確認する
   ```bash
   git diff --staged --stat
   ```
   空の場合は「ステージングに差分がありません」と伝えて終了する。

2. 差分の全内容を取得する
   ```bash
   git diff --staged
   ```

3. 以下の観点で差分をレビューする

   ### アーキテクチャ違反
   このプロジェクトの依存方向ルール（`docs/design.md` 参照）:
   ```
   adapter(handler) → usecase → domain
   driver           → adapter(interface)
   ```
   - `domain` が `usecase`・`driver`・`adapter` をインポートしていないか
   - `usecase` が `driver`・`adapter/handler` をインポートしていないか
   - `adapter/handler` が `driver` を直接インポートしていないか
   - `driver` が別の `driver` を直接インポートしていないか

   ### Typo
   - 変数名・関数名・コメント・文字列リテラルに明らかなスペルミスがないか
   - 日本語コメント・出力文字列に誤字・脱字がないか

   ### DRY 違反
   - 同じロジックや定数が複数箇所に重複していないか
   - 差分内で追加された似た処理が既存コードと重複していないか（重複が疑われる場合はファイル名と行番号を示す）

   ### エラーハンドリング漏れ
   - `err` を `_` で握り潰していないか
   - エラーをラップせず素通しで返していないか（`fmt.Errorf("...: %w", err)` の欠如）
   - エラーチェック自体が省略されていないか

   ### テスト漏れ
   - 新規追加した公開関数・メソッドに対応するテストが差分内に含まれているか
   - `docs/design.md` のテスト戦略（domain/usecase/driver の3層）に沿っているか
   - テストが存在しない場合は「テストなし」と指摘する（削除・リファクタのみの差分は除く）

   ### Go 命名規則違反
   - 頭字語が正しく大文字化されているか（`userId` → `userID`、`xmlParser` → `xmlParser` は OK、`Url` → `URL`）
   - 公開すべきでない型・関数が exported になっていないか
   - インターフェース名が `-er` suffix に沿っているか（`Reader`・`Writer` など）

   ### デバッグコードの残留
   - `fmt.Println` / `log.Println` が本番コードに残っていないか
   - `TODO` / `FIXME` コメントが意図せず混入していないか

   ### マジックナンバー・マジック文字列
   - 定数化すべきリテラルが直書きされていないか（タイムアウト秒数、繰り返し使われる文字列など）
   - SQL クエリの断片が複数箇所に散在していないか

   ### インターフェース肥大化
   - `adapter/` のインターフェースに不必要なメソッドが追加されていないか
   - 実装側でしか使わないメソッドがインターフェースに漏れていないか

4. 結果を以下の形式で報告する

   ```
   ## レビュー結果

   ### アーキテクチャ
   - [問題あり] internal/driver/xxx/xxx.go: usecase パッケージをインポートしている（依存逆転違反）
   - [OK] 問題なし

   ### Typo
   - [問題あり] internal/domain/article.go:42: `bookrmark` → `bookmark`
   - [OK] 問題なし

   ### DRY
   - [問題あり] internal/usecase/fetch.go:15 と internal/usecase/list.go:20 に同じエラーハンドリング
   - [OK] 問題なし

   ### エラーハンドリング
   - [問題あり] internal/driver/readerdb/article/article.go:30: err を _ で無視している
   - [OK] 問題なし

   ### テスト
   - [問題あり] internal/usecase/fetch.go に新規関数 `Execute` が追加されたがテストなし
   - [OK] 問題なし

   ### 命名規則
   - [問題あり] internal/domain/article.go:10: `articleId` → `articleID`
   - [OK] 問題なし

   ### デバッグコード
   - [問題あり] internal/driver/anthropic/summarize.go:55: `fmt.Println` が残っている
   - [OK] 問題なし

   ### マジックナンバー
   - [問題あり] internal/driver/anthropic/loop.go:20: タイムアウト `30` が直書きされている
   - [OK] 問題なし

   ### インターフェース
   - [問題あり] internal/adapter/repository/article/article.go: 実装専用の `rawQuery` がインターフェースに含まれている
   - [OK] 問題なし

   ---
   総評: X 件の問題が見つかりました。（または「問題は見つかりませんでした。」）
   ```

## 注意

- 差分にないファイルを推測で指摘しない
- 問題が見つかった場合はファイル名と行番号を必ず示す
- 軽微な style の好みではなく、明確な問題のみ指摘する
