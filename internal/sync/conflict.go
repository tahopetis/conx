package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/conx/cmdb/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

// ConflictType represents the type of conflict that can occur
type ConflictType string

const (
	ConflictTypeDataMismatch    ConflictType = "data_mismatch"
	ConflictTypeMissingEntity  ConflictType = "missing_entity"
	ConflictTypeRelationship   ConflictType = "relationship_conflict"
	ConflictTypeTimestamp      ConflictType = "timestamp_conflict"
	ConflictTypeVersion        ConflictType = "version_conflict"
)

// ConflictResolution represents how conflicts should be resolved
type ConflictResolution string

const (
	ResolutionPostgresWins ConflictResolution = "postgres_wins"
	ResolutionNeo4jWins   ConflictResolution = "neo4j_wins"
	ResolutionMerge       ConflictResolution = "merge"
	ResolutionManual      ConflictResolution = "manual"
	ResolutionTimestamp   ConflictResolution = "timestamp"
)

// Conflict represents a synchronization conflict
type Conflict struct {
	ID             string                 `json:"id"`
	EntityType     string                 `json:"entity_type"`
	EntityID       string                 `json:"entity_id"`
	ConflictType   ConflictType           `json:"conflict_type"`
	PostgresData   map[string]interface{} `json:"postgres_data"`
	Neo4jData      map[string]interface{} `json:"neo4j_data"`
	Resolution     ConflictResolution     `json:"resolution"`
	Resolved       bool                   `json:"resolved"`
	ResolvedBy     string                 `json:"resolved_by,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ConflictResolver handles conflict detection and resolution
type ConflictResolver struct {
	dbManager   *database.Manager
	strategy    ConflictResolution
	logger      *log.Logger
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(dbManager *database.Manager, strategy ConflictResolution, logger *log.Logger) *ConflictResolver {
	return &ConflictResolver{
		dbManager: dbManager,
		strategy:  strategy,
		logger:    logger,
	}
}

// DetectAndResolve detects conflicts and resolves them based on the configured strategy
func (cr *ConflictResolver) DetectAndResolve(ctx context.Context, event SyncEvent) (*Conflict, error) {
	conflict, err := cr.detectConflict(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to detect conflict: %w", err)
	}

	if conflict == nil {
		return nil, nil // No conflict detected
	}

	// Resolve the conflict
	if err := cr.resolveConflict(ctx, conflict); err != nil {
		return nil, fmt.Errorf("failed to resolve conflict: %w", err)
	}

	return conflict, nil
}

// detectConflict detects if there's a conflict between PostgreSQL and Neo4j data
func (cr *ConflictResolver) detectConflict(ctx context.Context, event SyncEvent) (*Conflict, error) {
	switch event.EntityType {
	case "configuration_item":
		return cr.detectCIConflict(ctx, event)
	case "relationship":
		return cr.detectRelationshipConflict(ctx, event)
	default:
		return nil, nil // Skip conflict detection for unsupported entity types
	}
}

// detectCIConflict detects conflicts for configuration items
func (cr *ConflictResolver) detectCIConflict(ctx context.Context, event SyncEvent) (*Conflict, error) {
	// Get data from PostgreSQL
	postgresData, err := cr.getCIDataFromPostgres(ctx, event.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CI data from PostgreSQL: %w", err)
	}

	// Get data from Neo4j
	neo4jData, err := cr.getCIDataFromNeo4j(ctx, event.EntityID)
	if err != nil {
		// If entity doesn't exist in Neo4j, it's not necessarily a conflict
		// This could be a CREATE operation
		return nil, nil
	}

	// Compare data to detect conflicts
	conflictType := cr.compareCIData(postgresData, neo4jData)
	if conflictType == "" {
		return nil, nil // No conflict detected
	}

	conflict := &Conflict{
		ID:           fmt.Sprintf("conflict_%d", time.Now().UnixNano()),
		EntityType:   event.EntityType,
		EntityID:     event.EntityID,
		ConflictType: conflictType,
		PostgresData: postgresData,
		Neo4jData:    neo4jData,
		Resolution:   cr.strategy,
		Resolved:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Store conflict in database
	if err := cr.storeConflict(ctx, conflict); err != nil {
		cr.logger.Error().Err(err).Str("conflict_id", conflict.ID).Msg("Failed to store conflict")
	}

	cr.logger.Warn().
		Str("conflict_id", conflict.ID).
		Str("entity_type", conflict.EntityType).
		Str("entity_id", conflict.EntityID).
		Str("conflict_type", string(conflict.ConflictType)).
		Msg("Conflict detected")

	return conflict, nil
}

// detectRelationshipConflict detects conflicts for relationships
func (cr *ConflictResolver) detectRelationshipConflict(ctx context.Context, event SyncEvent) (*Conflict, error) {
	// Get data from PostgreSQL
	postgresData, err := cr.getRelationshipDataFromPostgres(ctx, event.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationship data from PostgreSQL: %w", err)
	}

	// Get data from Neo4j
	neo4jData, err := cr.getRelationshipDataFromNeo4j(ctx, event.EntityID)
	if err != nil {
		// If relationship doesn't exist in Neo4j, it's not necessarily a conflict
		return nil, nil
	}

	// Compare data to detect conflicts
	conflictType := cr.compareRelationshipData(postgresData, neo4jData)
	if conflictType == "" {
		return nil, nil // No conflict detected
	}

	conflict := &Conflict{
		ID:           fmt.Sprintf("conflict_%d", time.Now().UnixNano()),
		EntityType:   event.EntityType,
		EntityID:     event.EntityID,
		ConflictType: conflictType,
		PostgresData: postgresData,
		Neo4jData:    neo4jData,
		Resolution:   cr.strategy,
		Resolved:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Store conflict in database
	if err := cr.storeConflict(ctx, conflict); err != nil {
		cr.logger.Error().Err(err).Str("conflict_id", conflict.ID).Msg("Failed to store conflict")
	}

	cr.logger.Warn().
		Str("conflict_id", conflict.ID).
		Str("entity_type", conflict.EntityType).
		Str("entity_id", conflict.EntityID).
		Str("conflict_type", string(conflict.ConflictType)).
		Msg("Conflict detected")

	return conflict, nil
}

// getCIDataFromPostgres retrieves CI data from PostgreSQL
func (cr *ConflictResolver) getCIDataFromPostgres(ctx context.Context, ciID string) (map[string]interface{}, error) {
	var dataJSON []byte
	err := cr.dbManager.Postgres.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'id', id, 'name', name, 'type', type, 'description', description,
			'status', status, 'attributes', attributes, 'tags', tags,
			'created_by', created_by, 'created_at', created_at, 'updated_at', updated_at
		) FROM configuration_items WHERE id = $1
	`, ciID).Scan(&dataJSON)

	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CI data: %w", err)
	}

	return data, nil
}

// getCIDataFromNeo4j retrieves CI data from Neo4j
func (cr *ConflictResolver) getCIDataFromNeo4j(ctx context.Context, ciID string) (map[string]interface{}, error) {
	session := cr.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (n:ConfigurationItem {id: $ciId})
		RETURN n.name as name, n.type as type, n.description as description, 
		       n.status as status, n.attributes as attributes, n.tags as tags,
		       n.created_by as created_by, n.created_at as created_at, n.updated_at as updated_at
	`, map[string]interface{}{"ciId": ciID})

	if err != nil {
		return nil, err
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("CI not found in Neo4j")
	}

	data := make(map[string]interface{})
	if err := result.Scan(&data["name"], &data["type"], &data["description"], 
		&data["status"], &data["attributes"], &data["tags"],
		&data["created_by"], &data["created_at"], &data["updated_at"]); err != nil {
		return nil, err
	}

	return data, nil
}

// getRelationshipDataFromPostgres retrieves relationship data from PostgreSQL
func (cr *ConflictResolver) getRelationshipDataFromPostgres(ctx context.Context, relID string) (map[string]interface{}, error) {
	var dataJSON []byte
	err := cr.dbManager.Postgres.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'id', id, 'source_id', source_id, 'target_id', target_id,
			'type', type, 'description', description, 'attributes', attributes,
			'strength', strength, 'created_by', created_by, 'created_at', created_at
		) FROM relationships WHERE id = $1
	`, relID).Scan(&dataJSON)

	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relationship data: %w", err)
	}

	return data, nil
}

// getRelationshipDataFromNeo4j retrieves relationship data from Neo4j
func (cr *ConflictResolver) getRelationshipDataFromNeo4j(ctx context.Context, relID string) (map[string]interface{}, error) {
	session := cr.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (source)-[r:RELATIONSHIP {id: $relId}]->(target)
		RETURN source.id as source_id, target.id as target_id, r.type as type, 
		       r.description as description, r.attributes as attributes,
		       r.strength as strength, r.created_by as created_by, r.created_at as created_at
	`, map[string]interface{}{"relId": relID})

	if err != nil {
		return nil, err
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("Relationship not found in Neo4j")
	}

	data := make(map[string]interface{})
	if err := result.Scan(&data["source_id"], &data["target_id"], &data["type"], 
		&data["description"], &data["attributes"], &data["strength"],
		&data["created_by"], &data["created_at"]); err != nil {
		return nil, err
	}

	return data, nil
}

// compareCIData compares CI data and returns the conflict type if there's a mismatch
func (cr *ConflictResolver) compareCIData(postgresData, neo4jData map[string]interface{}) ConflictType {
	// Compare basic fields
	if postgresData["name"] != neo4jData["name"] ||
		postgresData["type"] != neo4jData["type"] ||
		postgresData["status"] != neo4jData["status"] {
		return ConflictTypeDataMismatch
	}

	// Compare timestamps
	postgresUpdatedAt, _ := postgresData["updated_at"].(time.Time)
	neo4jUpdatedAt, _ := neo4jData["updated_at"].(time.Time)
	if !postgresUpdatedAt.Equal(neo4jUpdatedAt) {
		return ConflictTypeTimestamp
	}

	// Compare attributes (JSON comparison)
	postgresAttrs, _ := postgresData["attributes"].(map[string]interface{})
	neo4jAttrs, _ := neo4jData["attributes"].(map[string]interface{})
	if !cr.compareMaps(postgresAttrs, neo4jAttrs) {
		return ConflictTypeDataMismatch
	}

	// Compare tags
	postgresTags, _ := postgresData["tags"].([]interface{})
	neo4jTags, _ := neo4jData["tags"].([]interface{})
	if !cr.compareSlices(postgresTags, neo4jTags) {
		return ConflictTypeDataMismatch
	}

	return "" // No conflict
}

// compareRelationshipData compares relationship data and returns the conflict type if there's a mismatch
func (cr *ConflictResolver) compareRelationshipData(postgresData, neo4jData map[string]interface{}) ConflictType {
	// Compare basic fields
	if postgresData["type"] != neo4jData["type"] ||
		postgresData["source_id"] != neo4jData["source_id"] ||
		postgresData["target_id"] != neo4jData["target_id"] {
		return ConflictTypeRelationship
	}

	// Compare attributes
	postgresAttrs, _ := postgresData["attributes"].(map[string]interface{})
	neo4jAttrs, _ := neo4jData["attributes"].(map[string]interface{})
	if !cr.compareMaps(postgresAttrs, neo4jAttrs) {
		return ConflictTypeDataMismatch
	}

	return "" // No conflict
}

// compareMaps compares two maps for equality
func (cr *ConflictResolver) compareMaps(map1, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, val1 := range map1 {
		val2, exists := map2[key]
		if !exists {
			return false
		}
		if fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", val2) {
			return false
		}
	}

	return true
}

// compareSlices compares two slices for equality
func (cr *ConflictResolver) compareSlices(slice1, slice2 []interface{}) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	for i, val1 := range slice1 {
		if fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", slice2[i]) {
			return false
		}
	}

	return true
}

// resolveConflict resolves a conflict based on the configured strategy
func (cr *ConflictResolver) resolveConflict(ctx context.Context, conflict *Conflict) error {
	switch conflict.Resolution {
	case ResolutionPostgresWins:
		return cr.resolveWithPostgresData(ctx, conflict)
	case ResolutionNeo4jWins:
		return cr.resolveWithNeo4jData(ctx, conflict)
	case ResolutionMerge:
		return cr.resolveWithMerge(ctx, conflict)
	case ResolutionTimestamp:
		return cr.resolveWithTimestamp(ctx, conflict)
	case ResolutionManual:
		return fmt.Errorf("manual resolution not implemented")
	default:
		return fmt.Errorf("unknown conflict resolution strategy: %s", conflict.Resolution)
	}
}

// resolveWithPostgresData resolves conflict by using PostgreSQL data
func (cr *ConflictResolver) resolveWithPostgresData(ctx context.Context, conflict *Conflict) error {
	cr.logger.Info().
		Str("conflict_id", conflict.ID).
		Str("strategy", string(conflict.Resolution)).
		Msg("Resolving conflict with PostgreSQL data")

	// Update Neo4j with PostgreSQL data
	switch conflict.EntityType {
	case "configuration_item":
		return cr.updateNeo4jWithPostgresCIData(ctx, conflict)
	case "relationship":
		return cr.updateNeo4jWithPostgresRelationshipData(ctx, conflict)
	default:
		return fmt.Errorf("unsupported entity type for conflict resolution: %s", conflict.EntityType)
	}
}

// resolveWithNeo4jData resolves conflict by using Neo4j data
func (cr *ConflictResolver) resolveWithNeo4jData(ctx context.Context, conflict *Conflict) error {
	cr.logger.Info().
		Str("conflict_id", conflict.ID).
		Str("strategy", string(conflict.Resolution)).
		Msg("Resolving conflict with Neo4j data")

	// Update PostgreSQL with Neo4j data
	switch conflict.EntityType {
	case "configuration_item":
		return cr.updatePostgresWithNeo4jCIData(ctx, conflict)
	case "relationship":
		return cr.updatePostgresWithNeo4jRelationshipData(ctx, conflict)
	default:
		return fmt.Errorf("unsupported entity type for conflict resolution: %s", conflict.EntityType)
	}
}

// resolveWithMerge resolves conflict by merging data from both sources
func (cr *ConflictResolver) resolveWithMerge(ctx context.Context, conflict *Conflict) error {
	cr.logger.Info().
		Str("conflict_id", conflict.ID).
		Str("strategy", string(conflict.Resolution)).
		Msg("Resolving conflict with merged data")

	switch conflict.EntityType {
	case "configuration_item":
		return cr.mergeCIData(ctx, conflict)
	case "relationship":
		return cr.mergeRelationshipData(ctx, conflict)
	default:
		return fmt.Errorf("unsupported entity type for conflict resolution: %s", conflict.EntityType)
	}
}

// resolveWithTimestamp resolves conflict by using the most recently updated data
func (cr *ConflictResolver) resolveWithTimestamp(ctx context.Context, conflict *Conflict) error {
	cr.logger.Info().
		Str("conflict_id", conflict.ID).
		Str("strategy", string(conflict.Resolution)).
		Msg("Resolving conflict with timestamp-based strategy")

	// Get timestamps
	postgresUpdatedAt := cr.getTimestampFromData(conflict.PostgresData)
	neo4jUpdatedAt := cr.getTimestampFromData(conflict.Neo4jData)

	if postgresUpdatedAt.After(neo4jUpdatedAt) {
		// PostgreSQL data is newer
		return cr.resolveWithPostgresData(ctx, conflict)
	} else {
		// Neo4j data is newer
		return cr.resolveWithNeo4jData(ctx, conflict)
	}
}

// updateNeo4jWithPostgresCIData updates Neo4j with PostgreSQL CI data
func (cr *ConflictResolver) updateNeo4jWithPostgresCIData(ctx context.Context, conflict *Conflict) error {
	session := cr.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	data := conflict.PostgresData
	tags := cr.convertToTagsSlice(data["tags"])

	_, err := session.Run(ctx, `
		MERGE (n:ConfigurationItem {id: $id})
		SET n.name = $name,
		    n.type = $type,
		    n.description = $description,
		    n.status = $status,
		    n.attributes = $attributes,
		    n.tags = $tags,
		    n.created_by = $created_by,
		    n.created_at = $created_at,
		    n.updated_at = $updated_at,
		    n.conflict_resolved_at = datetime()
	`, map[string]interface{}{
		"id":          conflict.EntityID,
		"name":        data["name"],
		"type":        data["type"],
		"description": data["description"],
		"status":      data["status"],
		"attributes":  data["attributes"],
		"tags":        tags,
		"created_by":  data["created_by"],
		"created_at":  data["created_at"],
		"updated_at":  data["updated_at"],
	})

	return err
}

// updateNeo4jWithPostgresRelationshipData updates Neo4j with PostgreSQL relationship data
func (cr *ConflictResolver) updateNeo4jWithPostgresRelationshipData(ctx context.Context, conflict *Conflict) error {
	session := cr.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	data := conflict.PostgresData

	_, err := session.Run(ctx, `
		MATCH (source:ConfigurationItem {id: $sourceId})
		MATCH (target:ConfigurationItem {id: $targetId})
		MERGE (source)-[r:RELATIONSHIP {id: $id}]->(target)
		SET r.type = $type,
		    r.description = $description,
		    r.attributes = $attributes,
		    r.strength = $strength,
		    r.created_by = $created_by,
		    r.created_at = $created_at,
		    r.conflict_resolved_at = datetime()
	`, map[string]interface{}{
		"id":           conflict.EntityID,
		"sourceId":     data["source_id"],
		"targetId":     data["target_id"],
		"type":         data["type"],
		"description":  data["description"],
		"attributes":   data["attributes"],
		"strength":     data["strength"],
		"created_by":   data["created_by"],
		"created_at":   data["created_at"],
	})

	return err
}

// updatePostgresWithNeo4jCIData updates PostgreSQL with Neo4j CI data
func (cr *ConflictResolver) updatePostgresWithNeo4jCIData(ctx context.Context, conflict *Conflict) error {
	data := conflict.Neo4jData

	_, err := cr.dbManager.Postgres.Exec(ctx, `
		UPDATE configuration_items 
		SET name = $1, type = $2, description = $3, status = $4,
		    attributes = $4, tags = $5, updated_at = NOW(),
		    conflict_resolved_at = NOW()
		WHERE id = $6
	`, data["name"], data["type"], data["description"], data["status"],
		data["attributes"], data["tags"], conflict.EntityID)

	return err
}

// updatePostgresWithNeo4jRelationshipData updates PostgreSQL with Neo4j relationship data
func (cr *ConflictResolver) updatePostgresWithNeo4jRelationshipData(ctx context.Context, conflict *Conflict) error {
	data := conflict.Neo4jData

	_, err := cr.dbManager.Postgres.Exec(ctx, `
		UPDATE relationships 
		SET type = $1, description = $2, attributes = $3, strength = $4,
		    updated_at = NOW(), conflict_resolved_at = NOW()
		WHERE id = $5
	`, data["type"], data["description"], data["attributes"], data["strength"], conflict.EntityID)

	return err
}

// mergeCIData merges CI data from both sources
func (cr *ConflictResolver) mergeCIData(ctx context.Context, conflict *Conflict) error {
	// For CI data, we'll take PostgreSQL as the base but merge specific fields
	mergedData := make(map[string]interface{})
	
	// Copy all fields from PostgreSQL
	for k, v := range conflict.PostgresData {
		mergedData[k] = v
	}

	// Merge attributes (Neo4j might have additional attributes)
	postgresAttrs, _ := conflict.PostgresData["attributes"].(map[string]interface{})
	neo4jAttrs, _ := conflict.Neo4jData["attributes"].(map[string]interface{})
	
	mergedAttrs := make(map[string]interface{})
	for k, v := range postgresAttrs {
		mergedAttrs[k] = v
	}
	for k, v := range neo4jAttrs {
		mergedAttrs[k] = v // Neo4j attributes take precedence for conflicts
	}
	mergedData["attributes"] = mergedAttrs

	// Update both databases with merged data
	if err := cr.updateNeo4jWithPostgresCIData(ctx, &Conflict{PostgresData: mergedData}); err != nil {
		return err
	}

	return cr.updatePostgresWithNeo4jCIData(ctx, &Conflict{Neo4jData: mergedData})
}

// mergeRelationshipData merges relationship data from both sources
func (cr *ConflictResolver) mergeRelationshipData(ctx context.Context, conflict *Conflict) error {
	// For relationships, we'll take PostgreSQL as the base but merge attributes
	mergedData := make(map[string]interface{})
	
	// Copy all fields from PostgreSQL
	for k, v := range conflict.PostgresData {
		mergedData[k] = v
	}

	// Merge attributes
	postgresAttrs, _ := conflict.PostgresData["attributes"].(map[string]interface{})
	neo4jAttrs, _ := conflict.Neo4jData["attributes"].(map[string]interface{})
	
	mergedAttrs := make(map[string]interface{})
	for k, v := range postgresAttrs {
		mergedAttrs[k] = v
	}
	for k, v := range neo4jAttrs {
		mergedAttrs[k] = v // Neo4j attributes take precedence for conflicts
	}
	mergedData["attributes"] = mergedAttrs

	// Update both databases with merged data
	if err := cr.updateNeo4jWithPostgresRelationshipData(ctx, &Conflict{PostgresData: mergedData}); err != nil {
		return err
	}

	return cr.updatePostgresWithNeo4jRelationshipData(ctx, &Conflict{Neo4jData: mergedData})
}

// getTimestampFromData extracts timestamp from data map
func (cr *ConflictResolver) getTimestampFromData(data map[string]interface{}) time.Time {
	if updatedAt, ok := data["updated_at"].(time.Time); ok {
		return updatedAt
	}
	if createdAt, ok := data["created_at"].(time.Time); ok {
		return createdAt
	}
	return time.Time{} // Zero time if no timestamp found
}

// convertToTagsSlice converts interface to string slice
func (cr *ConflictResolver) convertToTagsSlice(tagsInterface interface{}) []string {
	var tags []string
	if tagsSlice, ok := tagsInterface.([]interface{}); ok {
		for _, tag := range tagsSlice {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}
	return tags
}

// storeConflict stores conflict information in the database
func (cr *ConflictResolver) storeConflict(ctx context.Context, conflict *Conflict) error {
	postgresJSON, _ := json.Marshal(conflict.PostgresData)
	neo4jJSON, _ := json.Marshal(conflict.Neo4jData)

	_, err := cr.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_conflicts (
			id, entity_type, entity_id, conflict_type, postgres_data, neo4j_data,
			resolution, resolved, resolved_by, resolved_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			entity_type = EXCLUDED.entity_type,
			entity_id = EXCLUDED.entity_id,
			conflict_type = EXCLUDED.conflict_type,
			postgres_data = EXCLUDED.postgres_data,
			neo4j_data = EXCLUDED.neo4j_data,
			resolution = EXCLUDED.resolution,
			resolved = EXCLUDED.resolved,
			resolved_by = EXCLUDED.resolved_by,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = EXCLUDED.updated_at
	`, conflict.ID, conflict.EntityType, conflict.EntityID, conflict.ConflictType,
		string(postgresJSON), string(neo4jJSON), conflict.Resolution, conflict.Resolved,
		conflict.ResolvedBy, conflict.ResolvedAt, conflict.CreatedAt, conflict.UpdatedAt)

	return err
}

// GetConflicts retrieves unresolved conflicts
func (cr *ConflictResolver) GetConflicts(ctx context.Context, limit int) ([]*Conflict, error) {
	rows, err := cr.dbManager.Postgres.Query(ctx, `
		SELECT id, entity_type, entity_id, conflict_type, postgres_data, neo4j_data,
		       resolution, resolved, resolved_by, resolved_at, created_at, updated_at
		FROM sync_conflicts 
		WHERE resolved = false 
		ORDER BY created_at DESC 
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []*Conflict
	for rows.Next() {
		conflict := &Conflict{}
		var postgresJSON, neo4jJSON []byte

		err := rows.Scan(&conflict.ID, &conflict.EntityType, &conflict.EntityID, &conflict.ConflictType,
			&postgresJSON, &neo4jJSON, &conflict.Resolution, &conflict.Resolved,
			&conflict.ResolvedBy, &conflict.ResolvedAt, &conflict.CreatedAt, &conflict.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON data
		json.Unmarshal(postgresJSON, &conflict.PostgresData)
		json.Unmarshal(neo4jJSON, &conflict.Neo4jData)

		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

// MarkConflictResolved marks a conflict as resolved
func (cr *ConflictResolver) MarkConflictResolved(ctx context.Context, conflictID, resolvedBy string) error {
	now := time.Now()
	_, err := cr.dbManager.Postgres.Exec(ctx, `
		UPDATE sync_conflicts 
		SET resolved = true, resolved_by = $1, resolved_at = $2, updated_at = $3
		WHERE id = $4
	`, resolvedBy, now, now, conflictID)

	return err
}
