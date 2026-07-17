// Package config contains a configuration loader for environment variables
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	GinMode     string
	DBHost      string
	DBPort      string
	Domain      string
	DBUser      string
	DBPassword  string
	DBName      string
	RedisHost   string
	RedisPort   string
	DatabaseURL string
	RedisURL    string
}

func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", ""),
		Domain:      getEnv("DOMAIN", ""),
		GinMode:     getEnv("GIN_MODE", ""),
		DBHost:      getEnv("DB_HOST", ""),
		DBPort:      getEnv("DB_PORT", ""),
		DBUser:      getEnv("DB_USER", ""),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", ""),
		RedisHost:   getEnv("REDIS_HOST", ""),
		RedisPort:   getEnv("REDIS_PORT", ""),
		RedisURL:    getEnv("REDIS_URL", ""),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key string, defaultValue string) string {
	err := godotenv.Load()
	if err != nil {
		return defaultValue
	}
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func ConstructDBConnectionString(cfg Config) string {
	if cfg.DatabaseURL != "" {
		return cfg.DatabaseURL
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=require",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBName,
	)
}

func ConstructRedisConnectionString(cfg Config) string {
	if cfg.RedisURL != "" {
		return cfg.RedisURL
	}
	return fmt.Sprintf(
		"%s:%s",
		cfg.RedisHost,
		cfg.RedisPort,
	)
}
