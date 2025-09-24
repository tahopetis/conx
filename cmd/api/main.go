package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/conx/cmdb/internal/api"
	"github.com/conx/cmdb/internal/auth"
	"github.com/conx/cmdb/internal/config"
	"github.com/conx/cmdb/internal/database"
	"github.com/conx/cmdb/internal/logger"
	"github.com/conx/cmdb/internal/repositories"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize logger
	appLogger := logger.New(cfg)
	appLogger.Info("Starting conx CMDB API server", "version", cfg.Version, "environment", cfg.Environment)

	// Initialize database connections
	dbManager, err := database.NewManager(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database connections")
	}

	// Test database connections
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := dbManager.Health(ctx); err != nil {
		log.Fatal().Err(err).Msg("Database health check failed")
	}

	// Initialize authentication services
	jwtService := auth.NewJWTService(
		cfg.Auth.SecretKey,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)

	passwordService := auth.NewPasswordService(auth.DefaultPasswordConfig())

	// Initialize repositories
	userRepository := repositories.NewUserRepository(dbManager.Postgres, passwordService)

	// Initialize API handlers
	authHandler := api.NewAuthHandler(cfg, appLogger, jwtService, userRepository, passwordService)
	ciHandler := api.NewCIHandler(cfg, appLogger, dbManager)
	relationshipHandler := api.NewRelationshipHandler(cfg, appLogger, dbManager)
	graphHandler := api.NewGraphHandler(cfg, appLogger, dbManager)
	healthHandler := api.NewHealthHandler(cfg, appLogger, dbManager)

	// Create router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.AllowContentType("application/json"))

	// CORS
	cors := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		ExposedHeaders:   cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	})
	router.Use(cors.Handler)

	// API version
	router.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", healthHandler.Health)

		// Authentication routes
		r.Mount("/auth", authHandler.Routes())

		// Protected routes
		r.Group(func(r chi.Router) {
			// Authentication middleware
			r.Use(auth.NewAuthMiddleware(auth.AuthConfig{
				JWTService: jwtService,
				Logger:     appLogger,
				ExcludePaths: []string{
					"/api/v1/health",
					"/api/v1/auth/login",
					"/api/v1/auth/register",
					"/api/v1/auth/refresh",
					"/api/v1/auth/password-reset-request",
					"/api/v1/auth/password-reset",
				},
				OptionalPaths: []string{},
			}).Middleware)

			// CI Management routes
			r.Mount("/cis", ciHandler.Routes())

			// Relationship Management routes
			r.Mount("/relationships", relationshipHandler.Routes())

			// Graph Service routes
			r.Mount("/graph", graphHandler.Routes())

			// User Management routes (admin only)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("admin"))

				// TODO: Add user management routes
				// r.Get("/users", userHandler.ListUsers)
				// r.Post("/users", userHandler.CreateUser)
				// r.Get("/users/{id}", userHandler.GetUser)
				// r.Put("/users/{id}", userHandler.UpdateUser)
				// r.Delete("/users/{id}", userHandler.DeleteUser)
				// r.Get("/users/{id}/roles", userHandler.GetUserRoles)
				// r.Post("/users/{id}/roles", userHandler.AssignRole)
				// r.Delete("/users/{id}/roles/{roleId}", userHandler.RevokeRole)
			})

			// Role Management routes (admin only)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("admin"))

				// TODO: Add role management routes
				// r.Get("/roles", roleHandler.ListRoles)
				// r.Post("/roles", roleHandler.CreateRole)
				// r.Get("/roles/{id}", roleHandler.GetRole)
				// r.Put("/roles/{id}", roleHandler.UpdateRole)
				// r.Delete("/roles/{id}", roleHandler.DeleteRole)
				// r.Get("/roles/{id}/permissions", roleHandler.GetRolePermissions)
				// r.Post("/roles/{id}/permissions", roleHandler.GrantPermission)
				// r.Delete("/roles/{id}/permissions/{permissionId}", roleHandler.RevokePermission)
			})

			// Permission Management routes (admin only)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("admin"))

				// TODO: Add permission management routes
				// r.Get("/permissions", permissionHandler.ListPermissions)
				// r.Post("/permissions", permissionHandler.CreatePermission)
				// r.Get("/permissions/{id}", permissionHandler.GetPermission)
				// r.Put("/permissions/{id}", permissionHandler.UpdatePermission)
				// r.Delete("/permissions/{id}", permissionHandler.DeletePermission)
			})
		})
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		appLogger.Info("Server starting", "port", cfg.Server.Port, "host", cfg.Server.Host)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	// Close database connections
	if err := dbManager.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close database connections")
	}

	appLogger.Info("Server stopped")
}
