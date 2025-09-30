package repositories

import (
	"context"
	"testing"

	"connect/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRoleRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	t.Run("Create valid role", func(t *testing.T) {
		role := &models.Role{
			Name:        "test_role",
			DisplayName: "Test Role",
			Description: "A test role for testing purposes",
			IsActive:    true,
		}

		createdRole, err := repo.Create(ctx, role)
		require.NoError(t, err)
		require.NotNil(t, createdRole)

		assert.NotEqual(t, uuid.Nil, createdRole.ID)
		assert.Equal(t, role.Name, createdRole.Name)
		assert.Equal(t, role.DisplayName, createdRole.DisplayName)
		assert.Equal(t, role.Description, createdRole.Description)
		assert.Equal(t, role.IsActive, createdRole.IsActive)
		assert.NotZero(t, createdRole.CreatedAt)
		assert.NotZero(t, createdRole.UpdatedAt)
	})

	t.Run("Create role with duplicate name", func(t *testing.T) {
		role1 := &models.Role{
			Name:        "duplicate_role",
			DisplayName: "Duplicate Role 1",
			Description: "First duplicate role",
			IsActive:    true,
		}

		role2 := &models.Role{
			Name:        "duplicate_role", // Same name
			DisplayName: "Duplicate Role 2",
			Description: "Second duplicate role",
			IsActive:    true,
		}

		// Create first role
		_, err := repo.Create(ctx, role1)
		require.NoError(t, err)

		// Try to create second role with same name
		_, err = repo.Create(ctx, role2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("Create role with empty required fields", func(t *testing.T) {
		role := &models.Role{
			Name:        "", // Empty name
			DisplayName: "Empty Name Role",
			Description: "Role with empty name",
			IsActive:    true,
		}

		_, err := repo.Create(ctx, role)
		assert.Error(t, err)
	})
}

func TestRoleRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role first
	testRole := &models.Role{
		Name:        "getbyid",
		DisplayName: "Get By ID Role",
		Description: "Role for testing GetByID",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	t.Run("Get existing role by ID", func(t *testing.T) {
		role, err := repo.GetByID(ctx, createdRole.ID)
		require.NoError(t, err)
		require.NotNil(t, role)

		assert.Equal(t, createdRole.ID, role.ID)
		assert.Equal(t, createdRole.Name, role.Name)
		assert.Equal(t, createdRole.DisplayName, role.DisplayName)
		assert.Equal(t, createdRole.Description, role.Description)
	})

	t.Run("Get non-existent role by ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		role, err := repo.GetByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, role)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestRoleRepository_GetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role first
	testRole := &models.Role{
		Name:        "getbyname",
		DisplayName: "Get By Name Role",
		Description: "Role for testing GetByName",
		IsActive:    true,
	}

	_, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	t.Run("Get existing role by name", func(t *testing.T) {
		role, err := repo.GetByName(ctx, testRole.Name)
		require.NoError(t, err)
		require.NotNil(t, role)

		assert.Equal(t, testRole.Name, role.Name)
		assert.Equal(t, testRole.DisplayName, role.DisplayName)
		assert.Equal(t, testRole.Description, role.Description)
	})

	t.Run("Get non-existent role by name", func(t *testing.T) {
		role, err := repo.GetByName(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, role)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestRoleRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role first
	testRole := &models.Role{
		Name:        "updaterole",
		DisplayName: "Update Role",
		Description: "Role for testing Update",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	t.Run("Update role successfully", func(t *testing.T) {
		updatedRole := &models.Role{
			ID:          createdRole.ID,
			Name:        createdRole.Name, // Name should not change
			DisplayName: "Updated Role",
			Description: "Updated role description",
			IsActive:    false,
		}

		result, err := repo.Update(ctx, updatedRole)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, updatedRole.ID, result.ID)
		assert.Equal(t, updatedRole.Name, result.Name)
		assert.Equal(t, updatedRole.DisplayName, result.DisplayName)
		assert.Equal(t, updatedRole.Description, result.Description)
		assert.Equal(t, updatedRole.IsActive, result.IsActive)
		assert.True(t, result.UpdatedAt.After(createdRole.UpdatedAt))
	})

	t.Run("Update non-existent role", func(t *testing.T) {
		nonExistentRole := &models.Role{
			ID:          uuid.New(),
			Name:        "nonexistent",
			DisplayName: "Non Existent Role",
			Description: "This role does not exist",
			IsActive:    true,
		}

		_, err := repo.Update(ctx, nonExistentRole)
		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("Update role with duplicate name", func(t *testing.T) {
		// Create another role
		anotherRole := &models.Role{
			Name:        "another",
			DisplayName: "Another Role",
			Description: "Another role for testing",
			IsActive:    true,
		}

		_, err := repo.Create(ctx, anotherRole)
		require.NoError(t, err)

		// Try to update first role with second role's name
		duplicateNameRole := &models.Role{
			ID:          createdRole.ID,
			Name:        "another", // Duplicate name
			DisplayName: "Updated Role",
			Description: "Updated role description",
			IsActive:    createdRole.IsActive,
		}

		_, err = repo.Update(ctx, duplicateNameRole)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})
}

func TestRoleRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role first
	testRole := &models.Role{
		Name:        "deleterole",
		DisplayName: "Delete Role",
		Description: "Role for testing Delete",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	t.Run("Delete existing role", func(t *testing.T) {
		err := repo.Delete(ctx, createdRole.ID)
		require.NoError(t, err)

		// Verify role is deleted
		role, err := repo.GetByID(ctx, createdRole.ID)
		assert.Error(t, err)
		assert.Nil(t, role)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("Delete non-existent role", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestRoleRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create test roles
	roles := []*models.Role{
		{
			Name:        "listrole1",
			DisplayName: "List Role 1",
			Description: "First list role",
			IsActive:    true,
		},
		{
			Name:        "listrole2",
			DisplayName: "List Role 2",
			Description: "Second list role",
			IsActive:    true,
		},
		{
			Name:        "listrole3",
			DisplayName: "List Role 3",
			Description: "Third list role",
			IsActive:    false, // Inactive role
		},
	}

	for _, role := range roles {
		_, err := repo.Create(ctx, role)
		require.NoError(t, err)
	}

	t.Run("List all roles", func(t *testing.T) {
		result, err := repo.List(ctx, &models.RoleListParams{
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.GreaterOrEqual(t, len(result.Roles), 3)
		assert.GreaterOrEqual(t, result.Total, int64(3))
	})

	t.Run("List roles with pagination", func(t *testing.T) {
		// Test first page
		result1, err := repo.List(ctx, &models.RoleListParams{
			Limit:  2,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, result1.Roles, 2)

		// Test second page
		result2, err := repo.List(ctx, &models.RoleListParams{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result2.Roles), 1)
	})

	t.Run("List roles filtered by active status", func(t *testing.T) {
		result, err := repo.List(ctx, &models.RoleListParams{
			IsActive: boolPtr(true),
			Limit:    10,
			Offset:   0,
		})
		require.NoError(t, err)

		// All returned roles should be active
		for _, role := range result.Roles {
			assert.True(t, role.IsActive)
		}
	})

	t.Run("List roles filtered by search term", func(t *testing.T) {
		result, err := repo.List(ctx, &models.RoleListParams{
			Search: "Role 1",
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)

		// Should find roles with "Role 1" in name, display name, or description
		assert.GreaterOrEqual(t, len(result.Roles), 1)
	})

	t.Run("List roles ordered by name", func(t *testing.T) {
		result, err := repo.List(ctx, &models.RoleListParams{
			OrderBy: "name",
			Order:   "asc",
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)

		// Check if roles are ordered by name
		for i := 1; i < len(result.Roles); i++ {
			assert.LessOrEqual(t, result.Roles[i-1].Name, result.Roles[i].Name)
		}
	})
}

func TestRoleRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Get initial count
	initialCount, err := repo.Count(ctx, nil)
	require.NoError(t, err)

	// Create test roles
	roles := []*models.Role{
		{
			Name:        "countrole1",
			DisplayName: "Count Role 1",
			Description: "First count role",
			IsActive:    true,
		},
		{
			Name:        "countrole2",
			DisplayName: "Count Role 2",
			Description: "Second count role",
			IsActive:    true,
		},
	}

	for _, role := range roles {
		_, err := repo.Create(ctx, role)
		require.NoError(t, err)
	}

	t.Run("Count all roles", func(t *testing.T) {
		count, err := repo.Count(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, initialCount+2, count)
	})

	t.Run("Count active roles", func(t *testing.T) {
		params := &models.RoleListParams{IsActive: boolPtr(true)}
		count, err := repo.Count(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, initialCount+2, count) // Both created roles are active
	})

	t.Run("Count roles with search filter", func(t *testing.T) {
		params := &models.RoleListParams{Search: "countrole"}
		count, err := repo.Count(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func TestRoleRepository_AddPermissionToRole(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role
	testRole := &models.Role{
		Name:        "permission_role",
		DisplayName: "Permission Role",
		Description: "Role for testing permissions",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	t.Run("Add permission to role successfully", func(t *testing.T) {
		permission := "ci:read"
		err := repo.AddPermissionToRole(ctx, createdRole.ID, permission)
		require.NoError(t, err)

		// Verify permission was added
		permissions, err := repo.GetRolePermissions(ctx, createdRole.ID)
		require.NoError(t, err)
		assert.Contains(t, permissions, permission)
	})

	t.Run("Add duplicate permission to role", func(t *testing.T) {
		permission := "ci:write"
		
		// Add permission first time
		err := repo.AddPermissionToRole(ctx, createdRole.ID, permission)
		require.NoError(t, err)

		// Try to add same permission again
		err = repo.AddPermissionToRole(ctx, createdRole.ID, permission)
		assert.Error(t, err) // Should fail or be idempotent depending on implementation
	})

	t.Run("Add permission to non-existent role", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.AddPermissionToRole(ctx, nonExistentID, "ci:delete")
		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestRoleRepository_RemovePermissionFromRole(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role
	testRole := &models.Role{
		Name:        "remove_permission_role",
		DisplayName: "Remove Permission Role",
		Description: "Role for testing permission removal",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	// Add a permission first
	permission := "ci:update"
	err = repo.AddPermissionToRole(ctx, createdRole.ID, permission)
	require.NoError(t, err)

	t.Run("Remove permission from role successfully", func(t *testing.T) {
		err := repo.RemovePermissionFromRole(ctx, createdRole.ID, permission)
		require.NoError(t, err)

		// Verify permission was removed
		permissions, err := repo.GetRolePermissions(ctx, createdRole.ID)
		require.NoError(t, err)
		assert.NotContains(t, permissions, permission)
	})

	t.Run("Remove non-existent permission from role", func(t *testing.T) {
		err := repo.RemovePermissionFromRole(ctx, createdRole.ID, "nonexistent:permission")
		// This might not error depending on implementation
		// For now, we'll assume it doesn't error
		require.NoError(t, err)
	})

	t.Run("Remove permission from non-existent role", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.RemovePermissionFromRole(ctx, nonExistentID, "ci:delete")
		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestRoleRepository_GetRolePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role
	testRole := &models.Role{
		Name:        "get_permissions_role",
		DisplayName: "Get Permissions Role",
		Description: "Role for testing GetRolePermissions",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	// Add permissions
	permissions := []string{"ci:read", "ci:write", "ci:update", "ci:delete"}
	for _, permission := range permissions {
		err := repo.AddPermissionToRole(ctx, createdRole.ID, permission)
		require.NoError(t, err)
	}

	t.Run("Get all permissions for role", func(t *testing.T) {
		retrievedPermissions, err := repo.GetRolePermissions(ctx, createdRole.ID)
		require.NoError(t, err)
		assert.Len(t, retrievedPermissions, len(permissions))
		
		for _, permission := range permissions {
			assert.Contains(t, retrievedPermissions, permission)
		}
	})

	t.Run("Get permissions for non-existent role", func(t *testing.T) {
		nonExistentID := uuid.New()
		permissions, err := repo.GetRolePermissions(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, permissions)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("Get permissions for role with no permissions", func(t *testing.T) {
		// Create another role with no permissions
		emptyRole := &models.Role{
			Name:        "empty_permissions_role",
			DisplayName: "Empty Permissions Role",
			Description: "Role with no permissions",
			IsActive:    true,
		}

		createdEmptyRole, err := repo.Create(ctx, emptyRole)
		require.NoError(t, err)

		permissions, err := repo.GetRolePermissions(ctx, createdEmptyRole.ID)
		require.NoError(t, err)
		assert.Empty(t, permissions)
	})
}

func TestRoleRepository_HasPermission(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, pool := setupTestDatabase(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer pool.Close()

	repo := NewRoleRepository(pool)

	// Create a test role
	testRole := &models.Role{
		Name:        "has_permission_role",
		DisplayName: "Has Permission Role",
		Description: "Role for testing HasPermission",
		IsActive:    true,
	}

	createdRole, err := repo.Create(ctx, testRole)
	require.NoError(t, err)

	// Add a permission
	permission := "ci:read"
	err = repo.AddPermissionToRole(ctx, createdRole.ID, permission)
	require.NoError(t, err)

	t.Run("Check existing permission", func(t *testing.T) {
		hasPermission, err := repo.HasPermission(ctx, createdRole.ID, permission)
		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("Check non-existent permission", func(t *testing.T) {
		hasPermission, err := repo.HasPermission(ctx, createdRole.ID, "nonexistent:permission")
		require.NoError(t, err)
		assert.False(t, hasPermission)
	})

	t.Run("Check permission for non-existent role", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.HasPermission(ctx, nonExistentID, "ci:read")
		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
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
