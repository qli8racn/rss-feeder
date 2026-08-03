package config

import (
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

// LogConfig はログ出力先・フォーマットの設定。
type LogConfig struct {
	Output string `mapstructure:"output"`
	Format string `mapstructure:"format"`
}

// DriverSQLite・DriverSupabase は db.driver に指定可能な値。
const (
	DriverSQLite   = "sqlite"
	DriverSupabase = "supabase"
)

// DBConfig は使用するDBドライバの選択と、その接続設定。
type DBConfig struct {
	Driver   string         `mapstructure:"driver"` // "sqlite"（デフォルト） | "supabase"
	Supabase SupabaseConfig `mapstructure:"supabase"`
}

// SupabaseConfig は Supabase（クラウド上のPostgresプロジェクト）への接続設定。
type SupabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

// IsSupabase は db.driver に "supabase" が指定されているかどうかを返す。
func (c DBConfig) IsSupabase() bool {
	return c.Driver == DriverSupabase
}

// Validate は db.driver に許可された値（未設定・"sqlite"・"supabase"）以外が
// 設定されていないかを確認する。タイプミス等による意図しないフォールバックを
// 起動時に検知するため。
func (c DBConfig) Validate() error {
	switch c.Driver {
	case "", DriverSQLite, DriverSupabase:
		return nil
	default:
		return fmt.Errorf("db.driver に不正な値 %q が指定されています（許可される値: %q, %q）", c.Driver, DriverSQLite, DriverSupabase)
	}
}

// Config は internal/config/config.yml から読み込む設定値。
type Config struct {
	AnthropicAPIKey string    `mapstructure:"anthropic_api_key"`
	Log             LogConfig `mapstructure:"log"`
	DB              DBConfig  `mapstructure:"db"`
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

	if err := cfg.DB.Validate(); err != nil {
		return nil, fmt.Errorf("config.yml の db.driver が不正です: %w", err)
	}

	return cfg, nil
}

// NewProvider は DI コンテナ向けに *Config を提供する provider。
func NewProvider(_ do.Injector) (*Config, error) {
	return Load()
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
