package database

import (
	"context"
	"fmt"
	"time"

	"github.com/conx/cmdb/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
)

// NewPostgresConnection creates a new PostgreSQL connection pool
func NewPostgresConnection(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	// Configure connection pool
	pool.Config().MaxConns = int32(cfg.Database.MaxOpenConns)
	pool.Config().MinConns = int32(cfg.Database.MaxIdleConns)
	pool.Config().MaxConnLifetime = cfg.Database.ConnMaxLifetime
	pool.Config().HealthCheckPeriod = 1 * time.Minute
	pool.Config().MaxConnIdleTime = 5 * time.Minute

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
		cfg.Neo4j.URL,
		neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = cfg.Neo4j.MaxPoolSize
			c.ConnectionAcquisitionTimeout = 30 * time.Second
			c.SocketTimeout = 30 * time.Second
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
		Addr:         cfg.Redis.URL,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: 5,
		MaxIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
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
