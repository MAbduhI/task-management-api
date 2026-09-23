package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	LogLevel       string
	LogFormat      string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	RedisHost      string
	RedisPassword  string
	RedisDB        int
	JWTSecret      string
	JWTExpiryHours int
	IdempotencyTTL time.Duration
	AdminKey       string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "8080"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "json"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "task_db"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		RedisHost:      getEnv("REDIS_HOST", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getEnvAsInt("REDIS_DB", 0),
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-jwt-key-replace-in-production"),
		JWTExpiryHours: getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		IdempotencyTTL: 24 * time.Hour,
		AdminKey:       getEnv("ADMIN_KEY", "admin-secret-key-123"),
	}

	if cfg.IsProduction() {
		if cfg.JWTSecret == "super-secret-jwt-key-replace-in-production" {
			panic("JWT_SECRET must be set in production")
		}
		if cfg.AdminKey == "admin-secret-key-123" {
			panic("ADMIN_KEY must be set in production")
		}
	}

	return cfg
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
