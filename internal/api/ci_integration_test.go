package api

import (
	"bytes"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CITestSuite struct {
	server      *Server
	authHandler *AuthHandler
	ciHandler   *CIHandler
	userRepo    *repositories.UserRepository
	ciRepo      *repositories.CIRepository
	jwtService  *auth.JWTService
	testUser    *models.User
	authToken   string
	refreshToken string
}

func setupCITestSuite(t *testing.T) *CITestSuite {
	// Create test configuration
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-for-jwt-tokens",
			AccessTokenTTL: 15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	// Create test logger
	appLogger := logger.NewLogger(cfg)

	// Mock database and other dependencies (in a real test, you'd use test containers)
	// For this test, we'll create a simplified setup
	userRepo := repositories.NewUserRepository(nil, appLogger)
	ciRepo := repositories.NewCIRepository(nil, appLogger)
	jwtService := auth.NewJWTService(cfg)
	passwordService := auth.NewPasswordService()

	// Create handlers
	authHandler := NewAuthHandler(cfg, appLogger, jwtService, userRepo, passwordService)
	ciHandler := NewCIHandler(cfg, appLogger, ciRepo, userRepo)

	// Create server
	server := &Server{
		cfg:         cfg,
		logger:      appLogger,
		authHandler: authHandler,
		ciHandler:   ciHandler,
		userRepository: userRepo,
		ciRepository: ciRepo,
		jwtService: jwtService,
	}

	// Setup routes
	server.setupRoutes()

	// Create test user
	testUser := &models.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return &CITestSuite{
		server:      server,
		authHandler: authHandler,
		ciHandler:   ciHandler,
		userRepo:    userRepo,
		ciRepo:      ciRepo,
		jwtService:  jwtService,
		testUser:    testUser,
	}
}

func (suite *CITestSuite) createTestUserAndToken(t *testing.T) {
	// Create test user (in real implementation, this would be saved to test database)
	suite.testUser = &models.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate JWT tokens for testing
	claims := auth.JWTClaims{
		UserID: suite.testUser.ID.String(),
		Email:  suite.testUser.Email,
		Roles:  []string{"viewer"},
	}

	token, err := suite.jwtService.GenerateAccessToken(claims)
	require.NoError(t, err)
	suite.authToken = token

	refreshToken, err := suite.jwtService.GenerateRefreshToken(claims)
	require.NoError(t, err)
	suite.refreshToken = refreshToken
}

func TestCIEndpoints(t *testing.T) {
	suite := setupCITestSuite(t)
	suite.createTestUserAndToken(t)

	t.Run("List CIs - Empty List", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, 0, response.Total)
		assert.Equal(t, 1, response.Page)
		assert.Equal(t, 20, response.Size)
		assert.Empty(t, response.CIs)
	})

	t.Run("Create CI - Valid Request", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "Test Server",
			Type:        "server",
			Description: "A test server for integration testing",
			Status:      "active",
			Criticality: "medium",
			Owner:       "IT Department",
			Location:    "Data Center 1",
			Version:     "1.0",
			IPAddress:   "192.168.1.100",
			Manufacturer: "Dell",
			Model:       "PowerEdge R740",
			OSVersion:   "Ubuntu 20.04 LTS",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response models.CIResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, "Test Server", response.Name)
		assert.Equal(t, "server", response.Type)
		assert.Equal(t, "active", response.Status)
		assert.Equal(t, "medium", response.Criticality)
		assert.NotEqual(t, uuid.Nil, response.ID)
	})

	t.Run("Create CI - Invalid Request", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			// Missing required fields
			Description: "Invalid CI request",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var errorResponse map[string]string
		err = json.NewDecoder(w.Body).Decode(&errorResponse)
		require.NoError(t, err)
		
		assert.Contains(t, errorResponse["error"], "Validation failed")
	})

	t.Run("Create CI - Unauthorized", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "Unauthorized CI",
			Type:        "server",
			Status:      "active",
			Criticality: "low",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Get CI - Not Found", func(t *testing.T) {
		ciID := uuid.New()
		
		req := httptest.NewRequest("GET", "/api/v1/cis/"+ciID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Get CI - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis/invalid-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var errorResponse map[string]string
		err := json.NewDecoder(w.Body).Decode(&errorResponse)
		require.NoError(t, err)
		
		assert.Equal(t, "Invalid CI ID", errorResponse["error"])
	})

	t.Run("Update CI - Not Found", func(t *testing.T) {
		ciID := uuid.New()
		updateReq := models.UpdateCIRequest{
			Name: ptr("Updated Server Name"),
		}

		reqBody, err := json.Marshal(updateReq)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/api/v1/cis/"+ciID.String(), bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Delete CI - Not Found", func(t *testing.T) {
		ciID := uuid.New()
		
		req := httptest.NewRequest("DELETE", "/api/v1/cis/"+ciID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("List CIs with Filtering", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?type=server&status=active&page=1&size=10", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, 1, response.Page)
		assert.Equal(t, 10, response.Size)
	})

	t.Run("Get CI Stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis/stats", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIStats
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.GreaterOrEqual(t, response.TotalCIs, 0)
		assert.NotNil(t, response.CIsByType)
		assert.NotNil(t, response.CIsByStatus)
		assert.NotNil(t, response.CIsByCriticality)
	})

	t.Run("Create CI Relationship - Valid Request", func(t *testing.T) {
		// First create two CIs
		ci1 := createTestCI(t, "Source CI", "server")
		ci2 := createTestCI(t, "Target CI", "database")

		relationshipReq := models.CreateCIRelationshipRequest{
			TargetCIID:  ci2.ID,
			Type:        "depends_on",
			Description: "Source CI depends on Target CI",
		}

		reqBody, err := json.Marshal(relationshipReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis/"+ci1.ID.String()+"/relationships", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response models.CIRelationshipResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, ci1.ID, response.SourceCIID)
		assert.Equal(t, ci2.ID, response.TargetCIID)
		assert.Equal(t, "depends_on", response.Type)
	})

	t.Run("Create CI Relationship - Circular Dependency", func(t *testing.T) {
		// Create two CIs
		ci1 := createTestCI(t, "CI 1", "server")
		ci2 := createTestCI(t, "CI 2", "server")

		// Create relationship from ci1 to ci2
		relationshipReq := models.CreateCIRelationshipRequest{
			TargetCIID:  ci2.ID,
			Type:        "depends_on",
			Description: "CI 1 depends on CI 2",
		}

		reqBody, err := json.Marshal(relationshipReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis/"+ci1.ID.String()+"/relationships", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Try to create reverse relationship (should fail due to circular dependency)
		reverseRelationshipReq := models.CreateCIRelationshipRequest{
			TargetCIID:  ci1.ID,
			Type:        "depends_on",
			Description: "CI 2 depends on CI 1",
		}

		reqBody, err = json.Marshal(reverseRelationshipReq)
		require.NoError(t, err)

		req = httptest.NewRequest("POST", "/api/v1/cis/"+ci2.ID.String()+"/relationships", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w = httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var errorResponse map[string]string
		err = json.NewDecoder(w.Body).Decode(&errorResponse)
		require.NoError(t, err)
		
		assert.Equal(t, "Circular dependency detected", errorResponse["error"])
	})

	t.Run("Get CI Relationships", func(t *testing.T) {
		// Create a CI
		ci := createTestCI(t, "Test CI", "server")

		req := httptest.NewRequest("GET", "/api/v1/cis/"+ci.ID.String()+"/relationships", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response []models.CIRelationshipResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		// Should be empty initially
		assert.Empty(t, response)
	})

	t.Run("Get CI Attributes - Not Implemented", func(t *testing.T) {
		ci := createTestCI(t, "Test CI", "server")

		req := httptest.NewRequest("GET", "/api/v1/cis/"+ci.ID.String()+"/attributes", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response []models.CIAttributeResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Empty(t, response)
	})

	t.Run("Create CI Attribute - Not Implemented", func(t *testing.T) {
		ci := createTestCI(t, "Test CI", "server")

		attributeReq := models.CreateCIAttributeRequest{
			Name:  "Custom Attribute",
			Value: "Custom Value",
			Type:  "string",
		}

		reqBody, err := json.Marshal(attributeReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis/"+ci.ID.String()+"/attributes", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
	})
}

// Helper functions
func ptr[T any](v T) *T {
	return &v
}

func createTestCI(t *testing.T, name, ciType string) *models.CI {
	return &models.CI{
		ID:           uuid.New(),
		Name:         name,
		Type:         ciType,
		Description:  "Test CI description",
		Status:       "active",
		Criticality:  "medium",
		Owner:        "IT Department",
		Location:     "Test Location",
		Version:      "1.0",
		IPAddress:    "192.168.1.1",
		Manufacturer: "Test Manufacturer",
		Model:        "Test Model",
		OSVersion:    "Test OS",
		IsActive:     true,
		IsDeleted:    false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestCIValidation(t *testing.T) {
	suite := setupCITestSuite(t)
	suite.createTestUserAndToken(t)

	t.Run("Create CI with Invalid Type", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "Test CI",
			Type:        "invalid_type",
			Status:      "active",
			Criticality: "low",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create CI with Invalid Status", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "Test CI",
			Type:        "server",
			Status:      "invalid_status",
			Criticality: "low",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create CI with Invalid Criticality", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "Test CI",
			Type:        "server",
			Status:      "active",
			Criticality: "invalid_criticality",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create CI with Invalid IP Address", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "Test CI",
			Type:        "server",
			Status:      "active",
			Criticality: "low",
			IPAddress:   "invalid_ip_address",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCIPaginationAndFiltering(t *testing.T) {
	suite := setupCITestSuite(t)
	suite.createTestUserAndToken(t)

	t.Run("Pagination - Page 2", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?page=2&size=5", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, 2, response.Page)
		assert.Equal(t, 5, response.Size)
	})

	t.Run("Filtering by Name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?name=Test", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
	})

	t.Run("Filtering by Type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?type=server", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
	})

	t.Run("Filtering by Status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?status=active", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
	})

	t.Run("Filtering by Criticality", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?criticality=medium", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
	})

	t.Run("Filtering by Active Status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?is_active=true", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
	})

	t.Run("Filtering by Date Range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?created_from=2023-01-01&created_to=2023-12-31", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
	})

	t.Run("Invalid Page Size", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?size=200", nil) // Size > 100
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.CIList
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		// Should default to maximum size
		assert.Equal(t, 100, response.Size)
	})
}

func TestCISecurity(t *testing.T) {
	suite := setupCITestSuite(t)
	suite.createTestUserAndToken(t)

	t.Run("Access Without Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis", nil)
		// No Authorization header
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Access With Invalid Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Access With Expired Token", func(t *testing.T) {
		// Create an expired token (this would require mocking time or using a very short TTL)
		// For now, we'll just test with a malformed token
		req := httptest.NewRequest("GET", "/api/v1/cis", nil)
		req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.payload")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("SQL Injection Attempt", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/cis?name='+OR+1=1--", nil)
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		// Should be handled gracefully, not cause SQL errors
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("XSS Attempt", func(t *testing.T) {
		createReq := models.CreateCIRequest{
			Name:        "<script>alert('xss')</script>",
			Type:        "server",
			Status:      "active",
			Criticality: "low",
		}

		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/cis", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		suite.server.Router().ServeHTTP(w, req)

		// Should be handled gracefully, input should be sanitized
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
