package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Agent    AgentConfig
	RustDesk RustDeskConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Environment  string
}

type DatabaseConfig struct {
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret           string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

type AgentConfig struct {
	HeartbeatInterval time.Duration
	OfflineThreshold  time.Duration
	MetricsInterval   time.Duration
}

type RustDeskConfig struct {
	APIEndpoint string
	APIToken    string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			Environment:  getEnv("SERVER_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnv("DB_PORT", "5432"),
			User:        getEnv("DB_USER", "rmm"),
			Password:    getEnv("DB_PASSWORD", "rmm_password"),
			Name:        getEnv("DB_NAME", "rmm_platform"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			MaxOpen:     getInt("DB_MAX_OPEN", 25),
			MaxIdle:     getInt("DB_MAX_IDLE", 5),
			MaxLifetime: getDuration("DB_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
			AccessTokenTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTokenTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Agent: AgentConfig{
			HeartbeatInterval: getDuration("AGENT_HEARTBEAT_INTERVAL", 30*time.Second),
			OfflineThreshold:  getDuration("AGENT_OFFLINE_THRESHOLD", 2*time.Minute),
			MetricsInterval:   getDuration("AGENT_METRICS_INTERVAL", 30*time.Second),
		},
		RustDesk: RustDeskConfig{
			APIEndpoint: getEnv("RUSTDESK_API_ENDPOINT", ""),
			APIToken:    getEnv("RUSTDESK_API_TOKEN", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
