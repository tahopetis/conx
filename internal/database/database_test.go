package database

import (
	"context"
	"testing"
	"time"

	"connect/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestNewPostgresConnection_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create PostgreSQL container
	ctx := context.Background()
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts("migrations/001_initial_schema.sql"),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:            connStr,
			MaxOpenConns:   10,
			MaxIdleConns:   5,
			ConnMaxLifetime: 5 * time.Minute,
		},
	}

	// Test connection creation
	pool, err := NewPostgresConnection(cfg)
	require.NoError(t, err)
	defer pool.Close()

	// Test that the pool is properly configured
	assert.NotNil(t, pool)
	assert.Equal(t, int32(10), pool.Config().MaxConns)
	assert.Equal(t, int32(5), pool.Config().MinConns)

	// Test that we can ping the database
	err = pool.Ping(ctx)
	assert.NoError(t, err)
}

func TestNewPostgresConnection_InvalidURL(t *testing.T) {
	// Create test configuration with invalid URL
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:            "postgres://invalid-url",
			MaxOpenConns:   10,
			MaxIdleConns:   5,
			ConnMaxLifetime: 5 * time.Minute,
		},
	}

	// Test connection creation with invalid URL
	pool, err := NewPostgresConnection(cfg)
	assert.Error(t, err)
	assert.Nil(t, pool)
	if pool != nil {
		pool.Close()
	}
}

func TestNewNeo4jConnection_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Neo4j container
	ctx := context.Background()
	neo4jContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Image:        "neo4j:5-community",
		ExposedPorts: []string{"7687/tcp"},
		Env: map[string]string{
			"NEO4J_AUTH": "neo4j/testpass",
		},
	})
	require.NoError(t, err)
	defer neo4jContainer.Terminate(ctx)

	// Get host and port
	host, err := neo4jContainer.Host(ctx)
	require.NoError(t, err)
	port, err := neo4jContainer.MappedPort(ctx, "7687")
	require.NoError(t, err)

	// Create test configuration
	cfg := &config.Config{
		Neo4j: config.Neo4jConfig{
			URL:         "bolt://" + host + ":" + port.Port(),
			Username:    "neo4j",
			Password:    "testpass",
			MaxPoolSize: 10,
		},
	}

	// Test connection creation
	driver, err := NewNeo4jConnection(cfg)
	require.NoError(t, err)
	defer driver.Close()

	// Test that we can verify connectivity
	err = driver.VerifyConnectivity(ctx)
	assert.NoError(t, err)
}

func TestNewNeo4jConnection_InvalidURL(t *testing.T) {
	// Create test configuration with invalid URL
	cfg := &config.Config{
		Neo4j: config.Neo4jConfig{
			URL:         "bolt://invalid-url",
			Username:    "neo4j",
			Password:    "testpass",
			MaxPoolSize: 10,
		},
	}

	// Test connection creation with invalid URL
	driver, err := NewNeo4jConnection(cfg)
	assert.Error(t, err)
	assert.Nil(t, driver)
	if driver != nil {
		driver.Close()
	}
}

func TestNewRedisConnection_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Redis container
	ctx := context.Background()
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)

	// Get connection string
	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Create test configuration
	cfg := &config.Config{
		Redis: config.RedisConfig{
			URL:      connStr,
			PoolSize: 5,
		},
	}

	// Test connection creation
	client, err := NewRedisConnection(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Test that we can ping Redis
	err = client.Ping(ctx).Err()
	assert.NoError(t, err)
}

func TestNewRedisConnection_InvalidURL(t *testing.T) {
	// Create test configuration with invalid URL
	cfg := &config.Config{
		Redis: config.RedisConfig{
			URL:      "redis://invalid-url",
			PoolSize: 5,
		},
	}

	// Test connection creation with invalid URL
	client, err := NewRedisConnection(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	if client != nil {
		client.Close()
	}
}

func TestCheckPostgresHealth_Healthy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create PostgreSQL container
	ctx := context.Background()
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create test configuration and connection
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:            connStr,
			MaxOpenConns:   10,
			MaxIdleConns:   5,
			ConnMaxLifetime: 5 * time.Minute,
		},
	}

	pool, err := NewPostgresConnection(cfg)
	require.NoError(t, err)
	defer pool.Close()

	// Test health check
	health := CheckPostgresHealth(pool)
	assert.Equal(t, "postgres", health.Name)
	assert.Equal(t, "healthy", health.Status)
	assert.Empty(t, health.Message)
}

func TestCheckPostgresHealth_Unhealthy(t *testing.T) {
	// Create a nil pool (simulating unhealthy state)
	var pool *pgxpool.Pool

	// Test health check with nil pool
	health := CheckPostgresHealth(pool)
	assert.Equal(t, "postgres", health.Name)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "nil")
}

func TestCheckNeo4jHealth_Healthy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Neo4j container
	ctx := context.Background()
	neo4jContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Image:        "neo4j:5-community",
		ExposedPorts: []string{"7687/tcp"},
		Env: map[string]string{
			"NEO4J_AUTH": "neo4j/testpass",
		},
	})
	require.NoError(t, err)
	defer neo4jContainer.Terminate(ctx)

	// Get host and port
	host, err := neo4jContainer.Host(ctx)
	require.NoError(t, err)
	port, err := neo4jContainer.MappedPort(ctx, "7687")
	require.NoError(t, err)

	// Create test configuration and connection
	cfg := &config.Config{
		Neo4j: config.Neo4jConfig{
			URL:         "bolt://" + host + ":" + port.Port(),
			Username:    "neo4j",
			Password:    "testpass",
			MaxPoolSize: 10,
		},
	}

	driver, err := NewNeo4jConnection(cfg)
	require.NoError(t, err)
	defer driver.Close()

	// Test health check
	health := CheckNeo4jHealth(driver)
	assert.Equal(t, "neo4j", health.Name)
	assert.Equal(t, "healthy", health.Status)
	assert.Empty(t, health.Message)
}

func TestCheckNeo4jHealth_Unhealthy(t *testing.T) {
	// Create a nil driver (simulating unhealthy state)
	var driver neo4j.DriverWithContext

	// Test health check with nil driver
	health := CheckNeo4jHealth(driver)
	assert.Equal(t, "neo4j", health.Name)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "nil")
}

func TestCheckRedisHealth_Healthy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Redis container
	ctx := context.Background()
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)

	// Get connection string
	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Create test configuration and connection
	cfg := &config.Config{
		Redis: config.RedisConfig{
			URL:      connStr,
			PoolSize: 5,
		},
	}

	client, err := NewRedisConnection(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Test health check
	health := CheckRedisHealth(client)
	assert.Equal(t, "redis", health.Name)
	assert.Equal(t, "healthy", health.Status)
	assert.Empty(t, health.Message)
}

func TestCheckRedisHealth_Unhealthy(t *testing.T) {
	// Create a nil client (simulating unhealthy state)
	var client *redis.Client

	// Test health check with nil client
	health := CheckRedisHealth(client)
	assert.Equal(t, "redis", health.Name)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Contains(t, health.Message, "nil")
}

func TestDatabaseConfig_ConnectionPoolSettings(t *testing.T) {
	// Test that connection pool settings are properly applied
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:            "postgres://test:test@localhost:5432/test",
			MaxOpenConns:   25,
			MaxIdleConns:   15,
			ConnMaxLifetime: 10 * time.Minute,
		},
	}

	// Note: We can't test actual connection without a running database,
	// but we can test that the configuration is properly structured
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 15, cfg.Database.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
}

func TestNeo4jConfig_ConnectionSettings(t *testing.T) {
	// Test that Neo4j connection settings are properly configured
	cfg := &config.Config{
		Neo4j: config.Neo4jConfig{
			URL:         "bolt://localhost:7687",
			Username:    "neo4j",
			Password:    "testpass",
			MaxPoolSize: 50,
		},
	}

	assert.Equal(t, "bolt://localhost:7687", cfg.Neo4j.URL)
	assert.Equal(t, "neo4j", cfg.Neo4j.Username)
	assert.Equal(t, "testpass", cfg.Neo4j.Password)
	assert.Equal(t, 50, cfg.Neo4j.MaxPoolSize)
}

func TestRedisConfig_ConnectionSettings(t *testing.T) {
	// Test that Redis connection settings are properly configured
	cfg := &config.Config{
		Redis: config.RedisConfig{
			URL:      "redis://localhost:6379",
			PoolSize: 20,
		},
	}

	assert.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
	assert.Equal(t, 20, cfg.Redis.PoolSize)
}

func TestHealthCheck_Struct(t *testing.T) {
	// Test HealthCheck struct properties
	health := HealthCheck{
		Name:    "test-db",
		Status:  "healthy",
		Message: "",
	}

	assert.Equal(t, "test-db", health.Name)
	assert.Equal(t, "healthy", health.Status)
	assert.Empty(t, health.Message)

	// Test with error message
	healthWithMessage := HealthCheck{
		Name:    "test-db",
		Status:  "unhealthy",
		Message: "connection failed",
	}

	assert.Equal(t, "test-db", healthWithMessage.Name)
	assert.Equal(t, "unhealthy", healthWithMessage.Status)
	assert.Equal(t, "connection failed", healthWithMessage.Message)
}

func TestMultipleHealthChecks(t *testing.T) {
	// Test multiple health checks in sequence
	// This simulates the health check endpoint behavior

	// Create unhealthy checks (nil connections)
	var pool *pgxpool.Pool
	var driver neo4j.DriverWithContext
	var client *redis.Client

	// Test all health checks
	pgHealth := CheckPostgresHealth(pool)
	neo4jHealth := CheckNeo4jHealth(driver)
	redisHealth := CheckRedisHealth(client)

	// All should be unhealthy
	assert.Equal(t, "unhealthy", pgHealth.Status)
	assert.Equal(t, "unhealthy", neo4jHealth.Status)
	assert.Equal(t, "unhealthy", redisHealth.Status)

	// All should have proper names
	assert.Equal(t, "postgres", pgHealth.Name)
	assert.Equal(t, "neo4j", neo4jHealth.Name)
	assert.Equal(t, "redis", redisHealth.Name)

	// All should have error messages
	assert.NotEmpty(t, pgHealth.Message)
	assert.NotEmpty(t, neo4jHealth.Message)
	assert.NotEmpty(t, redisHealth.Message)
}
