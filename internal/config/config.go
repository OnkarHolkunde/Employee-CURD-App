package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all environment-driven configuration for the app.
type Config struct {
	AppPort string

	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDBName   string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// CacheTTLSeconds controls how long the imported data / list cache lives
	// in Redis before it expires
	CacheTTLSeconds int

	// UploadDir is where uploaded excel files are temporarily stored.
	UploadDir string

	// GinMode is "debug", "release", or "test". Defaults to "release" so
	GinMode string

	// AllowedOrigin configures CORS; "*" allows any origin (fine for local
	// dev/demo, should be locked down to a real frontend domain in prod).
	AllowedOrigin string

	// ShutdownTimeoutSeconds bounds how long the server waits for
	// in-flight requests to finish during a graceful shutdown.
	ShutdownTimeoutSeconds int
}

var Cfg *Config

// Load reads the .env file (if present) and environment variables into Cfg.
// It is safe to call multiple times; subsequent calls are no-ops.
func Load() *Config {
	if Cfg != nil {
		return Cfg
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on real environment variables")
	}

	Cfg = &Config{
		AppPort: getEnv("APP_PORT", "8080"),

		MySQLHost:     getEnv("MYSQL_HOST", "127.0.0.1"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLUser:     getEnv("MYSQL_USER", "root"),
		MySQLPassword: getEnv("MYSQL_PASSWORD", "MyNewPassword@123"),
		MySQLDBName: getEnv("MYSQL_DB", "excel_crud_db"),

		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,

		CacheTTLSeconds: 300, 
		UploadDir:       getEnv("UPLOAD_DIR", "./uploads"),

		GinMode:                getEnv("GIN_MODE", "release"),
		AllowedOrigin:          getEnv("ALLOWED_ORIGIN", "*"),
		ShutdownTimeoutSeconds: 15,
	}

	return Cfg
}

// getEnv reads key from the environment, or returns a fallback value.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
