package config

import "os"

type Config struct {
	HTTPAddr       string
	DatabaseDSN    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        string
	RebalanceEvery string
}

func Load() Config {
	return Config{
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		DatabaseDSN:    getEnv("DATABASE_DSN", "file:adengine.db?_foreign_keys=on"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getEnv("REDIS_DB", "0"),
		RebalanceEvery: getEnv("REBALANCE_EVERY", "10s"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
