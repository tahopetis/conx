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

func TestUserRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	t.Run("Create valid user", func(t *testing.T) {
		user := &models.User{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Test",
			LastName:     "User",
			IsActive:     true,
			IsVerified:   false,
		}

		createdUser, err := repo.Create(ctx, user)
		require.NoError(t, err)
		require.NotNil(t, createdUser)

		assert.NotEqual(t, uuid.Nil, createdUser.ID)
		assert.Equal(t, user.Username, createdUser.Username)
		assert.Equal(t, user.Email, createdUser.Email)
		assert.Equal(t, user.PasswordHash, createdUser.PasswordHash)
		assert.Equal(t, user.FirstName, createdUser.FirstName)
		assert.Equal(t, user.LastName, createdUser.LastName)
		assert.Equal(t, user.IsActive, createdUser.IsActive)
		assert.Equal(t, user.IsVerified, createdUser.IsVerified)
		assert.NotZero(t, createdUser.CreatedAt)
		assert.NotZero(t, createdUser.UpdatedAt)
	})

	t.Run("Create user with duplicate username", func(t *testing.T) {
		user1 := &models.User{
			Username:     "duplicate",
			Email:        "test1@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Test",
			LastName:     "User1",
			IsActive:     true,
		}

		user2 := &models.User{
			Username:     "duplicate", // Same username
			Email:        "test2@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Test",
			LastName:     "User2",
			IsActive:     true,
		}

		// Create first user
		_, err := repo.Create(ctx, user1)
		require.NoError(t, err)

		// Try to create second user with same username
		_, err = repo.Create(ctx, user2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("Create user with duplicate email", func(t *testing.T) {
		user1 := &models.User{
			Username:     "user1",
			Email:        "duplicate@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Test",
			LastName:     "User1",
			IsActive:     true,
		}

		user2 := &models.User{
			Username:     "user2",
			Email:        "duplicate@example.com", // Same email
			PasswordHash: "hashedpassword123",
			FirstName:    "Test",
			LastName:     "User2",
			IsActive:     true,
		}

		// Create first user
		_, err := repo.Create(ctx, user1)
		require.NoError(t, err)

		// Try to create second user with same email
		_, err = repo.Create(ctx, user2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("Create user with empty required fields", func(t *testing.T) {
		user := &models.User{
			Username:     "", // Empty username
			Email:        "test@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Test",
			LastName:     "User",
		}

		_, err := repo.Create(ctx, user)
		assert.Error(t, err)
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create a test user first
	testUser := &models.User{
		Username:     "getbyid",
		Email:        "getbyid@example.com",
		PasswordHash: "hashedpassword123",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}

	createdUser, err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	t.Run("Get existing user by ID", func(t *testing.T) {
		user, err := repo.GetByID(ctx, createdUser.ID)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, createdUser.ID, user.ID)
		assert.Equal(t, createdUser.Username, user.Username)
		assert.Equal(t, createdUser.Email, user.Email)
	})

	t.Run("Get non-existent user by ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		user, err := repo.GetByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserRepository_GetByUsername(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create a test user first
	testUser := &models.User{
		Username:     "getbyusername",
		Email:        "getbyusername@example.com",
		PasswordHash: "hashedpassword123",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}

	_, err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	t.Run("Get existing user by username", func(t *testing.T) {
		user, err := repo.GetByUsername(ctx, testUser.Username)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, testUser.Username, user.Username)
		assert.Equal(t, testUser.Email, user.Email)
	})

	t.Run("Get non-existent user by username", func(t *testing.T) {
		user, err := repo.GetByUsername(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create a test user first
	testUser := &models.User{
		Username:     "getbyemail",
		Email:        "getbyemail@example.com",
		PasswordHash: "hashedpassword123",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}

	_, err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	t.Run("Get existing user by email", func(t *testing.T) {
		user, err := repo.GetByEmail(ctx, testUser.Email)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, testUser.Username, user.Username)
		assert.Equal(t, testUser.Email, user.Email)
	})

	t.Run("Get non-existent user by email", func(t *testing.T) {
		user, err := repo.GetByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create a test user first
	testUser := &models.User{
		Username:     "updateuser",
		Email:        "update@example.com",
		PasswordHash: "hashedpassword123",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}

	createdUser, err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	t.Run("Update user successfully", func(t *testing.T) {
		updatedUser := &models.User{
			ID:           createdUser.ID,
			Username:     createdUser.Username,
			Email:        "updated@example.com",
			PasswordHash: "newhashedpassword123",
			FirstName:    "Updated",
			LastName:     "User",
			IsActive:     false,
			IsVerified:   true,
		}

		result, err := repo.Update(ctx, updatedUser)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, updatedUser.ID, result.ID)
		assert.Equal(t, updatedUser.Username, result.Username)
		assert.Equal(t, updatedUser.Email, result.Email)
		assert.Equal(t, updatedUser.PasswordHash, result.PasswordHash)
		assert.Equal(t, updatedUser.FirstName, result.FirstName)
		assert.Equal(t, updatedUser.LastName, result.LastName)
		assert.Equal(t, updatedUser.IsActive, result.IsActive)
		assert.Equal(t, updatedUser.IsVerified, result.IsVerified)
		assert.True(t, result.UpdatedAt.After(createdUser.UpdatedAt))
	})

	t.Run("Update non-existent user", func(t *testing.T) {
		nonExistentUser := &models.User{
			ID:           uuid.New(),
			Username:     "nonexistent",
			Email:        "nonexistent@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Non",
			LastName:     "Existent",
			IsActive:     true,
		}

		_, err := repo.Update(ctx, nonExistentUser)
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("Update user with duplicate email", func(t *testing.T) {
		// Create another user
		anotherUser := &models.User{
			Username:     "another",
			Email:        "another@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Another",
			LastName:     "User",
			IsActive:     true,
		}

		_, err := repo.Create(ctx, anotherUser)
		require.NoError(t, err)

		// Try to update first user with second user's email
		duplicateEmailUser := &models.User{
			ID:           createdUser.ID,
			Username:     createdUser.Username,
			Email:        "another@example.com", // Duplicate email
			PasswordHash: createdUser.PasswordHash,
			FirstName:    createdUser.FirstName,
			LastName:     createdUser.LastName,
			IsActive:     createdUser.IsActive,
		}

		_, err = repo.Update(ctx, duplicateEmailUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})
}

func TestUserRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create a test user first
	testUser := &models.User{
		Username:     "deleteuser",
		Email:        "delete@example.com",
		PasswordHash: "hashedpassword123",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}

	createdUser, err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	t.Run("Delete existing user", func(t *testing.T) {
		err := repo.Delete(ctx, createdUser.ID)
		require.NoError(t, err)

		// Verify user is deleted
		user, err := repo.GetByID(ctx, createdUser.ID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("Delete non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create test users
	users := []*models.User{
		{
			Username:     "listuser1",
			Email:        "list1@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "List",
			LastName:     "User1",
			IsActive:     true,
		},
		{
			Username:     "listuser2",
			Email:        "list2@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "List",
			LastName:     "User2",
			IsActive:     true,
		},
		{
			Username:     "listuser3",
			Email:        "list3@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "List",
			LastName:     "User3",
			IsActive:     false, // Inactive user
		},
	}

	for _, user := range users {
		_, err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	t.Run("List all users", func(t *testing.T) {
		result, err := repo.List(ctx, &models.UserListParams{
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.GreaterOrEqual(t, len(result.Users), 3)
		assert.GreaterOrEqual(t, result.Total, int64(3))
	})

	t.Run("List users with pagination", func(t *testing.T) {
		// Test first page
		result1, err := repo.List(ctx, &models.UserListParams{
			Limit:  2,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, result1.Users, 2)

		// Test second page
		result2, err := repo.List(ctx, &models.UserListParams{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result2.Users), 1)
	})

	t.Run("List users filtered by active status", func(t *testing.T) {
		result, err := repo.List(ctx, &models.UserListParams{
			IsActive: boolPtr(true),
			Limit:    10,
			Offset:   0,
		})
		require.NoError(t, err)

		// All returned users should be active
		for _, user := range result.Users {
			assert.True(t, user.IsActive)
		}
	})

	t.Run("List users filtered by search term", func(t *testing.T) {
		result, err := repo.List(ctx, &models.UserListParams{
			Search: "User1",
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)

		// Should find users with "User1" in username, first name, last name, or email
		assert.GreaterOrEqual(t, len(result.Users), 1)
	})

	t.Run("List users ordered by username", func(t *testing.T) {
		result, err := repo.List(ctx, &models.UserListParams{
			OrderBy: "username",
			Order:   "asc",
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)

		// Check if users are ordered by username
		for i := 1; i < len(result.Users); i++ {
			assert.LessOrEqual(t, result.Users[i-1].Username, result.Users[i].Username)
		}
	})
}

func TestUserRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Get initial count
	initialCount, err := repo.Count(ctx, nil)
	require.NoError(t, err)

	// Create test users
	users := []*models.User{
		{
			Username:     "countuser1",
			Email:        "count1@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Count",
			LastName:     "User1",
			IsActive:     true,
		},
		{
			Username:     "countuser2",
			Email:        "count2@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "Count",
			LastName:     "User2",
			IsActive:     true,
		},
	}

	for _, user := range users {
		_, err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	t.Run("Count all users", func(t *testing.T) {
		count, err := repo.Count(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, initialCount+2, count)
	})

	t.Run("Count active users", func(t *testing.T) {
		params := &models.UserListParams{IsActive: boolPtr(true)}
		count, err := repo.Count(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, initialCount+2, count) // Both created users are active
	})

	t.Run("Count users with search filter", func(t *testing.T) {
		params := &models.UserListParams{Search: "countuser"}
		count, err := repo.Count(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewUserRepository(pool)

	// Create a test user
	testUser := &models.User{
		Username:     "lastlogin",
		Email:        "lastlogin@example.com",
		PasswordHash: "hashedpassword123",
		FirstName:    "Last",
		LastName:     "Login",
		IsActive:     true,
	}

	createdUser, err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	// Ensure LastLoginAt is nil initially
	assert.Nil(t, createdUser.LastLoginAt)

	t.Run("Update last login successfully", func(t *testing.T) {
		err := repo.UpdateLastLogin(ctx, createdUser.ID)
		require.NoError(t, err)

		// Verify LastLoginAt is updated
		user, err := repo.GetByID(ctx, createdUser.ID)
		require.NoError(t, err)
		require.NotNil(t, user.LastLoginAt)
		assert.WithinDuration(t, time.Now(), *user.LastLoginAt, 5*time.Second)
	})

	t.Run("Update last login for non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.UpdateLastLogin(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
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
