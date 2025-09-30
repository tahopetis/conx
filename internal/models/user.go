package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Username       string    `json:"username" db:"username"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	FirstName      string    `json:"first_name" db:"first_name"`
	LastName       string    `json:"last_name" db:"last_name"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	IsVerified     bool      `json:"is_verified" db:"is_verified"`
	LastLoginAt    *time.Time `json:"last_login_at" db:"last_login_at"`
	PasswordChangedAt *time.Time `json:"password_changed_at" db:"password_changed_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy      uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy      uuid.UUID `json:"updated_by" db:"updated_by"`
}

// UserResponse represents a user response without sensitive data
type UserResponse struct {
	ID             uuid.UUID  `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	IsActive       bool       `json:"is_active"`
	IsVerified     bool       `json:"is_verified"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Roles          []string   `json:"roles"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name" validate:"required,min=1,max=50"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	FirstName *string `json:"first_name" validate:"omitempty,min=1,max=50"`
	LastName  *string `json:"last_name" validate:"omitempty,min=1,max=50"`
	Email     *string `json:"email" validate:"omitempty,email"`
	IsActive   *bool   `json:"is_active"`
}

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username  string `json:"username" validate:"required"`
	Password  string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int64      `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UserFilter represents filters for user queries
type UserFilter struct {
	Username   string    `json:"username,omitempty"`
	Email      string    `json:"email,omitempty"`
	IsActive   *bool     `json:"is_active,omitempty"`
	IsVerified *bool     `json:"is_verified,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

// UserList represents a paginated list of users
type UserList struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse(roles []string) UserResponse {
	return UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		IsActive:       u.IsActive,
		IsVerified:     u.IsVerified,
		LastLoginAt:    u.LastLoginAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
		Roles:          roles,
	}
}

// Validate validates the CreateUserRequest
func (r *CreateUserRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// Validate validates the UpdateUserRequest
func (r *UpdateUserRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// Validate validates the ChangePasswordRequest
func (r *ChangePasswordRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// Validate validates the LoginRequest
func (r *LoginRequest) Validate() error {
	// Additional validation can be added here
	return nil
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers      int `json:"total_users"`
	ActiveUsers     int `json:"active_users"`
	InactiveUsers   int `json:"inactive_users"`
	VerifiedUsers   int `json:"verified_users"`
	UnverifiedUsers int `json:"unverified_users"`
	UsersCreatedToday int `json:"users_created_today"`
	UsersCreatedThisWeek int `json:"users_created_this_week"`
	UsersCreatedThisMonth int `json:"users_created_this_month"`
}

// UserProfile represents a user's profile with additional information
type UserProfile struct {
	UserResponse
	PasswordStrength int    `json:"password_strength"`
	PasswordFeedback string `json:"password_feedback"`
	LastPasswordChange *time.Time `json:"last_password_change"`
	LoginCount       int    `json:"login_count"`
	FailedLoginAttempts int `json:"failed_login_attempts"`
	AccountLocked     bool   `json:"account_locked"`
	AccountLockedUntil *time.Time `json:"account_locked_until"`
	SecurityQuestions []SecurityQuestion `json:"security_questions"`
	TwoFactorEnabled bool `json:"two_factor_enabled"`
}

// SecurityQuestion represents a security question and answer
type SecurityQuestion struct {
	ID        uuid.UUID `json:"id"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"` // This should be encrypted
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserActivity represents user activity logs
type UserActivity struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Theme          string    `json:"theme"`
	Language       string    `json:"language"`
	Timezone       string    `json:"timezone"`
	Notifications  bool      `json:"notifications"`
	EmailNotifications bool `json:"email_notifications"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserSession represents an active user session
type UserSession struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	IsActive     bool      `json:"is_active"`
}

// UserFilterOptions represents filtering options for user queries
type UserFilterOptions struct {
	Search       string    `json:"search,omitempty"`
	Roles        []string  `json:"roles,omitempty"`
	Status       string    `json:"status,omitempty"`
	CreatedFrom  *time.Time `json:"created_from,omitempty"`
	CreatedTo    *time.Time `json:"created_to,omitempty"`
	UpdatedFrom  *time.Time `json:"updated_from,omitempty"`
	UpdatedTo    *time.Time `json:"updated_to,omitempty"`
	SortBy       string    `json:"sort_by,omitempty"`
	SortOrder    string    `json:"sort_order,omitempty"`
	Page         int       `json:"page,omitempty"`
	Size         int       `json:"size,omitempty"`
}

// UserBulkAction represents bulk actions on users
type UserBulkAction struct {
	UserIDs   []uuid.UUID `json:"user_ids"`
	Action    string      `json:"action"` // activate, deactivate, delete, assign_role, remove_role
	Role      string      `json:"role,omitempty"` // Required for role actions
	Reason    string      `json:"reason,omitempty"`
}

// UserBulkActionResult represents the result of bulk actions
type UserBulkActionResult struct {
	SuccessCount int                     `json:"success_count"`
	FailedCount  int                     `json:"failed_count"`
	Results      []UserBulkActionItemResult `json:"results"`
}

// UserBulkActionItemResult represents the result of a single bulk action
type UserBulkActionItemResult struct {
	UserID uuid.UUID `json:"user_id"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordResetConfirmRequest represents a password reset confirmation
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// PasswordResetResponse represents a password reset response
type PasswordResetResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// EmailVerificationRequest represents an email verification request
type EmailVerificationRequest struct {
	Token string `json:"token" validate:"required"`
}

// EmailVerificationResponse represents an email verification response
type EmailVerificationResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// ResendVerificationEmailRequest represents a request to resend verification email
type ResendVerificationEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResendVerificationEmailResponse represents a response to resend verification email
type ResendVerificationEmailResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
