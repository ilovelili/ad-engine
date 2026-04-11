package config

import "os"

type Config struct {
	HTTPAddr            string
	DatabaseDSN         string
	RedisAddr           string
	RedisPassword       string
	RedisDB             string
	RebalanceEvery      string
	ConnectionSecret    string
	MetaGraphBaseURL    string
	MetaGraphAPIVersion string
	MetaAppID           string
	MetaAppSecret       string
	MetaRedirectURI     string
	MetaOAuthScopes     string
}

func Load() Config {
	return Config{
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DatabaseDSN:         getEnv("DATABASE_DSN", "file:adengine.db?_foreign_keys=on"),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnv("REDIS_DB", "0"),
		RebalanceEvery:      getEnv("REBALANCE_EVERY", "10s"),
		ConnectionSecret:    getEnv("CONNECTION_SECRET", "dev-only-32-byte-connection-secret"),
		MetaGraphBaseURL:    getEnv("META_GRAPH_BASE_URL", "https://graph.facebook.com"),
		MetaGraphAPIVersion: getEnv("META_GRAPH_API_VERSION", "v22.0"),
		MetaAppID:           getEnv("META_APP_ID", ""),
		MetaAppSecret:       getEnv("META_APP_SECRET", ""),
		MetaRedirectURI:     getEnv("META_REDIRECT_URI", "https://adsone.ngrok.io/api/v1/oauth/meta/callback"),
		MetaOAuthScopes:     getEnv("META_OAUTH_SCOPES", "ads_management,business_management"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
