package config

import (
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	// Настройки сервера
	Server struct {
		Port string `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	} `yaml:"server"`

	// Настройки логирования
	Logger struct {
		IsDev bool `yaml:"is_dev" env:"LOGGER_IS_DEV" env-default:"true"`
	} `yaml:"logger"`

	// Настройки симуляции
	Simulation struct {
		Deterministic   bool  `yaml:"deterministic" env:"SIMULATION_DETERMINISTIC" env-default:"false"`
		Seed            int64 `yaml:"seed" env:"SIMULATION_SEED" env-default:"1"`
		UseMockTopology bool  `yaml:"use_mock_topology" env:"SIMULATION_USE_MOCK_TOPOLOGY" env-default:"false"`
		EnableMetrics   bool  `yaml:"enable_metrics" env:"SIMULATION_ENABLE_METRICS" env-default:"false"`
	} `yaml:"simulation"`
}

var instance *Config

func MustLoad(configPath string) *Config {
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if configPath == "" {
		configPath = "config.yaml"
	}

	instance = &Config{}

	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			slog.Error("config file does not exist", "path", configPath)
			os.Exit(1)
		}
		if err := cleanenv.ReadConfig(configPath, instance); err != nil {
			slog.Error("failed to read config file", "path", configPath, "err", err)
			os.Exit(1)
		}
	} else {
		if err := cleanenv.ReadEnv(instance); err != nil {
			slog.Error("failed to read config from environment variables", "err", err)
			os.Exit(1)
		}
	}
	return instance
}
