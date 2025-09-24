package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/conx/cmdb/internal/database"
	"github.com/conx/cmdb/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRoleNotFound          = errors.New("role not found")
	ErrRoleAlreadyExists     = errors.New("role already exists")
	ErrPermissionNotFound    = errors.New("permission not found")
	ErrPermissionAlreadyExists = errors.New("permission already exists")
	ErrUserRoleNotFound      = errors.New("user role not found")
	ErrUserRoleAlreadyExists = errors.New("user role already exists")
	ErrRolePermissionNotFound = errors.New("role permission not found")
	ErrRolePermissionAlreadyExists = errors.New("role permission already exists")
	ErrCannotDeleteDefaultRole = errors.New("cannot delete default role")
	ErrCannotDeleteSystemPermission = errors.New("cannot delete system permission")
)

type RoleRepository struct {
	pool   *pgxpool.Pool
	logger *database.HealthCheck
}

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{
		pool:   pool,
		logger: &database.HealthCheck{Name: "role_repository"},
	}
}

// Role Management

// CreateRole creates a new role
func (r *RoleRepository) CreateRole(ctx context.Context, req *models.CreateRoleRequest) (*models.Role, error) {
	// Check if role already exists
	exists, err := r.RoleExists(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check role existence: %w", err)
	}
	if exists {
		return nil, ErrRoleAlreadyExists
	}

	role := &models.Role{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsDefault:   req.IsDefault,
		IsSystem:    req.IsSystem,
		IsActive:    req.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO roles (
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING 
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
	`

	err = r.pool.QueryRow(ctx, query,
		role.ID, role.Name, role.DisplayName, role.Description, role.IsDefault, role.IsSystem, role.IsActive, role.CreatedAt, role.UpdatedAt,
	).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// GetRoleByID retrieves a role by ID
func (r *RoleRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	query := `
		SELECT 
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
		FROM roles WHERE id = $1
	`

	role := &models.Role{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role by ID: %w", err)
	}

	return role, nil
}

// GetRoleByName retrieves a role by name
func (r *RoleRepository) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	query := `
		SELECT 
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
		FROM roles WHERE name = $1
	`

	role := &models.Role{}
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return role, nil
}

// UpdateRole updates a role
func (r *RoleRepository) UpdateRole(ctx context.Context, id uuid.UUID, req *models.UpdateRoleRequest) (*models.Role, error) {
	// Get existing role
	role, err := r.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cannot update system roles
	if role.IsSystem {
		return nil, ErrCannotDeleteDefaultRole
	}

	query := `
		UPDATE roles SET
			display_name = COALESCE($1, display_name),
			description = COALESCE($2, description),
			is_active = COALESCE($3, is_active),
			updated_at = $4
		WHERE id = $5
		RETURNING 
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
	`

	now := time.Now()
	err = r.pool.QueryRow(ctx, query,
		req.DisplayName, req.Description, req.IsActive, now, id,
	).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return role, nil
}

// DeleteRole deletes a role
func (r *RoleRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	// Get role to check if it's a default/system role
	role, err := r.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}

	if role.IsDefault || role.IsSystem {
		return ErrCannotDeleteDefaultRole
	}

	// Check if role is assigned to any users
	count, err := r.CountUsersByRole(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check role usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete role: %d users have this role", count)
	}

	query := `DELETE FROM roles WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRoleNotFound
	}

	return nil
}

// ListRoles retrieves a paginated list of roles
func (r *RoleRepository) ListRoles(ctx context.Context, filter *models.RoleFilterOptions, page, size int) (*models.RoleList, error) {
	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filter.Name != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR display_name ILIKE $%d)", argIndex, argIndex+1)
		searchPattern := "%" + filter.Name + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	if filter.IsActive != nil {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsDefault != nil {
		whereClause += fmt.Sprintf(" AND is_default = $%d", argIndex)
		args = append(args, *filter.IsDefault)
		argIndex++
	}

	if filter.IsSystem != nil {
		whereClause += fmt.Sprintf(" AND is_system = $%d", argIndex)
		args = append(args, *filter.IsSystem)
		argIndex++
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		allowedSortFields := map[string]bool{
			"name": true, "display_name": true, "created_at": true, "is_active": true,
		}
		if allowedSortFields[filter.SortBy] {
			orderBy = filter.SortBy
			if filter.SortOrder == "asc" {
				orderBy += " ASC"
			} else {
				orderBy += " DESC"
			}
		}
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM roles WHERE 1=1" + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count roles: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * size
	query := `
		SELECT 
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
		FROM roles WHERE 1=1` + whereClause + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argIndex, argIndex+1)
	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	// Convert to response format
	roleResponses := make([]models.RoleResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = role.ToResponse()
	}

	return &models.RoleList{
		Roles: roleResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetAllRoles retrieves all active roles
func (r *RoleRepository) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	query := `
		SELECT 
			id, name, display_name, description, is_default, is_system, is_active, created_at, updated_at
		FROM roles WHERE is_active = true ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// RoleExists checks if a role exists by name
func (r *RoleRepository) RoleExists(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check role existence: %w", err)
	}
	return exists, nil
}

// Permission Management

// CreatePermission creates a new permission
func (r *RoleRepository) CreatePermission(ctx context.Context, req *models.CreatePermissionRequest) (*models.Permission, error) {
	// Check if permission already exists
	exists, err := r.PermissionExists(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission existence: %w", err)
	}
	if exists {
		return nil, ErrPermissionAlreadyExists
	}

	permission := &models.Permission{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
		IsSystem:    req.IsSystem,
		IsActive:    req.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO permissions (
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING 
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
	`

	err = r.pool.QueryRow(ctx, query,
		permission.ID, permission.Name, permission.DisplayName, permission.Description, permission.Resource, permission.Action, permission.IsSystem, permission.IsActive, permission.CreatedAt, permission.UpdatedAt,
	).Scan(
		&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return permission, nil
}

// GetPermissionByID retrieves a permission by ID
func (r *RoleRepository) GetPermissionByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	query := `
		SELECT 
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
		FROM permissions WHERE id = $1
	`

	permission := &models.Permission{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("failed to get permission by ID: %w", err)
	}

	return permission, nil
}

// GetPermissionByName retrieves a permission by name
func (r *RoleRepository) GetPermissionByName(ctx context.Context, name string) (*models.Permission, error) {
	query := `
		SELECT 
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
		FROM permissions WHERE name = $1
	`

	permission := &models.Permission{}
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("failed to get permission by name: %w", err)
	}

	return permission, nil
}

// UpdatePermission updates a permission
func (r *RoleRepository) UpdatePermission(ctx context.Context, id uuid.UUID, req *models.UpdatePermissionRequest) (*models.Permission, error) {
	// Get existing permission
	permission, err := r.GetPermissionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cannot update system permissions
	if permission.IsSystem {
		return nil, ErrCannotDeleteSystemPermission
	}

	query := `
		UPDATE permissions SET
			display_name = COALESCE($1, display_name),
			description = COALESCE($2, description),
			is_active = COALESCE($3, is_active),
			updated_at = $4
		WHERE id = $5
		RETURNING 
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
	`

	now := time.Now()
	err = r.pool.QueryRow(ctx, query,
		req.DisplayName, req.Description, req.IsActive, now, id,
	).Scan(
		&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	return permission, nil
}

// DeletePermission deletes a permission
func (r *RoleRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	// Get permission to check if it's a system permission
	permission, err := r.GetPermissionByID(ctx, id)
	if err != nil {
		return err
	}

	if permission.IsSystem {
		return ErrCannotDeleteSystemPermission
	}

	// Check if permission is assigned to any roles
	count, err := r.CountRolesByPermission(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check permission usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete permission: %d roles have this permission", count)
	}

	query := `DELETE FROM permissions WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPermissionNotFound
	}

	return nil
}

// ListPermissions retrieves a paginated list of permissions
func (r *RoleRepository) ListPermissions(ctx context.Context, filter *models.PermissionFilterOptions, page, size int) (*models.PermissionList, error) {
	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filter.Name != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR display_name ILIKE $%d)", argIndex, argIndex+1)
		searchPattern := "%" + filter.Name + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	if filter.Resource != "" {
		whereClause += fmt.Sprintf(" AND resource = $%d", argIndex)
		args = append(args, filter.Resource)
		argIndex++
	}

	if filter.Action != "" {
		whereClause += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, filter.Action)
		argIndex++
	}

	if filter.IsActive != nil {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.IsSystem != nil {
		whereClause += fmt.Sprintf(" AND is_system = $%d", argIndex)
		args = append(args, *filter.IsSystem)
		argIndex++
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		allowedSortFields := map[string]bool{
			"name": true, "display_name": true, "resource": true, "created_at": true, "is_active": true,
		}
		if allowedSortFields[filter.SortBy] {
			orderBy = filter.SortBy
			if filter.SortOrder == "asc" {
				orderBy += " ASC"
			} else {
				orderBy += " DESC"
			}
		}
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM permissions WHERE 1=1" + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count permissions: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * size
	query := `
		SELECT 
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
		FROM permissions WHERE 1=1` + whereClause + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argIndex, argIndex+1)
	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	// Convert to response format
	permissionResponses := make([]models.PermissionResponse, len(permissions))
	for i, permission := range permissions {
		permissionResponses[i] = permission.ToResponse()
	}

	return &models.PermissionList{
		Permissions: permissionResponses,
		Total:       total,
		Page:        page,
		Size:        size,
	}, nil
}

// GetAllPermissions retrieves all active permissions
func (r *RoleRepository) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	query := `
		SELECT 
			id, name, display_name, description, resource, action, is_system, is_active, created_at, updated_at
		FROM permissions WHERE is_active = true ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// PermissionExists checks if a permission exists by name
func (r *RoleRepository) PermissionExists(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM permissions WHERE name = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check permission existence: %w", err)
	}
	return exists, nil
}

// User Role Management

// AssignRoleToUser assigns a role to a user
func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	// Check if user exists
	userRepo := NewUserRepository(r.pool)
	_, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if role exists
	_, err = r.GetRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Check if assignment already exists
	exists, err := r.UserRoleExists(ctx, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to check user role existence: %w", err)
	}
	if exists {
		return ErrUserRoleAlreadyExists
	}

	query := `
		INSERT INTO user_roles (user_id, role_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`

	_, err = r.pool.Exec(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	return nil
}

// RevokeRoleFromUser revokes a role from a user
func (r *RoleRepository) RevokeRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	// Check if assignment exists
	exists, err := r.UserRoleExists(ctx, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to check user role existence: %w", err)
	}
	if !exists {
		return ErrUserRoleNotFound
	}

	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	result, err := r.pool.Exec(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to revoke role from user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserRoleNotFound
	}

	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *RoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	query := `
		SELECT 
			r.id, r.name, r.display_name, r.description, r.is_default, r.is_system, r.is_active, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.is_active = true
		ORDER BY r.name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsDefault, &role.IsSystem, &role.IsActive, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserRoleNames retrieves role names for a user
func (r *RoleRepository) GetUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.is_active = true
		ORDER BY r.name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role names: %w", err)
	}
	defer rows.Close()

	var roleNames []string
	for rows.Next() {
		var roleName string
		err := rows.Scan(&roleName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role name: %w", err)
		}
		roleNames = append(roleNames, roleName)
	}

	return roleNames, nil
}

// UserRoleExists checks if a user has a specific role
func (r *RoleRepository) UserRoleExists(ctx context.Context, userID, roleID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, roleID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user role existence: %w", err)
	}
	return exists, nil
}

// CountUsersByRole counts users with a specific role
func (r *RoleRepository) CountUsersByRole(ctx context.Context, roleID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM user_roles WHERE role_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, roleID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}
	return count, nil
}

// Role Permission Management

// GrantPermissionToRole grants a permission to a role
func (r *RoleRepository) GrantPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	// Check if role exists
	_, err := r.GetRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Check if permission exists
	_, err = r.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return fmt.Errorf("failed to get permission: %w", err)
	}

	// Check if assignment already exists
	exists, err := r.RolePermissionExists(ctx, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to check role permission existence: %w", err)
	}
	if exists {
		return ErrRolePermissionAlreadyExists
	}

	query := `
		INSERT INTO role_permissions (role_id, permission_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`

	_, err = r.pool.Exec(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to grant permission to role: %w", err)
	}

	return nil
}

// RevokePermissionFromRole revokes a permission from a role
func (r *RoleRepository) RevokePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	// Check if assignment exists
	exists, err := r.RolePermissionExists(ctx, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to check role permission existence: %w", err)
	}
	if !exists {
		return ErrRolePermissionNotFound
	}

	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	result, err := r.pool.Exec(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to revoke permission from role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRolePermissionNotFound
	}

	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *RoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	query := `
		SELECT 
			p.id, p.name, p.display_name, p.description, p.resource, p.action, p.is_system, p.is_active, p.created_at, p.updated_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1 AND p.is_active = true
		ORDER BY p.name
	`

	rows, err := r.pool.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.DisplayName, &permission.Description, &permission.Resource, &permission.Action, &permission.IsSystem, &permission.IsActive, &permission.CreatedAt, &permission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// GetRolePermissionNames retrieves permission names for a role
func (r *RoleRepository) GetRolePermissionNames(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	query := `
		SELECT p.name
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1 AND p.is_active = true
		ORDER BY p.name
	`

	rows, err := r.pool.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permission names: %w", err)
	}
	defer rows.Close()

	var permissionNames []string
	for rows.Next() {
		var permissionName string
		err := rows.Scan(&permissionName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission name: %w", err)
		}
		permissionNames = append(permissionNames, permissionName)
	}

	return permissionNames, nil
}

// RolePermissionExists checks if a role has a specific permission
func (r *RoleRepository) RolePermissionExists(ctx context.Context, roleID, permissionID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM role_permissions WHERE role_id = $1 AND permission_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, roleID, permissionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check role permission existence: %w", err)
	}
	return exists, nil
}

// CountRolesByPermission counts roles with a specific permission
func (r *RoleRepository) CountRolesByPermission(ctx context.Context, permissionID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM role_permissions WHERE permission_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, permissionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count roles by permission: %w", err)
	}
	return count, nil
}

// Health checks the repository health
func (r *RoleRepository) Health() database.HealthCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.pool.Ping(ctx)
	if err != nil {
		r.logger.Status = "unhealthy"
		r.logger.Message = err.Error()
	} else {
		r.logger.Status = "healthy"
		r.logger.Message = ""
	}

	return *r.logger
}
