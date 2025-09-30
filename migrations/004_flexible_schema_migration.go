package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"connect/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FlexibleSchemaMigration handles migration from rigid technical specifications to flexible JSONB attributes
type FlexibleSchemaMigration struct {
	db *sqlx.DB
}

// NewFlexibleSchemaMigration creates a new migration instance
func NewFlexibleSchemaMigration(db *sqlx.DB) *FlexibleSchemaMigration {
	return &FlexibleSchemaMigration{db: db}
}

// Run executes the migration
func (m *FlexibleSchemaMigration) Run(ctx context.Context) error {
	log.Println("Starting flexible schema migration...")

	// Step 1: Add new columns to configuration_items table
	if err := m.addNewColumns(ctx); err != nil {
		return fmt.Errorf("failed to add new columns: %w", err)
	}

	// Step 2: Create default schemas
	if err := m.createDefaultSchemas(ctx); err != nil {
		return fmt.Errorf("failed to create default schemas: %w", err)
	}

	// Step 3: Migrate existing data to new format
	if err := m.migrateExistingData(ctx); err != nil {
		return fmt.Errorf("failed to migrate existing data: %w", err)
	}

	// Step 4: Create schema tables
	if err := m.createSchemaTables(ctx); err != nil {
		return fmt.Errorf("failed to create schema tables: %w", err)
	}

	// Step 5: Create relationship tables
	if err := m.createRelationshipTables(ctx); err != nil {
		return fmt.Errorf("failed to create relationship tables: %w", err)
	}

	log.Println("Flexible schema migration completed successfully")
	return nil
}

// addNewColumns adds new columns to the configuration_items table
func (m *FlexibleSchemaMigration) addNewColumns(ctx context.Context) error {
	log.Println("Adding new columns to configuration_items table...")

	queries := []string{
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS attributes JSONB`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS tags TEXT[]`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS install_date TIMESTAMP`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS warranty_expiry TIMESTAMP`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS last_updated TIMESTAMP`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS last_scanned TIMESTAMP`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT false`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS created_by UUID`,
		`ALTER TABLE configuration_items ADD COLUMN IF NOT EXISTS updated_by UUID`,
	}

	for _, query := range queries {
		if _, err := m.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query '%s': %w", query, err)
		}
	}

	return nil
}

// createDefaultSchemas creates default CI type schemas
func (m *FlexibleSchemaMigration) createDefaultSchemas(ctx context.Context) error {
	log.Println("Creating default CI type schemas...")

	// Get default schemas
	defaultSchemas := models.GetDefaultCISchemas()

	// Create each schema
	for _, schema := range defaultSchemas {
		// Convert attributes to JSON
		attributesJSON, err := json.Marshal(schema.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes for schema '%s': %w", schema.Name, err)
		}

		// Insert schema
		query := `
			INSERT INTO ci_type_schemas (id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (name) DO NOTHING`

		_, err = m.db.ExecContext(ctx, query,
			uuid.New(),
			schema.Name,
			schema.Description,
			attributesJSON,
			schema.IsActive,
			schema.CreatedAt,
			schema.UpdatedAt,
			schema.CreatedBy,
			schema.UpdatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to insert schema '%s': %w", schema.Name, err)
		}
	}

	return nil
}

// migrateExistingData migrates existing data to the new flexible format
func (m *FlexibleSchemaMigration) migrateExistingData(ctx context.Context) error {
	log.Println("Migrating existing data to new flexible format...")

	// Get all existing CIs
	query := `SELECT id, name, type, description, status, criticality, owner, location, 
	                 technical_specifications, install_date, warranty_expiry,
	                 updated_at, scanned_at
	          FROM configuration_items`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query existing CIs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id                     uuid.UUID
			name                   string
			ciType                 string
			description           sql.NullString
			status                 sql.NullString
			criticality            sql.NullString
			owner                  sql.NullString
			location               sql.NullString
			technicalSpecs        sql.NullString
			installDate           sql.NullTime
			warrantyExpiry        sql.NullTime
			updatedAt             sql.NullTime
			scannedAt             sql.NullTime
		)

		if err := rows.Scan(&id, &name, &ciType, &description, &status, &criticality, &owner, &location,
			&technicalSpecs, &installDate, &warrantyExpiry, &updatedAt, &scannedAt); err != nil {
			return fmt.Errorf("failed to scan CI row: %w", err)
		}

		// Convert technical specifications to flexible attributes
		attributes, err := m.convertTechSpecsToAttributes(technicalSpecs.String, ciType)
		if err != nil {
			log.Printf("Warning: failed to convert tech specs for CI '%s': %v", name, err)
			attributes = json.RawMessage("{}")
		}

		// Update the CI with new format
		updateQuery := `
			UPDATE configuration_items 
			SET attributes = $1, 
			    install_date = $2, 
			    warranty_expiry = $3, 
			    last_updated = $4, 
			    last_scanned = $5,
			    is_active = true,
			    is_deleted = false,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $6`

		_, err = m.db.ExecContext(ctx, updateQuery,
			attributes,
			installDate,
			warrantyExpiry,
			updatedAt,
			scannedAt,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to update CI '%s': %w", name, err)
		}
	}

	return nil
}

// convertTechSpecsToAttributes converts technical specifications JSON to flexible attributes
func (m *FlexibleSchemaMigration) convertTechSpecsToAttributes(techSpecs, ciType string) (json.RawMessage, error) {
	if techSpecs == "" {
		return json.RawMessage("{}"), nil
	}

	// Parse existing tech specs
	var specs map[string]interface{}
	if err := json.Unmarshal([]byte(techSpecs), &specs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tech specs: %w", err)
	}

	// Convert to flexible format based on CI type
	switch ciType {
	case "server":
		return m.convertServerSpecs(specs)
	case "application":
		return m.convertApplicationSpecs(specs)
	case "database":
		return m.convertDatabaseSpecs(specs)
	case "network_device":
		return m.convertNetworkDeviceSpecs(specs)
	default:
		// Generic conversion
		return json.Marshal(specs)
	}
}

// convertServerSpecs converts server technical specifications to flexible attributes
func (m *FlexibleSchemaMigration) convertServerSpecs(specs map[string]interface{}) (json.RawMessage, error) {
	attributes := make(map[string]interface{})

	// Map common server attributes
	if val, ok := specs["ip_address"]; ok {
		attributes["ip_address"] = val
	}
	if val, ok := specs["hostname"]; ok {
		attributes["hostname"] = val
	}
	if val, ok := specs["cpu"]; ok {
		attributes["cpu_cores"] = val
	}
	if val, ok := specs["memory"]; ok {
		attributes["memory_gb"] = val
	}
	if val, ok := specs["storage"]; ok {
		attributes["storage_gb"] = val
	}
	if val, ok := specs["os"]; ok {
		attributes["os_version"] = val
	}
	if val, ok := specs["environment"]; ok {
		attributes["environment"] = val
	}

	return json.Marshal(attributes)
}

// convertApplicationSpecs converts application technical specifications to flexible attributes
func (m *FlexibleSchemaMigration) convertApplicationSpecs(specs map[string]interface{}) (json.RawMessage, error) {
	attributes := make(map[string]interface{})

	// Map common application attributes
	if val, ok := specs["version"]; ok {
		attributes["version"] = val
	}
	if val, ok := specs["framework"]; ok {
		attributes["framework"] = val
	}
	if val, ok := specs["language"]; ok {
		attributes["language"] = val
	}
	if val, ok := specs["port"]; ok {
		attributes["port"] = val
	}
	if val, ok := specs["dependencies"]; ok {
		attributes["dependencies"] = val
	}
	if val, ok := specs["environment"]; ok {
		attributes["environment"] = val
	}

	return json.Marshal(attributes)
}

// convertDatabaseSpecs converts database technical specifications to flexible attributes
func (m *FlexibleSchemaMigration) convertDatabaseSpecs(specs map[string]interface{}) (json.RawMessage, error) {
	attributes := make(map[string]interface{})

	// Map common database attributes
	if val, ok := specs["engine"]; ok {
		attributes["engine"] = val
	}
	if val, ok := specs["version"]; ok {
		attributes["version"] = val
	}
	if val, ok := specs["size"]; ok {
		attributes["size_gb"] = val
	}
	if val, ok := specs["tables"]; ok {
		attributes["tables_count"] = val
	}
	if val, ok := specs["connection_string"]; ok {
		attributes["connection_string"] = val
	}

	return json.Marshal(attributes)
}

// convertNetworkDeviceSpecs converts network device technical specifications to flexible attributes
func (m *FlexibleSchemaMigration) convertNetworkDeviceSpecs(specs map[string]interface{}) (json.RawMessage, error) {
	attributes := make(map[string]interface{})

	// Map common network device attributes
	if val, ok := specs["device_type"]; ok {
		attributes["device_type"] = val
	}
	if val, ok := specs["management_ip"]; ok {
		attributes["management_ip"] = val
	}
	if val, ok := specs["ports"]; ok {
		attributes["ports_count"] = val
	}
	if val, ok := specs["vlan"]; ok {
		attributes["vlan"] = val
	}
	if val, ok := specs["model"]; ok {
		attributes["model"] = val
	}

	return json.Marshal(attributes)
}

// createSchemaTables creates schema management tables
func (m *FlexibleSchemaMigration) createSchemaTables(ctx context.Context) error {
	log.Println("Creating schema management tables...")

	queries := []string{
		`CREATE TABLE IF NOT EXISTS ci_type_schemas (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			attributes JSONB NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by UUID,
			updated_by UUID
		)`,

		`CREATE TABLE IF NOT EXISTS relationship_type_schemas (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			attributes JSONB NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by UUID,
			updated_by UUID
		)`,
	}

	for _, query := range queries {
		if _, err := m.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query '%s': %w", query, err)
		}
	}

	return nil
}

// createRelationshipTables creates relationship management tables
func (m *FlexibleSchemaMigration) createRelationshipTables(ctx context.Context) error {
	log.Println("Creating relationship management tables...")

	// Get default relationship schemas
	defaultRelSchemas := models.GetDefaultRelationshipSchemas()

	// Create relationship table
	query := `
		CREATE TABLE IF NOT EXISTS ci_relationships (
			id UUID PRIMARY KEY,
			source_ci_id UUID NOT NULL,
			target_ci_id UUID NOT NULL,
			type VARCHAR(255) NOT NULL,
			attributes JSONB,
			description TEXT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by UUID,
			updated_by UUID,
			FOREIGN KEY (source_ci_id) REFERENCES configuration_items(id),
			FOREIGN KEY (target_ci_id) REFERENCES configuration_items(id),
			UNIQUE(source_ci_id, target_ci_id, type)
		)`

	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create ci_relationships table: %w", err)
	}

	// Create default relationship schemas
	for _, schema := range defaultRelSchemas {
		// Convert attributes to JSON
		attributesJSON, err := json.Marshal(schema.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes for relationship schema '%s': %w", schema.Name, err)
		}

		// Insert schema
		insertQuery := `
			INSERT INTO relationship_type_schemas (id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (name) DO NOTHING`

		_, err = m.db.ExecContext(ctx, insertQuery,
			uuid.New(),
			schema.Name,
			schema.Description,
			attributesJSON,
			schema.IsActive,
			schema.CreatedAt,
			schema.UpdatedAt,
			schema.CreatedBy,
			schema.UpdatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to insert relationship schema '%s': %w", schema.Name, err)
		}
	}

	return nil
}

// Rollback rolls back the migration
func (m *FlexibleSchemaMigration) Rollback(ctx context.Context) error {
	log.Println("Rolling back flexible schema migration...")

	queries := []string{
		`DROP TABLE IF EXISTS ci_relationships`,
		`DROP TABLE IF EXISTS relationship_type_schemas`,
		`DROP TABLE IF EXISTS ci_type_schemas`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS attributes`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS tags`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS install_date`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS warranty_expiry`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS last_updated`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS last_scanned`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS is_active`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS is_deleted`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS created_at`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS updated_at`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS created_by`,
		`ALTER TABLE configuration_items DROP COLUMN IF EXISTS updated_by`,
	}

	for _, query := range queries {
		if _, err := m.db.ExecContext(ctx, query); err != nil {
			log.Printf("Warning: failed to execute rollback query '%s': %v", query, err)
		}
	}

	log.Println("Flexible schema migration rollback completed")
	return nil
}
