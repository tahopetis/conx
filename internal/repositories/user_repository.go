package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/conx/cmdb/internal/auth"
	"github.com/conx/cmdb/internal/database"
	"github.com/conx/cmdb/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserInactive      = errors.New("user is inactive")
)

type UserRepository struct {
	pool           *pgxpool.Pool
	passwordService *auth.PasswordService
	logger         *database.HealthCheck
}

func NewUserRepository(pool *pgxpool.Pool, passwordService *auth.PasswordService) *UserRepository {
	return &UserRepository{
		pool:           pool,
		passwordService: passwordService,
		logger:         database.HealthCheck{Name: "user_repository"},
	}
}

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, req *models.CreateUserRequest, createdBy uuid.UUID) (*models.User, error) {
	// Check if user already exists
	exists, err := r.UserExists(ctx, req.Username, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := r.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	now := time.Now()
	user := &models.User{
		ID:             uuid.New(),
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   passwordHash,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		IsActive:       true,
		IsVerified:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
		PasswordChangedAt: &now,
	}

	query := `
		INSERT INTO users (
			id, username, email, password_hash, first_name, last_name, 
			is_active, is_verified, created_at, updated_at, created_by, updated_by,
			password_changed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING 
			id, username, email, password_hash, first_name, last_name,
			is_active, is_verified, last_login_at, password_changed_at,
			created_at, updated_at, created_by, updated_by
	`

	err = r.pool.QueryRow(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.IsActive, user.IsVerified, user.CreatedAt, user.UpdatedAt, user.CreatedBy, user.UpdatedBy,
		user.PasswordChangedAt,
	).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, first_name, last_name,
			is_active, is_verified, last_login_at, password_changed_at,
			created_at, updated_at, created_by, updated_by
		FROM users WHERE id = $1
	`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, first_name, last_name,
			is_active, is_verified, last_login_at, password_changed_at,
			created_at, updated_at, created_by, updated_by
		FROM users WHERE username = $1
	`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT 
			id, username, email, password_hash, first_name, last_name,
			is_active, is_verified, last_login_at, password_changed_at,
			created_at, updated_at, created_by, updated_by
		FROM users WHERE email = $1
	`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// Update updates a user in the database
func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest, updatedBy uuid.UUID) (*models.User, error) {
	// Get existing user
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build update query
	query := `
		UPDATE users SET
			username = COALESCE($1, username),
			email = COALESCE($2, email),
			first_name = COALESCE($3, first_name),
			last_name = COALESCE($4, last_name),
			is_active = COALESCE($5, is_active),
			updated_at = $6,
			updated_by = $7
		WHERE id = $8
		RETURNING 
			id, username, email, password_hash, first_name, last_name,
			is_active, is_verified, last_login_at, password_changed_at,
			created_at, updated_at, created_by, updated_by
	`

	// Prepare values
	var username, email, firstName, lastName sql.NullString
	var isActive sql.NullBool

	if req.Username != nil {
		username.String = *req.Username
		username.Valid = true
	}
	if req.Email != nil {
		email.String = *req.Email
		email.Valid = true
	}
	if req.FirstName != nil {
		firstName.String = *req.FirstName
		firstName.Valid = true
	}
	if req.LastName != nil {
		lastName.String = *req.LastName
		lastName.Valid = true
	}
	if req.IsActive != nil {
		isActive.Bool = *req.IsActive
		isActive.Valid = true
	}

	now := time.Now()

	err = r.pool.QueryRow(ctx, query,
		username, email, firstName, lastName, isActive,
		now, updatedBy, id,
	).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.PasswordChangedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// Delete deletes a user from the database
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, filter *models.UserFilterOptions, page, size int) (*models.UserList, error) {
	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", 
			argIndex, argIndex+1, argIndex+2, argIndex+3)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		argIndex += 4
	}

	if filter.Status != "" {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, filter.Status == "active")
		argIndex++
	}

	if filter.CreatedFrom != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, filter.CreatedFrom)
		argIndex++
	}

	if filter.CreatedTo != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, filter.CreatedTo)
		argIndex++
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		allowedSortFields := map[string]bool{
			"username": true, "email": true, "first_name": true, "last_name": true,
			"created_at": true, "updated_at": true, "last_login_at": true,
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
	countQuery := "SELECT COUNT(*) FROM users WHERE 1=1" + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * size
	query := `
		SELECT 
			id, username, email, password_hash, first_name, last_name,
			is_active, is_verified, last_login_at, password_changed_at,
			created_at, updated_at, created_by, updated_by
		FROM users WHERE 1=1` + whereClause + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argIndex, argIndex+1)
	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
			&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.PasswordChangedAt,
			&user.CreatedAt, &user.UpdatedAt, &user.CreatedBy, &user.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		// TODO: Get user roles from role repository
		userResponses[i] = user.ToResponse([]string{})
	}

	return &models.UserList{
		Users: userResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// Authenticate authenticates a user with username and password
func (r *UserRepository) Authenticate(ctx context.Context, username, password string) (*models.User, error) {
	// Get user by username
	user, err := r.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user for authentication: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Verify password
	valid, err := r.passwordService.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidPassword
	}

	// Update last login time
	now := time.Now()
	updateQuery := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err = r.pool.Exec(ctx, updateQuery, now, user.ID)
	if err != nil {
		// Log error but don't fail authentication
		fmt.Printf("failed to update last login time: %v\n", err)
	}

	user.LastLoginAt = &now
	return user, nil
}

// ChangePassword changes a user's password
func (r *UserRepository) ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error {
	// Get user
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify current password
	valid, err := r.passwordService.VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to verify current password: %w", err)
	}
	if !valid {
		return ErrInvalidPassword
	}

	// Hash new password
	newPasswordHash, err := r.passwordService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	now := time.Now()
	query := `
		UPDATE users SET 
			password_hash = $1,
			password_changed_at = $2,
			updated_at = $3
		WHERE id = $4
	`

	_, err = r.pool.Exec(ctx, query, newPasswordHash, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ResetPassword resets a user's password (for forgot password flow)
func (r *UserRepository) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	// Hash new password
	newPasswordHash, err := r.passwordService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	now := time.Now()
	query := `
		UPDATE users SET 
			password_hash = $1,
			password_changed_at = $2,
			updated_at = $3
		WHERE id = $4
	`

	_, err = r.pool.Exec(ctx, query, newPasswordHash, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

// SetActive sets a user's active status
func (r *UserRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, active, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set user active status: %w", err)
	}
	return nil
}

// SetVerified sets a user's verified status
func (r *UserRepository) SetVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	query := `UPDATE users SET is_verified = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, verified, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set user verified status: %w", err)
	}
	return nil
}

// UserExists checks if a user exists with the given username or email
func (r *UserRepository) UserExists(ctx context.Context, username, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 OR email = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, username, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}

// GetStats retrieves user statistics
func (r *UserRepository) GetStats(ctx context.Context) (*models.UserStats, error) {
	stats := &models.UserStats{}

	// Get total users
	query := `SELECT COUNT(*) FROM users`
	err := r.pool.QueryRow(ctx, query).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get total users: %w", err)
	}

	// Get active users
	query = `SELECT COUNT(*) FROM users WHERE is_active = true`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	// Get inactive users
	query = `SELECT COUNT(*) FROM users WHERE is_active = false`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.InactiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive users: %w", err)
	}

	// Get verified users
	query = `SELECT COUNT(*) FROM users WHERE is_verified = true`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.VerifiedUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get verified users: %w", err)
	}

	// Get unverified users
	query = `SELECT COUNT(*) FROM users WHERE is_verified = false`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.UnverifiedUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get unverified users: %w", err)
	}

	// Get users created today
	query = `SELECT COUNT(*) FROM users WHERE DATE(created_at) = CURRENT_DATE`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.UsersCreatedToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get users created today: %w", err)
	}

	// Get users created this week
	query = `SELECT COUNT(*) FROM users WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE)`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.UsersCreatedThisWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get users created this week: %w", err)
	}

	// Get users created this month
	query = `SELECT COUNT(*) FROM users WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE)`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.UsersCreatedThisMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get users created this month: %w", err)
	}

	return stats, nil
}

// Health checks the repository health
func (r *UserRepository) Health() database.HealthCheck {
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

	return r.logger
}
