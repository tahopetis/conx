package auth

import (
	"testing"
	"time"

	"github.com/conx/cmdb/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService(t *testing.T) {
	secretKey := "test-secret-key-that-is-at-least-32-characters-long"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	jwtService := NewJWTService(secretKey, accessTTL, refreshTTL)

	t.Run("Generate and Validate Access Token", func(t *testing.T) {
		userID := uuid.New().String()
		username := "testuser"
		roles := []string{"viewer", "editor"}

		token, err := jwtService.GenerateAccessToken(userID, username, roles)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := jwtService.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, roles, claims.Roles)
	})

	t.Run("Generate and Validate Refresh Token", func(t *testing.T) {
		userID := uuid.New().String()
		username := "testuser"
		roles := []string{"viewer", "editor"}

		token, err := jwtService.GenerateRefreshToken(userID, username, roles)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := jwtService.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, roles, claims.Roles)
	})

	t.Run("Refresh Access Token", func(t *testing.T) {
		userID := uuid.New().String()
		username := "testuser"
		roles := []string{"viewer", "editor"}

		refreshToken, err := jwtService.GenerateRefreshToken(userID, username, roles)
		require.NoError(t, err)

		newAccessToken, err := jwtService.RefreshAccessToken(refreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)

		claims, err := jwtService.ValidateToken(newAccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, roles, claims.Roles)
	})

	t.Run("Extract User ID", func(t *testing.T) {
		userID := uuid.New().String()
		username := "testuser"
		roles := []string{"viewer"}

		token, err := jwtService.GenerateAccessToken(userID, username, roles)
		require.NoError(t, err)

		extractedID, err := jwtService.ExtractUserID(token)
		require.NoError(t, err)
		assert.Equal(t, userID, extractedID)
	})

	t.Run("Extract Roles", func(t *testing.T) {
		userID := uuid.New().String()
		username := "testuser"
		roles := []string{"viewer", "editor"}

		token, err := jwtService.GenerateAccessToken(userID, username, roles)
		require.NoError(t, err)

		extractedRoles, err := jwtService.ExtractRoles(token)
		require.NoError(t, err)
		assert.Equal(t, roles, extractedRoles)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		invalidToken := "invalid.token.here"
		
		_, err := jwtService.ValidateToken(invalidToken)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
	})

	t.Run("Token with Invalid Secret", func(t *testing.T) {
		userID := uuid.New().String()
		username := "testuser"
		roles := []string{"viewer"}

		// Generate token with original service
		token, err := jwtService.GenerateAccessToken(userID, username, roles)
		require.NoError(t, err)

		// Try to validate with different secret
		differentJWTService := NewJWTService("different-secret-key", accessTTL, refreshTTL)
		_, err = differentJWTService.ValidateToken(token)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
	})
}

func TestPasswordService(t *testing.T) {
	config := DefaultPasswordConfig()
	passwordService := NewPasswordService(config)

	t.Run("Hash and Verify Password", func(t *testing.T) {
		password := "TestPassword123!"

		hash, err := passwordService.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Contains(t, hash, "$argon2id$")

		valid, err := passwordService.VerifyPassword(password, hash)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("Verify Wrong Password", func(t *testing.T) {
		password := "TestPassword123!"
		wrongPassword := "WrongPassword123!"

		hash, err := passwordService.HashPassword(password)
		require.NoError(t, err)

		valid, err := passwordService.VerifyPassword(wrongPassword, hash)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("Validate Password Strength", func(t *testing.T) {
		testCases := []struct {
			password string
			expected error
		}{
			{"short", ErrPasswordTooShort},
			{"thispasswordistoolongandshouldexceedthemaximumallowedlengthof128characters", ErrPasswordTooLong},
			{"nouppercase123!", ErrPasswordMissingUpper},
			{"NOLOWERCASE123!", ErrPasswordMissingLower},
			{"NoNumbersHere!", ErrPasswordMissingNumber},
			{"NoSpecialChars123", ErrPasswordMissingSpecial},
			{"ValidPassword123!", nil},
		}

		for _, tc := range testCases {
			err := passwordService.ValidatePasswordStrength(tc.password)
			if tc.expected == nil {
				assert.NoError(t, err, "Password should be valid: %s", tc.password)
			} else {
				assert.ErrorIs(t, err, tc.expected, "Password should fail validation: %s", tc.password)
			}
		}
	})

	t.Run("Generate Password", func(t *testing.T) {
		password, err := passwordService.GeneratePassword(12)
		require.NoError(t, err)
		assert.Len(t, password, 12)

		// Check if generated password meets strength requirements
		err = passwordService.ValidatePasswordStrength(password)
		assert.NoError(t, err)
	})

	t.Run("Check Common Patterns", func(t *testing.T) {
		commonPasswords := []string{
			"password", "123456", "qwerty", "admin123",
		}

		for _, pwd := range commonPasswords {
			err := passwordService.CheckPasswordCommonPatterns(pwd)
			assert.Error(t, err, "Common password should be rejected: %s", pwd)
		}

		// Valid password should not trigger pattern detection
		validPwd := "SecureRandomPassword123!"
		err := passwordService.CheckPasswordCommonPatterns(validPwd)
		assert.NoError(t, err)
	})

	t.Run("Get Password Strength", func(t *testing.T) {
		testCases := []struct {
			password string
			minScore int
		}{
			{"Weak123", 30},
			{"BetterPassword123", 60},
			{"StrongPassword123!", 80},
			{"VeryStrongPassword123!WithMoreLength", 90},
		}

		for _, tc := range testCases {
			score, feedback := passwordService.GetPasswordStrength(tc.password)
			assert.GreaterOrEqual(t, score, tc.minScore, "Password score should be at least %d: %s", tc.minScore, tc.password)
			assert.NotEmpty(t, feedback, "Feedback should not be empty")
		}
	})

	t.Run("Invalid Hash Format", func(t *testing.T) {
		invalidHash := "invalid-hash-format"
		
		_, err := passwordService.VerifyPassword("password", invalidHash)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidHash, err)
	})
}

func TestAuthMiddleware(t *testing.T) {
	// This would require setting up a full HTTP server and middleware
	// For now, we'll test the helper functions
	
	t.Run("Context Helper Functions", func(t *testing.T) {
		ctx := context.Background()
		
		// Test with empty context
		userID, ok := GetUserIDFromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, userID)
		
		roles, ok := GetUserRolesFromContext(ctx)
		assert.False(t, ok)
		assert.Nil(t, roles)
		
		token, ok := GetTokenFromContext(ctx)
		assert.False(t, ok)
		assert.Empty(t, token)
	})
}

func TestUserModelValidation(t *testing.T) {
	t.Run("Create User Request Validation", func(t *testing.T) {
		validReq := &models.CreateUserRequest{
			Username:  "testuser",
			Email:     "test@example.com",
			Password:  "ValidPassword123!",
			FirstName: "Test",
			LastName:  "User",
		}
		
		err := validReq.Validate()
		assert.NoError(t, err)
		
		// Test invalid requests
		invalidReqs := []struct {
			name string
			req  *models.CreateUserRequest
		}{
			{
				"Empty Username",
				&models.CreateUserRequest{Username: "", Email: "test@example.com", Password: "ValidPassword123!", FirstName: "Test", LastName: "User"},
			},
			{
				"Invalid Email",
				&models.CreateUserRequest{Username: "testuser", Email: "invalid-email", Password: "ValidPassword123!", FirstName: "Test", LastName: "User"},
			},
			{
				"Empty Password",
				&models.CreateUserRequest{Username: "testuser", Email: "test@example.com", Password: "", FirstName: "Test", LastName: "User"},
			},
		}
		
		for _, tc := range invalidReqs {
			err := tc.req.Validate()
			assert.Error(t, err, "Validation should fail for: %s", tc.name)
		}
	})

	t.Run("Update User Request Validation", func(t *testing.T) {
		validReq := &models.UpdateUserRequest{
			FirstName: stringPtr("Updated"),
			LastName:  stringPtr("User"),
		}
		
		err := validReq.Validate()
		assert.NoError(t, err)
		
		// Test with nil values (should be valid)
		nilReq := &models.UpdateUserRequest{}
		err = nilReq.Validate()
		assert.NoError(t, err)
	})

	t.Run("Change Password Request Validation", func(t *testing.T) {
		validReq := &models.ChangePasswordRequest{
			CurrentPassword: "CurrentPassword123!",
			NewPassword:     "NewPassword123!",
		}
		
		err := validReq.Validate()
		assert.NoError(t, err)
		
		// Test invalid requests
		invalidReqs := []struct {
			name string
			req  *models.ChangePasswordRequest
		}{
			{
				"Empty Current Password",
				&models.ChangePasswordRequest{CurrentPassword: "", NewPassword: "NewPassword123!"},
			},
			{
				"Empty New Password",
				&models.ChangePasswordRequest{CurrentPassword: "CurrentPassword123!", NewPassword: ""},
			},
		}
		
		for _, tc := range invalidReqs {
			err := tc.req.Validate()
			assert.Error(t, err, "Validation should fail for: %s", tc.name)
		}
	})

	t.Run("Login Request Validation", func(t *testing.T) {
		validReq := &models.LoginRequest{
			Username: "testuser",
			Password: "ValidPassword123!",
		}
		
		err := validReq.Validate()
		assert.NoError(t, err)
		
		// Test invalid requests
		invalidReqs := []struct {
			name string
			req  *models.LoginRequest
		}{
			{
				"Empty Username",
				&models.LoginRequest{Username: "", Password: "ValidPassword123!"},
			},
			{
				"Empty Password",
				&models.LoginRequest{Username: "testuser", Password: ""},
			},
		}
		
		for _, tc := range invalidReqs {
			err := tc.req.Validate()
			assert.Error(t, err, "Validation should fail for: %s", tc.name)
		}
	})
}

func TestUserToResponse(t *testing.T) {
	user := &models.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		IsVerified: true,
	}
	
	roles := []string{"viewer", "editor"}
	
	response := user.ToResponse(roles)
	
	assert.Equal(t, user.ID, response.ID)
	assert.Equal(t, user.Username, response.Username)
	assert.Equal(t, user.Email, response.Email)
	assert.Equal(t, user.FirstName, response.FirstName)
	assert.Equal(t, user.LastName, response.LastName)
	assert.Equal(t, user.IsActive, response.IsActive)
	assert.Equal(t, user.IsVerified, response.IsVerified)
	assert.Equal(t, roles, response.Roles)
	assert.Empty(t, response.PasswordHash) // Should not be included
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
