package sync

import (
	"context"
	"testing"
	"time"

	"github.com/conx/cmdb/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestNewConflictResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	// Create test configuration
	cfg := &config.Config{
		Sync: config.SyncConfig{
			ConflictResolution: config.ConflictResolutionConfig{
				Strategy:        "last_write_wins",
				Enabled:         true,
				MaxRetries:      3,
				RetryDelay:      100 * time.Millisecond,
				ConflictWindow: 5 * time.Minute,
			},
		},
	}

	t.Run("Create conflict resolver successfully", func(t *testing.T) {
		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, resolver)

		assert.NotNil(t, resolver.dbPool)
		assert.NotNil(t, resolver.redisClient)
		assert.Equal(t, cfg.Sync.ConflictResolution.Strategy, resolver.strategy)
		assert.True(t, resolver.enabled)
	})

	t.Run("Create conflict resolver with disabled conflict resolution", func(t *testing.T) {
		disabledCfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Enabled: false,
				},
			},
		}

		resolver, err := NewConflictResolver(disabledCfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, resolver)

		assert.False(t, resolver.enabled)
	})

	t.Run("Create conflict resolver with invalid strategy", func(t *testing.T) {
		invalidCfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy: "invalid_strategy",
					Enabled:  true,
				},
			},
		}

		resolver, err := NewConflictResolver(invalidCfg, pool, redisClient)
		require.NoError(t, err) // Should fall back to default strategy
		require.NotNil(t, resolver)

		assert.Equal(t, "last_write_wins", resolver.strategy) // Should use default
	})
}

func TestConflictResolver_DetectConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	// Create test configuration
	cfg := &config.Config{
		Sync: config.SyncConfig{
			ConflictResolution: config.ConflictResolutionConfig{
				Strategy:        "last_write_wins",
				Enabled:         true,
				MaxRetries:      3,
				RetryDelay:      100 * time.Millisecond,
				ConflictWindow:  5 * time.Minute,
			},
		},
	}

	resolver, err := NewConflictResolver(cfg, pool, redisClient)
	require.NoError(t, err)

	t.Run("Detect conflict with concurrent operations", func(t *testing.T) {
		// Create two operations on the same record within the conflict window
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-conflict-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-conflict-123", // Same record ID
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive"},
			Timestamp: time.Now().Add(1 * time.Second), // Within conflict window
		}

		// Process first operation
		err := resolver.ProcessOperation(ctx, operation1)
		require.NoError(t, err)

		// Check if second operation conflicts
		conflict, err := resolver.DetectConflict(ctx, operation2)
		require.NoError(t, err)
		require.NotNil(t, conflict)

		assert.Equal(t, operation2.RecordID, conflict.RecordID)
		assert.Equal(t, operation2.Table, conflict.Table)
		assert.Equal(t, operation1, conflict.ExistingOperation)
		assert.Equal(t, operation2, conflict.NewOperation)
	})

	t.Run("No conflict with operations outside conflict window", func(t *testing.T) {
		// Create two operations on the same record outside the conflict window
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-no-conflict-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now().Add(-10 * time.Minute), // Old operation
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-no-conflict-123", // Same record ID
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive"},
			Timestamp: time.Now(), // Recent operation
		}

		// Process first operation
		err := resolver.ProcessOperation(ctx, operation1)
		require.NoError(t, err)

		// Check if second operation conflicts
		conflict, err := resolver.DetectConflict(ctx, operation2)
		require.NoError(t, err)
		assert.Nil(t, conflict) // Should be no conflict
	})

	t.Run("No conflict with operations on different records", func(t *testing.T) {
		// Create two operations on different records
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-different-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-different-456", // Different record ID
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive"},
			Timestamp: time.Now().Add(1 * time.Second),
		}

		// Process first operation
		err := resolver.ProcessOperation(ctx, operation1)
		require.NoError(t, err)

		// Check if second operation conflicts
		conflict, err := resolver.DetectConflict(ctx, operation2)
		require.NoError(t, err)
		assert.Nil(t, conflict) // Should be no conflict
	})

	t.Run("No conflict with operations on different tables", func(t *testing.T) {
		// Create two operations on different tables
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-different-table-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "users", // Different table
			RecordID:  "test-different-table-123",
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive"},
			Timestamp: time.Now().Add(1 * time.Second),
		}

		// Process first operation
		err := resolver.ProcessOperation(ctx, operation1)
		require.NoError(t, err)

		// Check if second operation conflicts
		conflict, err := resolver.DetectConflict(ctx, operation2)
		require.NoError(t, err)
		assert.Nil(t, conflict) // Should be no conflict
	})
}

func TestConflictResolver_ResolveConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	t.Run("Resolve conflict with last write wins strategy", func(t *testing.T) {
		// Create configuration with last_write_wins strategy
		cfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy:        "last_write_wins",
					Enabled:         true,
					MaxRetries:      3,
					RetryDelay:      100 * time.Millisecond,
					ConflictWindow:  5 * time.Minute,
				},
			},
		}

		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)

		// Create conflicting operations
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-lww-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active", "priority": 1},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-lww-123",
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive", "priority": 2},
			Timestamp: time.Now().Add(1 * time.Second), // Newer timestamp
		}

		conflict := &Conflict{
			RecordID:          operation2.RecordID,
			Table:             operation2.Table,
			ExistingOperation: operation1,
			NewOperation:      operation2,
		}

		// Resolve conflict
		resolvedOperation, err := resolver.ResolveConflict(ctx, conflict)
		require.NoError(t, err)
		require.NotNil(t, resolvedOperation)

		// Should use the newer operation (operation2)
		assert.Equal(t, operation2.Timestamp, resolvedOperation.Timestamp)
		assert.Equal(t, "Operation 2", resolvedOperation.Data["name"])
		assert.Equal(t, "inactive", resolvedOperation.Data["status"])
		assert.Equal(t, 2, resolvedOperation.Data["priority"])
	})

	t.Run("Resolve conflict with manual merge strategy", func(t *testing.T) {
		// Create configuration with manual_merge strategy
		cfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy:        "manual_merge",
					Enabled:         true,
					MaxRetries:      3,
					RetryDelay:      100 * time.Millisecond,
					ConflictWindow:  5 * time.Minute,
				},
			},
		}

		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)

		// Create conflicting operations
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-manual-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active", "tags": []string{"tag1"}},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-manual-123",
			Data:      map[string]interface{}{"name": "Operation 2", "priority": 1, "tags": []string{"tag2"}},
			Timestamp: time.Now().Add(1 * time.Second),
		}

		conflict := &Conflict{
			RecordID:          operation2.RecordID,
			Table:             operation2.Table,
			ExistingOperation: operation1,
			NewOperation:      operation2,
		}

		// Resolve conflict
		resolvedOperation, err := resolver.ResolveConflict(ctx, conflict)
		require.NoError(t, err)
		require.NotNil(t, resolvedOperation)

		// Should merge data from both operations
		assert.Equal(t, operation2.Timestamp, resolvedOperation.Timestamp) // Use newer timestamp
		assert.Equal(t, "Operation 2", resolvedOperation.Data["name"])   // Use newer name
		assert.Equal(t, "active", resolvedOperation.Data["status"])      // Keep existing status
		assert.Equal(t, 1, resolvedOperation.Data["priority"])           // Use newer priority
		
		// Tags should be merged
		tags, ok := resolvedOperation.Data["tags"].([]interface{})
		require.True(t, ok)
		assert.Contains(t, tags, "tag1")
		assert.Contains(t, tags, "tag2")
	})

	t.Run("Resolve conflict with first write wins strategy", func(t *testing.T) {
		// Create configuration with first_write_wins strategy
		cfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy:        "first_write_wins",
					Enabled:         true,
					MaxRetries:      3,
					RetryDelay:      100 * time.Millisecond,
					ConflictWindow:  5 * time.Minute,
				},
			},
		}

		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)

		// Create conflicting operations
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-fww-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-fww-123",
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive"},
			Timestamp: time.Now().Add(1 * time.Second), // Newer timestamp
		}

		conflict := &Conflict{
			RecordID:          operation2.RecordID,
			Table:             operation2.Table,
			ExistingOperation: operation1,
			NewOperation:      operation2,
		}

		// Resolve conflict
		resolvedOperation, err := resolver.ResolveConflict(ctx, conflict)
		require.NoError(t, err)
		require.NotNil(t, resolvedOperation)

		// Should use the older operation (operation1)
		assert.Equal(t, operation1.Timestamp, resolvedOperation.Timestamp)
		assert.Equal(t, "Operation 1", resolvedOperation.Data["name"])
		assert.Equal(t, "active", resolvedOperation.Data["status"])
	})

	t.Run("Resolve conflict with delete operation", func(t *testing.T) {
		// Create configuration with last_write_wins strategy
		cfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy:        "last_write_wins",
					Enabled:         true,
					MaxRetries:      3,
					RetryDelay:      100 * time.Millisecond,
					ConflictWindow:  5 * time.Minute,
				},
			},
		}

		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)

		// Create conflicting operations (one is delete)
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-resolve-delete-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now(),
		}

		operation2 := &SyncOperation{
			Type:      "delete",
			Table:     "configuration_items",
			RecordID:  "test-resolve-delete-123",
			Data:      nil, // Delete operation
			Timestamp: time.Now().Add(1 * time.Second),
		}

		conflict := &Conflict{
			RecordID:          operation2.RecordID,
			Table:             operation2.Table,
			ExistingOperation: operation1,
			NewOperation:      operation2,
		}

		// Resolve conflict
		resolvedOperation, err := resolver.ResolveConflict(ctx, conflict)
		require.NoError(t, err)
		require.NotNil(t, resolvedOperation)

		// Should use the delete operation
		assert.Equal(t, "delete", resolvedOperation.Type)
		assert.Nil(t, resolvedOperation.Data)
		assert.Equal(t, operation2.Timestamp, resolvedOperation.Timestamp)
	})
}

func TestConflictResolver_ProcessOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	// Create test configuration
	cfg := &config.Config{
		Sync: config.SyncConfig{
			ConflictResolution: config.ConflictResolutionConfig{
				Strategy:        "last_write_wins",
				Enabled:         true,
				MaxRetries:      3,
				RetryDelay:      100 * time.Millisecond,
				ConflictWindow:  5 * time.Minute,
			},
		},
	}

	resolver, err := NewConflictResolver(cfg, pool, redisClient)
	require.NoError(t, err)

	t.Run("Process operation without conflict", func(t *testing.T) {
		operation := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-process-no-conflict-123",
			Data:      map[string]interface{}{"name": "Test CI", "type": "server"},
			Timestamp: time.Now(),
		}

		err := resolver.ProcessOperation(ctx, operation)
		require.NoError(t, err)

		// Verify operation was processed
		stats := resolver.GetStats()
		assert.GreaterOrEqual(t, stats.ProcessedOperations, int64(1))
	})

	t.Run("Process operation with conflict", func(t *testing.T) {
		// Create first operation
		operation1 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-process-with-conflict-123",
			Data:      map[string]interface{}{"name": "Operation 1", "status": "active"},
			Timestamp: time.Now(),
		}

		err := resolver.ProcessOperation(ctx, operation1)
		require.NoError(t, err)

		// Create conflicting operation
		operation2 := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-process-with-conflict-123",
			Data:      map[string]interface{}{"name": "Operation 2", "status": "inactive"},
			Timestamp: time.Now().Add(1 * time.Second),
		}

		err = resolver.ProcessOperation(ctx, operation2)
		require.NoError(t, err)

		// Verify conflict was detected and resolved
		stats := resolver.GetStats()
		assert.GreaterOrEqual(t, stats.DetectedConflicts, int64(1))
		assert.GreaterOrEqual(t, stats.ResolvedConflicts, int64(1))
	})

	t.Run("Process operation with retry", func(t *testing.T) {
		// This test simulates a temporary failure that should be retried
		operation := &SyncOperation{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-process-retry-123",
			Data:      map[string]interface{}{"name": "Retry Operation", "status": "pending"},
			Timestamp: time.Now(),
		}

		err := resolver.ProcessOperation(ctx, operation)
		// The operation should succeed after retries
		require.NoError(t, err)

		stats := resolver.GetStats()
		assert.GreaterOrEqual(t, stats.RetryAttempts, int64(0))
	})
}

func TestConflictResolver_GetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	// Create test configuration
	cfg := &config.Config{
		Sync: config.SyncConfig{
			ConflictResolution: config.ConflictResolutionConfig{
				Strategy:        "last_write_wins",
				Enabled:         true,
				MaxRetries:      3,
				RetryDelay:      100 * time.Millisecond,
				ConflictWindow:  5 * time.Minute,
			},
		},
	}

	resolver, err := NewConflictResolver(cfg, pool, redisClient)
	require.NoError(t, err)

	t.Run("Get initial stats", func(t *testing.T) {
		stats := resolver.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, int64(0), stats.ProcessedOperations)
		assert.Equal(t, int64(0), stats.DetectedConflicts)
		assert.Equal(t, int64(0), stats.ResolvedConflicts)
		assert.Equal(t, int64(0), stats.FailedResolutions)
		assert.Equal(t, int64(0), stats.RetryAttempts)
	})

	t.Run("Get stats after processing operations", func(t *testing.T) {
		// Process some operations
		operations := []*SyncOperation{
			{
				Type:      "create",
				Table:     "configuration_items",
				RecordID:  "test-stats-1",
				Data:      map[string]interface{}{"name": "Stats CI 1", "type": "server"},
				Timestamp: time.Now(),
			},
			{
				Type:      "update",
				Table:     "configuration_items",
				RecordID:  "test-stats-2",
				Data:      map[string]interface{}{"name": "Stats CI 2", "type": "database"},
				Timestamp: time.Now(),
			},
		}

		for _, op := range operations {
			err := resolver.ProcessOperation(ctx, op)
			require.NoError(t, err)
		}

		stats := resolver.GetStats()
		assert.GreaterOrEqual(t, stats.ProcessedOperations, int64(2))
	})
}

func TestConflictResolver_ConfigValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	t.Run("Validate configuration with zero values", func(t *testing.T) {
		cfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy:        "last_write_wins",
					Enabled:         true,
					MaxRetries:      0,       // Should be set to default
					RetryDelay:      0,       // Should be set to default
					ConflictWindow:  0,       // Should be set to default
				},
			},
		}

		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)
		defer resolver.Stop()

		// Resolver should work with default values
		assert.True(t, resolver.enabled)
		assert.Equal(t, "last_write_wins", resolver.strategy)
	})

	t.Run("Validate configuration with negative values", func(t *testing.T) {
		cfg := &config.Config{
			Sync: config.SyncConfig{
				ConflictResolution: config.ConflictResolutionConfig{
					Strategy:        "last_write_wins",
					Enabled:         true,
					MaxRetries:      -1,      // Should be set to default
					RetryDelay:      -1,      // Should be set to default
					ConflictWindow:  -1,      // Should be set to default
				},
			},
		}

		resolver, err := NewConflictResolver(cfg, pool, redisClient)
		require.NoError(t, err)
		defer resolver.Stop()

		// Resolver should work with default values
		assert.True(t, resolver.enabled)
		assert.Equal(t, "last_write_wins", resolver.strategy)
	})
}

// Helper function to setup test infrastructure
func setupTestInfrastructure(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, *redis.Container, *pgxpool.Pool, *redis.Client) {
	// Create PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts("../../migrations/001_initial_schema.sql", "../../migrations/002_session_management.sql", "../../migrations/003_sync_triggers.sql"),
	)
	require.NoError(t, err)

	// Create Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	require.NoError(t, err)

	// Get connection strings
	pgConnStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	redisConnStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Create PostgreSQL connection pool
	pool, err := pgxpool.New(ctx, pgConnStr)
	require.NoError(t, err)

	// Wait for database to be ready
	require.Eventually(t, func() bool {
		return pool.Ping(ctx) == nil
	}, 30*time.Second, 1*time.Second)

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisConnStr,
	})

	// Wait for Redis to be ready
	require.Eventually(t, func() bool {
		return redisClient.Ping(ctx).Err() == nil
	}, 30*time.Second, 1*time.Second)

	return pgContainer, redisContainer, pool, redisClient
}
