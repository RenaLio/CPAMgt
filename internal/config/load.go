package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func NewConfig(path string) (*Config, error) {
	envConf := os.Getenv("APP_CONF")
	if envConf == "" {
		envConf = path
	}
	fmt.Println("load conf file:", envConf)
	return getConfig(envConf)
}

const (
	Prefix = "APP"
)

func getConfig(configFile string) (*Config, error) {
	if configFile == "" {
		return nil, errors.New("config file is empty")
	}

	v := viper.New()
	v.SetConfigFile(configFile)

	// default values
	v.SetDefault("app.env", "dev")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout_sec", 15)
	v.SetDefault("server.write_timeout_sec", 15)
	v.SetDefault("data.redis.db", 0)

	// env override: APP_DATA_DB_USER_DSN -> data.db.user.dsn
	v.SetEnvPrefix(strings.ToUpper(Prefix))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	return &cfg, nil
}
