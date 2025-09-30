package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connect/internal/database"
	"connect/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired      = errors.New("session expired")
	ErrSessionRevoked      = errors.New("session revoked")
	ErrSessionInvalid      = errors.New("session invalid")
	ErrTooManySessions     = errors.New("too many concurrent sessions")
	ErrSessionAlreadyExists = errors.New("session already exists")
)

type SessionRepository struct {
	pool   *pgxpool.Pool
	logger *database.HealthCheck
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		pool:   pool,
		logger: &database.HealthCheck{Name: "session_repository"},
	}
}

// Create creates a new session in the database
func (r *SessionRepository) Create(ctx context.Context, req *models.CreateSessionRequest) (*models.Session, error) {
	// Check if user already has too many active sessions
	activeSessions, err := r.CountActiveSessions(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active sessions: %w", err)
	}

	if activeSessions >= models.MaxConcurrentSessions {
		// Revoke oldest active session
		err := r.RevokeOldestActiveSession(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to revoke oldest session: %w", err)
		}
	}

	// Create session
	now := time.Now()
	session := &models.Session{
		ID:           uuid.New(),
		UserID:       req.UserID,
		Token:        req.Token,
		RefreshToken: req.RefreshToken,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ExpiresAt:    req.ExpiresAt,
		LastActiveAt: now,
		CreatedAt:    now,
		IsActive:     true,
	}

	query := `
		INSERT INTO sessions (
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
	`

	err = r.pool.QueryRow(ctx, query,
		session.ID, session.UserID, session.Token, session.RefreshToken, session.IPAddress, session.UserAgent,
		session.ExpiresAt, session.LastActiveAt, session.CreatedAt, session.IsActive,
	).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Log session creation activity
	activity := &models.CreateSessionActivityRequest{
		SessionID: session.ID,
		Action:    models.SessionActionCreated,
		Details:   "Session created",
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	}
	_, err = r.CreateActivity(ctx, activity)
	if err != nil {
		// Log error but don't fail session creation
		fmt.Printf("failed to log session activity: %v\n", err)
	}

	return session, nil
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	query := `
		SELECT 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
		FROM sessions WHERE id = $1
	`

	session := &models.Session{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}

	return session, nil
}

// GetByToken retrieves a session by token
func (r *SessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	query := `
		SELECT 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
		FROM sessions WHERE token = $1
	`

	session := &models.Session{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	return session, nil
}

// GetByRefreshToken retrieves a session by refresh token
func (r *SessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	query := `
		SELECT 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
		FROM sessions WHERE refresh_token = $1
	`

	session := &models.Session{}
	err := r.pool.QueryRow(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by refresh token: %w", err)
	}

	return session, nil
}

// Update updates a session in the database
func (r *SessionRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateSessionRequest) (*models.Session, error) {
	// Get existing session
	session, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build update query
	query := `
		UPDATE sessions SET
			last_active_at = COALESCE($1, last_active_at),
			expires_at = COALESCE($2, expires_at),
			is_active = COALESCE($3, is_active),
			revoked_at = COALESCE($4, revoked_at),
			updated_at = $5
		WHERE id = $6
		RETURNING 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
	`

	now := time.Now()
	err = r.pool.QueryRow(ctx, query,
		req.LastActiveAt, req.ExpiresAt, req.IsActive, req.RevokedAt, now, id,
	).Scan(
		&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// Delete deletes a session from the database
func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// Revoke revokes a session
func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID, reason string) error {
	session, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if session.IsRevoked() {
		return ErrSessionRevoked
	}

	now := time.Now()
	query := `
		UPDATE sessions SET
			is_active = false,
			revoked_at = $1,
			updated_at = $2
		WHERE id = $3
	`

	_, err = r.pool.Exec(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Log revocation activity
	activity := &models.CreateSessionActivityRequest{
		SessionID: session.ID,
		Action:    models.SessionActionRevoked,
		Details:   fmt.Sprintf("Session revoked: %s", reason),
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
	}
	_, err = r.CreateActivity(ctx, activity)
	if err != nil {
		// Log error but don't fail revocation
		fmt.Printf("failed to log session revocation activity: %v\n", err)
	}

	return nil
}

// ValidateSession validates a session and updates last active time
func (r *SessionRepository) ValidateSession(ctx context.Context, token string, ipAddress, userAgent string) (*models.Session, error) {
	session, err := r.GetByToken(ctx, token)
	if err != nil {
		if err == ErrSessionNotFound {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session for validation: %w", err)
	}

	// Check if session is valid
	if !session.IsValid() {
		if session.IsExpired() {
			// Mark as inactive if expired
			_, err := r.Update(ctx, session.ID, &models.UpdateSessionRequest{
				IsActive: boolPtr(false),
			})
			if err != nil {
				fmt.Printf("failed to mark expired session as inactive: %v\n", err)
			}
			return nil, ErrSessionExpired
		}
		if session.IsRevoked() {
			return nil, ErrSessionRevoked
		}
		if !session.IsActive {
			return nil, ErrSessionInvalid
		}
	}

	// Update last active time
	_, err = r.Update(ctx, session.ID, &models.UpdateSessionRequest{
		LastActiveAt: timePtr(time.Now()),
	})
	if err != nil {
		fmt.Printf("failed to update session last active time: %v\n", err)
	}

	// Log access activity
	activity := &models.CreateSessionActivityRequest{
		SessionID: session.ID,
		Action:    models.SessionActionAccessed,
		Details:   "Session accessed",
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	_, err = r.CreateActivity(ctx, activity)
	if err != nil {
		// Log error but don't fail validation
		fmt.Printf("failed to log session access activity: %v\n", err)
	}

	return session, nil
}

// List retrieves a paginated list of sessions
func (r *SessionRepository) List(ctx context.Context, filter *models.SessionFilterOptions, page, size int) (*models.SessionList, error) {
	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != nil {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.IsActive != nil {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.ExpiresFrom != nil {
		whereClause += fmt.Sprintf(" AND expires_at >= $%d", argIndex)
		args = append(args, filter.ExpiresFrom)
		argIndex++
	}

	if filter.ExpiresTo != nil {
		whereClause += fmt.Sprintf(" AND expires_at <= $%d", argIndex)
		args = append(args, filter.ExpiresTo)
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

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (ip_address ILIKE $%d OR user_agent ILIKE $%d)", argIndex, argIndex+1)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		allowedSortFields := map[string]bool{
			"created_at": true, "last_active_at": true, "expires_at": true,
			"ip_address": true, "user_agent": true,
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
	countQuery := "SELECT COUNT(*) FROM sessions WHERE 1=1" + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * size
	query := `
		SELECT 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
		FROM sessions WHERE 1=1` + whereClause + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argIndex, argIndex+1)
	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var session models.Session
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
			&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	// Convert to response format
	sessionResponses := make([]models.SessionResponse, len(sessions))
	for i, session := range sessions {
		sessionResponses[i] = session.ToResponse()
	}

	return &models.SessionList{
		Sessions: sessionResponses,
		Total:    total,
		Page:     page,
		Size:     size,
	}, nil
}

// GetByUserID retrieves all sessions for a user
func (r *SessionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	query := `
		SELECT 
			id, user_id, token, refresh_token, ip_address, user_agent,
			expires_at, last_active_at, created_at, revoked_at, is_active
		FROM sessions WHERE user_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var session models.Session
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token, &session.RefreshToken, &session.IPAddress, &session.UserAgent,
			&session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt, &session.RevokedAt, &session.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// RevokeAllByUserID revokes all sessions for a user
func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID, reason string) error {
	sessions, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	now := time.Now()
	for _, session := range sessions {
		if session.IsActive && !session.IsRevoked() {
			err := r.Revoke(ctx, session.ID, reason)
			if err != nil {
				fmt.Printf("failed to revoke session %s: %v\n", session.ID, err)
			}
		}
	}

	return nil
}

// CountActiveSessions counts active sessions for a user
func (r *SessionRepository) CountActiveSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND is_active = true AND revoked_at IS NULL AND expires_at > NOW()`
	
	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}

	return count, nil
}

// RevokeOldestActiveSession revokes the oldest active session for a user
func (r *SessionRepository) RevokeOldestActiveSession(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE sessions 
		SET is_active = false, revoked_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM sessions 
			WHERE user_id = $1 AND is_active = true AND revoked_at IS NULL AND expires_at > NOW()
			ORDER BY created_at ASC 
			LIMIT 1
		)
		RETURNING id
	`

	var sessionID uuid.UUID
	err := r.pool.QueryRow(ctx, query, userID).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // No active sessions to revoke
		}
		return fmt.Errorf("failed to revoke oldest session: %w", err)
	}

	// Log revocation activity
	activity := &models.CreateSessionActivityRequest{
		SessionID: sessionID,
		Action:    models.SessionActionRevoked,
		Details:   "Session revoked due to concurrent session limit",
		IPAddress: "system",
		UserAgent: "system",
	}
	_, err = r.CreateActivity(ctx, activity)
	if err != nil {
		// Log error but don't fail revocation
		fmt.Printf("failed to log session revocation activity: %v\n", err)
	}

	return nil
}

// CleanupExpiredSessions cleans up expired sessions
func (r *SessionRepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	query := `
		UPDATE sessions 
		SET is_active = false, updated_at = NOW()
		WHERE expires_at <= NOW() AND is_active = true
	`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// GetStats retrieves session statistics
func (r *SessionRepository) GetStats(ctx context.Context) (*models.SessionStats, error) {
	stats := &models.SessionStats{}

	// Get total sessions
	query := `SELECT COUNT(*) FROM sessions`
	err := r.pool.QueryRow(ctx, query).Scan(&stats.TotalSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total sessions: %w", err)
	}

	// Get active sessions
	query = `SELECT COUNT(*) FROM sessions WHERE is_active = true AND revoked_at IS NULL AND expires_at > NOW()`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.ActiveSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	// Get expired sessions
	query = `SELECT COUNT(*) FROM sessions WHERE expires_at <= NOW()`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.ExpiredSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired sessions: %w", err)
	}

	// Get revoked sessions
	query = `SELECT COUNT(*) FROM sessions WHERE revoked_at IS NOT NULL`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.RevokedSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get revoked sessions: %w", err)
	}

	// Get sessions created today
	query = `SELECT COUNT(*) FROM sessions WHERE DATE(created_at) = CURRENT_DATE`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.SessionsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions created today: %w", err)
	}

	// Get sessions created this week
	query = `SELECT COUNT(*) FROM sessions WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE)`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.SessionsThisWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions created this week: %w", err)
	}

	// Get sessions created this month
	query = `SELECT COUNT(*) FROM sessions WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE)`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.SessionsThisMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions created this month: %w", err)
	}

	// Get average session time (in hours)
	query = `
		SELECT AVG(EXTRACT(EPOCH FROM (COALESCE(revoked_at, expires_at) - created_at)) / 3600)
		FROM sessions 
		WHERE revoked_at IS NOT NULL OR expires_at <= NOW()
	`
	err = r.pool.QueryRow(ctx, query).Scan(&stats.AverageSessionTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get average session time: %w", err)
	}

	return stats, nil
}

// CreateActivity creates a session activity record
func (r *SessionRepository) CreateActivity(ctx context.Context, req *models.CreateSessionActivityRequest) (*models.SessionActivity, error) {
	activity := &models.SessionActivity{
		ID:        uuid.New(),
		SessionID: req.SessionID,
		Action:    req.Action,
		Details:   req.Details,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO session_activities (
			id, session_id, action, details, ip_address, user_agent, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING 
			id, session_id, action, details, ip_address, user_agent, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		activity.ID, activity.SessionID, activity.Action, activity.Details, activity.IPAddress, activity.UserAgent, activity.CreatedAt,
	).Scan(
		&activity.ID, &activity.SessionID, &activity.Action, &activity.Details, &activity.IPAddress, &activity.UserAgent, &activity.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session activity: %w", err)
	}

	return activity, nil
}

// ListActivities retrieves a paginated list of session activities
func (r *SessionRepository) ListActivities(ctx context.Context, filter *models.SessionActivityFilterOptions, page, size int) (*models.SessionActivityList, error) {
	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if filter.SessionID != nil {
		whereClause += fmt.Sprintf(" AND sa.session_id = $%d", argIndex)
		args = append(args, *filter.SessionID)
		argIndex++
	}

	if filter.UserID != nil {
		whereClause += fmt.Sprintf(" AND s.user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.Action != "" {
		whereClause += fmt.Sprintf(" AND sa.action = $%d", argIndex)
		args = append(args, filter.Action)
		argIndex++
	}

	if filter.From != nil {
		whereClause += fmt.Sprintf(" AND sa.created_at >= $%d", argIndex)
		args = append(args, filter.From)
		argIndex++
	}

	if filter.To != nil {
		whereClause += fmt.Sprintf(" AND sa.created_at <= $%d", argIndex)
		args = append(args, filter.To)
		argIndex++
	}

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (sa.details ILIKE $%d OR sa.ip_address ILIKE $%d OR sa.user_agent ILIKE $%d)", argIndex, argIndex+1, argIndex+2)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	// Build ORDER BY clause
	orderBy := "sa.created_at DESC"
	if filter.SortBy != "" {
		allowedSortFields := map[string]bool{
			"created_at": true, "action": true, "ip_address": true,
		}
		if allowedSortFields[filter.SortBy] {
			orderBy = "sa." + filter.SortBy
			if filter.SortOrder == "asc" {
				orderBy += " ASC"
			} else {
				orderBy += " DESC"
			}
		}
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM session_activities sa JOIN sessions s ON sa.session_id = s.id WHERE 1=1` + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count session activities: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * size
	query := `
		SELECT 
			sa.id, sa.session_id, sa.action, sa.details, sa.ip_address, sa.user_agent, sa.created_at
		FROM session_activities sa
		JOIN sessions s ON sa.session_id = s.id
		WHERE 1=1` + whereClause + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argIndex, argIndex+1)
	args = append(args, size, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list session activities: %w", err)
	}
	defer rows.Close()

	var activities []models.SessionActivity
	for rows.Next() {
		var activity models.SessionActivity
		err := rows.Scan(
			&activity.ID, &activity.SessionID, &activity.Action, &activity.Details, &activity.IPAddress, &activity.UserAgent, &activity.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session activity: %w", err)
		}
		activities = append(activities, activity)
	}

	// Convert to response format
	activityResponses := make([]models.SessionActivityResponse, len(activities))
	for i, activity := range activities {
		activityResponses[i] = activity.ToResponse()
	}

	return &models.SessionActivityList{
		Activities: activityResponses,
		Total:      total,
		Page:       page,
		Size:       size,
	}, nil
}

// Health checks the repository health
func (r *SessionRepository) Health() database.HealthCheck {
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

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
