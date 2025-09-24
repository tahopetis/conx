package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a user role in the system
type Role struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	IsSystem    bool      `json:"is_system" db:"is_system"` // System roles cannot be deleted
	Priority    int       `json:"priority" db:"priority"`    // Higher priority roles override lower ones
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy   uuid.UUID `json:"updated_by" db:"updated_by"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Description string    `json:"description" db:"description"`
	Resource    string    `json:"resource" db:"resource"`      // API resource or feature
	Action      string    `json:"action" db:"action"`        // CRUD action or custom action
	IsActive    bool      `json:"is_active" db:"is_active"`
	IsSystem    bool      `json:"is_system" db:"is_system"`   // System permissions cannot be deleted
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy   uuid.UUID `json:"updated_by" db:"updated_by"`
}

// UserRole represents the relationship between users and roles
type UserRole struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	RoleID    uuid.UUID `json:"role_id" db:"role_id"`
	GrantedBy uuid.UUID `json:"granted_by" db:"granted_by"`
	GrantedAt time.Time `json:"granted_at" db:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RolePermission represents the relationship between roles and permissions
type RolePermission struct {
	ID           uuid.UUID `json:"id" db:"id"`
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	GrantedBy    uuid.UUID `json:"granted_by" db:"granted_by"`
	GrantedAt    time.Time `json:"granted_at" db:"granted_at"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateRoleRequest represents a request to create a new role
type CreateRoleRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=50,alphanum"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=500"`
	Priority    int    `json:"priority" validate:"min=0,max=100"`
}

// UpdateRoleRequest represents a request to update a role
type UpdateRoleRequest struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Priority    *int    `json:"priority" validate:"omitempty,min=0,max=100"`
	IsActive    *bool   `json:"is_active"`
}

// CreatePermissionRequest represents a request to create a new permission
type CreatePermissionRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100,alphanum"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=500"`
	Resource    string `json:"resource" validate:"required,min=1,max=50"`
	Action      string `json:"action" validate:"required,min=1,max=50"`
}

// UpdatePermissionRequest represents a request to update a permission
type UpdatePermissionRequest struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Resource    *string `json:"resource" validate:"omitempty,min=1,max=50"`
	Action      *string `json:"action" validate:"omitempty,min=1,max=50"`
	IsActive    *bool   `json:"is_active"`
}

// AssignRoleRequest represents a request to assign a role to a user
type AssignRoleRequest struct {
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	RoleID    uuid.UUID `json:"role_id" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// RevokeRoleRequest represents a request to revoke a role from a user
type RevokeRoleRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	RoleID uuid.UUID `json:"role_id" validate:"required"`
}

// GrantPermissionRequest represents a request to grant a permission to a role
type GrantPermissionRequest struct {
	RoleID       uuid.UUID `json:"role_id" validate:"required"`
	PermissionID uuid.UUID `json:"permission_id" validate:"required"`
}

// RevokePermissionRequest represents a request to revoke a permission from a role
type RevokePermissionRequest struct {
	RoleID       uuid.UUID `json:"role_id" validate:"required"`
	PermissionID uuid.UUID `json:"permission_id" validate:"required"`
}

// RoleResponse represents a role response
type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	IsSystem    bool      `json:"is_system"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Permissions []PermissionResponse `json:"permissions"`
	UserCount   int      `json:"user_count"`
}

// PermissionResponse represents a permission response
type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	IsActive    bool      `json:"is_active"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	RoleCount   int       `json:"role_count"`
}

// UserRoleResponse represents a user role response
type UserRoleResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	Role      RoleResponse `json:"role"`
	GrantedBy uuid.UUID  `json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	IsActive  bool       `json:"is_active"`
}

// RolePermissionResponse represents a role permission response
type RolePermissionResponse struct {
	ID           uuid.UUID         `json:"id"`
	RoleID       uuid.UUID         `json:"role_id"`
	PermissionID uuid.UUID         `json:"permission_id"`
	Permission  PermissionResponse `json:"permission"`
	GrantedBy    uuid.UUID         `json:"granted_by"`
	GrantedAt    time.Time         `json:"granted_at"`
	IsActive     bool              `json:"is_active"`
}

// RoleList represents a paginated list of roles
type RoleList struct {
	Roles []RoleResponse `json:"roles"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// PermissionList represents a paginated list of permissions
type PermissionList struct {
	Permissions []PermissionResponse `json:"permissions"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page"`
	Size        int                  `json:"size"`
}

// UserRoleList represents a paginated list of user roles
type UserRoleList struct {
	UserRoles []UserRoleResponse `json:"user_roles"`
	Total     int                `json:"total"`
	Page      int                `json:"page"`
	Size      int                `json:"size"`
}

// RolePermissionList represents a paginated list of role permissions
type RolePermissionList struct {
	RolePermissions []RolePermissionResponse `json:"role_permissions"`
	Total           int                        `json:"total"`
	Page            int                        `json:"page"`
	Size            int                        `json:"size"`
}

// ToResponse converts a Role to RoleResponse
func (r *Role) ToResponse(permissions []PermissionResponse, userCount int) RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		IsActive:    r.IsActive,
		IsSystem:    r.IsSystem,
		Priority:    r.Priority,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Permissions: permissions,
		UserCount:   userCount,
	}
}

// ToResponse converts a Permission to PermissionResponse
func (p *Permission) ToResponse(roleCount int) PermissionResponse {
	return PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
		Description: p.Description,
		Resource:    p.Resource,
		Action:      p.Action,
		IsActive:    p.IsActive,
		IsSystem:    p.IsSystem,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		RoleCount:   roleCount,
	}
}

// ToResponse converts a UserRole to UserRoleResponse
func (ur *UserRole) ToResponse(role RoleResponse) UserRoleResponse {
	return UserRoleResponse{
		ID:        ur.ID,
		UserID:    ur.UserID,
		RoleID:    ur.RoleID,
		Role:      role,
		GrantedBy: ur.GrantedBy,
		GrantedAt: ur.GrantedAt,
		ExpiresAt: ur.ExpiresAt,
		IsActive:  ur.IsActive,
	}
}

// ToResponse converts a RolePermission to RolePermissionResponse
func (rp *RolePermission) ToResponse(permission PermissionResponse) RolePermissionResponse {
	return RolePermissionResponse{
		ID:           rp.ID,
		RoleID:       rp.RoleID,
		PermissionID: rp.PermissionID,
		Permission:  permission,
		GrantedBy:    rp.GrantedBy,
		GrantedAt:    rp.GrantedAt,
		IsActive:     rp.IsActive,
	}
}

// Validate validates the CreateRoleRequest
func (r *CreateRoleRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// Validate validates the UpdateRoleRequest
func (r *UpdateRoleRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// Validate validates the CreatePermissionRequest
func (r *CreatePermissionRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// Validate validates the UpdatePermissionRequest
func (r *UpdatePermissionRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// RoleFilter represents filters for role queries
type RoleFilter struct {
	Name        string    `json:"name,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	IsActive    *bool     `json:"is_active,omitempty"`
	IsSystem    *bool     `json:"is_system,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// PermissionFilter represents filters for permission queries
type PermissionFilter struct {
	Name        string    `json:"name,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Resource    string    `json:"resource,omitempty"`
	Action      string    `json:"action,omitempty"`
	IsActive    *bool     `json:"is_active,omitempty"`
	IsSystem    *bool     `json:"is_system,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// UserRoleFilter represents filters for user role queries
type UserRoleFilter struct {
	UserID    uuid.UUID `json:"user_id,omitempty"`
	RoleID    uuid.UUID `json:"role_id,omitempty"`
	IsActive  *bool     `json:"is_active,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// RolePermissionFilter represents filters for role permission queries
type RolePermissionFilter struct {
	RoleID       uuid.UUID `json:"role_id,omitempty"`
	PermissionID uuid.UUID `json:"permission_id,omitempty"`
	IsActive     *bool     `json:"is_active,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// RoleStats represents role statistics
type RoleStats struct {
	TotalRoles       int `json:"total_roles"`
	ActiveRoles      int `json:"active_roles"`
	InactiveRoles    int `json:"inactive_roles"`
	SystemRoles      int `json:"system_roles"`
	CustomRoles      int `json:"custom_roles"`
	RolesCreatedToday int `json:"roles_created_today"`
}

// PermissionStats represents permission statistics
type PermissionStats struct {
	TotalPermissions       int `json:"total_permissions"`
	ActivePermissions      int `json:"active_permissions"`
	InactivePermissions    int `json:"inactive_permissions"`
	SystemPermissions      int `json:"system_permissions"`
	CustomPermissions      int `json:"custom_permissions"`
	PermissionsCreatedToday int `json:"permissions_created_today"`
}

// UserRoleStats represents user role statistics
type UserRoleStats struct {
	TotalUserRoles    int `json:"total_user_roles"`
	ActiveUserRoles   int `json:"active_user_roles"`
	ExpiredUserRoles  int `json:"expired_user_roles"`
	UsersWithRoles    int `json:"users_with_roles"`
	AverageRolesPerUser float64 `json:"average_roles_per_user"`
}

// RolePermissionStats represents role permission statistics
type RoleStats struct {
	TotalRolePermissions    int `json:"total_role_permissions"`
	ActiveRolePermissions   int `json:"active_role_permissions"`
	RolesWithPermissions    int `json:"roles_with_permissions"`
	AveragePermissionsPerRole float64 `json:"average_permissions_per_role"`
}

// DefaultRoles returns the default system roles
func DefaultRoles() []Role {
	return []Role{
		{
			Name:        "admin",
			DisplayName: "Administrator",
			Description: "Full system access with all permissions",
			IsActive:    true,
			IsSystem:    true,
			Priority:    100,
		},
		{
			Name:        "ci_manager",
			DisplayName: "CI Manager",
			Description: "Can manage configuration items and relationships",
			IsActive:    true,
			IsSystem:    true,
			Priority:    80,
		},
		{
			Name:        "viewer",
			DisplayName: "Viewer",
			Description: "Read-only access to configuration items",
			IsActive:    true,
			IsSystem:    true,
			Priority:    20,
		},
		{
			Name:        "auditor",
			DisplayName: "Auditor",
			Description: "Can view configuration items and audit logs",
			IsActive:    true,
			IsSystem:    true,
			Priority:    30,
		},
	}
}

// DefaultPermissions returns the default system permissions
func DefaultPermissions() []Permission {
	return []Permission{
		// CI permissions
		{Name: "ci:create", DisplayName: "Create CI", Description: "Create new configuration items", Resource: "ci", Action: "create"},
		{Name: "ci:read", DisplayName: "Read CI", Description: "Read configuration items", Resource: "ci", Action: "read"},
		{Name: "ci:update", DisplayName: "Update CI", Description: "Update configuration items", Resource: "ci", Action: "update"},
		{Name: "ci:delete", DisplayName: "Delete CI", Description: "Delete configuration items", Resource: "ci", Action: "delete"},
		
		// Relationship permissions
		{Name: "relationship:manage", DisplayName: "Manage Relationships", Description: "Create, update, and delete relationships", Resource: "relationship", Action: "manage"},
		
		// User permissions
		{Name: "user:create", DisplayName: "Create User", Description: "Create new users", Resource: "user", Action: "create"},
		{Name: "user:read", DisplayName: "Read User", Description: "Read user information", Resource: "user", Action: "read"},
		{Name: "user:update", DisplayName: "Update User", Description: "Update user information", Resource: "user", Action: "update"},
		{Name: "user:delete", DisplayName: "Delete User", Description: "Delete users", Resource: "user", Action: "delete"},
		{Name: "user:manage", DisplayName: "Manage Users", Description: "Full user management", Resource: "user", Action: "manage"},
		
		// Role permissions
		{Name: "role:create", DisplayName: "Create Role", Description: "Create new roles", Resource: "role", Action: "create"},
		{Name: "role:read", DisplayName: "Read Role", Description: "Read role information", Resource: "role", Action: "read"},
		{Name: "role:update", DisplayName: "Update Role", Description: "Update role information", Resource: "role", Action: "update"},
		{Name: "role:delete", DisplayName: "Delete Role", Description: "Delete roles", Resource: "role", Action: "delete"},
		{Name: "role:manage", DisplayName: "Manage Roles", Description: "Full role management", Resource: "role", Action: "manage"},
		
		// Permission permissions
		{Name: "permission:create", DisplayName: "Create Permission", Description: "Create new permissions", Resource: "permission", Action: "create"},
		{Name: "permission:read", DisplayName: "Read Permission", Description: "Read permission information", Resource: "permission", Action: "read"},
		{Name: "permission:update", DisplayName: "Update Permission", Description: "Update permission information", Resource: "permission", Action: "update"},
		{Name: "permission:delete", DisplayName: "Delete Permission", Description: "Delete permissions", Resource: "permission", Action: "delete"},
		{Name: "permission:manage", DisplayName: "Manage Permissions", Description: "Full permission management", Resource: "permission", Action: "manage"},
		
		// Audit permissions
		{Name: "audit_log:read", DisplayName: "Read Audit Log", Description: "Read audit logs", Resource: "audit_log", Action: "read"},
		
		// Import permissions
		{Name: "import:csv", DisplayName: "Import CSV", Description: "Import data from CSV files", Resource: "import", Action: "csv"},
		
		// Export permissions
		{Name: "export:csv", DisplayName: "Export CSV", Description: "Export data to CSV files", Resource: "export", Action: "csv"},
		{Name: "export:json", DisplayName: "Export JSON", Description: "Export data to JSON files", Resource: "export", Action: "json"},
		
		// System permissions
		{Name: "system:health", DisplayName: "System Health", Description: "View system health status", Resource: "system", Action: "health"},
		{Name: "system:metrics", DisplayName: "System Metrics", Description: "View system metrics", Resource: "system", Action: "metrics"},
		{Name: "system:config", DisplayName: "System Config", Description: "View and update system configuration", Resource: "system", Action: "config"},
	}
}

// DefaultRolePermissions returns the default role-permission mappings
func DefaultRolePermissions() map[string][]string {
	return map[string][]string{
		"admin": {
			"ci:create", "ci:read", "ci:update", "ci:delete",
			"relationship:manage",
			"user:create", "user:read", "user:update", "user:delete", "user:manage",
			"role:create", "role:read", "role:update", "role:delete", "role:manage",
			"permission:create", "permission:read", "permission:update", "permission:delete", "permission:manage",
			"audit_log:read",
			"import:csv",
			"export:csv", "export:json",
			"system:health", "system:metrics", "system:config",
		},
		"ci_manager": {
			"ci:create", "ci:read", "ci:update", "ci:delete",
			"relationship:manage",
			"import:csv",
			"export:csv",
		},
		"viewer": {
			"ci:read",
		},
		"auditor": {
			"ci:read",
			"audit_log:read",
		},
	}
}
