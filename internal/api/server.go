package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/conx/conx/internal/config"
	"github.com/conx/conx/internal/repositories"
	"github.com/gorilla/mux"
)

// Server represents the HTTP server
type Server struct {
	cfg         *config.Config
	router      *mux.Router
	ciRepo      *repositories.CIRepository
	ciHandler   *CIHandler
	schemaHandler *SchemaHandler
	httpServer  *http.Server
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, ciRepo *repositories.CIRepository) *Server {
	router := mux.NewRouter()
	
	// Create handlers
	ciHandler := NewCIHandler(ciRepo)
	schemaHandler := NewSchemaHandler(ciRepo)
	
	// Register routes
	ciHandler.RegisterRoutes(router)
	schemaHandler.RegisterRoutes(router)
	
	// Add CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	return &Server{
		cfg:          cfg,
		router:       router,
		ciRepo:       ciRepo,
		ciHandler:    ciHandler,
		schemaHandler: schemaHandler,
		httpServer: &http.Server{
			Addr:         ":" + cfg.Server.Port,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on port %s", s.cfg.Server.Port)
	
	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}
	
	log.Println("Server exited")
	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	log.Println("Stopping server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}
	
	log.Println("Server stopped")
	return nil
}

// GetRouter returns the router for testing purposes
func (s *Server) GetRouter() *mux.Router {
	return s.router
}
