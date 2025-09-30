package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Version     string            `yaml:"version"`
	Environment string            `yaml:"environment"`
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Auth        AuthConfig        `yaml:"auth"`
	CORS        CORSConfig        `yaml:"cors"`
	Logging     LoggingConfig     `yaml:"logging"`
	Sync        *SyncConfig       `yaml:"sync,omitempty"`
}

type SyncConfig struct {
	Enabled           *bool         `yaml:"enabled,omitempty"`
	BatchSize         *int          `yaml:"batch_size,omitempty"`
	WorkerCount       *int          `yaml:"worker_count,omitempty"`
	RetryLimit        *int          `yaml:"retry_limit,omitempty"`
	RetryDelay        *string       `yaml:"retry_delay,omitempty"`
	SyncInterval      *string       `yaml:"sync_interval,omitempty"`
	ConflictStrategy  *string       `yaml:"conflict_strategy,omitempty"`
	EventTTL          *string       `yaml:"event_ttl,omitempty"`
	CleanupInterval   *string       `yaml:"cleanup_interval,omitempty"`
	MaxConcurrentSync *int          `yaml:"max_concurrent_sync,omitempty"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type DatabaseConfig struct {
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
	Neo4j     Neo4jConfig     `yaml:"neo4j"`
	Redis     RedisConfig     `yaml:"redis"`
}

type PostgreSQLConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Database        string        `yaml:"database"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	SSLMode         string        `yaml:"ssl_mode"`
}

type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type AuthConfig struct {
	SecretKey        string        `yaml:"secret_key"`
	AccessTokenTTL   time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `yaml:"refresh_token_ttl"`
	PasswordMinLength int         `yaml:"password_min_length"`
	PasswordMaxLength int         `yaml:"password_max_length"`
	MaxLoginAttempts int          `yaml:"max_login_attempts"`
	LockoutDuration  time.Duration `yaml:"lockout_duration"`
}

type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/conx-cmdb")

	// Set defaults
	setDefaults()

	// Enable environment variable override
	viper.AutomaticEnv()

	// Read configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults and environment variables
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Version and Environment
	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("environment", "development")

	// Server
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")

	// PostgreSQL
	viper.SetDefault("database.postgresql.host", "localhost")
	viper.SetDefault("database.postgresql.port", 5432)
	viper.SetDefault("database.postgresql.database", "cmdb")
	viper.SetDefault("database.postgresql.username", "cmdb_user")
	viper.SetDefault("database.postgresql.password", "dev_password")
	viper.SetDefault("database.postgresql.max_open_conns", 25)
	viper.SetDefault("database.postgresql.max_idle_conns", 5)
	viper.SetDefault("database.postgresql.conn_max_lifetime", "5m")
	viper.SetDefault("database.postgresql.conn_max_idle_time", "5m")
	viper.SetDefault("database.postgresql.ssl_mode", "disable")

	// Neo4j
	viper.SetDefault("database.neo4j.uri", "bolt://localhost:7687")
	viper.SetDefault("database.neo4j.username", "neo4j")
	viper.SetDefault("database.neo4j.password", "neo4j_password")
	viper.SetDefault("database.neo4j.database", "neo4j")

	// Redis
	viper.SetDefault("database.redis.host", "localhost")
	viper.SetDefault("database.redis.port", 6379)
	viper.SetDefault("database.redis.password", "")
	viper.SetDefault("database.redis.db", 0)

	// Authentication
	viper.SetDefault("auth.secret_key", "your-secret-key-change-in-production")
	viper.SetDefault("auth.access_token_ttl", "15m")
	viper.SetDefault("auth.refresh_token_ttl", "7d")
	viper.SetDefault("auth.password_min_length", 8)
	viper.SetDefault("auth.password_max_length", 128)
	viper.SetDefault("auth.max_login_attempts", 5)
	viper.SetDefault("auth.lockout_duration", "15m")

	// CORS
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"})
	viper.SetDefault("cors.exposed_headers", []string{"Link"})
	viper.SetDefault("cors.allow_credentials", false)
	viper.SetDefault("cors.max_age", 300)

	// Logging
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
}

func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate database configuration
	if config.Database.PostgreSQL.Port <= 0 || config.Database.PostgreSQL.Port > 65535 {
		return fmt.Errorf("invalid PostgreSQL port: %d", config.Database.PostgreSQL.Port)
	}

	if config.Database.PostgreSQL.MaxOpenConns <= 0 {
		return fmt.Errorf("invalid PostgreSQL max open connections: %d", config.Database.PostgreSQL.MaxOpenConns)
	}

	if config.Database.PostgreSQL.MaxIdleConns < 0 {
		return fmt.Errorf("invalid PostgreSQL max idle connections: %d", config.Database.PostgreSQL.MaxIdleConns)
	}

	if config.Database.PostgreSQL.MaxIdleConns > config.Database.PostgreSQL.MaxOpenConns {
		return fmt.Errorf("PostgreSQL max idle connections cannot exceed max open connections")
	}

	// Validate Neo4j configuration
	if config.Database.Neo4j.URI == "" {
		return fmt.Errorf("Neo4j URI cannot be empty")
	}

	// Validate Redis configuration
	if config.Database.Redis.Port <= 0 || config.Database.Redis.Port > 65535 {
		return fmt.Errorf("invalid Redis port: %d", config.Database.Redis.Port)
	}

	if config.Database.Redis.DB < 0 || config.Database.Redis.DB > 15 {
		return fmt.Errorf("invalid Redis DB: %d", config.Database.Redis.DB)
	}

	// Validate authentication configuration
	if config.Auth.SecretKey == "" {
		return fmt.Errorf("auth secret key cannot be empty")
	}

	if len(config.Auth.SecretKey) < 32 {
		return fmt.Errorf("auth secret key must be at least 32 characters long")
	}

	if config.Auth.AccessTokenTTL <= 0 {
		return fmt.Errorf("access token TTL must be positive")
	}

	if config.Auth.RefreshTokenTTL <= 0 {
		return fmt.Errorf("refresh token TTL must be positive")
	}

	if config.Auth.PasswordMinLength < 8 {
		return fmt.Errorf("password minimum length must be at least 8")
	}

	if config.Auth.PasswordMaxLength > 128 {
		return fmt.Errorf("password maximum length cannot exceed 128")
	}

	if config.Auth.PasswordMinLength > config.Auth.PasswordMaxLength {
		return fmt.Errorf("password minimum length cannot exceed maximum length")
	}

	if config.Auth.MaxLoginAttempts <= 0 {
		return fmt.Errorf("max login attempts must be positive")
	}

	if config.Auth.LockoutDuration <= 0 {
		return fmt.Errorf("lockout duration must be positive")
	}

	// Validate CORS configuration
	if len(config.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("at least one allowed origin must be specified")
	}

	if len(config.CORS.AllowedMethods) == 0 {
		return fmt.Errorf("at least one allowed method must be specified")
	}

	if len(config.CORS.AllowedHeaders) == 0 {
		return fmt.Errorf("at least one allowed header must be specified")
	}

	if config.CORS.MaxAge <= 0 {
		return fmt.Errorf("CORS max age must be positive")
	}

	// Validate logging configuration
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validLogFormats[config.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	validLogOutputs := map[string]bool{
		"stdout": true, "stderr": true, "file": true,
	}
	if !validLogOutputs[config.Logging.Output] {
		return fmt.Errorf("invalid log output: %s", config.Logging.Output)
	}

	return nil
}

// GetPostgreSQLConnectionString returns the PostgreSQL connection string
func (c *Config) GetPostgreSQLConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.PostgreSQL.Host,
		c.Database.PostgreSQL.Port,
		c.Database.PostgreSQL.Username,
		c.Database.PostgreSQL.Password,
		c.Database.PostgreSQL.Database,
		c.Database.PostgreSQL.SSLMode,
	)
}

// GetRedisConnectionString returns the Redis connection string
func (c *Config) GetRedisConnectionString() string {
	if c.Database.Redis.Password != "" {
		return fmt.Sprintf("redis://%s@%s:%d/%d",
			c.Database.Redis.Password,
			c.Database.Redis.Host,
			c.Database.Redis.Port,
			c.Database.Redis.DB,
		)
	}
	return fmt.Sprintf("redis://%s:%d/%d",
		c.Database.Redis.Host,
		c.Database.Redis.Port,
		c.Database.Redis.DB,
	)
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsTesting returns true if the environment is testing
func (c *Config) IsTesting() bool {
	return c.Environment == "testing"
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := getEnv(key, ""); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := getEnv(key, ""); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := getEnv(key, ""); value != "" {
		if durationValue, err := time.ParseDuration(value); err == nil {
			return durationValue
		}
	}
	return defaultValue
}
