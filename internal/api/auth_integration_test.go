package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connect/internal/auth"
	"connect/internal/config"
	"connect/internal/logger"
	"connect/internal/models"
	"connect/internal/repositories"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestServer struct {
	server     *Server
	db         *pgxpool.Pool
	neo4jDriver neo4j.DriverWithContext
	redis      *redis.Client
	httpServer *httptest.Server
}

func setupTestServer(t *testing.T) *TestServer {
	ctx := context.Background()
	
	// Create test configuration
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecretKey:     "test-secret-key-that-is-at-least-32-characters-long",
			AccessTokenTTL:   15 * time.Minute,
			RefreshTokenTTL:  7 * 24 * time.Hour,
			Issuer:          "conx-test",
			Audience:        "conx-users",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "conx_test",
			Username: "test",
			Password: "test",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Neo4j: config.Neo4jConfig{
			URI:      "neo4j://localhost:7687",
			Username: "neo4j",
			Password: "test",
		},
	}

	// Create logger
	appLogger := logger.NewLogger(cfg)

	// Create database connections (using mock or test database)
	db, err := pgxpool.New(ctx, cfg.Database.GetConnectionString())
	require.NoError(t, err)
	
	// Create Neo4j driver
	neo4jDriver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	require.NoError(t, err)
	
	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetAddress(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Create server
	server := NewServer(cfg, db, neo4jDriver, redisClient, appLogger)
	
	// Create HTTP test server
	httpServer := httptest.NewServer(server.Router())
	
	return &TestServer{
		server:     server,
		db:         db,
		neo4jDriver: neo4jDriver,
		redis:      redisClient,
		httpServer: httpServer,
	}
}

func (ts *TestServer) Close() {
	ts.httpServer.Close()
	ts.neo4jDriver.Close(context.Background())
	ts.db.Close()
	ts.redis.Close()
}

func (ts *TestServer) makeRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, ts.httpServer.URL+url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	return client.Do(req)
}

func (ts *TestServer) createTestUser(t *testing.T) *models.User {
	ctx := context.Background()
	userRepo := repositories.NewUserRepository(ts.db, ts.server.logger)
	
	createReq := &models.CreateUserRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "TestPassword123!",
		FirstName: "Test",
		LastName:  "User",
	}
	
	user, err := userRepo.Create(ctx, createReq, uuid.Nil)
	require.NoError(t, err)
	
	return user
}

func TestAuthenticationEndpoints(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("User Registration", func(t *testing.T) {
		registerReq := models.CreateUserRequest{
			Username:  "newuser",
			Email:     "newuser@example.com",
			Password:  "NewPassword123!",
			FirstName: "New",
			LastName:  "User",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/register", registerReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response models.LoginResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Greater(t, response.ExpiresIn, int64(0))
		assert.Equal(t, "newuser", response.User.Username)
		assert.Equal(t, "newuser@example.com", response.User.Email)
	})

	t.Run("User Login", func(t *testing.T) {
		// Create a test user first
		testUser := ts.createTestUser(t)

		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "TestPassword123!",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/login", loginReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response models.LoginResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Greater(t, response.ExpiresIn, int64(0))
		assert.Equal(t, testUser.Username, response.User.Username)
		assert.Equal(t, testUser.Email, response.User.Email)
	})

	t.Run("Invalid Login", func(t *testing.T) {
		loginReq := models.LoginRequest{
			Username: "nonexistent",
			Password: "WrongPassword123!",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/login", loginReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var errorResponse map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "Invalid credentials", errorResponse["error"])
	})

	t.Run("Token Refresh", func(t *testing.T) {
		// Login first to get refresh token
		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "TestPassword123!",
		}

		loginResp, err := ts.makeRequest("POST", "/api/v1/auth/login", loginReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, loginResp.StatusCode)

		var loginResponse models.LoginResponse
		err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
		require.NoError(t, err)

		// Now refresh the token
		refreshReq := models.RefreshTokenRequest{
			RefreshToken: loginResponse.RefreshToken,
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/refresh", refreshReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var refreshResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&refreshResponse)
		require.NoError(t, err)
		
		assert.NotEmpty(t, refreshResponse["access_token"])
		assert.Equal(t, "Bearer", refreshResponse["token_type"])
		assert.Greater(t, refreshResponse["expires_in"], 0.0)
	})

	t.Run("Invalid Token Refresh", func(t *testing.T) {
		refreshReq := models.RefreshTokenRequest{
			RefreshToken: "invalid-refresh-token",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/refresh", refreshReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var errorResponse map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "Invalid or expired refresh token", errorResponse["error"])
	})
}

func TestProtectedEndpoints(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Create a test user and login
	testUser := ts.createTestUser(t)
	
	loginReq := models.LoginRequest{
		Username: "testuser",
		Password: "TestPassword123!",
	}

	loginResp, err := ts.makeRequest("POST", "/api/v1/auth/login", loginReq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	var loginResponse models.LoginResponse
	err = json.NewDecoder(loginResp.Body).Decode(&loginResponse)
	require.NoError(t, err)

	accessToken := loginResponse.AccessToken

	t.Run("Get Profile Without Token", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/api/v1/auth/profile", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Get Profile With Valid Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.httpServer.URL+"/api/v1/auth/profile", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var profileResponse models.UserResponse
		err = json.NewDecoder(resp.Body).Decode(&profileResponse)
		require.NoError(t, err)
		
		assert.Equal(t, testUser.ID, profileResponse.ID)
		assert.Equal(t, testUser.Username, profileResponse.Username)
		assert.Equal(t, testUser.Email, profileResponse.Email)
	})

	t.Run("Get Current User (/me)", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.httpServer.URL+"/api/v1/me", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var userResponse models.UserResponse
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		
		assert.Equal(t, testUser.ID, userResponse.ID)
		assert.Equal(t, testUser.Username, userResponse.Username)
		assert.Equal(t, testUser.Email, userResponse.Email)
	})

	t.Run("Update Profile", func(t *testing.T) {
		updateReq := models.UpdateUserRequest{
			FirstName: stringPtr("Updated"),
			LastName:  stringPtr("User"),
		}

		req, err := http.NewRequest("PUT", ts.httpServer.URL+"/api/v1/auth/profile", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		// Add request body
		reqBody, err := json.Marshal(updateReq)
		require.NoError(t, err)
		req.Body = bytes.NewBuffer(reqBody)
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedUser models.UserResponse
		err = json.NewDecoder(resp.Body).Decode(&updatedUser)
		require.NoError(t, err)
		
		assert.Equal(t, "Updated", updatedUser.FirstName)
		assert.Equal(t, "User", updatedUser.LastName)
	})

	t.Run("Change Password", func(t *testing.T) {
		changePasswordReq := models.ChangePasswordRequest{
			CurrentPassword: "TestPassword123!",
			NewPassword:     "NewTestPassword123!",
		}

		req, err := http.NewRequest("POST", ts.httpServer.URL+"/api/v1/auth/change-password", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		// Add request body
		reqBody, err := json.Marshal(changePasswordReq)
		require.NoError(t, err)
		req.Body = bytes.NewBuffer(reqBody)
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]string
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Password changed successfully", response["message"])
	})

	t.Run("Logout", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.httpServer.URL+"/api/v1/auth/logout", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]string
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Logged out successfully", response["message"])
	})
}

func TestPasswordResetEndpoints(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("Request Password Reset", func(t *testing.T) {
		resetReq := models.PasswordResetRequest{
			Email: "test@example.com",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/password-reset-request", resetReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]string
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "If your email is registered, you will receive a password reset link", response["message"])
	})

	t.Run("Request Password Reset for Non-existent Email", func(t *testing.T) {
		resetReq := models.PasswordResetRequest{
			Email: "nonexistent@example.com",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/password-reset-request", resetReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode) // Should return 200 for security

		var response map[string]string
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "If your email is registered, you will receive a password reset link", response["message"])
	})

	t.Run("Reset Password", func(t *testing.T) {
		resetConfirmReq := models.PasswordResetConfirmRequest{
			Token:       "dummy-token", // In real implementation, this would be a valid token
			NewPassword: "ResetPassword123!",
		}

		resp, err := ts.makeRequest("POST", "/api/v1/auth/password-reset", resetConfirmReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]string
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Password reset successfully", response["message"])
	})
}

func TestAuthenticationMiddleware(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("Access Protected Endpoint Without Token", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/api/v1/auth/profile", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access Protected Endpoint With Invalid Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.httpServer.URL+"/api/v1/auth/profile", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Access Protected Endpoint With Malformed Token", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.httpServer.URL+"/api/v1/auth/profile", nil)
		require.NoError(t, err)
		
		req.Header.Set("Authorization", "Malformed token")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestHealthCheck(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("Health Check Endpoint", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/health", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var healthResponse struct {
			Status string                 `json:"status"`
			Checks map[string]interface{} `json:"checks"`
		}
		err = json.NewDecoder(resp.Body).Decode(&healthResponse)
		require.NoError(t, err)
		
		assert.Equal(t, "healthy", healthResponse.Status)
		assert.Contains(t, healthResponse.Checks, "postgres")
		assert.Contains(t, healthResponse.Checks, "neo4j")
		assert.Contains(t, healthResponse.Checks, "redis")
	})
}

func TestInputValidation(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("Registration Validation", func(t *testing.T) {
		invalidRequests := []struct {
			name string
			req  models.CreateUserRequest
		}{
			{
				"Empty Username",
				models.CreateUserRequest{Username: "", Email: "test@example.com", Password: "ValidPassword123!", FirstName: "Test", LastName: "User"},
			},
			{
				"Invalid Email",
				models.CreateUserRequest{Username: "testuser", Email: "invalid-email", Password: "ValidPassword123!", FirstName: "Test", LastName: "User"},
			},
			{
				"Weak Password",
				models.CreateUserRequest{Username: "testuser", Email: "test@example.com", Password: "weak", FirstName: "Test", LastName: "User"},
			},
		}

		for _, tc := range invalidRequests {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := ts.makeRequest("POST", "/api/v1/auth/register", tc.req)
				require.NoError(t, err)
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})
		}
	})

	t.Run("Login Validation", func(t *testing.T) {
		invalidRequests := []struct {
			name string
			req  models.LoginRequest
		}{
			{
				"Empty Username",
				models.LoginRequest{Username: "", Password: "ValidPassword123!"},
			},
			{
				"Empty Password",
				models.LoginRequest{Username: "testuser", Password: ""},
			},
		}

		for _, tc := range invalidRequests {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := ts.makeRequest("POST", "/api/v1/auth/login", tc.req)
				require.NoError(t, err)
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})
		}
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
