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
	AccrualHost     string
	AccrualPort     string
}

func ParseFlags() *Config {
	cfg := &Config{}

	RunAddr := flag.String("a", ":8080", "address and port to run server")
	logLevel := flag.String("l", "INFO", "log level")
	dataBaseDSN := flag.String("d", "", "connect to postgres")
	cookieSecretKey := flag.String("s", "default-secret-key", "you secret key for cookie")
	accrualHost := flag.String("accrual-host", "http://localhost", "accrual system host")
	accrualPort := flag.String("accrual-port", "8081", "accrual system port")

	flag.Parse()

	cfg.RunAddr = getEnvOrDefault("RUN_ADDRESS", *RunAddr)
	cfg.LogLevel = strings.ToUpper(getEnvOrDefault("LOG_LEVEL", *logLevel))
	cfg.DataBaseDSN = getEnvOrDefault("DATABASE_URI", *dataBaseDSN)
	cfg.CookieSecretKey = getEnvOrDefault("COOKIE_SECRET_KEY", *cookieSecretKey)
	cfg.AccrualHost = getEnvOrDefault("ACCRUAL_SYSTEM_ADDRESS", *accrualHost)
	cfg.AccrualPort = getEnvOrDefault("ACCRUAL_SYSTEM_PORT", *accrualPort)

	return cfg
}

func getEnvOrDefault(envName, defaultValue string) string {
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	return defaultValue
}
