package database

import (
	"context"
	"fmt"
	"time"

	"connect/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
)

// NewPostgresConnection creates a new PostgreSQL connection pool
func NewPostgresConnection(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.GetPostgreSQLConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	// Configure connection pool
	pool.Config().MaxConns = int32(cfg.Database.PostgreSQL.MaxOpenConns)
	pool.Config().MinConns = int32(cfg.Database.PostgreSQL.MaxIdleConns)
	pool.Config().MaxConnLifetime = cfg.Database.PostgreSQL.ConnMaxLifetime
	pool.Config().HealthCheckPeriod = 1 * time.Minute
	pool.Config().MaxConnIdleTime = cfg.Database.PostgreSQL.ConnMaxIdleTime

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return pool, nil
}

// NewNeo4jConnection creates a new Neo4j driver
func NewNeo4jConnection(cfg *config.Config) (neo4j.DriverWithContext, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create Neo4j driver
	driver, err := neo4j.NewDriverWithContext(
		cfg.Database.Neo4j.URI,
		neo4j.BasicAuth(cfg.Database.Neo4j.Username, cfg.Database.Neo4j.Password, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = 50 // Default value
			c.ConnectionAcquisitionTimeout = 30 * time.Second
			c.MaxTransactionRetryTime = 30 * time.Second
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return driver, nil
}

// NewRedisConnection creates a new Redis client
func NewRedisConnection(cfg *config.Config) (*redis.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisConnectionString(),
		PoolSize:     cfg.Database.Redis.PoolSize,
		MinIdleConns: cfg.Database.Redis.MinIdleConns,
		MaxIdleConns: cfg.Database.Redis.PoolSize,
		DialTimeout:  cfg.Database.Redis.DialTimeout,
		ReadTimeout:  cfg.Database.Redis.ReadTimeout,
		WriteTimeout: cfg.Database.Redis.WriteTimeout,
		PoolTimeout:  cfg.Database.Redis.PoolTimeout,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

// HealthCheck represents the health status of a database
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// CheckPostgresHealth checks the health of PostgreSQL
func CheckPostgresHealth(pool *pgxpool.Pool) HealthCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return HealthCheck{
			Name:    "postgres",
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return HealthCheck{
		Name:   "postgres",
		Status: "healthy",
	}
}

// CheckNeo4jHealth checks the health of Neo4j
func CheckNeo4jHealth(driver neo4j.DriverWithContext) HealthCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	_, err := session.Run(ctx, "RETURN 1", nil)
	if err != nil {
		return HealthCheck{
			Name:    "neo4j",
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return HealthCheck{
		Name:   "neo4j",
		Status: "healthy",
	}
}

// CheckRedisHealth checks the health of Redis
func CheckRedisHealth(client *redis.Client) HealthCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return HealthCheck{
			Name:    "redis",
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return HealthCheck{
		Name:   "redis",
		Status: "healthy",
	}
}
