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
	Client struct {
		Grpc struct {
			Addr string `yaml:"addr"`
		} `yaml:"grpc"`
	} `yaml:"client"`
	Storage StorageConfig `yaml:"database"`
	Jwt     struct {
		PrivKeyPath    string        `yaml:"priv_key_path"`
		AccessTokenTtl time.Duration `yaml:"access_token_ttl"`
	} `yaml:"jwt"`
	OtpSecretPath string `yaml:"otp_secret_path"`
}

type StorageConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

const configPath = "config.yaml"

var (
	once   sync.Once
	config *Config
)

func GetConfig(logger *slog.Logger) *Config {
	once.Do(func() {
		logger.Info("read application config")

		config = &Config{}

		err := cleanenv.ReadConfig(configPath, config)
		if err != nil {
			help, _ := cleanenv.GetDescription(config, nil)
			logger.Info(help)
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
