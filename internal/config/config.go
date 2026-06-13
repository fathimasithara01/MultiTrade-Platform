package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DBConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
}

type AppConfig struct {
	Env  string `mapstructure:"APP_ENV"`
	Port string `mapstructure:"PORT"`
}

type DBConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASS"`
	Name     string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASS"`
}

type KafkaConfig struct {
	Brokers string `mapstructure:"KAFKA_BROKERS"`
}

type JWTConfig struct {
	Secret        string        `mapstructure:"JWT_SECRET"`
	AccessExpiry  time.Duration `mapstructure:"JWT_ACCESS_EXPIRY"`
	RefreshExpiry time.Duration `mapstructure:"JWT_REFRESH_EXPIRY"`
}

func LoadConfig() (*Config, error) {
	// Find .env file by walking up parent directories
	configFile := ".env"
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(configFile); err == nil {
			viper.SetConfigFile(configFile)
			break
		}
		configFile = "../" + configFile
	}
	viper.AutomaticEnv()

	// Allows nested configurations to be read properly via environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Default configurations
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASS", "postgrespassword")
	viper.SetDefault("DB_NAME", "tradecore")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("KAFKA_BROKERS", "localhost:9092")
	viper.SetDefault("JWT_SECRET", "supersecretjwtkeyforportfolio")
	viper.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRY", "168h")

	// Read config file if it exists, otherwise fall back to environment variables
	if err := viper.ReadInConfig(); err != nil {
		// Only return error if the config file was found but failed to read
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read .env file: %w", err)
		}
	}

	var cfg Config

	// Unmarshal configs from environment and files directly into flat structures
	// (or map manually to handle structure grouping cleanly with Viper)
	cfg.App.Env = viper.GetString("APP_ENV")
	cfg.App.Port = viper.GetString("PORT")

	cfg.Database.Host = viper.GetString("DB_HOST")
	cfg.Database.Port = viper.GetString("DB_PORT")
	cfg.Database.User = viper.GetString("DB_USER")
	cfg.Database.Password = viper.GetString("DB_PASS")
	cfg.Database.Name = viper.GetString("DB_NAME")
	cfg.Database.SSLMode = viper.GetString("DB_SSLMODE")

	cfg.Redis.Host = viper.GetString("REDIS_HOST")
	cfg.Redis.Port = viper.GetString("REDIS_PORT")
	cfg.Redis.Password = viper.GetString("REDIS_PASS")

	cfg.Kafka.Brokers = viper.GetString("KAFKA_BROKERS")

	cfg.JWT.Secret = viper.GetString("JWT_SECRET")

	// Parse Access and Refresh Expiries
	accessDuration, err := time.ParseDuration(viper.GetString("JWT_ACCESS_EXPIRY"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}
	cfg.JWT.AccessExpiry = accessDuration

	refreshDuration, err := time.ParseDuration(viper.GetString("JWT_REFRESH_EXPIRY"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}
	cfg.JWT.RefreshExpiry = refreshDuration

	return &cfg, nil
}
