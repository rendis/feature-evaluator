package config

import (
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server    ServerConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	Auth      AuthConfig
	OIDC      OIDCConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	Log       LogConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int
	ShutdownTimeout time.Duration
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	DatabaseURL       string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthcheckPeriod time.Duration
	ConnectTimeout    time.Duration
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URI           string
	RedisPassword string
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Disabled         bool
	DevUserEmail     string
	DevUserRole      string
	SecretsMasterKey string
}

// OIDCConfig holds OIDC provider settings.
type OIDCConfig struct {
	Issuer   string
	Audience string
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowOrigins []string
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	EvalPerSecond  int
	AdminPerSecond int
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string
	Format string
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "10s")
	v.SetDefault("POSTGRES_MAX_CONNS", 20)
	v.SetDefault("POSTGRES_MIN_CONNS", 2)
	v.SetDefault("POSTGRES_MAX_CONN_LIFETIME", "30m")
	v.SetDefault("POSTGRES_MAX_CONN_IDLE_TIME", "5m")
	v.SetDefault("POSTGRES_HEALTHCHECK_PERIOD", "1m")
	v.SetDefault("POSTGRES_CONNECT_TIMEOUT", "10s")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("AUTH_DISABLED", false)
	v.SetDefault("DEV_USER_EMAIL", "dev@local.dev")
	v.SetDefault("DEV_USER_ROLE", "owner")
	v.SetDefault("AUTH_SECRETS_MASTER_KEY", "")
	v.SetDefault("OIDC_ISSUER", "")
	v.SetDefault("OIDC_AUDIENCE", "")
	v.SetDefault("CORS_ALLOW_ORIGINS", "http://localhost:5173")
	v.SetDefault("RATE_LIMIT_EVAL", 500)
	v.SetDefault("RATE_LIMIT_ADMIN", 60)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "text")
}

func parseDurationOrDefault(v *viper.Viper, key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(v.GetString(key))
	if err != nil {
		return fallback
	}

	return value
}

func parseShutdownTimeout(v *viper.Viper) time.Duration {
	shutdownTimeout, err := time.ParseDuration(v.GetString("SERVER_SHUTDOWN_TIMEOUT"))
	if err != nil {
		return 10 * time.Second
	}

	return shutdownTimeout
}

func parseOrigins(v *viper.Viper) []string {
	origins := strings.Split(v.GetString("CORS_ALLOW_ORIGINS"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return origins
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)

	cfg := &Config{
		Server: ServerConfig{
			Port:            v.GetInt("SERVER_PORT"),
			ShutdownTimeout: parseShutdownTimeout(v),
		},
		Postgres: PostgresConfig{
			DatabaseURL:       v.GetString("DATABASE_URL"),
			MaxConns:          int32(v.GetInt("POSTGRES_MAX_CONNS")),
			MinConns:          int32(v.GetInt("POSTGRES_MIN_CONNS")),
			MaxConnLifetime:   parseDurationOrDefault(v, "POSTGRES_MAX_CONN_LIFETIME", 30*time.Minute),
			MaxConnIdleTime:   parseDurationOrDefault(v, "POSTGRES_MAX_CONN_IDLE_TIME", 5*time.Minute),
			HealthcheckPeriod: parseDurationOrDefault(v, "POSTGRES_HEALTHCHECK_PERIOD", time.Minute),
			ConnectTimeout:    parseDurationOrDefault(v, "POSTGRES_CONNECT_TIMEOUT", 10*time.Second),
		},
		Redis: RedisConfig{
			URI:           v.GetString("REDIS_URI"),
			RedisPassword: v.GetString("REDIS_PASSWORD"),
		},
		Auth: AuthConfig{
			Disabled:         v.GetBool("AUTH_DISABLED"),
			DevUserEmail:     v.GetString("DEV_USER_EMAIL"),
			DevUserRole:      v.GetString("DEV_USER_ROLE"),
			SecretsMasterKey: v.GetString("AUTH_SECRETS_MASTER_KEY"),
		},
		OIDC: OIDCConfig{
			Issuer:   v.GetString("OIDC_ISSUER"),
			Audience: v.GetString("OIDC_AUDIENCE"),
		},
		CORS: CORSConfig{
			AllowOrigins: parseOrigins(v),
		},
		RateLimit: RateLimitConfig{
			EvalPerSecond:  v.GetInt("RATE_LIMIT_EVAL"),
			AdminPerSecond: v.GetInt("RATE_LIMIT_ADMIN"),
		},
		Log: LogConfig{
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
	}

	slog.Info("configuration loaded",
		"server.port", cfg.Server.Port,
		"postgres.maxConns", cfg.Postgres.MaxConns,
		"auth.disabled", cfg.Auth.Disabled,
		"log.level", cfg.Log.Level,
		"log.format", cfg.Log.Format,
	)

	return cfg, nil
}
