package config

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	RunAddr         string
	LogLevel        string
	DataBaseDSN     string
	CookieSecretKey string
}

func ParseFlags() *Config {
	cfg := &Config{}

	RunAddr := flag.String("a", ":8080", "address and port to run server")
	logLevel := flag.String("l", "INFO", "log level")
	dataBaseDSN := flag.String("d", "", "connect to postgres")
	cookieSecretKey := flag.String("s", "default-secret-key", "you secret key for cookie")

	flag.Parse()

	cfg.RunAddr = getEnvOrDefault("SERVER_ADDRESS", *RunAddr)
	cfg.LogLevel = strings.ToUpper(getEnvOrDefault("LOG_LEVEL", *logLevel))
	cfg.DataBaseDSN = getEnvOrDefault("DATABASE_DSN", *dataBaseDSN)
	cfg.CookieSecretKey = getEnvOrDefault("COOKIE_SECRET_KEY", *cookieSecretKey)

	return cfg
}

func getEnvOrDefault(envName, defaultValue string) string {
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	return defaultValue
}
