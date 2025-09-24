package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/conx/conx/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// CIRepository handles database operations for CIs
type CIRepository struct {
	db *sqlx.DB
}

// NewCIRepository creates a new CI repository
func NewCIRepository(db *sqlx.DB) *CIRepository {
	return &CIRepository{db: db}
}

// CreateCI creates a new CI in the database
func (r *CIRepository) CreateCI(ctx context.Context, ci *models.CI) (*models.CI, error) {
	query := `
		INSERT INTO configuration_items (
			id, name, type, description, status, criticality, owner, location,
			attributes, tags, install_date, warranty_expiry, last_updated, last_scanned,
			is_active, is_deleted, created_at, updated_at, created_by, updated_by
		) VALUES (
			:id, :name, :type, :description, :status, :criticality, :owner, :location,
			:attributes, :tags, :install_date, :warranty_expiry, :last_updated, :last_scanned,
			:is_active, :is_deleted, :created_at, :updated_at, :created_by, :updated_by
		)
		RETURNING id, name, type, description, status, criticality, owner, location,
		          attributes, tags, install_date, warranty_expiry, last_updated, last_scanned,
		          is_active, is_deleted, created_at, updated_at, created_by, updated_by`

	// Set timestamps if not provided
	if ci.CreatedAt.IsZero() {
		ci.CreatedAt = time.Now()
	}
	if ci.UpdatedAt.IsZero() {
		ci.UpdatedAt = time.Now()
	}

	// Set default values
	if ci.Status == "" {
		ci.Status = models.CIStatusActive
	}
	if ci.Criticality == "" {
		ci.Criticality = models.CICriticalityMedium
	}
	if !ci.IsActive {
		ci.IsActive = true
	}

	rows, err := r.db.NamedQueryContext(ctx, query, ci)
	if err != nil {
		return nil, fmt.Errorf("failed to create CI: %w", err)
	}
	defer rows.Close()

	var createdCI models.CI
	if rows.Next() {
		if err := rows.StructScan(&createdCI); err != nil {
			return nil, fmt.Errorf("failed to scan created CI: %w", err)
		}
	}

	return &createdCI, nil
}

// GetCI retrieves a CI by ID
func (r *CIRepository) GetCI(ctx context.Context, id uuid.UUID) (*models.CI, error) {
	query := `
		SELECT id, name, type, description, status, criticality, owner, location,
		       attributes, tags, install_date, warranty_expiry, last_updated, last_scanned,
		       is_active, is_deleted, created_at, updated_at, created_by, updated_by
		FROM configuration_items 
		WHERE id = $1 AND is_deleted = false`

	var ci models.CI
	err := r.db.GetContext(ctx, &ci, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CI not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	return &ci, nil
}

// UpdateCI updates an existing CI
func (r *CIRepository) UpdateCI(ctx context.Context, ci *models.CI) (*models.CI, error) {
	query := `
		UPDATE configuration_items SET
			name = :name,
			type = :type,
			description = :description,
			status = :status,
			criticality = :criticality,
			owner = :owner,
			location = :location,
			attributes = :attributes,
			tags = :tags,
			install_date = :install_date,
			warranty_expiry = :warranty_expiry,
			last_updated = :last_updated,
			last_scanned = :last_scanned,
			is_active = :is_active,
			updated_at = :updated_at,
			updated_by = :updated_by
		WHERE id = :id AND is_deleted = false
		RETURNING id, name, type, description, status, criticality, owner, location,
		          attributes, tags, install_date, warranty_expiry, last_updated, last_scanned,
		          is_active, is_deleted, created_at, updated_at, created_by, updated_by`

	// Set updated timestamp
	ci.UpdatedAt = time.Now()

	rows, err := r.db.NamedQueryContext(ctx, query, ci)
	if err != nil {
		return nil, fmt.Errorf("failed to update CI: %w", err)
	}
	defer rows.Close()

	var updatedCI models.CI
	if rows.Next() {
		if err := rows.StructScan(&updatedCI); err != nil {
			return nil, fmt.Errorf("failed to scan updated CI: %w", err)
		}
	} else {
		return nil, fmt.Errorf("CI not found")
	}

	return &updatedCI, nil
}

// DeleteCI soft-deletes a CI
func (r *CIRepository) DeleteCI(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE configuration_items 
		SET is_deleted = true, updated_at = $1
		WHERE id = $2 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete CI: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("CI not found")
	}

	return nil
}

// ListCIs retrieves CIs with pagination and filtering
func (r *CIRepository) ListCIs(ctx context.Context, req *models.ListCIsRequest) (*models.ListCIsResponse, error) {
	// Build WHERE clause
	whereConditions := []string{"is_deleted = false"}
	args := []interface{}{}
	argCount := 1

	if req.Search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf(
			"(name ILIKE $%d OR type ILIKE $%d OR description ILIKE $%d OR owner ILIKE $%d OR location ILIKE $%d)",
			argCount, argCount+1, argCount+2, argCount+3, argCount+4,
		))
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
		argCount += 5
	}

	if req.Type != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("type = $%d", argCount))
		args = append(args, req.Type)
		argCount++
	}

	if req.Status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, req.Status)
		argCount++
	}

	if req.Criticality != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("criticality = $%d", argCount))
		args = append(args, req.Criticality)
		argCount++
	}

	if req.Owner != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("owner = $%d", argCount))
		args = append(args, req.Owner)
		argCount++
	}

	if req.Location != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("location = $%d", argCount))
		args = append(args, req.Location)
		argCount++
	}

	if len(req.Tags) > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("tags && $%d", argCount))
		args = append(args, pq.Array(req.Tags))
		argCount++
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if req.SortBy != "" {
		validSortFields := map[string]bool{
			"name": true, "type": true, "status": true, "criticality": true,
			"owner": true, "location": true, "created_at": true, "updated_at": true,
		}
		if validSortFields[req.SortBy] {
			orderBy = req.SortBy
			if req.SortOrder == "desc" {
				orderBy += " DESC"
			} else {
				orderBy += " ASC"
			}
		}
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM configuration_items WHERE %s", whereClause)
	var totalCount int64
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count CIs: %w", err)
	}

	// Calculate pagination
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize
	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	// Build SELECT query
	query := fmt.Sprintf(`
		SELECT id, name, type, description, status, criticality, owner, location,
		       attributes, tags, install_date, warranty_expiry, last_updated, last_scanned,
		       is_active, is_deleted, created_at, updated_at, created_by, updated_by
		FROM configuration_items 
		WHERE %s 
		ORDER BY %s 
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, argCount, argCount+1)

	args = append(args, req.PageSize, offset)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list CIs: %w", err)
	}
	defer rows.Close()

	var cis []models.CI
	for rows.Next() {
		var ci models.CI
		if err := rows.StructScan(&ci); err != nil {
			return nil, fmt.Errorf("failed to scan CI: %w", err)
		}
		cis = append(cis, &ci)
	}

	return &models.ListCIsResponse{
		CIs:        cis,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateRelationship creates a new relationship between CIs
func (r *CIRepository) CreateRelationship(ctx context.Context, rel *models.CIRelationship) (*models.CIRelationship, error) {
	query := `
		INSERT INTO ci_relationships (
			id, source_ci_id, target_ci_id, type, attributes, description,
			is_active, created_at, updated_at, created_by, updated_by
		) VALUES (
			:id, :source_ci_id, :target_ci_id, :type, :attributes, :description,
			:is_active, :created_at, :updated_at, :created_by, :updated_by
		)
		RETURNING id, source_ci_id, target_ci_id, type, attributes, description,
		          is_active, created_at, updated_at, created_by, updated_by`

	// Set timestamps if not provided
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = time.Now()
	}
	if rel.UpdatedAt.IsZero() {
		rel.UpdatedAt = time.Now()
	}

	// Set default values
	if !rel.IsActive {
		rel.IsActive = true
	}

	rows, err := r.db.NamedQueryContext(ctx, query, rel)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}
	defer rows.Close()

	var createdRel models.CIRelationship
	if rows.Next() {
		if err := rows.StructScan(&createdRel); err != nil {
			return nil, fmt.Errorf("failed to scan created relationship: %w", err)
		}
	}

	return &createdRel, nil
}

// GetRelationship retrieves a relationship by ID
func (r *CIRepository) GetRelationship(ctx context.Context, id uuid.UUID) (*models.CIRelationship, error) {
	query := `
		SELECT id, source_ci_id, target_ci_id, type, attributes, description,
		       is_active, created_at, updated_at, created_by, updated_by
		FROM ci_relationships 
		WHERE id = $1`

	var rel models.CIRelationship
	err := r.db.GetContext(ctx, &rel, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("relationship not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get relationship: %w", err)
	}

	return &rel, nil
}

// UpdateRelationship updates an existing relationship
func (r *CIRepository) UpdateRelationship(ctx context.Context, rel *models.CIRelationship) (*models.CIRelationship, error) {
	query := `
		UPDATE ci_relationships SET
			type = :type,
			attributes = :attributes,
			description = :description,
			is_active = :is_active,
			updated_at = :updated_at,
			updated_by = :updated_by
		WHERE id = :id
		RETURNING id, source_ci_id, target_ci_id, type, attributes, description,
		          is_active, created_at, updated_at, created_by, updated_by`

	// Set updated timestamp
	rel.UpdatedAt = time.Now()

	rows, err := r.db.NamedQueryContext(ctx, query, rel)
	if err != nil {
		return nil, fmt.Errorf("failed to update relationship: %w", err)
	}
	defer rows.Close()

	var updatedRel models.CIRelationship
	if rows.Next() {
		if err := rows.StructScan(&updatedRel); err != nil {
			return nil, fmt.Errorf("failed to scan updated relationship: %w", err)
		}
	} else {
		return nil, fmt.Errorf("relationship not found")
	}

	return &updatedRel, nil
}

// DeleteRelationship deletes a relationship
func (r *CIRepository) DeleteRelationship(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ci_relationships WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("relationship not found")
	}

	return nil
}

// GetRelationshipsByCI retrieves all relationships for a CI
func (r *CIRepository) GetRelationshipsByCI(ctx context.Context, ciID uuid.UUID) ([]*models.CIRelationship, error) {
	query := `
		SELECT id, source_ci_id, target_ci_id, type, attributes, description,
		       is_active, created_at, updated_at, created_by, updated_by
		FROM ci_relationships 
		WHERE (source_ci_id = $1 OR target_ci_id = $1) AND is_active = true`

	rows, err := r.db.QueryxContext(ctx, query, ciID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships by CI: %w", err)
	}
	defer rows.Close()

	var relationships []*models.CIRelationship
	for rows.Next() {
		var rel models.CIRelationship
		if err := rows.StructScan(&rel); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		relationships = append(relationships, &rel)
	}

	return relationships, nil
}

// CheckCircularDependency checks for circular dependencies in relationships
func (r *CIRepository) CheckCircularDependency(ctx context.Context, sourceCIID, targetCIID uuid.UUID, relationshipType string) (bool, error) {
	// This is a simplified check - in a real implementation, you'd use graph traversal
	// For now, we'll check if there's already a reverse relationship of the same type
	query := `
		SELECT COUNT(*) FROM ci_relationships 
		WHERE source_ci_id = $1 AND target_ci_id = $2 AND type = $3 AND is_active = true`

	var count int
	err := r.db.GetContext(ctx, &count, query, targetCIID, sourceCIID, relationshipType)
	if err != nil {
		return false, fmt.Errorf("failed to check circular dependency: %w", err)
	}

	return count > 0, nil
}

// Schema Management Methods

// CreateCITypeSchema creates a new CI type schema
func (r *CIRepository) CreateCITypeSchema(ctx context.Context, schema *models.CITypeSchema) (*models.CITypeSchema, error) {
	query := `
		INSERT INTO ci_type_schemas (
			id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		) VALUES (
			:id, :name, :description, :attributes, :is_active, :created_at, :updated_at, :created_by, :updated_by
		)
		RETURNING id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by`

	// Set timestamps if not provided
	if schema.CreatedAt.IsZero() {
		schema.CreatedAt = time.Now()
	}
	if schema.UpdatedAt.IsZero() {
		schema.UpdatedAt = time.Now()
	}

	// Set default values
	if !schema.IsActive {
		schema.IsActive = true
	}

	// Convert attributes to JSON
	attributesJSON, err := json.Marshal(schema.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Create a map for the query
	schemaMap := map[string]interface{}{
		"id":          schema.ID,
		"name":        schema.Name,
		"description": schema.Description,
		"attributes":  attributesJSON,
		"is_active":   schema.IsActive,
		"created_at":  schema.CreatedAt,
		"updated_at":  schema.UpdatedAt,
		"created_by":  schema.CreatedBy,
		"updated_by":  schema.UpdatedBy,
	}

	rows, err := r.db.NamedQueryContext(ctx, query, schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create CI type schema: %w", err)
	}
	defer rows.Close()

	var createdSchema models.CITypeSchema
	if rows.Next() {
		if err := rows.StructScan(&createdSchema); err != nil {
			return nil, fmt.Errorf("failed to scan created CI type schema: %w", err)
		}
		// Unmarshal attributes
		if err := json.Unmarshal(createdSchema.Attributes, &createdSchema.Attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	}

	return &createdSchema, nil
}

// GetCITypeSchema retrieves a CI type schema by ID
func (r *CIRepository) GetCITypeSchema(ctx context.Context, id uuid.UUID) (*models.CITypeSchema, error) {
	query := `
		SELECT id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		FROM ci_type_schemas 
		WHERE id = $1`

	var schema models.CITypeSchema
	err := r.db.GetContext(ctx, &schema, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CI type schema not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get CI type schema: %w", err)
	}

	// Unmarshal attributes
	if err := json.Unmarshal(schema.Attributes, &schema.Attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	return &schema, nil
}

// GetCITypeSchemaByName retrieves a CI type schema by name
func (r *CIRepository) GetCITypeSchemaByName(ctx context.Context, name string) (*models.CITypeSchema, error) {
	query := `
		SELECT id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		FROM ci_type_schemas 
		WHERE name = $1 AND is_active = true`

	var schema models.CITypeSchema
	err := r.db.GetContext(ctx, &schema, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CI type schema not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get CI type schema by name: %w", err)
	}

	// Unmarshal attributes
	if err := json.Unmarshal(schema.Attributes, &schema.Attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	return &schema, nil
}

// UpdateCITypeSchema updates an existing CI type schema
func (r *CIRepository) UpdateCITypeSchema(ctx context.Context, schema *models.CITypeSchema) (*models.CITypeSchema, error) {
	query := `
		UPDATE ci_type_schemas SET
			name = :name,
			description = :description,
			attributes = :attributes,
			is_active = :is_active,
			updated_at = :updated_at,
			updated_by = :updated_by
		WHERE id = :id
		RETURNING id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by`

	// Set updated timestamp
	schema.UpdatedAt = time.Now()

	// Convert attributes to JSON
	attributesJSON, err := json.Marshal(schema.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Create a map for the query
	schemaMap := map[string]interface{}{
		"id":          schema.ID,
		"name":        schema.Name,
		"description": schema.Description,
		"attributes":  attributesJSON,
		"is_active":   schema.IsActive,
		"updated_at":  schema.UpdatedAt,
		"updated_by":  schema.UpdatedBy,
	}

	rows, err := r.db.NamedQueryContext(ctx, query, schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to update CI type schema: %w", err)
	}
	defer rows.Close()

	var updatedSchema models.CITypeSchema
	if rows.Next() {
		if err := rows.StructScan(&updatedSchema); err != nil {
			return nil, fmt.Errorf("failed to scan updated CI type schema: %w", err)
		}
		// Unmarshal attributes
		if err := json.Unmarshal(updatedSchema.Attributes, &updatedSchema.Attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	} else {
		return nil, fmt.Errorf("CI type schema not found")
	}

	return &updatedSchema, nil
}

// DeleteCITypeSchema deletes a CI type schema
func (r *CIRepository) DeleteCITypeSchema(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ci_type_schemas WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete CI type schema: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("CI type schema not found")
	}

	return nil
}

// ListCITypeSchemas retrieves CI type schemas with pagination
func (r *CIRepository) ListCITypeSchemas(ctx context.Context, page, pageSize int) ([]*models.CITypeSchema, int64, error) {
	// Count total records
	var totalCount int64
	err := r.db.GetContext(ctx, &totalCount, "SELECT COUNT(*) FROM ci_type_schemas")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count CI type schemas: %w", err)
	}

	// Calculate pagination
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := `
		SELECT id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		FROM ci_type_schemas 
		ORDER BY name 
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryxContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list CI type schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*models.CITypeSchema
	for rows.Next() {
		var schema models.CITypeSchema
		if err := rows.StructScan(&schema); err != nil {
			return nil, 0, fmt.Errorf("failed to scan CI type schema: %w", err)
		}
		// Unmarshal attributes
		if err := json.Unmarshal(schema.Attributes, &schema.Attributes); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
		schemas = append(schemas, &schema)
	}

	return schemas, totalCount, nil
}

// Relationship Type Schema Methods

// CreateRelationshipTypeSchema creates a new relationship type schema
func (r *CIRepository) CreateRelationshipTypeSchema(ctx context.Context, schema *models.RelationshipTypeSchema) (*models.RelationshipTypeSchema, error) {
	query := `
		INSERT INTO relationship_type_schemas (
			id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		) VALUES (
			:id, :name, :description, :attributes, :is_active, :created_at, :updated_at, :created_by, :updated_by
		)
		RETURNING id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by`

	// Set timestamps if not provided
	if schema.CreatedAt.IsZero() {
		schema.CreatedAt = time.Now()
	}
	if schema.UpdatedAt.IsZero() {
		schema.UpdatedAt = time.Now()
	}

	// Set default values
	if !schema.IsActive {
		schema.IsActive = true
	}

	// Convert attributes to JSON
	attributesJSON, err := json.Marshal(schema.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Create a map for the query
	schemaMap := map[string]interface{}{
		"id":          schema.ID,
		"name":        schema.Name,
		"description": schema.Description,
		"attributes":  attributesJSON,
		"is_active":   schema.IsActive,
		"created_at":  schema.CreatedAt,
		"updated_at":  schema.UpdatedAt,
		"created_by":  schema.CreatedBy,
		"updated_by":  schema.UpdatedBy,
	}

	rows, err := r.db.NamedQueryContext(ctx, query, schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship type schema: %w", err)
	}
	defer rows.Close()

	var createdSchema models.RelationshipTypeSchema
	if rows.Next() {
		if err := rows.StructScan(&createdSchema); err != nil {
			return nil, fmt.Errorf("failed to scan created relationship type schema: %w", err)
		}
		// Unmarshal attributes
		if err := json.Unmarshal(createdSchema.Attributes, &createdSchema.Attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	}

	return &createdSchema, nil
}

// GetRelationshipTypeSchema retrieves a relationship type schema by ID
func (r *CIRepository) GetRelationshipTypeSchema(ctx context.Context, id uuid.UUID) (*models.RelationshipTypeSchema, error) {
	query := `
		SELECT id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		FROM relationship_type_schemas 
		WHERE id = $1`

	var schema models.RelationshipTypeSchema
	err := r.db.GetContext(ctx, &schema, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("relationship type schema not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get relationship type schema: %w", err)
	}

	// Unmarshal attributes
	if err := json.Unmarshal(schema.Attributes, &schema.Attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	return &schema, nil
}

// GetRelationshipTypeSchemaByName retrieves a relationship type schema by name
func (r *CIRepository) GetRelationshipTypeSchemaByName(ctx context.Context, name string) (*models.RelationshipTypeSchema, error) {
	query := `
		SELECT id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		FROM relationship_type_schemas 
		WHERE name = $1 AND is_active = true`

	var schema models.RelationshipTypeSchema
	err := r.db.GetContext(ctx, &schema, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("relationship type schema not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get relationship type schema by name: %w", err)
	}

	// Unmarshal attributes
	if err := json.Unmarshal(schema.Attributes, &schema.Attributes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	return &schema, nil
}

// UpdateRelationshipTypeSchema updates an existing relationship type schema
func (r *CIRepository) UpdateRelationshipTypeSchema(ctx context.Context, schema *models.RelationshipTypeSchema) (*models.RelationshipTypeSchema, error) {
	query := `
		UPDATE relationship_type_schemas SET
			name = :name,
			description = :description,
			attributes = :attributes,
			is_active = :is_active,
			updated_at = :updated_at,
			updated_by = :updated_by
		WHERE id = :id
		RETURNING id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by`

	// Set updated timestamp
	schema.UpdatedAt = time.Now()

	// Convert attributes to JSON
	attributesJSON, err := json.Marshal(schema.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Create a map for the query
	schemaMap := map[string]interface{}{
		"id":          schema.ID,
		"name":        schema.Name,
		"description": schema.Description,
		"attributes":  attributesJSON,
		"is_active":   schema.IsActive,
		"updated_at":  schema.UpdatedAt,
		"updated_by":  schema.UpdatedBy,
	}

	rows, err := r.db.NamedQueryContext(ctx, query, schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to update relationship type schema: %w", err)
	}
	defer rows.Close()

	var updatedSchema models.RelationshipTypeSchema
	if rows.Next() {
		if err := rows.StructScan(&updatedSchema); err != nil {
			return nil, fmt.Errorf("failed to scan updated relationship type schema: %w", err)
		}
		// Unmarshal attributes
		if err := json.Unmarshal(updatedSchema.Attributes, &updatedSchema.Attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	} else {
		return nil, fmt.Errorf("relationship type schema not found")
	}

	return &updatedSchema, nil
}

// DeleteRelationshipTypeSchema deletes a relationship type schema
func (r *CIRepository) DeleteRelationshipTypeSchema(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM relationship_type_schemas WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete relationship type schema: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("relationship type schema not found")
	}

	return nil
}

// ListRelationshipTypeSchemas retrieves relationship type schemas with pagination
func (r *CIRepository) ListRelationshipTypeSchemas(ctx context.Context, page, pageSize int) ([]*models.RelationshipTypeSchema, int64, error) {
	// Count total records
	var totalCount int64
	err := r.db.GetContext(ctx, &totalCount, "SELECT COUNT(*) FROM relationship_type_schemas")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count relationship type schemas: %w", err)
	}

	// Calculate pagination
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := `
		SELECT id, name, description, attributes, is_active, created_at, updated_at, created_by, updated_by
		FROM relationship_type_schemas 
		ORDER BY name 
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryxContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list relationship type schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*models.RelationshipTypeSchema
	for rows.Next() {
		var schema models.RelationshipTypeSchema
		if err := rows.StructScan(&schema); err != nil {
			return nil, 0, fmt.Errorf("failed to scan relationship type schema: %w", err)
		}
		// Unmarshal attributes
		if err := json.Unmarshal(schema.Attributes, &schema.Attributes); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
		schemas = append(schemas, &schema)
	}

	return schemas, totalCount, nil
}

// Schema Validation Methods

// ValidateCIAgainstSchema validates CI data against a CI type schema
func (r *CIRepository) ValidateCIAgainstSchema(ctx context.Context, ci *models.CI, schema *models.CITypeSchema) (*models.ValidationResult, error) {
	validator := models.NewSchemaValidator()
	result := validator.ValidateCIAgainstSchema(*ci, *schema)
	return &result, nil
}

// ValidateRelationshipAgainstSchema validates relationship data against a relationship type schema
func (r *CIRepository) ValidateRelationshipAgainstSchema(ctx context.Context, relationship *models.CIRelationship, schema *models.RelationshipTypeSchema) (*models.ValidationResult, error) {
	validator := models.NewSchemaValidator()
	result := validator.ValidateRelationshipAgainstSchema(*relationship, *schema)
	return &result, nil
}

// GetCISchemaByType retrieves the CI type schema for a given CI type
func (r *CIRepository) GetCISchemaByType(ctx context.Context, ciType string) (*models.CITypeSchema, error) {
	return r.GetCITypeSchemaByName(ctx, ciType)
}

// GetRelationshipSchemaByType retrieves the relationship type schema for a given relationship type
func (r *CIRepository) GetRelationshipSchemaByType(ctx context.Context, relType string) (*models.RelationshipTypeSchema, error) {
	return r.GetRelationshipTypeSchemaByName(ctx, relType)
}

// CreateCIWithValidation creates a CI with schema validation
func (r *CIRepository) CreateCIWithValidation(ctx context.Context, ci *models.CI, schema *models.CITypeSchema) (*models.CI, error) {
	// Validate against schema
	validationResult, err := r.ValidateCIAgainstSchema(ctx, ci, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to validate CI: %w", err)
	}

	if !validationResult.IsValid {
		return nil, fmt.Errorf("CI validation failed: %v", validationResult.Errors)
	}

	// Apply defaults if needed
	var attributes map[string]interface{}
	if len(ci.Attributes) > 0 {
		if err := json.Unmarshal(ci.Attributes, &attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	} else {
		attributes = make(map[string]interface{})
	}

	validator := models.NewSchemaValidator()
	attributes = validator.ApplyDefaults(attributes, *schema)

	// Marshal attributes back to JSON
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}
	ci.Attributes = attributesJSON

	// Create the CI
	return r.CreateCI(ctx, ci)
}

// UpdateCIWithValidation updates a CI with schema validation
func (r *CIRepository) UpdateCIWithValidation(ctx context.Context, ci *models.CI, schema *models.CITypeSchema) (*models.CI, error) {
	// Validate against schema
	validationResult, err := r.ValidateCIAgainstSchema(ctx, ci, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to validate CI: %w", err)
	}

	if !validationResult.IsValid {
		return nil, fmt.Errorf("CI validation failed: %v", validationResult.Errors)
	}

	// Apply defaults if needed
	var attributes map[string]interface{}
	if len(ci.Attributes) > 0 {
		if err := json.Unmarshal(ci.Attributes, &attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	} else {
		attributes = make(map[string]interface{})
	}

	validator := models.NewSchemaValidator()
	attributes = validator.ApplyDefaults(attributes, *schema)

	// Marshal attributes back to JSON
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}
	ci.Attributes = attributesJSON

	// Update the CI
	return r.UpdateCI(ctx, ci)
}

// CreateRelationshipWithValidation creates a relationship with schema validation
func (r *CIRepository) CreateRelationshipWithValidation(ctx context.Context, relationship *models.CIRelationship, schema *models.RelationshipTypeSchema) (*models.CIRelationship, error) {
	// Validate against schema
	validationResult, err := r.ValidateRelationshipAgainstSchema(ctx, relationship, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to validate relationship: %w", err)
	}

	if !validationResult.IsValid {
		return nil, fmt.Errorf("relationship validation failed: %v", validationResult.Errors)
	}

	// Apply defaults if needed
	var attributes map[string]interface{}
	if len(relationship.Attributes) > 0 {
		if err := json.Unmarshal(relationship.Attributes, &attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	} else {
		attributes = make(map[string]interface{})
	}

	validator := models.NewSchemaValidator()
	attributes = validator.ApplyDefaults(attributes, models.CITypeSchema{Attributes: schema.Attributes})

	// Marshal attributes back to JSON
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}
	relationship.Attributes = attributesJSON

	// Create the relationship
	return r.CreateRelationship(ctx, relationship)
}

// UpdateRelationshipWithValidation updates a relationship with schema validation
func (r *CIRepository) UpdateRelationshipWithValidation(ctx context.Context, relationship *models.CIRelationship, schema *models.RelationshipTypeSchema) (*models.CIRelationship, error) {
	// Validate against schema
	validationResult, err := r.ValidateRelationshipAgainstSchema(ctx, relationship, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to validate relationship: %w", err)
	}

	if !validationResult.IsValid {
		return nil, fmt.Errorf("relationship validation failed: %v", validationResult.Errors)
	}

	// Apply defaults if needed
	var attributes map[string]interface{}
	if len(relationship.Attributes) > 0 {
		if err := json.Unmarshal(relationship.Attributes, &attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attributes: %w", err)
		}
	} else {
		attributes = make(map[string]interface{})
	}

	validator := models.NewSchemaValidator()
	attributes = validator.ApplyDefaults(attributes, models.CITypeSchema{Attributes: schema.Attributes})

	// Marshal attributes back to JSON
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}
	relationship.Attributes = attributesJSON

	// Update the relationship
	return r.UpdateRelationship(ctx, relationship)
}
