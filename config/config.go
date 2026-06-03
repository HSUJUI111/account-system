package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Mysql MysqlConfig `mapstructure:"MySQL"`
	Redis RedisConfig `mapstructure:"Redis"`
	Jwt   JWTConfig   `mapstructure:"Jwt"`
}

type MysqlConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
type JWTConfig struct {
	Secret                string
	AccessTokenTTLMinutes int `mapstructure:"access_token_ttl_minutes"`
	RefreshTokenTTLHours  int `mapstructure:"refresh_token_ttl_hours"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}
