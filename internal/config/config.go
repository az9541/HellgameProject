package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type DbConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"password"`
	DBName   string `yaml:"dbname" env:"DB_NAME" env-default:"game_saves"`
	SSLMode  string `yaml:"sslmode" env:"DB_SSLMODE" env-default:"disable"`
}

type Config struct {
	// Настройки сервера
	Server struct {
		EnableREST bool   `yaml:"enable_rest" env-default:"true"`
		RESTPort   string `yaml:"port" env:"SERVER_PORT" env-default:"8080"`

		EnableGRPC bool   `yaml:"enable_grpc" env-default:"false"`
		GRPCPort   string `yaml:"grpc_port" env-default:":50051"`
	} `yaml:"server"`

	// Настройки логирования
	Logger struct {
		IsDev bool `yaml:"is_dev" env:"LOGGER_IS_DEV" env-default:"true"`
	} `yaml:"logger"`

	// Настройки симуляции
	Simulation struct {
		Deterministic   bool   `yaml:"deterministic" env:"SIMULATION_DETERMINISTIC" env-default:"false"`
		Seed            int64  `yaml:"seed" env:"SIMULATION_SEED" env-default:"1"`
		UseMockTopology bool   `yaml:"use_mock_topology" env:"SIMULATION_USE_MOCK_TOPOLOGY" env-default:"false"`
		EnableMetrics   bool   `yaml:"enable_metrics" env:"SIMULATION_ENABLE_METRICS" env-default:"false"`
		LoadFromSave    bool   `yaml:"load_from_save" env:"SIMULATION_LOAD_FROM_SAVE" env-default:"false"`
		DBType          string `yaml:"db_type" env:"SIMULATION_DB_TYPE" env-default:"json"` // "json" или "postgres"
	} `yaml:"simulation"`

	// Настройки базы данных postgres
	Database DbConfig `yaml:"database"`
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

func (d DbConfig) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}
