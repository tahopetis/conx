package config

import (
	"fmt"
	"strconv"
	"time"
)

// RedisConfig holds the configuration for Redis connections
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	IdleTimeout  time.Duration
	TTL          time.Duration
	Enabled      bool
}

// LoadRedisConfig loads Redis configuration from environment variables
func LoadRedisConfig() (*RedisConfig, error) {
	config := &RedisConfig{
		Host:         getEnv("REDIS_HOST", "localhost"),
		Port:         getEnvAsInt("REDIS_PORT", 6379),
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           getEnvAsInt("REDIS_DB", 0),
		PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 2),
		MaxRetries:   getEnvAsInt("REDIS_MAX_RETRIES", 3),
		TTL:          getEnvAsDuration("REDIS_TTL", 24*time.Hour),
		Enabled:      getEnvAsBool("REDIS_ENABLED", true),
	}

	// Parse timeout durations
	var err error
	config.DialTimeout = getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second)
	config.ReadTimeout = getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second)
	config.WriteTimeout = getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second)
	config.PoolTimeout = getEnvAsDuration("REDIS_POOL_TIMEOUT", 4*time.Second)
	config.IdleTimeout = getEnvAsDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute)

	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis timeout configuration: %w", err)
	}

	// Validate configuration
	if config.PoolSize <= 0 {
		return nil, fmt.Errorf("REDIS_POOL_SIZE must be greater than 0")
	}
	if config.MinIdleConns < 0 || config.MinIdleConns > config.PoolSize {
		return nil, fmt.Errorf("REDIS_MIN_IDLE_CONNS must be between 0 and REDIS_POOL_SIZE")
	}
	if config.MaxRetries < 0 {
		return nil, fmt.Errorf("REDIS_MAX_RETRIES must be greater than or equal to 0")
	}

	return config, nil
}

// GetRedisAddr returns the Redis connection address
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetRedisURL returns the Redis connection URL
func (c *RedisConfig) GetRedisURL() string {
	if c.Password != "" {
		return fmt.Sprintf("redis://%s:%s@%s:%d/%d", c.Password, c.Password, c.Host, c.Port, c.DB)
	}
	return fmt.Sprintf("redis://%s:%d/%d", c.Host, c.Port, c.DB)
}

// Validate validates the Redis configuration
func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("REDIS_PORT must be between 1 and 65535")
	}
	if c.DB < 0 || c.DB > 15 {
		return fmt.Errorf("REDIS_DB must be between 0 and 15")
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("REDIS_POOL_SIZE must be greater than 0")
	}
	if c.MinIdleConns < 0 || c.MinIdleConns > c.PoolSize {
		return fmt.Errorf("REDIS_MIN_IDLE_CONNS must be between 0 and REDIS_POOL_SIZE")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("REDIS_MAX_RETRIES must be greater than or equal to 0")
	}
	if c.DialTimeout <= 0 {
		return fmt.Errorf("REDIS_DIAL_TIMEOUT must be greater than 0")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("REDIS_READ_TIMEOUT must be greater than 0")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("REDIS_WRITE_TIMEOUT must be greater than 0")
	}
	if c.PoolTimeout <= 0 {
		return fmt.Errorf("REDIS_POOL_TIMEOUT must be greater than 0")
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("REDIS_IDLE_TIMEOUT must be greater than 0")
	}
	if c.TTL <= 0 {
		return fmt.Errorf("REDIS_TTL must be greater than 0")
	}
	return nil
}

// String returns a string representation of the Redis configuration
func (c *RedisConfig) String() string {
	return fmt.Sprintf(
		"RedisConfig{Host: %s, Port: %d, DB: %d, PoolSize: %d, MinIdleConns: %d, Enabled: %v}",
		c.Host, c.Port, c.DB, c.PoolSize, c.MinIdleConns, c.Enabled,
	)
}

// Helper functions for environment variable parsing
func getEnvAsInt(key string, defaultValue int) int {
	if value := getEnv(key, ""); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
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
