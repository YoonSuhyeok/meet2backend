package config

import (
	"net"
	"net/url"
	"os"
)

type Config struct {
	AppPort     string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	DatabaseURL string
}

func Load() Config {
	return Config{
		AppPort:     envOrDefault("APP_PORT", "8080"),
		DBHost:      envOrDefault("DB_HOST", "localhost"),
		DBPort:      envOrDefault("DB_PORT", "25432"),
		DBUser:      envOrDefault("DB_USER", "postgres"),
		DBPassword:  envOrDefault("DB_PASSWORD", "postgres"),
		DBName:      envOrDefault("DB_NAME", "meetback"),
		DBSSLMode:   envOrDefault("DB_SSLMODE", "disable"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

func (c Config) PostgresDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}

	query := url.Values{}
	query.Set("sslmode", c.DBSSLMode)

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     net.JoinHostPort(c.DBHost, c.DBPort),
		Path:     c.DBName,
		RawQuery: query.Encode(),
	}).String()
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
