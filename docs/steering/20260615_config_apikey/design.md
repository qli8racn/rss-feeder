# 設計：設定ファイルによる ANTHROPIC_API_KEY 管理

## 設定ファイル

`internal/config/` に配置する（設定の読み込みロジックと設定ファイル自体を同じディレクトリに置く）。

### `internal/config/config.example.yml`（リポジトリ管理）

```yaml
anthropic_api_key: "sk-ant-xxxxxxxx"
```

### `internal/config/config.yml`（Git 管理対象外）

`config.example.yml` をコピーして作成し、実際の API キーを設定する。

```bash
cp internal/config/config.example.yml internal/config/config.yml
```

`.gitignore` に `/internal/config/config.yml` を追加し、誤コミットを防ぐ。

## `internal/config` パッケージ

`github.com/spf13/viper` を使い、`internal/config/config.yml` を読み込んで構造体にマッピングする。

```go
package config

type Config struct {
    AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
}

func Load() (*Config, error) {
    v := viper.New()
    v.SetConfigName("config")
    v.SetConfigType("yml")
    v.AddConfigPath("internal/config")

    cfg := &Config{}
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            return cfg, nil // config.yml が無くてもエラーにしない
        }
        return nil, fmt.Errorf("config.yml の読み込みに失敗しました: %w", err)
    }
    if err := v.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("config.yml の解析に失敗しました: %w", err)
    }
    return cfg, nil
}
```

## `cmd/agent/main.go` での利用

`main()` の冒頭で `config.Load()` を呼び、`AnthropicAPIKey` が設定されていれば
`ANTHROPIC_API_KEY` 環境変数にセットする。Anthropic SDK の `anthropic.NewClient()` は
環境変数 `ANTHROPIC_API_KEY` を自動で読むため、各エージェント側の実装変更は不要。

```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    if cfg.AnthropicAPIKey != "" {
        os.Setenv("ANTHROPIC_API_KEY", cfg.AnthropicAPIKey)
    }
    // ... 既存の DI / cobra セットアップ
}
```

`config.yml` に値が設定されていればそれを優先し、なければ既存どおり
シェルの `ANTHROPIC_API_KEY` 環境変数がそのまま使われる。

## go.mod

`internal/config` が `github.com/spf13/viper` を直接利用するため、`go mod tidy` で
`viper` を直接依存（direct dependency）に変更する。
