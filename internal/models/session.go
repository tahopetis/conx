package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidUserID         = errors.New("invalid user ID")
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrInvalidIPAddress      = errors.New("invalid IP address")
	ErrInvalidExpirationTime = errors.New("invalid expiration time")
	ErrInvalidRevocationTime = errors.New("invalid revocation time")
	ErrInvalidSessionID      = errors.New("invalid session ID")
	ErrInvalidAction         = errors.New("invalid action")
)

// Session represents a user session
type Session struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	Token        string     `json:"token" db:"token"`
	RefreshToken string     `json:"refresh_token" db:"refresh_token"`
	IPAddress    string     `json:"ip_address" db:"ip_address"`
	UserAgent    string     `json:"user_agent" db:"user_agent"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	LastActiveAt time.Time  `json:"last_active_at" db:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	IsActive     bool       `json:"is_active" db:"is_active"`
}

// SessionResponse represents a session for API responses
type SessionResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time  `json:"expires_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at"`
	IsActive     bool      `json:"is_active"`
}

// ToResponse converts a Session to SessionResponse
func (s *Session) ToResponse() SessionResponse {
	return SessionResponse{
		ID:           s.ID,
		UserID:       s.UserID,
		IPAddress:    s.IPAddress,
		UserAgent:    s.UserAgent,
		ExpiresAt:    s.ExpiresAt,
		LastActiveAt: s.LastActiveAt,
		CreatedAt:    s.CreatedAt,
		IsActive:     s.IsActive,
	}
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked checks if the session is revoked
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil && !s.RevokedAt.IsZero()
}

// IsValid checks if the session is valid (not expired and not revoked)
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked() && s.IsActive
}

// SessionFilterOptions represents filtering options for listing sessions
type SessionFilterOptions struct {
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	IsActive   *bool      `json:"is_active,omitempty"`
	ExpiresFrom *time.Time `json:"expires_from,omitempty"`
	ExpiresTo   *time.Time `json:"expires_to,omitempty"`
	CreatedFrom *time.Time `json:"created_from,omitempty"`
	CreatedTo   *time.Time `json:"created_to,omitempty"`
	Search      string     `json:"search,omitempty"`
	SortBy      string     `json:"sort_by,omitempty"`
	SortOrder   string     `json:"sort_order,omitempty"`
}

// SessionList represents a paginated list of sessions
type SessionList struct {
	Sessions []SessionResponse `json:"sessions"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	Size     int               `json:"size"`
}

// SessionStats represents session statistics
type SessionStats struct {
	TotalSessions      int     `json:"total_sessions"`
	ActiveSessions     int     `json:"active_sessions"`
	ExpiredSessions    int     `json:"expired_sessions"`
	RevokedSessions    int     `json:"revoked_sessions"`
	SessionsToday      int     `json:"sessions_today"`
	SessionsThisWeek   int     `json:"sessions_this_week"`
	SessionsThisMonth  int     `json:"sessions_this_month"`
	AverageSessionTime float64 `json:"average_session_time_hours"`
}

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	UserID       uuid.UUID `json:"user_id" validate:"required"`
	Token        string    `json:"token" validate:"required"`
	RefreshToken string    `json:"refresh_token" validate:"required"`
	IPAddress    string    `json:"ip_address" validate:"required"`
	UserAgent    string    `json:"user_agent" validate:"required"`
	ExpiresAt    time.Time `json:"expires_at" validate:"required"`
}

// Validate validates the CreateSessionRequest
func (r *CreateSessionRequest) Validate() error {
	if r.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if r.Token == "" {
		return ErrInvalidToken
	}
	if r.RefreshToken == "" {
		return ErrInvalidRefreshToken
	}
	if r.IPAddress == "" {
		return ErrInvalidIPAddress
	}
	if r.ExpiresAt.IsZero() || time.Now().After(r.ExpiresAt) {
		return ErrInvalidExpirationTime
	}
	return nil
}

// UpdateSessionRequest represents a request to update a session
type UpdateSessionRequest struct {
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// Validate validates the UpdateSessionRequest
func (r *UpdateSessionRequest) Validate() error {
	if r.ExpiresAt != nil && (r.ExpiresAt.IsZero() || time.Now().After(*r.ExpiresAt)) {
		return ErrInvalidExpirationTime
	}
	if r.RevokedAt != nil && r.RevokedAt.IsZero() {
		return ErrInvalidRevocationTime
	}
	return nil
}

// RevokeSessionRequest represents a request to revoke a session
type RevokeSessionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// Validate validates the RevokeSessionRequest
func (r *RevokeSessionRequest) Validate() error {
	// No validation required for revocation
	return nil
}

// SessionActivity represents user session activity
type SessionActivity struct {
	ID        uuid.UUID `json:"id" db:"id"`
	SessionID uuid.UUID `json:"session_id" db:"session_id"`
	Action    string    `json:"action" db:"action"`
	Details   string    `json:"details" db:"details"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SessionActivityResponse represents session activity for API responses
type SessionActivityResponse struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts SessionActivity to SessionActivityResponse
func (a *SessionActivity) ToResponse() SessionActivityResponse {
	return SessionActivityResponse{
		ID:        a.ID,
		SessionID: a.SessionID,
		Action:    a.Action,
		Details:   a.Details,
		IPAddress: a.IPAddress,
		UserAgent: a.UserAgent,
		CreatedAt: a.CreatedAt,
	}
}

// SessionActivityFilterOptions represents filtering options for session activities
type SessionActivityFilterOptions struct {
	SessionID  *uuid.UUID `json:"session_id,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Action     string     `json:"action,omitempty"`
	From       *time.Time `json:"from,omitempty"`
	To         *time.Time `json:"to,omitempty"`
	Search     string     `json:"search,omitempty"`
	SortBy     string     `json:"sort_by,omitempty"`
	SortOrder  string     `json:"sort_order,omitempty"`
}

// SessionActivityList represents a paginated list of session activities
type SessionActivityList struct {
	Activities []SessionActivityResponse `json:"activities"`
	Total      int                        `json:"total"`
	Page       int                        `json:"page"`
	Size       int                        `json:"size"`
}

// CreateSessionActivityRequest represents a request to create session activity
type CreateSessionActivityRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"required"`
	Action    string    `json:"action" validate:"required"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address" validate:"required"`
	UserAgent string    `json:"user_agent" validate:"required"`
}

// Validate validates the CreateSessionActivityRequest
func (r *CreateSessionActivityRequest) Validate() error {
	if r.SessionID == uuid.Nil {
		return ErrInvalidSessionID
	}
	if r.Action == "" {
		return ErrInvalidAction
	}
	if r.IPAddress == "" {
		return ErrInvalidIPAddress
	}
	return nil
}

// Constants for session actions
const (
	SessionActionCreated    = "created"
	SessionActionRefreshed  = "refreshed"
	SessionActionAccessed   = "accessed"
	SessionActionRevoked    = "revoked"
	SessionActionExpired    = "expired"
	SessionActionLoggedOut  = "logged_out"
)

// Constants for session validation
const (
	DefaultSessionTTL       = 24 * time.Hour
	MaxSessionTTL          = 30 * 24 * time.Hour
	MinSessionTTL          = 15 * time.Minute
	MaxConcurrentSessions  = 5
	SessionCleanupInterval = 1 * time.Hour
)
