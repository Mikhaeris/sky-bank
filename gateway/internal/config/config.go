package config

import (
	"log/slog"
	"os"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Rest struct {
		Addr string `yaml:"addr"`
	} `yaml:"rest"`
	Auth struct {
		Addr string `yaml:"addr"`
	} `yaml:"auth"`
	Customer struct {
		Addr string `yaml:"addr"`
	} `yaml:"customer"`
	Jwt struct {
		PubKeyPath string `yaml:"pub_key_path"`
	} `yaml:"jwt"`
	RateLimiter struct {
		Enabled      bool    `yaml:"enabled"`
		PublicRPS    float64 `yaml:"public_rps"`
		PublicBurst  int     `yaml:"public_burst"`
		PrivateRPS   float64 `yaml:"private_rps"`
		PrivateBurst int     `yaml:"private_burst"`
	} `yaml:"rate_limiter"`
	OtlpEndpoint string `yaml:"otlp_endpoint"`
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
