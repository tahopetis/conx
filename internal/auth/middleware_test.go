package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/conx/cmdb/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	// Create JWT service for testing
	secretKey := "test-secret-key-that-is-at-least-32-characters-long"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	jwtService := NewJWTService(secretKey, accessTTL, refreshTTL)

	// Create test user
	userID := uuid.New().String()
	username := "testuser"
	roles := []string{"viewer", "editor"}

	// Generate valid token
	validToken, err := jwtService.GenerateAccessToken(userID, username, roles)
	require.NoError(t, err)

	// Generate expired token
	expiredJWTService := NewJWTService(secretKey, -1*time.Minute, refreshTTL)
	expiredToken, err := expiredJWTService.GenerateAccessToken(userID, username, roles)
	require.NoError(t, err)

	// Create middleware
	middleware := AuthMiddleware(jwtService)

	t.Run("Valid token", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if user context is set
			ctxUserID, ok := GetUserIDFromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, userID, ctxUserID)

			ctxRoles, ok := GetUserRolesFromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, roles, ctxRoles)

			ctxToken, ok := GetTokenFromContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, validToken, ctxToken)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middleware
		wrappedHandler := middleware(handler)

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create test request without authorization header
		req := httptest.NewRequest("GET", "/test", nil)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middleware
		wrappedHandler := middleware(handler)

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "missing authorization header")
	})

	t.Run("Invalid authorization header format", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create test request with invalid authorization header format
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat "+validToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middleware
		wrappedHandler := middleware(handler)

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "invalid authorization header format")
	})

	t.Run("Invalid token", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create test request with invalid token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middleware
		wrappedHandler := middleware(handler)

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "invalid token")
	})

	t.Run("Expired token", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create test request with expired token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middleware
		wrappedHandler := middleware(handler)

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "token is expired")
	})
}

func TestRequireRole(t *testing.T) {
	// Create JWT service for testing
	secretKey := "test-secret-key-that-is-at-least-32-characters-long"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	jwtService := NewJWTService(secretKey, accessTTL, refreshTTL)

	// Create test users with different roles
	adminUserID := uuid.New().String()
	adminUsername := "adminuser"
	adminRoles := []string{"admin", "editor"}

	editorUserID := uuid.New().String()
	editorUsername := "editoruser"
	editorRoles := []string{"editor"}

	viewerUserID := uuid.New().String()
	viewerUsername := "vieweruser"
	viewerRoles := []string{"viewer"}

	// Generate tokens
	adminToken, err := jwtService.GenerateAccessToken(adminUserID, adminUsername, adminRoles)
	require.NoError(t, err)

	editorToken, err := jwtService.GenerateAccessToken(editorUserID, editorUsername, editorRoles)
	require.NoError(t, err)

	viewerToken, err := jwtService.GenerateAccessToken(viewerUserID, viewerUsername, viewerRoles)
	require.NoError(t, err)

	// Create auth middleware
	authMiddleware := AuthMiddleware(jwtService)

	t.Run("Require admin role - admin user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("admin access granted"))
		})

		// Create require role middleware
		requireAdmin := RequireRole("admin")

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireAdmin(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be successful
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "admin access granted", recorder.Body.String())
	})

	t.Run("Require admin role - editor user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create require role middleware
		requireAdmin := RequireRole("admin")

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+editorToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireAdmin(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "insufficient permissions")
	})

	t.Run("Require editor role - editor user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("editor access granted"))
		})

		// Create require role middleware
		requireEditor := RequireRole("editor")

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+editorToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireEditor(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be successful
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "editor access granted", recorder.Body.String())
	})

	t.Run("Require editor role - viewer user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create require role middleware
		requireEditor := RequireRole("editor")

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireEditor(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "insufficient permissions")
	})

	t.Run("Require role without auth context", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create require role middleware
		requireAdmin := RequireRole("admin")

		// Create test request without authorization
		req := httptest.NewRequest("GET", "/test", nil)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middleware (no auth middleware)
		wrappedHandler := requireAdmin(handler)

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "user context not found")
	})
}

func TestRequireAnyRole(t *testing.T) {
	// Create JWT service for testing
	secretKey := "test-secret-key-that-is-at-least-32-characters-long"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	jwtService := NewJWTService(secretKey, accessTTL, refreshTTL)

	// Create test users with different roles
	adminUserID := uuid.New().String()
	adminUsername := "adminuser"
	adminRoles := []string{"admin", "editor"}

	editorUserID := uuid.New().String()
	editorUsername := "editoruser"
	editorRoles := []string{"editor"}

	viewerUserID := uuid.New().String()
	viewerUsername := "vieweruser"
	viewerRoles := []string{"viewer"}

	// Generate tokens
	adminToken, err := jwtService.GenerateAccessToken(adminUserID, adminUsername, adminRoles)
	require.NoError(t, err)

	editorToken, err := jwtService.GenerateAccessToken(editorUserID, editorUsername, editorRoles)
	require.NoError(t, err)

	viewerToken, err := jwtService.GenerateAccessToken(viewerUserID, viewerUsername, viewerRoles)
	require.NoError(t, err)

	// Create auth middleware
	authMiddleware := AuthMiddleware(jwtService)

	t.Run("Require admin or editor - admin user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("admin or editor access granted"))
		})

		// Create require any role middleware
		requireAdminOrEditor := RequireAnyRole([]string{"admin", "editor"})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireAdminOrEditor(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be successful
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "admin or editor access granted", recorder.Body.String())
	})

	t.Run("Require admin or editor - editor user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("admin or editor access granted"))
		})

		// Create require any role middleware
		requireAdminOrEditor := RequireAnyRole([]string{"admin", "editor"})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+editorToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireAdminOrEditor(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be successful
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "admin or editor access granted", recorder.Body.String())
	})

	t.Run("Require admin or editor - viewer user", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create require any role middleware
		requireAdminOrEditor := RequireAnyRole([]string{"admin", "editor"})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireAdminOrEditor(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "insufficient permissions")
	})

	t.Run("Require any role with empty roles list", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("access granted"))
		})

		// Create require any role middleware with empty roles
		requireAnyRole := RequireAnyRole([]string{})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Wrap handler with middlewares
		wrappedHandler := authMiddleware(requireAnyRole(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be successful (empty roles means no restriction)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "access granted", recorder.Body.String())
	})
}

func TestContextHelperFunctions(t *testing.T) {
	t.Run("GetUserIDFromContext", func(t *testing.T) {
		// Test with empty context
		userID, ok := GetUserIDFromContext(context.Background())
		assert.False(t, ok)
		assert.Empty(t, userID)

		// Test with context containing user ID
		ctx := context.WithValue(context.Background(), userIDKey, "test-user-id")
		userID, ok = GetUserIDFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test-user-id", userID)
	})

	t.Run("GetUsernameFromContext", func(t *testing.T) {
		// Test with empty context
		username, ok := GetUsernameFromContext(context.Background())
		assert.False(t, ok)
		assert.Empty(t, username)

		// Test with context containing username
		ctx := context.WithValue(context.Background(), usernameKey, "testuser")
		username, ok = GetUsernameFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "testuser", username)
	})

	t.Run("GetUserRolesFromContext", func(t *testing.T) {
		// Test with empty context
		roles, ok := GetUserRolesFromContext(context.Background())
		assert.False(t, ok)
		assert.Nil(t, roles)

		// Test with context containing roles
		testRoles := []string{"admin", "editor"}
		ctx := context.WithValue(context.Background(), rolesKey, testRoles)
		roles, ok = GetUserRolesFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, testRoles, roles)
	})

	t.Run("GetTokenFromContext", func(t *testing.T) {
		// Test with empty context
		token, ok := GetTokenFromContext(context.Background())
		assert.False(t, ok)
		assert.Empty(t, token)

		// Test with context containing token
		testToken := "test-token"
		ctx := context.WithValue(context.Background(), tokenKey, testToken)
		token, ok = GetTokenFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, testToken, token)
	})
}

func TestSetUserContext(t *testing.T) {
	t.Run("Set user context", func(t *testing.T) {
		// Create test claims
		claims := &JWTClaims{
			UserID:   "test-user-id",
			Username: "testuser",
			Roles:    []string{"admin", "editor"},
		}

		// Create empty context
		ctx := context.Background()

		// Set user context
		userCtx := SetUserContext(ctx, claims, "test-token")

		// Verify context values
		userID, ok := GetUserIDFromContext(userCtx)
		assert.True(t, ok)
		assert.Equal(t, claims.UserID, userID)

		username, ok := GetUsernameFromContext(userCtx)
		assert.True(t, ok)
		assert.Equal(t, claims.Username, username)

		roles, ok := GetUserRolesFromContext(userCtx)
		assert.True(t, ok)
		assert.Equal(t, claims.Roles, roles)

		token, ok := GetTokenFromContext(userCtx)
		assert.True(t, ok)
		assert.Equal(t, "test-token", token)
	})

	t.Run("Set user context with nil claims", func(t *testing.T) {
		// Create empty context
		ctx := context.Background()

		// Set user context with nil claims
		userCtx := SetUserContext(ctx, nil, "test-token")

		// Verify context values are empty
		userID, ok := GetUserIDFromContext(userCtx)
		assert.False(t, ok)
		assert.Empty(t, userID)

		username, ok := GetUsernameFromContext(userCtx)
		assert.False(t, ok)
		assert.Empty(t, username)

		roles, ok := GetUserRolesFromContext(userCtx)
		assert.False(t, ok)
		assert.Nil(t, roles)

		token, ok := GetTokenFromContext(userCtx)
		assert.True(t, ok)
		assert.Equal(t, "test-token", token)
	})
}

func TestWriteErrorResponse(t *testing.T) {
	t.Run("Write error response", func(t *testing.T) {
		// Create response recorder
		recorder := httptest.NewRecorder()

		// Write error response
		writeErrorResponse(recorder, http.StatusUnauthorized, "test error message")

		// Check response
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		
		// Check response body
		expectedBody := `{"error":"test error message"}`
		assert.Equal(t, expectedBody, strings.TrimSpace(recorder.Body.String()))
	})

	t.Run("Write error response with different status codes", func(t *testing.T) {
		testCases := []struct {
			statusCode int
			message    string
		}{
			{http.StatusBadRequest, "bad request"},
			{http.StatusForbidden, "forbidden"},
			{http.StatusInternalServerError, "internal server error"},
		}

		for _, tc := range testCases {
			recorder := httptest.NewRecorder()
			writeErrorResponse(recorder, tc.statusCode, tc.message)

			assert.Equal(t, tc.statusCode, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			
			expectedBody := `{"error":"` + tc.message + `"}`
			assert.Equal(t, expectedBody, strings.TrimSpace(recorder.Body.String()))
		}
	})
}

func TestMiddlewareChain(t *testing.T) {
	// Create JWT service for testing
	secretKey := "test-secret-key-that-is-at-least-32-characters-long"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	jwtService := NewJWTService(secretKey, accessTTL, refreshTTL)

	// Create test user
	userID := uuid.New().String()
	username := "testuser"
	roles := []string{"admin", "editor"}

	// Generate valid token
	validToken, err := jwtService.GenerateAccessToken(userID, username, roles)
	require.NoError(t, err)

	// Create middlewares
	authMiddleware := AuthMiddleware(jwtService)
	requireAdmin := RequireRole("admin")

	t.Run("Middleware chain with valid token and correct role", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("middleware chain success"))
		})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Chain middlewares
		wrappedHandler := authMiddleware(requireAdmin(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be successful
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "middleware chain success", recorder.Body.String())
	})

	t.Run("Middleware chain with valid token but wrong role", func(t *testing.T) {
		// Create user with different role
		viewerUserID := uuid.New().String()
		viewerUsername := "vieweruser"
		viewerRoles := []string{"viewer"}

		viewerToken, err := jwtService.GenerateAccessToken(viewerUserID, viewerUsername, viewerRoles)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not reach here"))
		})

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)

		// Create response recorder
		recorder := httptest.NewRecorder()

		// Chain middlewares
		wrappedHandler := authMiddleware(requireAdmin(handler))

		// Serve request
		wrappedHandler.ServeHTTP(recorder, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "insufficient permissions")
	})
}
