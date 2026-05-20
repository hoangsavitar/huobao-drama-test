package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Storage   StorageConfig   `mapstructure:"storage"`
	AI        AIConfig        `mapstructure:"ai"`
	Narrative NarrativeConfig `mapstructure:"narrative"`
}

// NarrativeConfig: Huobao Text AI + embedded prompts only (NarrativePackageService). No HTTP delegate.
// FallbackStub: if true, use local template DAG when LLM fails; if false, return error.
type NarrativeConfig struct {
	FallbackStub bool `mapstructure:"fallback_stub"`
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Version  string `mapstructure:"version"`
	Debug    bool   `mapstructure:"debug"`
	Language string `mapstructure:"language"` // zh 或 en
}

type ServerConfig struct {
	Port         int      `mapstructure:"port"`
	Host         string   `mapstructure:"host"`
	CORSOrigins  []string `mapstructure:"cors_origins"`
	ReadTimeout  int      `mapstructure:"read_timeout"`
	WriteTimeout int      `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Type     string `mapstructure:"type"` // sqlite, mysql
	Path     string `mapstructure:"path"` // SQLite数据库文件路径
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
	MaxIdle  int    `mapstructure:"max_idle"`
	MaxOpen  int    `mapstructure:"max_open"`
}

type StorageConfig struct {
	Type      string `mapstructure:"type"`       // local, minio
	LocalPath string `mapstructure:"local_path"` // 本地存储路径
	BaseURL   string `mapstructure:"base_url"`   // 访问URL前缀
}

type AIConfig struct {
	DefaultTextProvider  string `mapstructure:"default_text_provider"`
	DefaultImageProvider string `mapstructure:"default_image_provider"`
	DefaultVideoProvider string `mapstructure:"default_video_provider"`
}

func LoadConfig() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	_ = viper.BindEnv("narrative.fallback_stub", "NARRATIVE_FALLBACK_STUB")

	configFile := "./configs/config.yaml"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "./configs/config.example.yaml"
		if _, err2 := os.Stat(configFile); os.IsNotExist(err2) {
			return nil, fmt.Errorf("missing configs/config.yaml — run: copy configs\\config.example.yaml configs\\config.yaml (%w)", err)
		}
	}
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config %s: %w", configFile, err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if fb := strings.TrimSpace(os.Getenv("NARRATIVE_FALLBACK_STUB")); fb != "" {
		config.Narrative.FallbackStub = strings.EqualFold(fb, "true") || fb == "1"
	}

	return &config, nil
}

func (c *DatabaseConfig) DSN() string {
	if c.Type == "sqlite" {
		return c.Path
	}
	// MySQL DSN
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Charset,
	)
}
