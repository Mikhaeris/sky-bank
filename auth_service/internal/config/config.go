package config

import (
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server struct {
		Grpc struct {
			Addr string `yaml:"addr"`
		} `yaml:"grpc"`
	} `yaml:"server"`
	Storage StorageConfig `yaml:"database"`
	Jwt     struct {
		PrivKeyPath    string        `yaml:"priv_key_path"`
		AccessTokenTtl time.Duration `yaml:"access_token_ttl"`
	} `yaml:"jwt"`
}

type StorageConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

const configPath = "config.yml"

var (
	once   sync.Once
	config *Config
)

func GetConfig(logger *slog.Logger) *Config {
	var once sync.Once

	config := &Config{}

	once.Do(func() {
		logger.Info("read application config")
		err := cleanenv.ReadConfig(configPath, config)
		if err != nil {
			logger.Error(
				"can't read config",
				"path", configPath,
				"error", err,
			)
			os.Exit(1)
		}
	})

	return config
}
