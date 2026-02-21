package config

import "os"

type Config struct {
	DatabaseURL    string
	RedisURL       string
	JaegerEndpoint string
	Port           string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	return &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		JaegerEndpoint: os.Getenv("JAEGER_ENDPOINT"),
		Port:           port,
	}
}
