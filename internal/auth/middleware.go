package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/conx/cmdb/internal/logger"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoToken          = errors.New("no authorization token provided")
	ErrInvalidTokenType = errors.New("invalid token type")
	ErrUnauthorized      = errors.New("unauthorized access")
)

type contextKey string

const (
	UserContextKey   contextKey = "user"
	RolesContextKey  contextKey = "roles"
	TokenContextKey  contextKey = "token"
)

type AuthMiddleware struct {
	jwtService     *JWTService
	logger         *logger.Logger
	excludePaths   map[string]bool
	optionalPaths  map[string]bool
}

type AuthConfig struct {
	JWTService     *JWTService
	Logger         *logger.Logger
	ExcludePaths   []string
	OptionalPaths  []string
}

func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	excludePaths := make(map[string]bool)
	for _, path := range config.ExcludePaths {
		excludePaths[path] = true
	}

	optionalPaths := make(map[string]bool)
	for _, path := range config.OptionalPaths {
		optionalPaths[path] = true
	}

	return &AuthMiddleware{
		jwtService:    config.JWTService,
		logger:        config.Logger,
		excludePaths:  excludePaths,
		optionalPaths: optionalPaths,
	}
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is excluded from authentication
		if m.isPathExcluded(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		tokenString, err := m.extractToken(r)
		if err != nil {
			// For optional paths, continue without authentication
			if m.isPathOptional(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			m.logger.ErrorRequest(r, err, "Authentication failed")
			m.respondWithError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		// Validate token
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			m.logger.ErrorRequest(r, err, "Token validation failed")
			m.respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Add user context to request
		ctx := m.addUserContext(r.Context(), claims, tokenString)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, ok := r.Context().Value(RolesContextKey).([]string)
			if !ok {
				m.logger.ErrorRequest(r, ErrUnauthorized, "User roles not found in context")
				m.respondWithError(w, http.StatusUnauthorized, "User roles not found")
				return
			}

			if !m.hasRequiredRole(userRoles, roles) {
				m.logger.ErrorRequest(r, ErrUnauthorized, "Insufficient permissions")
				m.respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	// This is a simplified permission check
	// In a real implementation, you would map permissions to roles
	rolePermissions := map[string][]string{
		"admin": {
			"ci:create", "ci:read", "ci:update", "ci:delete",
			"relationship:manage", "audit_log:read", "user:manage", "import:csv",
		},
		"ci_manager": {
			"ci:create", "ci:read", "ci:update", "ci:delete",
			"relationship:manage", "import:csv",
		},
		"viewer": {
			"ci:read",
		},
		"auditor": {
			"ci:read", "audit_log:read",
		},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, ok := r.Context().Value(RolesContextKey).([]string)
			if !ok {
				m.logger.ErrorRequest(r, ErrUnauthorized, "User roles not found in context")
				m.respondWithError(w, http.StatusUnauthorized, "User roles not found")
				return
			}

			if !m.hasRequiredPermission(userRoles, permission, rolePermissions) {
				m.logger.ErrorRequest(r, ErrUnauthorized, "Insufficient permissions")
				m.respondWithError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *AuthMiddleware) extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoToken
	}

	// Check for Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidTokenType
	}

	return parts[1], nil
}

func (m *AuthMiddleware) isPathExcluded(path string) bool {
	return m.excludePaths[path]
}

func (m *AuthMiddleware) isPathOptional(path string) bool {
	return m.optionalPaths[path]
}

func (m *AuthMiddleware) addUserContext(ctx context.Context, claims *Claims, tokenString string) context.Context {
	// Add user ID
	ctx = context.WithValue(ctx, UserContextKey, claims.UserID)

	// Add user roles
	ctx = context.WithValue(ctx, RolesContextKey, claims.Roles)

	// Add token
	ctx = context.WithValue(ctx, TokenContextKey, tokenString)

	return ctx
}

func (m *AuthMiddleware) hasRequiredRole(userRoles []string, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true
	}

	for _, requiredRole := range requiredRoles {
		for _, userRole := range userRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}

	return false
}

func (m *AuthMiddleware) hasRequiredPermission(userRoles []string, requiredPermission string, rolePermissions map[string][]string) bool {
	for _, role := range userRoles {
		if permissions, exists := rolePermissions[role]; exists {
			for _, permission := range permissions {
				if permission == requiredPermission {
					return true
				}
			}
		}
	}

	return false
}

func (m *AuthMiddleware) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// Helper functions to extract user information from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserContextKey).(string)
	return userID, ok
}

func GetUserRolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(RolesContextKey).([]string)
	return roles, ok
}

func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(TokenContextKey).(string)
	return token, ok
}

// OptionalAuthMiddleware creates middleware that doesn't require authentication
// but will authenticate the user if a token is provided
func OptionalAuthMiddleware(jwtService *JWTService, appLogger *logger.Logger) func(http.Handler) http.Handler {
	authConfig := AuthConfig{
		JWTService:    jwtService,
		Logger:        appLogger,
		ExcludePaths:  []string{}, // No excluded paths
		OptionalPaths: []string{"/"}, // All paths are optional
	}

	authMiddleware := NewAuthMiddleware(authConfig)
	return authMiddleware.Middleware
}

// AdminOnlyMiddleware creates middleware that requires admin role
func AdminOnlyMiddleware(jwtService *JWTService, appLogger *logger.Logger) func(http.Handler) http.Handler {
	authConfig := AuthConfig{
		JWTService:    jwtService,
		Logger:        appLogger,
		ExcludePaths:  []string{},
		OptionalPaths: []string{},
	}

	authMiddleware := NewAuthMiddleware(authConfig)
	return func(next http.Handler) http.Handler {
		return authMiddleware.RequireRole("admin")(next)
	}
}

// RoleMiddleware creates middleware that requires specific roles
func RoleMiddleware(jwtService *JWTService, appLogger *logger.Logger, roles ...string) func(http.Handler) http.Handler {
	authConfig := AuthConfig{
		JWTService:    jwtService,
		Logger:        appLogger,
		ExcludePaths:  []string{},
		OptionalPaths: []string{},
	}

	authMiddleware := NewAuthMiddleware(authConfig)
	return authMiddleware.RequireRole(roles...)
}

// PermissionMiddleware creates middleware that requires specific permissions
func PermissionMiddleware(jwtService *JWTService, appLogger *logger.Logger, permission string) func(http.Handler) http.Handler {
	authConfig := AuthConfig{
		JWTService:    jwtService,
		Logger:        appLogger,
		ExcludePaths:  []string{},
		OptionalPaths: []string{},
	}

	authMiddleware := NewAuthMiddleware(authConfig)
	return authMiddleware.RequirePermission(permission)
}
