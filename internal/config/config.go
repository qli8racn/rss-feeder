package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config は internal/config/config.yml から読み込む設定値。
type Config struct {
	AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
}

// Load は internal/config/config.yml を読み込んで Config にマッピングする。
// config.yml が存在しない場合はゼロ値の Config を返す。
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yml")
	v.AddConfigPath("internal/config")

	cfg := &Config{}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return cfg, nil
		}
		return nil, fmt.Errorf("config.yml の読み込みに失敗しました: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config.yml の解析に失敗しました: %w", err)
	}

	return cfg, nil
}

// SetupAnthropicAPIKey は config.yml から AnthropicAPIKey を読み込み、
// 設定されていれば ANTHROPIC_API_KEY 環境変数にセットする。
// cmd/agent ・ cmd/rss-feeder の両方が Anthropic SDK に直接依存し、
// 起動時に同じ初期化処理を必要とするため共通化する。
func SetupAnthropicAPIKey() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if cfg.AnthropicAPIKey == "" {
		return nil
	}
	return os.Setenv("ANTHROPIC_API_KEY", cfg.AnthropicAPIKey)
}
