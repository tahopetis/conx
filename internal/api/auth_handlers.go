package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/conx/cmdb/internal/auth"
	"github.com/conx/cmdb/internal/config"
	"github.com/conx/cmdb/internal/logger"
	"github.com/conx/cmdb/internal/models"
	"github.com/conx/cmdb/internal/repositories"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type AuthHandler struct {
	config         *config.Config
	logger         *logger.Logger
	jwtService     *auth.JWTService
	userRepository *repositories.UserRepository
	passwordService *auth.PasswordService
}

func NewAuthHandler(
	config *config.Config,
	appLogger *logger.Logger,
	jwtService *auth.JWTService,
	userRepository *repositories.UserRepository,
	passwordService *auth.PasswordService,
) *AuthHandler {
	return &AuthHandler{
		config:         config,
		logger:         appLogger,
		jwtService:     jwtService,
		userRepository: userRepository,
		passwordService: passwordService,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode registration request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid registration request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// Create user
	user, err := h.userRepository.Create(r.Context(), &req, uuid.Nil) // uuid.Nil for system-created users
	if err != nil {
		if err == repositories.ErrUserAlreadyExists {
			h.logger.ErrorRequest(r, err, "User already exists")
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "User already exists"})
			return
		}
		h.logger.ErrorRequest(r, err, "Failed to create user")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create user"})
		return
	}

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID.String(), user.Username, []string{"viewer"})
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to generate access token")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate tokens"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID.String(), user.Username, []string{"viewer"})
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to generate refresh token")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate tokens"})
		return
	}

	// Return response
	response := models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.config.Auth.AccessTokenTTL.Seconds()),
		User:         user.ToResponse([]string{"viewer"}),
	}

	h.logger.InfoRequest(r, "User registered successfully", "user_id", user.ID)
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode login request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid login request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// Authenticate user
	user, err := h.userRepository.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		if err == repositories.ErrUserNotFound || err == repositories.ErrInvalidPassword {
			h.logger.ErrorRequest(r, err, "Authentication failed")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "Invalid credentials"})
			return
		}
		if err == repositories.ErrUserInactive {
			h.logger.ErrorRequest(r, err, "User account is inactive")
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, map[string]string{"error": "Account is inactive"})
			return
		}
		h.logger.ErrorRequest(r, err, "Authentication error")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Authentication failed"})
		return
	}

	// TODO: Get user roles from role repository
	userRoles := []string{"viewer"} // Default role for now

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID.String(), user.Username, userRoles)
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to generate access token")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate tokens"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID.String(), user.Username, userRoles)
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to generate refresh token")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to generate tokens"})
		return
	}

	// Return response
	response := models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.config.Auth.AccessTokenTTL.Seconds()),
		User:         user.ToResponse(userRoles),
	}

	h.logger.InfoRequest(r, "User logged in successfully", "user_id", user.ID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode refresh token request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid refresh token request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// Refresh access token
	accessToken, err := h.jwtService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to refresh access token")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid or expired refresh token"})
		return
	}

	// Extract user info from refresh token
	claims, err := h.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to validate refresh token")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Invalid refresh token"})
		return
	}

	// Get user info
	user, err := h.userRepository.GetByID(r.Context(), uuid.MustParse(claims.UserID))
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to get user info")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not found"})
		return
	}

	// Return response
	response := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int64(h.config.Auth.AccessTokenTTL.Seconds()),
		"user":         user.ToResponse(claims.Roles),
	}

	h.logger.InfoRequest(r, "Token refreshed successfully", "user_id", user.ID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		h.logger.ErrorRequest(r, nil, "User ID not found in context")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Unauthorized"})
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode change password request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid change password request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// Change password
	err = h.userRepository.ChangePassword(r.Context(), uuid.MustParse(userID), req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err == repositories.ErrInvalidPassword {
			h.logger.ErrorRequest(r, err, "Invalid current password")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "Invalid current password"})
			return
		}
		h.logger.ErrorRequest(r, err, "Failed to change password")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to change password"})
		return
	}

	h.logger.InfoRequest(r, "Password changed successfully", "user_id", userID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Password changed successfully"})
}

// GetProfile handles getting user profile
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		h.logger.ErrorRequest(r, nil, "User ID not found in context")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Unauthorized"})
		return
	}

	// Get user info
	user, err := h.userRepository.GetByID(r.Context(), uuid.MustParse(userID))
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to get user info")
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "User not found"})
		return
	}

	// TODO: Get user roles from role repository
	userRoles := []string{"viewer"} // Default role for now

	response := user.ToResponse(userRoles)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// UpdateProfile handles updating user profile
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		h.logger.ErrorRequest(r, nil, "User ID not found in context")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode update profile request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid update profile request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// Update user
	user, err := h.userRepository.Update(r.Context(), uuid.MustParse(userID), &req, uuid.MustParse(userID))
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to update user profile")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update profile"})
		return
	}

	// TODO: Get user roles from role repository
	userRoles := []string{"viewer"} // Default role for now

	response := user.ToResponse(userRoles)
	h.logger.InfoRequest(r, "User profile updated successfully", "user_id", userID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// RequestPasswordReset handles password reset request
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode password reset request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid password reset request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// Get user by email
	user, err := h.userRepository.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if err == repositories.ErrUserNotFound {
			// Don't reveal if user exists or not
			h.logger.InfoRequest(r, "Password reset requested for non-existent email", "email", req.Email)
			render.Status(r, http.StatusOK)
			render.JSON(w, r, map[string]string{"message": "If your email is registered, you will receive a password reset link"})
			return
		}
		h.logger.ErrorRequest(r, err, "Failed to get user by email")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to process request"})
		return
	}

	// TODO: Generate password reset token and send email
	// For now, just return success message
	h.logger.InfoRequest(r, "Password reset requested", "user_id", user.ID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "If your email is registered, you will receive a password reset link"})
}

// ResetPassword handles password reset confirmation
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorRequest(r, err, "Failed to decode password reset confirmation request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.ErrorRequest(r, err, "Invalid password reset confirmation request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request data"})
		return
	}

	// TODO: Validate reset token and extract user ID
	// For now, use a dummy user ID
	userID := uuid.New()

	// Reset password
	err := h.userRepository.ResetPassword(r.Context(), userID, req.NewPassword)
	if err != nil {
		h.logger.ErrorRequest(r, err, "Failed to reset password")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to reset password"})
		return
	}

	h.logger.InfoRequest(r, "Password reset successfully", "user_id", userID)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Password reset successfully"})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would:
	// 1. Add the token to a blacklist
	// 2. Remove the token from Redis cache
	// 3. Update user session status

	// For now, just return success
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Logged out successfully"})
}

// Routes returns the auth routes
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.RefreshToken)
	r.Post("/password-reset-request", h.RequestPasswordReset)
	r.Post("/password-reset", h.ResetPassword)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.NewAuthMiddleware(auth.AuthConfig{
			JWTService: h.jwtService,
			Logger:     h.logger,
			ExcludePaths: []string{},
			OptionalPaths: []string{},
		}).Middleware)

		r.Get("/profile", h.GetProfile)
		r.Put("/profile", h.UpdateProfile)
		r.Post("/change-password", h.ChangePassword)
		r.Post("/logout", h.Logout)
	})

	return r
}
