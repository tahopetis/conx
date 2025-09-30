package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"connect/internal/config"
)

// RedisClient wraps the Redis client with additional functionality
type RedisClient struct {
	client   *redis.Client
	config   *config.RedisConfig
	logger   *logrus.Logger
	enabled  bool
}

// NewRedisClient creates a new Redis client instance
func NewRedisClient(cfg *config.RedisConfig, logger *logrus.Logger) (*RedisClient, error) {
	if !cfg.Enabled {
		logger.Info("Redis is disabled, creating mock client")
		return &RedisClient{
			config:  cfg,
			logger:  logger,
			enabled: false,
		}, nil
	}

	// Create Redis options
	opts := &redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"db":       cfg.DB,
		"poolSize": cfg.PoolSize,
	}).Info("Successfully connected to Redis")

	return &RedisClient{
		client:  client,
		config:  cfg,
		logger:  logger,
		enabled: true,
	}, nil
}

// IsEnabled returns whether Redis is enabled
func (r *RedisClient) IsEnabled() bool {
	return r.enabled
}

// Get retrieves a value from Redis by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if !r.enabled {
		return "", fmt.Errorf("Redis is disabled")
	}

	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return result, nil
}

// Set stores a value in Redis with the default TTL
func (r *RedisClient) Set(ctx context.Context, key, value string) error {
	return r.SetWithTTL(ctx, key, value, r.config.TTL)
}

// SetWithTTL stores a value in Redis with a specific TTL
func (r *RedisClient) SetWithTTL(ctx context.Context, key, value string, ttl time.Duration) error {
	if !r.enabled {
		return fmt.Errorf("Redis is disabled")
	}

	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key from Redis
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	if !r.enabled {
		return fmt.Errorf("Redis is disabled")
	}

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists in Redis
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if !r.enabled {
		return false, fmt.Errorf("Redis is disabled")
	}

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}

	return result > 0, nil
}

// GetJSON retrieves a JSON value from Redis and unmarshals it
func (r *RedisClient) GetJSON(ctx context.Context, key string, target interface{}) error {
	if !r.enabled {
		return fmt.Errorf("Redis is disabled")
	}

	value, err := r.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(value), target)
}

// SetJSON marshals a value to JSON and stores it in Redis
func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}) error {
	return r.SetJSONWithTTL(ctx, key, value, r.config.TTL)
}

// SetJSONWithTTL marshals a value to JSON and stores it in Redis with a specific TTL
func (r *RedisClient) SetJSONWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !r.enabled {
		return fmt.Errorf("Redis is disabled")
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
	}

	return r.SetWithTTL(ctx, key, string(jsonData), ttl)
}

// Increment increments the numeric value of a key
func (r *RedisClient) Increment(ctx context.Context, key string) (int64, error) {
	if !r.enabled {
		return 0, fmt.Errorf("Redis is disabled")
	}

	result, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}

	return result, nil
}

// Decrement decrements the numeric value of a key
func (r *RedisClient) Decrement(ctx context.Context, key string) (int64, error) {
	if !r.enabled {
		return 0, fmt.Errorf("Redis is disabled")
	}

	result, err := r.client.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement key %s: %w", key, err)
	}

	return result, nil
}

// Expire sets the expiration time for a key
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if !r.enabled {
		return fmt.Errorf("Redis is disabled")
	}

	err := r.client.Expire(ctx, key, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration for key %s: %w", key, err)
	}

	return nil
}

// TTL returns the remaining time to live for a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !r.enabled {
		return 0, fmt.Errorf("Redis is disabled")
	}

	result, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	return result, nil
}

// Keys returns all keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	if !r.enabled {
		return nil, fmt.Errorf("Redis is disabled")
	}

	result, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys with pattern %s: %w", pattern, err)
	}

	return result, nil
}

// FlushDB deletes all keys in the current database
func (r *RedisClient) FlushDB(ctx context.Context) error {
	if !r.enabled {
		return fmt.Errorf("Redis is disabled")
	}

	err := r.client.FlushDB(ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to flush database: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if !r.enabled {
		return nil
	}

	err := r.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	r.logger.Info("Redis connection closed")
	return nil
}

// Health checks the Redis connection health
func (r *RedisClient) Health(ctx context.Context) error {
	if !r.enabled {
		return nil
	}

	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// GetStats returns Redis statistics
func (r *RedisClient) GetStats(ctx context.Context) (map[string]string, error) {
	if !r.enabled {
		return nil, fmt.Errorf("Redis is disabled")
	}

	info, err := r.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	return redis.ParseInfo(info)
}

// CacheService provides high-level caching functionality
type CacheService struct {
	redis   *RedisClient
	prefix  string
	logger  *logrus.Logger
}

// NewCacheService creates a new cache service instance
func NewCacheService(redis *RedisClient, prefix string, logger *logrus.Logger) *CacheService {
	return &CacheService{
		redis:  redis,
		prefix: prefix,
		logger: logger,
	}
}

// Get retrieves a value from cache
func (c *CacheService) Get(ctx context.Context, key string) (string, error) {
	cacheKey := c.buildKey(key)
	return c.redis.Get(ctx, cacheKey)
}

// Set stores a value in cache
func (c *CacheService) Set(ctx context.Context, key, value string) error {
	cacheKey := c.buildKey(key)
	return c.redis.Set(ctx, cacheKey, value)
}

// SetWithTTL stores a value in cache with specific TTL
func (c *CacheService) SetWithTTL(ctx context.Context, key, value string, ttl time.Duration) error {
	cacheKey := c.buildKey(key)
	return c.redis.SetWithTTL(ctx, cacheKey, value, ttl)
}

// Delete removes a value from cache
func (c *CacheService) Delete(ctx context.Context, key string) error {
	cacheKey := c.buildKey(key)
	return c.redis.Delete(ctx, cacheKey)
}

// GetJSON retrieves a JSON value from cache
func (c *CacheService) GetJSON(ctx context.Context, key string, target interface{}) error {
	cacheKey := c.buildKey(key)
	return c.redis.GetJSON(ctx, cacheKey, target)
}

// SetJSON stores a JSON value in cache
func (c *CacheService) SetJSON(ctx context.Context, key string, value interface{}) error {
	cacheKey := c.buildKey(key)
	return c.redis.SetJSON(ctx, cacheKey, value)
}

// SetJSONWithTTL stores a JSON value in cache with specific TTL
func (c *CacheService) SetJSONWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	cacheKey := c.buildKey(key)
	return c.redis.SetJSONWithTTL(ctx, cacheKey, value, ttl)
}

// ClearByPattern removes all cache entries matching a pattern
func (c *CacheService) ClearByPattern(ctx context.Context, pattern string) error {
	cachePattern := c.buildKey(pattern)
	keys, err := c.redis.Keys(ctx, cachePattern)
	if err != nil {
		return fmt.Errorf("failed to get cache keys with pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	err = c.redis.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache keys: %w", err)
	}

	c.logger.WithField("pattern", pattern).WithField("keys_deleted", len(keys)).Debug("Cleared cache entries by pattern")
	return nil
}

// buildKey constructs the full cache key with prefix
func (c *CacheService) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", c.prefix, key)
}
