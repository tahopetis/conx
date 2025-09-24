package config

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultConfiguration(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Test default values
	assert.Equal(t, "postgres://cmdb_user:password@localhost:5432/cmdb?sslmode=disable", cfg.Database.URL)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 25, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

	assert.Equal(t, "bolt://localhost:7687", cfg.Neo4j.URL)
	assert.Equal(t, "neo4j", cfg.Neo4j.Username)
	assert.Equal(t, "neo4j", cfg.Neo4j.Password)
	assert.Equal(t, 50, cfg.Neo4j.MaxPoolSize)

	assert.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
	assert.Equal(t, 10, cfg.Redis.PoolSize)

	assert.Equal(t, "your-secret-key-change-in-production", cfg.JWT.Secret)
	assert.Equal(t, 24*time.Hour, cfg.JWT.AccessTokenTTL)
	assert.Equal(t, 168*time.Hour, cfg.JWT.RefreshTokenTTL)

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)

	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "api", cfg.Logging.Service)

	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, "/metrics", cfg.Metrics.Path)

	assert.Equal(t, []string{"http://localhost:3000"}, cfg.CORS.AllowedOrigins)
	assert.Equal(t, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, cfg.CORS.AllowedMethods)
	assert.Equal(t, []string{"Origin", "Content-Type", "Accept", "Authorization"}, cfg.CORS.AllowedHeaders)
	assert.True(t, cfg.CORS.AllowCredentials)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set environment variables
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("NEO4J_URL", "bolt://test:5432")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("SERVER_PORT", "9090")

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Test environment variable overrides
	assert.Equal(t, "postgres://test:test@localhost:5432/test", cfg.Database.URL)
	assert.Equal(t, "bolt://test:5432", cfg.Neo4j.URL)
	assert.Equal(t, "test-secret", cfg.JWT.Secret)
	assert.Equal(t, "9090", cfg.Server.Port)
}

func TestLoad_ConfigFile(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Create a temporary config file
	configContent := `
database:
  url: "postgres://file:file@localhost:5432/file"
  max_open_conns: 50
neo4j:
  url: "bolt://file:7687"
  username: "fileuser"
jwt:
  secret: "file-secret"
server:
  port: "9999"
`

	// Set config content (in a real scenario, this would be a file)
	viper.SetConfigType("yaml")
	require.NoError(t, viper.ReadConfig(strings.NewReader(configContent)))

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Test config file values
	assert.Equal(t, "postgres://file:file@localhost:5432/file", cfg.Database.URL)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, "bolt://file:7687", cfg.Neo4j.URL)
	assert.Equal(t, "fileuser", cfg.Neo4j.Username)
	assert.Equal(t, "file-secret", cfg.JWT.Secret)
	assert.Equal(t, "9999", cfg.Server.Port)
}

func TestValidateConfig_DefaultSecret(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Load configuration with default secret
	cfg, err := Load()
	require.NoError(t, err)

	// Test validation error for default JWT secret
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT secret must be changed from default value")
}

func TestValidateConfig_MissingURLs(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set environment variables to empty values
	t.Setenv("DATABASE_URL", "")
	t.Setenv("NEO4J_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("JWT_SECRET", "valid-secret")

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Test validation errors for missing URLs
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database URL is required")
}

func TestValidateConfig_ValidConfiguration(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set valid environment variables
	t.Setenv("DATABASE_URL", "postgres://valid:valid@localhost:5432/valid")
	t.Setenv("NEO4J_URL", "bolt://valid:7687")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("JWT_SECRET", "valid-secret-key")

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Test validation passes for valid configuration
	err = validateConfig(cfg)
	assert.NoError(t, err)
}

func TestConfig_DurationParsing(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set custom duration values
	t.Setenv("DATABASE_CONN_MAX_LIFETIME", "10m")
	t.Setenv("JWT_ACCESS_TOKEN_TTL", "2h")
	t.Setenv("JWT_REFRESH_TOKEN_TTL", "72h")
	t.Setenv("SERVER_READ_TIMEOUT", "15s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "30s")
	t.Setenv("SERVER_IDLE_TIMEOUT", "120s")

	// Load configuration
	cfg, err := Load()
	require.NoError(t, err)

	// Test duration parsing
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 2*time.Hour, cfg.JWT.AccessTokenTTL)
	assert.Equal(t, 72*time.Hour, cfg.JWT.RefreshTokenTTL)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.Server.IdleTimeout)
}
