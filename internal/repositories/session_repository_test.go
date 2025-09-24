package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/conx/cmdb/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestSessionRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	t.Run("Create valid session", func(t *testing.T) {
		session := &models.Session{
			UserID:       uuid.New(),
			RefreshToken: "test_refresh_token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		}

		createdSession, err := repo.Create(ctx, session)
		require.NoError(t, err)
		require.NotNil(t, createdSession)

		assert.NotEqual(t, uuid.Nil, createdSession.ID)
		assert.Equal(t, session.UserID, createdSession.UserID)
		assert.Equal(t, session.RefreshToken, createdSession.RefreshToken)
		assert.Equal(t, session.ExpiresAt, createdSession.ExpiresAt)
		assert.Equal(t, session.UserAgent, createdSession.UserAgent)
		assert.Equal(t, session.IPAddress, createdSession.IPAddress)
		assert.Equal(t, session.IsActive, createdSession.IsActive)
		assert.NotZero(t, createdSession.CreatedAt)
		assert.NotZero(t, createdSession.UpdatedAt)
	})

	t.Run("Create session with duplicate refresh token", func(t *testing.T) {
		userID := uuid.New()
		session1 := &models.Session{
			UserID:       userID,
			RefreshToken: "duplicate_refresh_token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-1",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		}

		session2 := &models.Session{
			UserID:       userID,
			RefreshToken: "duplicate_refresh_token", // Same refresh token
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-2",
			IPAddress:    "192.168.1.101",
			IsActive:     true,
		}

		// Create first session
		_, err := repo.Create(ctx, session1)
		require.NoError(t, err)

		// Try to create second session with same refresh token
		_, err = repo.Create(ctx, session2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refresh_token")
	})

	t.Run("Create session with empty required fields", func(t *testing.T) {
		session := &models.Session{
			UserID:       uuid.New(),
			RefreshToken: "", // Empty refresh token
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		}

		_, err := repo.Create(ctx, session)
		assert.Error(t, err)
	})

	t.Run("Create session with expired time", func(t *testing.T) {
		session := &models.Session{
			UserID:       uuid.New(),
			RefreshToken: "expired_token",
			ExpiresAt:    time.Now().Add(-1 * time.Hour), // Already expired
			UserAgent:    "test-agent",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		}

		_, err := repo.Create(ctx, session)
		// This might be allowed depending on business logic
		// For now, we'll assume it's allowed but will be filtered out in queries
		require.NoError(t, err)
	})
}

func TestSessionRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	// Create a test session first
	testSession := &models.Session{
		UserID:       uuid.New(),
		RefreshToken: "getbyid_token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		UserAgent:    "test-agent",
		IPAddress:    "192.168.1.100",
		IsActive:     true,
	}

	createdSession, err := repo.Create(ctx, testSession)
	require.NoError(t, err)

	t.Run("Get existing session by ID", func(t *testing.T) {
		session, err := repo.GetByID(ctx, createdSession.ID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, createdSession.ID, session.ID)
		assert.Equal(t, createdSession.UserID, session.UserID)
		assert.Equal(t, createdSession.RefreshToken, session.RefreshToken)
		assert.Equal(t, createdSession.ExpiresAt, session.ExpiresAt)
	})

	t.Run("Get non-existent session by ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		session, err := repo.GetByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Equal(t, ErrSessionNotFound, err)
	})
}

func TestSessionRepository_GetByRefreshToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	// Create a test session first
	testSession := &models.Session{
		UserID:       uuid.New(),
		RefreshToken: "getbyrefreshtoken",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		UserAgent:    "test-agent",
		IPAddress:    "192.168.1.100",
		IsActive:     true,
	}

	_, err := repo.Create(ctx, testSession)
	require.NoError(t, err)

	t.Run("Get existing session by refresh token", func(t *testing.T) {
		session, err := repo.GetByRefreshToken(ctx, testSession.RefreshToken)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, testSession.RefreshToken, session.RefreshToken)
		assert.Equal(t, testSession.UserID, session.UserID)
	})

	t.Run("Get non-existent session by refresh token", func(t *testing.T) {
		session, err := repo.GetByRefreshToken(ctx, "nonexistent_token")
		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Equal(t, ErrSessionNotFound, err)
	})
}

func TestSessionRepository_GetByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	userID := uuid.New()

	// Create test sessions for the same user
	sessions := []*models.Session{
		{
			UserID:       userID,
			RefreshToken: "token1",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-1",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "token2",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-2",
			IPAddress:    "192.168.1.101",
			IsActive:     true,
		},
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Get sessions by user ID", func(t *testing.T) {
		retrievedSessions, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, retrievedSessions, 2)

		// Verify all sessions belong to the correct user
		for _, session := range retrievedSessions {
			assert.Equal(t, userID, session.UserID)
		}
	})

	t.Run("Get sessions for non-existent user", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		sessions, err := repo.GetByUserID(ctx, nonExistentUserID)
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestSessionRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	// Create a test session first
	testSession := &models.Session{
		UserID:       uuid.New(),
		RefreshToken: "update_token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		UserAgent:    "test-agent",
		IPAddress:    "192.168.1.100",
		IsActive:     true,
	}

	createdSession, err := repo.Create(ctx, testSession)
	require.NoError(t, err)

	t.Run("Update session successfully", func(t *testing.T) {
		updatedSession := &models.Session{
			ID:           createdSession.ID,
			UserID:       createdSession.UserID,
			RefreshToken: "updated_refresh_token",
			ExpiresAt:    time.Now().Add(48 * time.Hour),
			UserAgent:    "updated-agent",
			IPAddress:    "192.168.1.200",
			IsActive:     false,
		}

		result, err := repo.Update(ctx, updatedSession)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, updatedSession.ID, result.ID)
		assert.Equal(t, updatedSession.UserID, result.UserID)
		assert.Equal(t, updatedSession.RefreshToken, result.RefreshToken)
		assert.Equal(t, updatedSession.ExpiresAt, result.ExpiresAt)
		assert.Equal(t, updatedSession.UserAgent, result.UserAgent)
		assert.Equal(t, updatedSession.IPAddress, result.IPAddress)
		assert.Equal(t, updatedSession.IsActive, result.IsActive)
		assert.True(t, result.UpdatedAt.After(createdSession.UpdatedAt))
	})

	t.Run("Update non-existent session", func(t *testing.T) {
		nonExistentSession := &models.Session{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			RefreshToken: "nonexistent_token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "nonexistent-agent",
			IPAddress:    "192.168.1.300",
			IsActive:     true,
		}

		_, err := repo.Update(ctx, nonExistentSession)
		assert.Error(t, err)
		assert.Equal(t, ErrSessionNotFound, err)
	})

	t.Run("Update session with duplicate refresh token", func(t *testing.T) {
		// Create another session
		anotherSession := &models.Session{
			UserID:       uuid.New(),
			RefreshToken: "another_token",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "another-agent",
			IPAddress:    "192.168.1.150",
			IsActive:     true,
		}

		_, err := repo.Create(ctx, anotherSession)
		require.NoError(t, err)

		// Try to update first session with second session's refresh token
		duplicateTokenSession := &models.Session{
			ID:           createdSession.ID,
			UserID:       createdSession.UserID,
			RefreshToken: "another_token", // Duplicate refresh token
			ExpiresAt:    createdSession.ExpiresAt,
			UserAgent:    createdSession.UserAgent,
			IPAddress:    createdSession.IPAddress,
			IsActive:     createdSession.IsActive,
		}

		_, err = repo.Update(ctx, duplicateTokenSession)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refresh_token")
	})
}

func TestSessionRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	// Create a test session first
	testSession := &models.Session{
		UserID:       uuid.New(),
		RefreshToken: "delete_token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		UserAgent:    "test-agent",
		IPAddress:    "192.168.1.100",
		IsActive:     true,
	}

	createdSession, err := repo.Create(ctx, testSession)
	require.NoError(t, err)

	t.Run("Delete existing session", func(t *testing.T) {
		err := repo.Delete(ctx, createdSession.ID)
		require.NoError(t, err)

		// Verify session is deleted
		session, err := repo.GetByID(ctx, createdSession.ID)
		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Equal(t, ErrSessionNotFound, err)
	})

	t.Run("Delete non-existent session", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, ErrSessionNotFound, err)
	})
}

func TestSessionRepository_DeleteByRefreshToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	// Create a test session first
	testSession := &models.Session{
		UserID:       uuid.New(),
		RefreshToken: "deletebyrefreshtoken",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		UserAgent:    "test-agent",
		IPAddress:    "192.168.1.100",
		IsActive:     true,
	}

	_, err := repo.Create(ctx, testSession)
	require.NoError(t, err)

	t.Run("Delete session by refresh token", func(t *testing.T) {
		err := repo.DeleteByRefreshToken(ctx, testSession.RefreshToken)
		require.NoError(t, err)

		// Verify session is deleted
		session, err := repo.GetByRefreshToken(ctx, testSession.RefreshToken)
		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Equal(t, ErrSessionNotFound, err)
	})

	t.Run("Delete non-existent session by refresh token", func(t *testing.T) {
		err := repo.DeleteByRefreshToken(ctx, "nonexistent_token")
		assert.Error(t, err)
		assert.Equal(t, ErrSessionNotFound, err)
	})
}

func TestSessionRepository_DeleteByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	userID := uuid.New()

	// Create test sessions for the same user
	sessions := []*models.Session{
		{
			UserID:       userID,
			RefreshToken: "deletebyuserid_token1",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-1",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "deletebyuserid_token2",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-2",
			IPAddress:    "192.168.1.101",
			IsActive:     true,
		},
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Delete sessions by user ID", func(t *testing.T) {
		err := repo.DeleteByUserID(ctx, userID)
		require.NoError(t, err)

		// Verify all sessions for the user are deleted
		retrievedSessions, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, retrievedSessions)
	})

	t.Run("Delete sessions for non-existent user", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		err := repo.DeleteByUserID(ctx, nonExistentUserID)
		// This should not error even if user doesn't exist
		require.NoError(t, err)
	})
}

func TestSessionRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	// Create test sessions
	userID := uuid.New()
	sessions := []*models.Session{
		{
			UserID:       userID,
			RefreshToken: "list_token1",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-1",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "list_token2",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-2",
			IPAddress:    "192.168.1.101",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "list_token3",
			ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired session
			UserAgent:    "test-agent-3",
			IPAddress:    "192.168.1.102",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "list_token4",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-4",
			IPAddress:    "192.168.1.103",
			IsActive:     false, // Inactive session
		},
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("List all sessions", func(t *testing.T) {
		result, err := repo.List(ctx, &models.SessionListParams{
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.GreaterOrEqual(t, len(result.Sessions), 4)
		assert.GreaterOrEqual(t, result.Total, int64(4))
	})

	t.Run("List sessions with pagination", func(t *testing.T) {
		// Test first page
		result1, err := repo.List(ctx, &models.SessionListParams{
			Limit:  2,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, result1.Sessions, 2)

		// Test second page
		result2, err := repo.List(ctx, &models.SessionListParams{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result2.Sessions), 2)
	})

	t.Run("List sessions filtered by user ID", func(t *testing.T) {
		result, err := repo.List(ctx, &models.SessionListParams{
			UserID: &userID,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)

		// All returned sessions should belong to the specified user
		for _, session := range result.Sessions {
			assert.Equal(t, userID, session.UserID)
		}
	})

	t.Run("List sessions filtered by active status", func(t *testing.T) {
		result, err := repo.List(ctx, &models.SessionListParams{
			IsActive: boolPtr(true),
			Limit:    10,
			Offset:   0,
		})
		require.NoError(t, err)

		// All returned sessions should be active
		for _, session := range result.Sessions {
			assert.True(t, session.IsActive)
		}
	})

	t.Run("List sessions filtered by expiry", func(t *testing.T) {
		result, err := repo.List(ctx, &models.SessionListParams{
			NotExpired: boolPtr(true),
			Limit:      10,
			Offset:     0,
		})
		require.NoError(t, err)

		// All returned sessions should not be expired
		for _, session := range result.Sessions {
			assert.True(t, session.ExpiresAt.After(time.Now()))
		}
	})

	t.Run("List sessions ordered by created at", func(t *testing.T) {
		result, err := repo.List(ctx, &models.SessionListParams{
			OrderBy: "created_at",
			Order:   "desc",
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)

		// Check if sessions are ordered by created_at (newest first)
		for i := 1; i < len(result.Sessions); i++ {
			assert.True(t, result.Sessions[i-1].CreatedAt.After(result.Sessions[i].CreatedAt) ||
				result.Sessions[i-1].CreatedAt.Equal(result.Sessions[i].CreatedAt))
		}
	})
}

func TestSessionRepository_CleanupExpiredSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	userID := uuid.New()

	// Create test sessions
	sessions := []*models.Session{
		{
			UserID:       userID,
			RefreshToken: "expired_token1",
			ExpiresAt:    time.Now().Add(-2 * time.Hour), // Expired
			UserAgent:    "test-agent-1",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "expired_token2",
			ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
			UserAgent:    "test-agent-2",
			IPAddress:    "192.168.1.101",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "active_token",
			ExpiresAt:    time.Now().Add(24 * time.Hour), // Active
			UserAgent:    "test-agent-3",
			IPAddress:    "192.168.1.102",
			IsActive:     true,
		},
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Cleanup expired sessions", func(t *testing.T) {
		deletedCount, err := repo.CleanupExpiredSessions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deletedCount) // Should delete 2 expired sessions

		// Verify expired sessions are deleted
		expiredSession1, err := repo.GetByRefreshToken(ctx, "expired_token1")
		assert.Error(t, err)
		assert.Nil(t, expiredSession1)
		assert.Equal(t, ErrSessionNotFound, err)

		expiredSession2, err := repo.GetByRefreshToken(ctx, "expired_token2")
		assert.Error(t, err)
		assert.Nil(t, expiredSession2)
		assert.Equal(t, ErrSessionNotFound, err)

		// Verify active session still exists
		activeSession, err := repo.GetByRefreshToken(ctx, "active_token")
		require.NoError(t, err)
		require.NotNil(t, activeSession)
	})
}

func TestSessionRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewSessionRepository(pool)

	userID := uuid.New()

	// Get initial count
	initialCount, err := repo.Count(ctx, nil)
	require.NoError(t, err)

	// Create test sessions
	sessions := []*models.Session{
		{
			UserID:       userID,
			RefreshToken: "count_token1",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-1",
			IPAddress:    "192.168.1.100",
			IsActive:     true,
		},
		{
			UserID:       userID,
			RefreshToken: "count_token2",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserAgent:    "test-agent-2",
			IPAddress:    "192.168.1.101",
			IsActive:     false, // Inactive session
		},
	}

	for _, session := range sessions {
		_, err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Count all sessions", func(t *testing.T) {
		count, err := repo.Count(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, initialCount+2, count)
	})

	t.Run("Count active sessions", func(t *testing.T) {
		params := &models.SessionListParams{IsActive: boolPtr(true)}
		count, err := repo.Count(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, initialCount+1, count) // One active session created
	})

	t.Run("Count sessions by user ID", func(t *testing.T) {
		params := &models.SessionListParams{UserID: &userID}
		count, err := repo.Count(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count) // Both sessions created for this user
	})
}

// Helper function to setup test database
func setupTestDatabase(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, *pgxpool.Pool) {
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts("../../migrations/001_initial_schema.sql", "../../migrations/002_session_management.sql"),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Wait for database to be ready
	require.Eventually(t, func() bool {
		return pool.Ping(ctx) == nil
	}, 30*time.Second, 1*time.Second)

	return pgContainer, pool
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
