package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"

	"connect/internal/config"
	"connect/internal/database"
)

var (
	logger = logrus.New()
)

// SeedData represents the structure for seed data
type SeedData struct {
	Roles       []Role       `json:"roles"`
	Permissions []Permission `json:"permissions"`
	Users       []User       `json:"users"`
	CIs         []CI         `json:"cis"`
	Relationships []Relationship `json:"relationships"`
}

// Role represents a role for seeding
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Permission represents a permission for seeding
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// User represents a user for seeding
type User struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
	IsActive bool     `json:"is_active"`
}

// CI represents a Configuration Item for seeding
type CI struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Attributes  map[string]interface{} `json:"attributes"`
	Tags        []string               `json:"tags"`
	CreatedBy   string                 `json:"created_by"`
}

// Relationship represents a relationship between CIs for seeding
type Relationship struct {
	ID           string `json:"id"`
	SourceCI     string `json:"source_ci"`
	TargetCI     string `json:"target_ci"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Strength     int    `json:"strength"`
	CreatedBy    string `json:"created_by"`
}

func main() {
	// Initialize logger
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Load seed data
	seedData, err := loadSeedData()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load seed data")
	}

	// Seed PostgreSQL
	if err := seedPostgreSQL(cfg, seedData); err != nil {
		logger.WithError(err).Fatal("Failed to seed PostgreSQL")
	}

	// Seed Neo4j
	if err := seedNeo4j(cfg, seedData); err != nil {
		logger.WithError(err).Fatal("Failed to seed Neo4j")
	}

	logger.Info("Database seeding completed successfully")
}

// loadSeedData loads seed data from JSON files or generates default data
func loadSeedData() (*SeedData, error) {
	// Try to load from JSON file first
	if data, err := os.ReadFile("migrations/seed_data.json"); err == nil {
		var seedData SeedData
		if err := json.Unmarshal(data, &seedData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal seed data: %w", err)
		}
		return &seedData, nil
	}

	// If no JSON file, generate default seed data
	return generateDefaultSeedData(), nil
}

// generateDefaultSeedData generates default seed data for the application
func generateDefaultSeedData() *SeedData {
	return &SeedData{
		Roles: []Role{
			{
				ID:          "role_admin",
				Name:        "Administrator",
				Description: "Full system access",
				Permissions: []string{"perm_user_create", "perm_user_read", "perm_user_update", "perm_user_delete",
					"perm_role_create", "perm_role_read", "perm_role_update", "perm_role_delete",
					"perm_ci_create", "perm_ci_read", "perm_ci_update", "perm_ci_delete",
					"perm_graph_read", "perm_search_read"},
			},
			{
				ID:          "role_manager",
				Name:        "Manager",
				Description: "Can manage CIs and view reports",
				Permissions: []string{"perm_user_read", "perm_role_read",
					"perm_ci_create", "perm_ci_read", "perm_ci_update",
					"perm_graph_read", "perm_search_read"},
			},
			{
				ID:          "role_user",
				Name:        "User",
				Description: "Read-only access",
				Permissions: []string{"perm_ci_read", "perm_graph_read", "perm_search_read"},
			},
		},
		Permissions: []Permission{
			{ID: "perm_user_create", Name: "Create Users", Description: "Ability to create new users", Resource: "users", Action: "create"},
			{ID: "perm_user_read", Name: "Read Users", Description: "Ability to read user information", Resource: "users", Action: "read"},
			{ID: "perm_user_update", Name: "Update Users", Description: "Ability to update user information", Resource: "users", Action: "update"},
			{ID: "perm_user_delete", Name: "Delete Users", Description: "Ability to delete users", Resource: "users", Action: "delete"},
			{ID: "perm_role_create", Name: "Create Roles", Description: "Ability to create new roles", Resource: "roles", Action: "create"},
			{ID: "perm_role_read", Name: "Read Roles", Description: "Ability to read role information", Resource: "roles", Action: "read"},
			{ID: "perm_role_update", Name: "Update Roles", Description: "Ability to update role information", Resource: "roles", Action: "update"},
			{ID: "perm_role_delete", Name: "Delete Roles", Description: "Ability to delete roles", Resource: "roles", Action: "delete"},
			{ID: "perm_ci_create", Name: "Create CIs", Description: "Ability to create configuration items", Resource: "cis", Action: "create"},
			{ID: "perm_ci_read", Name: "Read CIs", Description: "Ability to read configuration items", Resource: "cis", Action: "read"},
			{ID: "perm_ci_update", Name: "Update CIs", Description: "Ability to update configuration items", Resource: "cis", Action: "update"},
			{ID: "perm_ci_delete", Name: "Delete CIs", Description: "Ability to delete configuration items", Resource: "cis", Action: "delete"},
			{ID: "perm_graph_read", Name: "Read Graph", Description: "Ability to read graph data", Resource: "graph", Action: "read"},
			{ID: "perm_search_read", Name: "Search", Description: "Ability to search the system", Resource: "search", Action: "read"},
		},
		Users: []User{
			{
				ID:       "user_admin",
				Name:     "System Administrator",
				Email:    "admin@conx-cmdb.com",
				Password: "admin123", // In production, use properly hashed passwords
				Roles:    []string{"role_admin"},
				IsActive: true,
			},
			{
				ID:       "user_manager",
				Name:     "IT Manager",
				Email:    "manager@conx-cmdb.com",
				Password: "manager123",
				Roles:    []string{"role_manager"},
				IsActive: true,
			},
			{
				ID:       "user_user",
				Name:     "IT User",
				Email:    "user@conx-cmdb.com",
				Password: "user123",
				Roles:    []string{"role_user"},
				IsActive: true,
			},
		},
		CIs: []CI{
			{
				ID:          "ci_web_server_01",
				Name:        "Web Server 01",
				Type:        "server",
				Description: "Primary web server for production environment",
				Status:      "active",
				Attributes: map[string]interface{}{
					"ip_address":     "192.168.1.10",
					"operating_system": "Ubuntu 20.04",
					"cpu_cores":       4,
					"memory_gb":       8,
					"storage_gb":      500,
				},
				Tags:      []string{"production", "web", "linux"},
				CreatedBy: "user_admin",
			},
			{
				ID:          "ci_database_01",
				Name:        "Database Server 01",
				Type:        "database",
				Description: "Primary database server for application data",
				Status:      "active",
				Attributes: map[string]interface{}{
					"ip_address":     "192.168.1.20",
					"operating_system": "Ubuntu 20.04",
					"cpu_cores":       8,
					"memory_gb":       16,
					"storage_gb":      1000,
					"database_type":   "PostgreSQL",
					"version":        "13.4",
				},
				Tags:      []string{"production", "database", "linux"},
				CreatedBy: "user_admin",
			},
			{
				ID:          "ci_load_balancer_01",
				Name:        "Load Balancer 01",
				Type:        "load_balancer",
				Description: "Load balancer for web servers",
				Status:      "active",
				Attributes: map[string]interface{}{
					"ip_address":    "192.168.1.5",
					"type":          "nginx",
					"version":       "1.20.0",
					"algorithm":     "round_robin",
				},
				Tags:      []string{"production", "network", "nginx"},
				CreatedBy: "user_admin",
			},
		},
		Relationships: []Relationship{
			{
				ID:           "rel_web_server_01_to_db_01",
				SourceCI:     "ci_web_server_01",
				TargetCI:     "ci_database_01",
				Type:         "DEPENDS_ON",
				Description:  "Web server depends on database server",
				Strength:     5,
				CreatedBy:    "user_admin",
			},
			{
				ID:           "rel_web_server_01_to_lb_01",
				SourceCI:     "ci_web_server_01",
				TargetCI:     "ci_load_balancer_01",
				Type:         "HOSTS",
				Description:  "Load balancer hosts web server",
				Strength:     3,
				CreatedBy:    "user_admin",
			},
		},
	}
}

// seedPostgreSQL seeds the PostgreSQL database with data
func seedPostgreSQL(cfg *config.Config, seedData *SeedData) error {
	// Connect to PostgreSQL
	db, err := database.NewPostgresConnection(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Seed permissions
	for _, permission := range seedData.Permissions {
		query := `
			INSERT INTO permissions (id, name, description, resource, action, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				resource = EXCLUDED.resource,
				action = EXCLUDED.action,
				updated_at = EXCLUDED.updated_at
		`
		
		_, err := db.ExecContext(ctx, query,
			permission.ID, permission.Name, permission.Description,
			permission.Resource, permission.Action, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to seed permission %s: %w", permission.ID, err)
		}
		logger.WithField("permission_id", permission.ID).Info("Seeded permission")
	}

	// Seed roles
	for _, role := range seedData.Roles {
		query := `
			INSERT INTO roles (id, name, description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				updated_at = EXCLUDED.updated_at
		`
		
		_, err := db.ExecContext(ctx, query,
			role.ID, role.Name, role.Description, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to seed role %s: %w", role.ID, err)
		}
		logger.WithField("role_id", role.ID).Info("Seeded role")

		// Assign permissions to role
		for _, permissionID := range role.Permissions {
			query := `
				INSERT INTO role_permissions (role_id, permission_id, created_at)
				VALUES ($1, $2, $3)
				ON CONFLICT (role_id, permission_id) DO NOTHING
			`
			
			_, err := db.ExecContext(ctx, query, role.ID, permissionID, time.Now())
			if err != nil {
				return fmt.Errorf("failed to assign permission %s to role %s: %w", permissionID, role.ID, err)
			}
		}
	}

	// Seed users
	for _, user := range seedData.Users {
		// Hash password (in production, use proper password hashing)
		hashedPassword := user.Password // This should be properly hashed
		
		query := `
			INSERT INTO users (id, name, email, password_hash, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				email = EXCLUDED.email,
				password_hash = EXCLUDED.password_hash,
				is_active = EXCLUDED.is_active,
				updated_at = EXCLUDED.updated_at
		`
		
		_, err := db.ExecContext(ctx, query,
			user.ID, user.Name, user.Email, hashedPassword, user.IsActive, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to seed user %s: %w", user.ID, err)
		}
		logger.WithField("user_id", user.ID).Info("Seeded user")

		// Assign roles to user
		for _, roleID := range user.Roles {
			query := `
				INSERT INTO user_roles (user_id, role_id, created_at)
				VALUES ($1, $2, $3)
				ON CONFLICT (user_id, role_id) DO NOTHING
			`
			
			_, err := db.ExecContext(ctx, query, user.ID, roleID, time.Now())
			if err != nil {
				return fmt.Errorf("failed to assign role %s to user %s: %w", roleID, user.ID, err)
			}
		}
	}

	// Seed CIs
	for _, ci := range seedData.CIs {
		attributesJSON, err := json.Marshal(ci.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes for CI %s: %w", ci.ID, err)
		}

		tagsJSON, err := json.Marshal(ci.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags for CI %s: %w", ci.ID, err)
		}

		query := `
			INSERT INTO configuration_items (id, name, type, description, status, attributes, tags, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				type = EXCLUDED.type,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				attributes = EXCLUDED.attributes,
				tags = EXCLUDED.tags,
				created_by = EXCLUDED.created_by,
				updated_at = EXCLUDED.updated_at
		`
		
		_, err = db.ExecContext(ctx, query,
			ci.ID, ci.Name, ci.Type, ci.Description, ci.Status,
			attributesJSON, tagsJSON, ci.CreatedBy, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to seed CI %s: %w", ci.ID, err)
		}
		logger.WithField("ci_id", ci.ID).Info("Seeded CI")
	}

	// Seed relationships
	for _, relationship := range seedData.Relationships {
		query := `
			INSERT INTO relationships (id, source_ci_id, target_ci_id, type, description, strength, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				source_ci_id = EXCLUDED.source_ci_id,
				target_ci_id = EXCLUDED.target_ci_id,
				type = EXCLUDED.type,
				description = EXCLUDED.description,
				strength = EXCLUDED.strength,
				created_by = EXCLUDED.created_by,
				updated_at = EXCLUDED.updated_at
		`
		
		_, err := db.ExecContext(ctx, query,
			relationship.ID, relationship.SourceCI, relationship.TargetCI,
			relationship.Type, relationship.Description, relationship.Strength,
			relationship.CreatedBy, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to seed relationship %s: %w", relationship.ID, err)
		}
		logger.WithField("relationship_id", relationship.ID).Info("Seeded relationship")
	}

	logger.Info("PostgreSQL seeding completed successfully")
	return nil
}

// seedNeo4j seeds the Neo4j database with data
func seedNeo4j(cfg *config.Config, seedData *SeedData) error {
	// Connect to Neo4j
	driver, err := database.NewNeo4jDriver(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	defer driver.Close(ctx)

	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Seed CIs as nodes
	for _, ci := range seedData.CIs {
		query := `
			MERGE (ci:ConfigurationItem {id: $id})
			SET ci.name = $name,
			    ci.type = $type,
			    ci.description = $description,
			    ci.status = $status,
			    ci.attributes = $attributes,
			    ci.tags = $tags,
			    ci.created_by = $created_by,
			    ci.created_at = $created_at,
			    ci.updated_at = $updated_at
		`
		
		attributesJSON, _ := json.Marshal(ci.Attributes)
		tagsJSON, _ := json.Marshal(ci.Tags)
		
		_, err := session.Run(ctx, query,
			map[string]interface{}{
				"id":          ci.ID,
				"name":        ci.Name,
				"type":        ci.Type,
				"description": ci.Description,
				"status":      ci.Status,
				"attributes":  string(attributesJSON),
				"tags":        string(tagsJSON),
				"created_by":  ci.CreatedBy,
				"created_at":  time.Now(),
				"updated_at":  time.Now(),
			})
		if err != nil {
			return fmt.Errorf("failed to seed CI node %s: %w", ci.ID, err)
		}
		logger.WithField("ci_id", ci.ID).Info("Seeded CI node")
	}

	// Seed relationships as edges
	for _, relationship := range seedData.Relationships {
		query := `
			MATCH (source:ConfigurationItem {id: $source_id})
			MATCH (target:ConfigurationItem {id: $target_id})
			MERGE (source)-[r:RELATIONSHIP {id: $id}]->(target)
			SET r.type = $type,
			    r.description = $description,
			    r.strength = $strength,
			    r.created_by = $created_by,
			    r.created_at = $created_at,
			    r.updated_at = $updated_at
		`
		
		_, err := session.Run(ctx, query,
			map[string]interface{}{
				"id":           relationship.ID,
				"source_id":    relationship.SourceCI,
				"target_id":    relationship.TargetCI,
				"type":         relationship.Type,
				"description":  relationship.Description,
				"strength":     relationship.Strength,
				"created_by":   relationship.CreatedBy,
				"created_at":   time.Now(),
				"updated_at":   time.Now(),
			})
		if err != nil {
			return fmt.Errorf("failed to seed relationship edge %s: %w", relationship.ID, err)
		}
		logger.WithField("relationship_id", relationship.ID).Info("Seeded relationship edge")
	}

	logger.Info("Neo4j seeding completed successfully")
	return nil
}
