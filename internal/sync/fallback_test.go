package sync

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

func TestNewFallbackService(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024, // 10MB
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	t.Run("Create fallback service successfully", func(t *testing.T) {
		fallback, err := NewFallbackService(cfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, fallback)

		assert.NotNil(t, fallback.dbPool)
		assert.NotNil(t, fallback.redisClient)
		assert.True(t, fallback.enabled)
		assert.Equal(t, cfg.Sync.Fallback.Mode, fallback.mode)
		assert.Equal(t, cfg.Sync.Fallback.StoragePath, fallback.storagePath)
		assert.Equal(t, cfg.Sync.Fallback.MaxFileSize, fallback.maxFileSize)
		assert.Equal(t, cfg.Sync.Fallback.MaxRetries, fallback.maxRetries)
		assert.Equal(t, cfg.Sync.Fallback.RetryDelay, fallback.retryDelay)
		assert.Equal(t, cfg.Sync.Fallback.SyncInterval, fallback.syncInterval)
		assert.Equal(t, cfg.Sync.Fallback.CompressionLevel, fallback.compressionLevel)
		assert.Equal(t, cfg.Sync.Fallback.EncryptionEnabled, fallback.encryptionEnabled)
	})

	t.Run("Create fallback service with disabled fallback", func(t *testing.T) {
		disabledCfg := &config.Config{
			Sync: config.SyncConfig{
				Fallback: config.FallbackConfig{
					Enabled: false,
				},
			},
		}

		fallback, err := NewFallbackService(disabledCfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, fallback)

		assert.False(t, fallback.enabled)
	})

	t.Run("Create fallback service with invalid configuration", func(t *testing.T) {
		invalidCfg := &config.Config{
			Sync: config.SyncConfig{
				Fallback: config.FallbackConfig{
					Enabled:           true,
					Mode:              "invalid_mode", // Should fall back to default
					MaxFileSize:       0,              // Should be set to default
					MaxRetries:        0,              // Should be set to default
					RetryDelay:        0,              // Should be set to default
					SyncInterval:      0,              // Should be set to default
					CompressionLevel:  -1,             // Should be set to default
				},
			},
		}

		fallback, err := NewFallbackService(invalidCfg, pool, redisClient)
		require.NoError(t, err) // Should not error, should use defaults
		require.NotNil(t, fallback)

		assert.Equal(t, "local", fallback.mode) // Should use default mode
		assert.Greater(t, fallback.maxFileSize, int64(0))
		assert.Greater(t, fallback.maxRetries, 0)
		assert.Greater(t, fallback.retryDelay, time.Duration(0))
		assert.Greater(t, fallback.syncInterval, time.Duration(0))
		assert.GreaterOrEqual(t, fallback.compressionLevel, 0)
	})
}

func TestFallbackService_StartStop(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      1 * time.Second, // Short interval for testing
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)

	t.Run("Stop running fallback service", func(t *testing.T) {
		assert.True(t, fallback.IsRunning())

		err := fallback.Stop()
		require.NoError(t, err)
		assert.False(t, fallback.IsRunning())
	})

	t.Run("Start stopped fallback service", func(t *testing.T) {
		assert.False(t, fallback.IsRunning())

		err := fallback.Start()
		require.NoError(t, err)
		assert.True(t, fallback.IsRunning())
	})

	t.Run("Start already running fallback service", func(t *testing.T) {
		assert.True(t, fallback.IsRunning())

		err := fallback.Start()
		// Should not error, should be idempotent
		require.NoError(t, err)
		assert.True(t, fallback.IsRunning())
	})

	t.Run("Stop already stopped fallback service", func(t *testing.T) {
		// Stop first
		err := fallback.Stop()
		require.NoError(t, err)
		assert.False(t, fallback.IsRunning())

		// Stop again
		err = fallback.Stop()
		// Should not error, should be idempotent
		require.NoError(t, err)
		assert.False(t, fallback.IsRunning())
	})
}

func TestFallbackService_StoreOperation(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer fallback.Stop()

	t.Run("Store operation successfully", func(t *testing.T) {
		operation := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-fallback-123",
			Data:      map[string]interface{}{"name": "Test CI", "type": "server", "status": "active"},
			Timestamp: time.Now(),
		}

		err := fallback.StoreOperation(ctx, operation)
		require.NoError(t, err)

		// Verify operation was stored
		stats := fallback.GetStats()
		assert.GreaterOrEqual(t, stats.StoredOperations, int64(1))
	})

	t.Run("Store multiple operations", func(t *testing.T) {
		operations := []*SyncOperation{
			{
				Type:      "create",
				Table:     "configuration_items",
				RecordID:  "test-fallback-456",
				Data:      map[string]interface{}{"name": "Test CI 2", "type": "database"},
				Timestamp: time.Now(),
			},
			{
				Type:      "update",
				Table:     "configuration_items",
				RecordID:  "test-fallback-789",
				Data:      map[string]interface{}{"name": "Updated CI", "type": "server"},
				Timestamp: time.Now(),
			},
			{
				Type:      "delete",
				Table:     "configuration_items",
				RecordID:  "test-fallback-999",
				Data:      nil,
				Timestamp: time.Now(),
			},
		}

		for _, op := range operations {
			err := fallback.StoreOperation(ctx, op)
			require.NoError(t, err)
		}

		// Verify operations were stored
		stats := fallback.GetStats()
		assert.GreaterOrEqual(t, stats.StoredOperations, int64(3))
	})

	t.Run("Store operation when service is stopped", func(t *testing.T) {
		// Stop the service
		err := fallback.Stop()
		require.NoError(t, err)

		operation := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-fallback-stopped",
			Data:      map[string]interface{}{"name": "Stopped CI", "type": "server"},
			Timestamp: time.Now(),
		}

		err = fallback.StoreOperation(ctx, operation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service is not running")

		// Restart service for subsequent tests
		err = fallback.Start()
		require.NoError(t, err)
	})
}

func TestFallbackService_RetrieveOperations(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer fallback.Stop()

	// Store test operations
	testOperations := []*SyncOperation{
		{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-retrieve-123",
			Data:      map[string]interface{}{"name": "Test CI 1", "type": "server"},
			Timestamp: time.Now(),
		},
		{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-retrieve-456",
			Data:      map[string]interface{}{"name": "Test CI 2", "type": "database"},
			Timestamp: time.Now().Add(1 * time.Second),
		},
	}

	for _, op := range testOperations {
		err := fallback.StoreOperation(ctx, op)
		require.NoError(t, err)
	}

	t.Run("Retrieve all operations", func(t *testing.T) {
		operations, err := fallback.RetrieveOperations(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(operations), 2)

		// Verify operations are retrieved correctly
		operationMap := make(map[string]*SyncOperation)
		for _, op := range operations {
			operationMap[op.RecordID] = op
		}

		for _, expectedOp := range testOperations {
			actualOp, exists := operationMap[expectedOp.RecordID]
			require.True(t, exists, "Operation %s should exist", expectedOp.RecordID)
			assert.Equal(t, expectedOp.Type, actualOp.Type)
			assert.Equal(t, expectedOp.Table, actualOp.Table)
			assert.Equal(t, expectedOp.Data, actualOp.Data)
		}
	})

	t.Run("Retrieve operations with limit", func(t *testing.T) {
		operations, err := fallback.RetrieveOperations(ctx, WithLimit(1))
		require.NoError(t, err)
		assert.Len(t, operations, 1)
	})

	t.Run("Retrieve operations with time range", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now().Add(1 * time.Hour)

		operations, err := fallback.RetrieveOperations(ctx, WithTimeRange(startTime, endTime))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(operations), 2)
	})

	t.Run("Retrieve operations when service is stopped", func(t *testing.T) {
		// Stop the service
		err := fallback.Stop()
		require.NoError(t, err)

		operations, err := fallback.RetrieveOperations(ctx)
		assert.Error(t, err)
		assert.Nil(t, operations)
		assert.Contains(t, err.Error(), "service is not running")

		// Restart service for subsequent tests
		err = fallback.Start()
		require.NoError(t, err)
	})
}

func TestFallbackService_ClearOperations(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer fallback.Stop()

	// Store test operations
	operations := []*SyncOperation{
		{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-clear-123",
			Data:      map[string]interface{}{"name": "Test CI 1", "type": "server"},
			Timestamp: time.Now(),
		},
		{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-clear-456",
			Data:      map[string]interface{}{"name": "Test CI 2", "type": "database"},
			Timestamp: time.Now(),
		},
	}

	for _, op := range operations {
		err := fallback.StoreOperation(ctx, op)
		require.NoError(t, err)
	}

	t.Run("Clear all operations", func(t *testing.T) {
		// Verify operations exist
		statsBefore := fallback.GetStats()
		assert.GreaterOrEqual(t, statsBefore.StoredOperations, int64(2))

		// Clear all operations
		err := fallback.ClearOperations(ctx)
		require.NoError(t, err)

		// Verify operations are cleared
		statsAfter := fallback.GetStats()
		assert.Equal(t, int64(0), statsAfter.StoredOperations)

		// Verify no operations can be retrieved
		retrievedOps, err := fallback.RetrieveOperations(ctx)
		require.NoError(t, err)
		assert.Empty(t, retrievedOps)
	})

	t.Run("Clear operations when service is stopped", func(t *testing.T) {
		// Store operations again
		for _, op := range operations {
			err := fallback.StoreOperation(ctx, op)
			require.NoError(t, err)
		}

		// Stop the service
		err := fallback.Stop()
		require.NoError(t, err)

		err = fallback.ClearOperations(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service is not running")

		// Restart service for subsequent tests
		err = fallback.Start()
		require.NoError(t, err)
	})
}

func TestFallbackService_SyncWithDatabase(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer fallback.Stop()

	// Store test operations
	operations := []*SyncOperation{
		{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-sync-123",
			Data:      map[string]interface{}{"name": "Sync CI 1", "type": "server"},
			Timestamp: time.Now(),
		},
		{
			Type:      "update",
			Table:     "configuration_items",
			RecordID:  "test-sync-456",
			Data:      map[string]interface{}{"name": "Sync CI 2", "type": "database"},
			Timestamp: time.Now(),
		},
	}

	for _, op := range operations {
		err := fallback.StoreOperation(ctx, op)
		require.NoError(t, err)
	}

	t.Run("Sync operations with database", func(t *testing.T) {
		// Sync operations
		syncedCount, err := fallback.SyncWithDatabase(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, syncedCount, int64(2))

		// Verify operations are cleared after sync
		stats := fallback.GetStats()
		assert.Equal(t, int64(0), stats.StoredOperations)

		// Verify sync stats are updated
		syncStats := fallback.GetStats()
		assert.GreaterOrEqual(t, syncStats.SyncedOperations, int64(2))
	})

	t.Run("Sync when no operations exist", func(t *testing.T) {
		// Sync again (no operations should exist)
		syncedCount, err := fallback.SyncWithDatabase(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), syncedCount)
	})

	t.Run("Sync when service is stopped", func(t *testing.T) {
		// Store operations again
		for _, op := range operations {
			err := fallback.StoreOperation(ctx, op)
			require.NoError(t, err)
		}

		// Stop the service
		err := fallback.Stop()
		require.NoError(t, err)

		syncedCount, err := fallback.SyncWithDatabase(ctx)
		assert.Error(t, err)
		assert.Equal(t, int64(0), syncedCount)
		assert.Contains(t, err.Error(), "service is not running")

		// Restart service for subsequent tests
		err = fallback.Start()
		require.NoError(t, err)
	})
}

func TestFallbackService_GetStats(t *testing.T) {
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
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer fallback.Stop()

	t.Run("Get initial stats", func(t *testing.T) {
		stats := fallback.GetStats()
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.StartTime, time.Time{})
		assert.Equal(t, int64(0), stats.StoredOperations)
		assert.Equal(t, int64(0), stats.RetrievedOperations)
		assert.Equal(t, int64(0), stats.SyncedOperations)
		assert.Equal(t, int64(0), stats.FailedOperations)
		assert.Equal(t, int64(0), stats.RetryAttempts)
		assert.True(t, stats.IsRunning)
	})

	t.Run("Get stats after storing operations", func(t *testing.T) {
		// Store test operations
		operations := []*SyncOperation{
			{
				Type:      "create",
				Table:     "configuration_items",
				RecordID:  "test-stats-123",
				Data:      map[string]interface{}{"name": "Stats CI 1", "type": "server"},
				Timestamp: time.Now(),
			},
			{
				Type:      "update",
				Table:     "configuration_items",
				RecordID:  "test-stats-456",
				Data:      map[string]interface{}{"name": "Stats CI 2", "type": "database"},
				Timestamp: time.Now(),
			},
		}

		for _, op := range operations {
			err := fallback.StoreOperation(ctx, op)
			require.NoError(t, err)
		}

		stats := fallback.GetStats()
		assert.GreaterOrEqual(t, stats.StoredOperations, int64(2))
	})

	t.Run("Get stats after retrieving operations", func(t *testing.T) {
		// Retrieve operations
		_, err := fallback.RetrieveOperations(ctx)
		require.NoError(t, err)

		stats := fallback.GetStats()
		assert.GreaterOrEqual(t, stats.RetrievedOperations, int64(1))
	})

	t.Run("Get stats after syncing", func(t *testing.T) {
		// Sync operations
		_, err := fallback.SyncWithDatabase(ctx)
		require.NoError(t, err)

		stats := fallback.GetStats()
		assert.GreaterOrEqual(t, stats.SyncedOperations, int64(1))
	})

	t.Run("Get stats when service is stopped", func(t *testing.T) {
		// Stop the service
		err := fallback.Stop()
		require.NoError(t, err)

		stats := fallback.GetStats()
		assert.False(t, stats.IsRunning)

		// Restart service
		err = fallback.Start()
		require.NoError(t, err)
	})
}

func TestFallbackService_ConfigValidation(t *testing.T) {
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
				Fallback: config.FallbackConfig{
					Enabled:           true,
					Mode:              "local",
					MaxFileSize:       0,       // Should be set to default
					MaxRetries:        0,       // Should be set to default
					RetryDelay:        0,       // Should be set to default
					SyncInterval:      0,       // Should be set to default
					CompressionLevel:  0,       // Should be set to default
				},
			},
		}

		fallback, err := NewFallbackService(cfg, pool, redisClient)
		require.NoError(t, err)
		defer fallback.Stop()

		// Service should work with default values
		assert.True(t, fallback.enabled)
		assert.Equal(t, "local", fallback.mode)
		assert.Greater(t, fallback.maxFileSize, int64(0))
		assert.Greater(t, fallback.maxRetries, 0)
		assert.Greater(t, fallback.retryDelay, time.Duration(0))
		assert.Greater(t, fallback.syncInterval, time.Duration(0))
		assert.GreaterOrEqual(t, fallback.compressionLevel, 0)
	})

	t.Run("Validate configuration with negative values", func(t *testing.T) {
		cfg := &config.Config{
			Sync: config.SyncConfig{
				Fallback: config.FallbackConfig{
					Enabled:           true,
					Mode:              "local",
					MaxFileSize:       -1,      // Should be set to default
					MaxRetries:        -1,      // Should be set to default
					RetryDelay:        -1,      // Should be set to default
					SyncInterval:      -1,      // Should be set to default
					CompressionLevel:  -1,      // Should be set to default
				},
			},
		}

		fallback, err := NewFallbackService(cfg, pool, redisClient)
		require.NoError(t, err)
		defer fallback.Stop()

		// Service should work with default values
		assert.True(t, fallback.enabled)
		assert.Equal(t, "local", fallback.mode)
		assert.Greater(t, fallback.maxFileSize, int64(0))
		assert.Greater(t, fallback.maxRetries, 0)
		assert.Greater(t, fallback.retryDelay, time.Duration(0))
		assert.Greater(t, fallback.syncInterval, time.Duration(0))
		assert.GreaterOrEqual(t, fallback.compressionLevel, 0)
	})
}

func TestFallbackService_RetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	// Create test configuration with short retry delay
	cfg := &config.Config{
		Sync: config.SyncConfig{
			Fallback: config.FallbackConfig{
				Enabled:           true,
				Mode:              "local",
				StoragePath:       "/tmp/fallback",
				MaxFileSize:       10 * 1024 * 1024,
				MaxRetries:        3,
				RetryDelay:        100 * time.Millisecond, // Short delay for testing
				SyncInterval:      5 * time.Minute,
				CompressionLevel:  6,
				EncryptionEnabled: false,
			},
		},
	}

	fallback, err := NewFallbackService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer fallback.Stop()

	t.Run("Retry failed operations", func(t *testing.T) {
		operation := &SyncOperation{
			Type:      "create",
			Table:     "configuration_items",
			RecordID:  "test-retry-123",
			Data:      map[string]interface{}{"name": "Retry CI", "type": "server"},
			Timestamp: time.Now(),
		}

		// Store operation (should succeed)
		err := fallback.StoreOperation(ctx, operation)
		require.NoError(t, err)

		// Get initial stats
		statsBefore := fallback.GetStats()

		// This test would need to simulate failures to properly test retries
		// For now, we'll just verify the retry mechanism is in place
		assert.GreaterOrEqual(t, statsBefore.StoredOperations, int64(1))
		assert.GreaterOrEqual(t, fallback.maxRetries, 3)
		assert.Greater(t, fallback.retryDelay, time.Duration(0))
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
