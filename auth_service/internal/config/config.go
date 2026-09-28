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
	Kafka   KafkaConfig   `yaml:"kafka"`
	Jwt     struct {
		PrivKeyPath    string        `yaml:"priv_key_path"`
		AccessTokenTtl time.Duration `yaml:"access_token_ttl"`
	} `yaml:"jwt"`
	OtpSecretPath string    `yaml:"otp_secret_path"`
	OtpLimits     OtpLimits `yaml:"otp_limits"`
	OtlpEndpoint  string    `yaml:"otlp_endpoint"`
}

type StorageConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
}

type OtpLimits struct {
	FreeIssues              int           `yaml:"free_issues"`
	InitialCooldown         time.Duration `yaml:"initial_cooldown"`
	MaxCooldown             time.Duration `yaml:"max_cooldown"`
	IssueWindow             time.Duration `yaml:"issue_window"`
	MaxIssues               int           `yaml:"max_issues"`
	MaxAttemptsPerChallenge int           `yaml:"max_attempts_per_challenge"`
	FailureWindow           time.Duration `yaml:"failure_window"`
	MaxFailures             int           `yaml:"max_failures"`
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
		if config.OtpLimits.FreeIssues <= 0 || config.OtpLimits.InitialCooldown <= 0 ||
			config.OtpLimits.MaxCooldown < config.OtpLimits.InitialCooldown ||
			config.OtpLimits.IssueWindow <= 0 || config.OtpLimits.MaxIssues <= config.OtpLimits.FreeIssues ||
			config.OtpLimits.MaxAttemptsPerChallenge <= 0 || config.OtpLimits.FailureWindow <= 0 || config.OtpLimits.MaxFailures <= 0 {
			logger.Error("invalid OTP limit configuration")
			os.Exit(1)
		}
	})

	return config
}
