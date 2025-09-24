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

func TestNewMonitorService(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	t.Run("Create monitor service successfully", func(t *testing.T) {
		monitor, err := NewMonitorService(cfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, monitor)

		assert.NotNil(t, monitor.dbPool)
		assert.NotNil(t, monitor.redisClient)
		assert.True(t, monitor.enabled)
		assert.Equal(t, cfg.Sync.Monitoring.CheckInterval, monitor.checkInterval)
		assert.Equal(t, cfg.Sync.Monitoring.HealthCheckTTL, monitor.healthCheckTTL)
		assert.Equal(t, cfg.Sync.Monitoring.MaxLatency, monitor.maxLatency)
		assert.Equal(t, cfg.Sync.Monitoring.AlertThreshold, monitor.alertThreshold)
	})

	t.Run("Create monitor service with disabled monitoring", func(t *testing.T) {
		disabledCfg := &config.Config{
			Sync: config.SyncConfig{
				Monitoring: config.MonitoringConfig{
					Enabled: false,
				},
			},
		}

		monitor, err := NewMonitorService(disabledCfg, pool, redisClient)
		require.NoError(t, err)
		require.NotNil(t, monitor)

		assert.False(t, monitor.enabled)
	})

	t.Run("Create monitor service with invalid configuration", func(t *testing.T) {
		invalidCfg := &config.Config{
			Sync: config.SyncConfig{
				Monitoring: config.MonitoringConfig{
					Enabled:        true,
					CheckInterval:  0,       // Should be set to default
					HealthCheckTTL: 0,       // Should be set to default
					MaxLatency:     0,       // Should be set to default
					AlertThreshold: 0,       // Should be set to default
				},
			},
		}

		monitor, err := NewMonitorService(invalidCfg, pool, redisClient)
		require.NoError(t, err) // Should not error, should use defaults
		require.NotNil(t, monitor)

		assert.Greater(t, monitor.checkInterval, time.Duration(0))
		assert.Greater(t, monitor.healthCheckTTL, time.Duration(0))
		assert.Greater(t, monitor.maxLatency, time.Duration(0))
		assert.Greater(t, monitor.alertThreshold, 0)
	})
}

func TestMonitorService_StartStop(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  1 * time.Second, // Short interval for testing
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)

	t.Run("Stop running monitor", func(t *testing.T) {
		assert.True(t, monitor.IsRunning())

		err := monitor.Stop()
		require.NoError(t, err)
		assert.False(t, monitor.IsRunning())
	})

	t.Run("Start stopped monitor", func(t *testing.T) {
		assert.False(t, monitor.IsRunning())

		err := monitor.Start()
		require.NoError(t, err)
		assert.True(t, monitor.IsRunning())
	})

	t.Run("Start already running monitor", func(t *testing.T) {
		assert.True(t, monitor.IsRunning())

		err := monitor.Start()
		// Should not error, should be idempotent
		require.NoError(t, err)
		assert.True(t, monitor.IsRunning())
	})

	t.Run("Stop already stopped monitor", func(t *testing.T) {
		// Stop first
		err := monitor.Stop()
		require.NoError(t, err)
		assert.False(t, monitor.IsRunning())

		// Stop again
		err = monitor.Stop()
		// Should not error, should be idempotent
		require.NoError(t, err)
		assert.False(t, monitor.IsRunning())
	})
}

func TestMonitorService_CheckHealth(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	t.Run("Check health with healthy services", func(t *testing.T) {
		health, err := monitor.CheckHealth(ctx)
		require.NoError(t, err)
		require.NotNil(t, health)

		assert.True(t, health.Healthy)
		assert.GreaterOrEqual(t, health.Timestamp, time.Time{})
		assert.NotNil(t, health.DatabaseStatus)
		assert.NotNil(t, health.RedisStatus)
		assert.NotNil(t, health.SyncStatus)
		assert.GreaterOrEqual(t, health.DatabaseLatency, time.Duration(0))
		assert.GreaterOrEqual(t, health.RedisLatency, time.Duration(0))
	})

	t.Run("Check health multiple times", func(t *testing.T) {
		// Check health multiple times to verify consistency
		for i := 0; i < 5; i++ {
			health, err := monitor.CheckHealth(ctx)
			require.NoError(t, err)
			require.NotNil(t, health)

			assert.True(t, health.Healthy)
			assert.GreaterOrEqual(t, health.Timestamp, time.Time{})
		}
	})
}

func TestMonitorService_RegisterComponent(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	t.Run("Register component successfully", func(t *testing.T) {
		component := &Component{
			ID:          "test-component-1",
			Name:        "Test Component",
			Type:        "service",
			Status:      "healthy",
			LastChecked: time.Now(),
			Metadata:    map[string]interface{}{"version": "1.0.0", "environment": "test"},
		}

		err := monitor.RegisterComponent(ctx, component)
		require.NoError(t, err)

		// Verify component is registered
		registeredComponent, err := monitor.GetComponent(ctx, component.ID)
		require.NoError(t, err)
		require.NotNil(t, registeredComponent)

		assert.Equal(t, component.ID, registeredComponent.ID)
		assert.Equal(t, component.Name, registeredComponent.Name)
		assert.Equal(t, component.Type, registeredComponent.Type)
		assert.Equal(t, component.Status, registeredComponent.Status)
		assert.Equal(t, component.Metadata, registeredComponent.Metadata)
	})

	t.Run("Register multiple components", func(t *testing.T) {
		components := []*Component{
			{
				ID:          "test-component-2",
				Name:        "Test Component 2",
				Type:        "service",
				Status:      "healthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "2.0.0"},
			},
			{
				ID:          "test-component-3",
				Name:        "Test Component 3",
				Type:        "worker",
				Status:      "healthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "1.5.0"},
			},
		}

		for _, component := range components {
			err := monitor.RegisterComponent(ctx, component)
			require.NoError(t, err)
		}

		// Verify all components are registered
		allComponents, err := monitor.GetAllComponents(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allComponents), 2) // At least the 2 we just registered

		// Verify specific components
		for _, component := range components {
			registeredComponent, err := monitor.GetComponent(ctx, component.ID)
			require.NoError(t, err)
			require.NotNil(t, registeredComponent)
			assert.Equal(t, component.ID, registeredComponent.ID)
		}
	})

	t.Run("Register component with duplicate ID", func(t *testing.T) {
		component1 := &Component{
			ID:          "test-duplicate",
			Name:        "Test Duplicate 1",
			Type:        "service",
			Status:      "healthy",
			LastChecked: time.Now(),
			Metadata:    map[string]interface{}{"version": "1.0.0"},
		}

		component2 := &Component{
			ID:          "test-duplicate", // Same ID
			Name:        "Test Duplicate 2",
			Type:        "service",
			Status:      "unhealthy",
			LastChecked: time.Now(),
			Metadata:    map[string]interface{}{"version": "2.0.0"},
		}

		// Register first component
		err := monitor.RegisterComponent(ctx, component1)
		require.NoError(t, err)

		// Register second component with same ID (should update)
		err = monitor.RegisterComponent(ctx, component2)
		require.NoError(t, err)

		// Verify component was updated
		registeredComponent, err := monitor.GetComponent(ctx, component1.ID)
		require.NoError(t, err)
		require.NotNil(t, registeredComponent)

		assert.Equal(t, component2.Name, registeredComponent.Name)
		assert.Equal(t, component2.Status, registeredComponent.Status)
		assert.Equal(t, component2.Metadata, registeredComponent.Metadata)
	})
}

func TestMonitorService_UpdateComponentStatus(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	// Register a test component
	component := &Component{
		ID:          "test-update",
		Name:        "Test Update Component",
		Type:        "service",
		Status:      "healthy",
		LastChecked: time.Now(),
		Metadata:    map[string]interface{}{"version": "1.0.0"},
	}

	err = monitor.RegisterComponent(ctx, component)
	require.NoError(t, err)

	t.Run("Update component status successfully", func(t *testing.T) {
		newStatus := "unhealthy"
		newMetadata := map[string]interface{}{
			"version":     "1.0.1",
			"environment": "production",
			"error":       "connection failed",
		}

		err := monitor.UpdateComponentStatus(ctx, component.ID, newStatus, newMetadata)
		require.NoError(t, err)

		// Verify component was updated
		updatedComponent, err := monitor.GetComponent(ctx, component.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedComponent)

		assert.Equal(t, newStatus, updatedComponent.Status)
		assert.Equal(t, newMetadata, updatedComponent.Metadata)
		assert.True(t, updatedComponent.LastChecked.After(component.LastChecked))
	})

	t.Run("Update non-existent component", func(t *testing.T) {
		err := monitor.UpdateComponentStatus(ctx, "non-existent-component", "healthy", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
	})
}

func TestMonitorService_GetComponent(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	// Register a test component
	component := &Component{
		ID:          "test-get",
		Name:        "Test Get Component",
		Type:        "service",
		Status:      "healthy",
		LastChecked: time.Now(),
		Metadata:    map[string]interface{}{"version": "1.0.0"},
	}

	err = monitor.RegisterComponent(ctx, component)
	require.NoError(t, err)

	t.Run("Get existing component", func(t *testing.T) {
		retrievedComponent, err := monitor.GetComponent(ctx, component.ID)
		require.NoError(t, err)
		require.NotNil(t, retrievedComponent)

		assert.Equal(t, component.ID, retrievedComponent.ID)
		assert.Equal(t, component.Name, retrievedComponent.Name)
		assert.Equal(t, component.Type, retrievedComponent.Type)
		assert.Equal(t, component.Status, retrievedComponent.Status)
		assert.Equal(t, component.Metadata, retrievedComponent.Metadata)
	})

	t.Run("Get non-existent component", func(t *testing.T) {
		retrievedComponent, err := monitor.GetComponent(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, retrievedComponent)
		assert.Contains(t, err.Error(), "component not found")
	})
}

func TestMonitorService_GetAllComponents(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	t.Run("Get all components when empty", func(t *testing.T) {
		components, err := monitor.GetAllComponents(ctx)
		require.NoError(t, err)
		assert.Empty(t, components)
	})

	t.Run("Get all components with registered components", func(t *testing.T) {
		// Register test components
		testComponents := []*Component{
			{
				ID:          "test-all-1",
				Name:        "Test All Component 1",
				Type:        "service",
				Status:      "healthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "1.0.0"},
			},
			{
				ID:          "test-all-2",
				Name:        "Test All Component 2",
				Type:        "worker",
				Status:      "healthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "2.0.0"},
			},
			{
				ID:          "test-all-3",
				Name:        "Test All Component 3",
				Type:        "service",
				Status:      "unhealthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "1.5.0"},
			},
		}

		for _, component := range testComponents {
			err := monitor.RegisterComponent(ctx, component)
			require.NoError(t, err)
		}

		// Get all components
		allComponents, err := monitor.GetAllComponents(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allComponents), 3)

		// Verify all test components are present
		componentMap := make(map[string]*Component)
		for _, component := range allComponents {
			componentMap[component.ID] = component
		}

		for _, expectedComponent := range testComponents {
			actualComponent, exists := componentMap[expectedComponent.ID]
			require.True(t, exists, "Component %s should exist", expectedComponent.ID)
			assert.Equal(t, expectedComponent.Name, actualComponent.Name)
			assert.Equal(t, expectedComponent.Type, actualComponent.Type)
			assert.Equal(t, expectedComponent.Status, actualComponent.Status)
		}
	})
}

func TestMonitorService_GetStats(t *testing.T) {
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
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  30 * time.Second,
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 3,
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	t.Run("Get initial stats", func(t *testing.T) {
		stats := monitor.GetStats()
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.StartTime, time.Time{})
		assert.Equal(t, int64(0), stats.HealthChecks)
		assert.Equal(t, int64(0), stats.AlertsTriggered)
		assert.Equal(t, int64(0), stats.ComponentsRegistered)
		assert.True(t, stats.IsRunning)
	})

	t.Run("Get stats after registering components", func(t *testing.T) {
		// Register test components
		components := []*Component{
			{
				ID:          "test-stats-1",
				Name:        "Test Stats Component 1",
				Type:        "service",
				Status:      "healthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "1.0.0"},
			},
			{
				ID:          "test-stats-2",
				Name:        "Test Stats Component 2",
				Type:        "worker",
				Status:      "healthy",
				LastChecked: time.Now(),
				Metadata:    map[string]interface{}{"version": "2.0.0"},
			},
		}

		for _, component := range components {
			err := monitor.RegisterComponent(ctx, component)
			require.NoError(t, err)
		}

		// Perform health check
		_, err = monitor.CheckHealth(ctx)
		require.NoError(t, err)

		stats := monitor.GetStats()
		assert.GreaterOrEqual(t, stats.ComponentsRegistered, int64(2))
		assert.GreaterOrEqual(t, stats.HealthChecks, int64(1))
	})

	t.Run("Get stats when service is stopped", func(t *testing.T) {
		// Stop the service
		err := monitor.Stop()
		require.NoError(t, err)

		stats := monitor.GetStats()
		assert.False(t, stats.IsRunning)

		// Restart service
		err = monitor.Start()
		require.NoError(t, err)
	})
}

func TestMonitorService_MonitorComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, redisContainer, pool, redisClient := setupTestInfrastructure(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer redisContainer.Terminate(ctx)
	defer pool.Close()
	defer redisClient.Close()

	// Create test configuration with short check interval
	cfg := &config.Config{
		Sync: config.SyncConfig{
			Monitoring: config.MonitoringConfig{
				Enabled:         true,
				CheckInterval:  100 * time.Millisecond, // Very short for testing
				HealthCheckTTL: 5 * time.Minute,
				MaxLatency:     1 * time.Second,
				AlertThreshold: 2, // Low threshold for testing
			},
		},
	}

	monitor, err := NewMonitorService(cfg, pool, redisClient)
	require.NoError(t, err)
	defer monitor.Stop()

	// Register test components
	components := []*Component{
		{
			ID:          "test-monitor-1",
			Name:        "Test Monitor Component 1",
			Type:        "service",
			Status:      "healthy",
			LastChecked: time.Now(),
			Metadata:    map[string]interface{}{"version": "1.0.0"},
		},
		{
			ID:          "test-monitor-2",
			Name:        "Test Monitor Component 2",
			Type:        "worker",
			Status:      "healthy",
			LastChecked: time.Now(),
			Metadata:    map[string]interface{}{"version": "2.0.0"},
		},
	}

	for _, component := range components {
		err := monitor.RegisterComponent(ctx, component)
		require.NoError(t, err)
	}

	t.Run("Monitor components automatically", func(t *testing.T) {
		// Wait for monitoring to occur
		time.Sleep(500 * time.Millisecond)

		// Check if components were monitored
		stats := monitor.GetStats()
		assert.GreaterOrEqual(t, stats.HealthChecks, int64(1))

		// Get components to see if they were updated
		allComponents, err := monitor.GetAllComponents(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allComponents), 2)

		// Verify last checked times were updated
		for _, component := range allComponents {
			if component.ID == "test-monitor-1" || component.ID == "test-monitor-2" {
				assert.True(t, component.LastChecked.After(components[0].LastChecked) ||
					component.LastChecked.Equal(components[0].LastChecked))
			}
		}
	})
}

func TestMonitorService_ConfigValidation(t *testing.T) {
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
				Monitoring: config.MonitoringConfig{
					Enabled:        true,
					CheckInterval:  0,       // Should be set to default
					HealthCheckTTL: 0,       // Should be set to default
					MaxLatency:     0,       // Should be set to default
					AlertThreshold: 0,       // Should be set to default
				},
			},
		}

		monitor, err := NewMonitorService(cfg, pool, redisClient)
		require.NoError(t, err)
		defer monitor.Stop()

		// Service should work with default values
		assert.True(t, monitor.enabled)
		assert.Greater(t, monitor.checkInterval, time.Duration(0))
		assert.Greater(t, monitor.healthCheckTTL, time.Duration(0))
		assert.Greater(t, monitor.maxLatency, time.Duration(0))
		assert.Greater(t, monitor.alertThreshold, 0)
	})

	t.Run("Validate configuration with negative values", func(t *testing.T) {
		cfg := &config.Config{
			Sync: config.SyncConfig{
				Monitoring: config.MonitoringConfig{
					Enabled:        true,
					CheckInterval:  -1,      // Should be set to default
					HealthCheckTTL: -1,      // Should be set to default
					MaxLatency:     -1,      // Should be set to default
					AlertThreshold: -1,      // Should be set to default
				},
			},
		}

		monitor, err := NewMonitorService(cfg, pool, redisClient)
		require.NoError(t, err)
		defer monitor.Stop()

		// Service should work with default values
		assert.True(t, monitor.enabled)
		assert.Greater(t, monitor.checkInterval, time.Duration(0))
		assert.Greater(t, monitor.healthCheckTTL, time.Duration(0))
		assert.Greater(t, monitor.maxLatency, time.Duration(0))
		assert.Greater(t, monitor.alertThreshold, 0)
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
