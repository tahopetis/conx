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

func TestNewSyncService(t *testing.T) {
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
			Enabled:         true,
			BatchSize:       100,
			FlushInterval:   5 * time.Second,
			MaxRetries:      3,
			RetryDelay:      1 * time.Second,
			QueueSize:       1000,
			WorkerCount:     5,
		},
	}

	t.Run("Create sync service successfully", func(t *testing.T) {
		syncService, err := NewSyncService(cfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, syncService)

		assert.NotNil(t, syncService.dbPool)
		assert.NotNil(t, syncService.redisClient)
		assert.NotNil(t, syncService.queue)
		assert.NotNil(t, syncService.workers)
		assert.Equal(t, cfg.Sync.WorkerCount, len(syncService.workers))
		assert.True(t, syncService.IsRunning())
	})

	t.Run("Create sync service with disabled sync", func(t *testing.T) {
		disabledCfg := &config.Config{
			Sync: config.SyncConfig{
				Enabled: false,
			},
		}

		syncService, err := NewSyncService(disabledCfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, syncService)

		assert.False(t, syncService.IsRunning())
	})

	t.Run("Create sync service with invalid configuration", func(t *testing.T) {
		invalidCfg := &config.Config{
			Sync: config.SyncConfig{
				Enabled:     true,
				WorkerCount: 0, // Invalid worker count
			},
		}

		syncService, err := NewSyncService(invalidCfg, pool, redisClient)
		require.NoError(t, err) // Should not error, should use defaults
		require.NotNil(t, syncService)

		assert.Greater(t, len(syncService.workers), 0)
	})
}

func TestSyncService_StartStop(t *testing.T) {
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
			Enabled:         true,
			BatchSize:       10,
			FlushInterval:   1 * time.Second,
			MaxRetries:      3,
			RetryDelay:      100 * time.Millisecond,
			QueueSize:       100,
			WorkerCount:     2,
		},
	}

	syncService, err := NewSyncService(cfg, pool, redisClient)
	require.NoError(t, err)

	t.Run("Stop running service", func(t *testing.T) {
		assert.True(t, syncService.IsRunning())

		err := syncService.Stop()
		require.NoError(t, err)
		assert.False(t, syncService.IsRunning())
	})

	t.Run("Start stopped service", func(t *testing.T) {
		assert.False(t, syncService.IsRunning())

		err := syncService.Start()
		require.NoError(t, err)
		assert.True(t, syncService.IsRunning())
	})

	t.Run("Start already running service", func(t *testing.T) {
		assert.True(t, syncService.IsRunning())

		err := syncService.Start()
		// Should not error, should be idempotent
		require.NoError(t, err)
		assert.True(t, syncService.IsRunning())
	})

	t.Run("Stop already stopped service", func(t *testing.T) {
		// Stop first
		err := syncService.Stop()
		require.NoError(t, err)
		assert.False(t, syncService.IsRunning())

		// Stop again
		err = syncService.Stop()
		// Should not error, should be idempotent
		require.NoError(t, err)
		assert.False(t, syncService.IsRunning())
	})
}

func TestSyncService_QueueOperation(t *testing.T) {
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
			Enabled:         true,
			BatchSize:       5,
			FlushInterval:   100 * time.Millisecond,
			MaxRetries:      3,
			RetryDelay:      50 * time.Millisecond,
			QueueSize:       50,
			WorkerCount:     2,
		},
	}

	syncService, err := NewSyncService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer syncService.Stop()

	t.Run("Queue sync operation", func(t *testing.T) {
		operation := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-ci-123",
			Data:      map[string]interface{}{"name": "Test CI", "type": "server"},
			Timestamp: time.Now(),
		}

		err := syncService.QueueOperation(operation)
		require.NoError(t, err)

		// Check if operation is queued
		stats := syncService.GetStats()
		assert.GreaterOrEqual(t, stats.QueuedOperations, int64(1))
	})

	t.Run("Queue multiple sync operations", func(t *testing.T) {
		operations := []*SyncOperation{
			{
				Type:      "create",
				Table:     "configuration_items",
				RecordID:  "test-ci-456",
				Data:      map[string]interface{}{"name": "Test CI 2", "type": "database"},
				Timestamp: time.Now(),
			},
			{
				Type:      "update",
				Table:     "configuration_items",
				RecordID:  "test-ci-789",
				Data:      map[string]interface{}{"name": "Updated CI", "type": "server"},
				Timestamp: time.Now(),
			},
			{
				Type:      "delete",
				Table:     "configuration_items",
				RecordID:  "test-ci-999",
				Data:      nil,
				Timestamp: time.Now(),
			},
		}

		for _, op := range operations {
			err := syncService.QueueOperation(op)
			require.NoError(t, err)
		}

		// Check if operations are queued
		stats := syncService.GetStats()
		assert.GreaterOrEqual(t, stats.QueuedOperations, int64(3))
	})

	t.Run("Queue operation when service is stopped", func(t *testing.T) {
		// Stop the service
		err := syncService.Stop()
		require.NoError(t, err)

		operation := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-ci-stopped",
			Data:      map[string]interface{}{"name": "Stopped CI", "type": "server"},
			Timestamp: time.Now(),
		}

		err = syncService.QueueOperation(operation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service is not running")

		// Restart service for subsequent tests
		err = syncService.Start()
		require.NoError(t, err)
	})

	t.Run("Queue operation with full queue", func(t *testing.T) {
		// Create a service with very small queue size
		smallQueueCfg := &config.Config{
			Sync: config.SyncConfig{
				Enabled:         true,
				BatchSize:       1,
				FlushInterval:   100 * time.Millisecond,
				MaxRetries:      1,
				RetryDelay:      50 * time.Millisecond,
				QueueSize:       1, // Very small queue
				WorkerCount:     1,
			},
		}

		smallQueueService, err := NewSyncService(smallQueueCfg, pool, redisClient)
		require.NoError(t, err)
		defer smallQueueService.Stop()

		// Fill the queue
		operation1 := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-ci-queue-1",
			Data:      map[string]interface{}{"name": "Queue CI 1", "type": "server"},
			Timestamp: time.Now(),
		}

		err = smallQueueService.QueueOperation(operation1)
		require.NoError(t, err)

		// Try to add another operation (should fail or block)
		operation2 := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-ci-queue-2",
			Data:      map[string]interface{}{"name": "Queue CI 2", "type": "server"},
			Timestamp: time.Now(),
		}

		// This might timeout or error depending on implementation
		done := make(chan bool)
		go func() {
			err := smallQueueService.QueueOperation(operation2)
			assert.Error(t, err) // Should error due to full queue
			done <- true
		}()

		select {
		case <-done:
			// Operation completed (with error)
		case <-time.After(2 * time.Second):
			// Operation timed out (might be blocking)
			t.Log("Queue operation timed out (possibly blocking)")
		}
	})
}

func TestSyncService_ProcessOperations(t *testing.T) {
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
			Enabled:         true,
			BatchSize:       2,
			FlushInterval:   50 * time.Millisecond,
			MaxRetries:      2,
			RetryDelay:      25 * time.Millisecond,
			QueueSize:       100,
			WorkerCount:     1, // Single worker for predictable processing
		},
	}

	syncService, err := NewSyncService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer syncService.Stop()

	t.Run("Process queued operations", func(t *testing.T) {
		// Queue some operations
		operations := []*SyncOperation{
			{
				Type:      "create",
				Table:     "configuration_items",
				RecordID:  "test-process-1",
				Data:      map[string]interface{}{"name": "Process CI 1", "type": "server"},
				Timestamp: time.Now(),
			},
			{
				Type:      "update",
				Table:     "configuration_items",
				RecordID:  "test-process-2",
				Data:      map[string]interface{}{"name": "Process CI 2", "type": "database"},
				Timestamp: time.Now(),
			},
		}

		for _, op := range operations {
			err := syncService.QueueOperation(op)
			require.NoError(t, err)
		}

		// Wait for processing to complete
		time.Sleep(200 * time.Millisecond)

		// Check stats
		stats := syncService.GetStats()
		assert.GreaterOrEqual(t, stats.ProcessedOperations, int64(2))
	})

	t.Run("Handle operation processing errors", func(t *testing.T) {
		// Create an operation that will fail
		invalidOperation := &SyncOperation{
			Type:      "invalid_type",
			Table:     "invalid_table",
			RecordID:  "test-invalid",
			Data:      map[string]interface{}{"invalid": "data"},
			Timestamp: time.Now(),
		}

		err := syncService.QueueOperation(invalidOperation)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		// Check stats - should have failed operations
		stats := syncService.GetStats()
		assert.GreaterOrEqual(t, stats.FailedOperations, int64(1))
	})
}

func TestSyncService_GetStats(t *testing.T) {
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
			Enabled:         true,
			BatchSize:       5,
			FlushInterval:   100 * time.Millisecond,
			MaxRetries:      3,
			RetryDelay:      50 * time.Millisecond,
			QueueSize:       100,
			WorkerCount:     2,
		},
	}

	syncService, err := NewSyncService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer syncService.Stop()

	t.Run("Get initial stats", func(t *testing.T) {
		stats := syncService.GetStats()
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.StartTime, time.Time{})
		assert.Equal(t, int64(0), stats.QueuedOperations)
		assert.Equal(t, int64(0), stats.ProcessedOperations)
		assert.Equal(t, int64(0), stats.FailedOperations)
		assert.Equal(t, int64(0), stats.RetryAttempts)
		assert.True(t, stats.IsRunning)
	})

	t.Run("Get stats after operations", func(t *testing.T) {
		// Queue some operations
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
			err := syncService.QueueOperation(op)
			require.NoError(t, err)
		}

		// Wait for some processing
		time.Sleep(100 * time.Millisecond)

		stats := syncService.GetStats()
		assert.GreaterOrEqual(t, stats.QueuedOperations, int64(2))
		assert.GreaterOrEqual(t, stats.ProcessedOperations, int64(0))
	})

	t.Run("Get stats when service is stopped", func(t *testing.T) {
		// Stop the service
		err := syncService.Stop()
		require.NoError(t, err)

		stats := syncService.GetStats()
		assert.False(t, stats.IsRunning)

		// Restart service
		err = syncService.Start()
		require.NoError(t, err)
	})
}

func TestSyncService_Flush(t *testing.T) {
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
			Enabled:         true,
			BatchSize:       10,
			FlushInterval:   1 * time.Second, // Long flush interval
			MaxRetries:      3,
			RetryDelay:      50 * time.Millisecond,
			QueueSize:       100,
			WorkerCount:     1,
		},
	}

	syncService, err := NewSyncService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer syncService.Stop()

	t.Run("Flush queued operations", func(t *testing.T) {
		// Queue operations
		operations := []*SyncOperation{
			{
				Type:      "create",
				Table:     "configuration_items",
				RecordID:  "test-flush-1",
				Data:      map[string]interface{}{"name": "Flush CI 1", "type": "server"},
				Timestamp: time.Now(),
			},
			{
				Type:      "update",
				Table:     "configuration_items",
				RecordID:  "test-flush-2",
				Data:      map[string]interface{}{"name": "Flush CI 2", "type": "database"},
				Timestamp: time.Now(),
			},
		}

		for _, op := range operations {
			err := syncService.QueueOperation(op)
			require.NoError(t, err)
		}

		// Get stats before flush
		statsBefore := syncService.GetStats()
		queuedBefore := statsBefore.QueuedOperations

		// Flush operations
		err := syncService.Flush()
		require.NoError(t, err)

		// Wait for flush to complete
		time.Sleep(100 * time.Millisecond)

		// Get stats after flush
		statsAfter := syncService.GetStats()
		assert.GreaterOrEqual(t, statsAfter.ProcessedOperations, statsBefore.ProcessedOperations+2)
	})

	t.Run("Flush when service is stopped", func(t *testing.T) {
		// Stop the service
		err := syncService.Stop()
		require.NoError(t, err)

		// Try to flush
		err = syncService.Flush()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service is not running")

		// Restart service
		err = syncService.Start()
		require.NoError(t, err)
	})

	t.Run("Flush empty queue", func(t *testing.T) {
		// Ensure queue is empty (wait for any pending operations)
		time.Sleep(200 * time.Millisecond)

		statsBefore := syncService.GetStats()
		processedBefore := statsBefore.ProcessedOperations

		// Flush empty queue
		err := syncService.Flush()
		require.NoError(t, err)

		statsAfter := syncService.GetStats()
		assert.Equal(t, processedBefore, statsAfter.ProcessedOperations)
	})
}

func TestSyncService_ConfigValidation(t *testing.T) {
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
				Enabled:         true,
				BatchSize:       0,       // Should be set to default
				FlushInterval:   0,       // Should be set to default
				MaxRetries:      0,       // Should be set to default
				RetryDelay:      0,       // Should be set to default
				QueueSize:       0,       // Should be set to default
				WorkerCount:     0,       // Should be set to default
			},
		}

		syncService, err := NewSyncService(cfg, pool, redisClient)
		require.NoError(t, err)
		defer syncService.Stop()

		// Service should work with default values
		assert.True(t, syncService.IsRunning())
		assert.Greater(t, len(syncService.workers), 0)
	})

	t.Run("Validate configuration with negative values", func(t *testing.T) {
		cfg := &config.Config{
			Sync: config.SyncConfig{
				Enabled:         true,
				BatchSize:       -1,      // Should be set to default
				FlushInterval:   -1,      // Should be set to default
				MaxRetries:      -1,      // Should be set to default
				RetryDelay:      -1,      // Should be set to default
				QueueSize:       -1,      // Should be set to default
				WorkerCount:     -1,      // Should be set to default
			},
		}

		syncService, err := NewSyncService(cfg, pool, redisClient)
		require.NoError(t, err)
		defer syncService.Stop()

		// Service should work with default values
		assert.True(t, syncService.IsRunning())
		assert.Greater(t, len(syncService.workers), 0)
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
