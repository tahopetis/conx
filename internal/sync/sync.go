package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/conx/cmdb/internal/config"
	"github.com/conx/cmdb/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

// SyncService handles data synchronization between PostgreSQL and Neo4j
type SyncService struct {
	config       *config.Config
	dbManager    *database.Manager
	redisClient  *database.RedisClient
	eventChan    chan SyncEvent
	errorChan    chan SyncError
	stats        *SyncStats
	logger       *log.Logger
}

// SyncEvent represents a synchronization event
type SyncEvent struct {
	ID          string                 `json:"id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"` // CREATE, UPDATE, DELETE
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"` // PENDING, PROCESSING, COMPLETED, FAILED
	RetryCount  int                    `json:"retry_count"`
	Error       string                 `json:"error,omitempty"`
}

// SyncError represents a synchronization error
type SyncError struct {
	EventID    string    `json:"event_id"`
	Error      error     `json:"error"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"retry_count"`
}

// SyncStats represents synchronization statistics
type SyncStats struct {
	TotalEvents      int64         `json:"total_events"`
	SuccessfulEvents int64         `json:"successful_events"`
	FailedEvents     int64         `json:"failed_events"`
	PendingEvents    int64         `json:"pending_events"`
	LastSyncTime     time.Time     `json:"last_sync_time"`
	AverageSyncTime  time.Duration `json:"average_sync_time"`
	LastError        *SyncError    `json:"last_error,omitempty"`
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	Enabled           bool          `yaml:"enabled"`
	BatchSize         int           `yaml:"batch_size"`
	WorkerCount       int           `yaml:"worker_count"`
	RetryLimit        int           `yaml:"retry_limit"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
	SyncInterval      time.Duration `yaml:"sync_interval"`
	ConflictStrategy  string        `yaml:"conflict_strategy"` // "postgres_wins", "neo4j_wins", "merge"
	EventTTL          time.Duration `yaml:"event_ttl"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
	MaxConcurrentSync int          `yaml:"max_concurrent_sync"`
}

// NewSyncService creates a new synchronization service
func NewSyncService(cfg *config.Config, dbManager *database.Manager, redisClient *database.RedisClient, logger *log.Logger) (*SyncService, error) {
	syncConfig := SyncConfig{
		Enabled:           true,
		BatchSize:         100,
		WorkerCount:       5,
		RetryLimit:        3,
		RetryDelay:        5 * time.Second,
		SyncInterval:      30 * time.Second,
		ConflictStrategy:  "postgres_wins",
		EventTTL:          24 * time.Hour,
		CleanupInterval:   1 * time.Hour,
		MaxConcurrentSync: 10,
	}

	// Override with config if available
	if cfg.Sync != nil {
		if cfg.Sync.Enabled != nil {
			syncConfig.Enabled = *cfg.Sync.Enabled
		}
		if cfg.Sync.BatchSize != nil {
			syncConfig.BatchSize = *cfg.Sync.BatchSize
		}
		if cfg.Sync.WorkerCount != nil {
			syncConfig.WorkerCount = *cfg.Sync.WorkerCount
		}
		if cfg.Sync.RetryLimit != nil {
			syncConfig.RetryLimit = *cfg.Sync.RetryLimit
		}
		if cfg.Sync.RetryDelay != nil {
			syncConfig.RetryDelay = *cfg.Sync.RetryDelay
		}
		if cfg.Sync.SyncInterval != nil {
			syncConfig.SyncInterval = *cfg.Sync.SyncInterval
		}
		if cfg.Sync.ConflictStrategy != nil {
			syncConfig.ConflictStrategy = *cfg.Sync.ConflictStrategy
		}
		if cfg.Sync.EventTTL != nil {
			syncConfig.EventTTL = *cfg.Sync.EventTTL
		}
		if cfg.Sync.CleanupInterval != nil {
			syncConfig.CleanupInterval = *cfg.Sync.CleanupInterval
		}
		if cfg.Sync.MaxConcurrentSync != nil {
			syncConfig.MaxConcurrentSync = *cfg.Sync.MaxConcurrentSync
		}
	}

	service := &SyncService{
		config:      cfg,
		dbManager:   dbManager,
		redisClient: redisClient,
		eventChan:   make(chan SyncEvent, 1000),
		errorChan:   make(chan SyncError, 100),
		stats:       &SyncStats{},
		logger:      logger,
	}

	// Initialize sync tables and procedures
	if err := service.initializeSyncInfrastructure(); err != nil {
		return nil, fmt.Errorf("failed to initialize sync infrastructure: %w", err)
	}

	// Start background workers
	go service.startEventProcessor()
	go service.startErrorProcessor()
	go service.startCleanupWorker()
	go service.startStatsCollector()

	return service, nil
}

// initializeSyncInfrastructure creates necessary database objects for synchronization
func (s *SyncService) initializeSyncInfrastructure() error {
	ctx := context.Background()

	// Create sync_events table in PostgreSQL
	_, err := s.dbManager.Postgres.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sync_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID NOT NULL,
			action VARCHAR(20) NOT NULL,
			data JSONB NOT NULL DEFAULT '{}',
			status VARCHAR(20) DEFAULT 'PENDING',
			retry_count INTEGER DEFAULT 0,
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			processed_at TIMESTAMP WITH TIME ZONE,
			
			CONSTRAINT valid_action CHECK (action IN ('CREATE', 'UPDATE', 'DELETE')),
			CONSTRAINT valid_status CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'))
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sync_events table: %w", err)
	}

	// Create indexes for sync_events
	_, err = s.dbManager.Postgres.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_sync_events_status ON sync_events(status);
		CREATE INDEX IF NOT EXISTS idx_sync_events_entity ON sync_events(entity_type, entity_id);
		CREATE INDEX IF NOT EXISTS idx_sync_events_created_at ON sync_events(created_at);
		CREATE INDEX IF NOT EXISTS idx_sync_events_retry_count ON sync_events(retry_count) WHERE status = 'FAILED';
	`)
	if err != nil {
		return fmt.Errorf("failed to create sync_events indexes: %w", err)
	}

	// Create sync_stats table
	_, err = s.dbManager.Postgres.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sync_stats (
			id SERIAL PRIMARY KEY,
			total_events BIGINT DEFAULT 0,
			successful_events BIGINT DEFAULT 0,
			failed_events BIGINT DEFAULT 0,
			pending_events BIGINT DEFAULT 0,
			last_sync_time TIMESTAMP WITH TIME ZONE,
			average_sync_time INTERVAL,
			last_error TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sync_stats table: %w", err)
	}

	// Create sync_log table for audit trail
	_, err = s.dbManager.Postgres.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sync_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			event_id UUID NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID NOT NULL,
			action VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL,
			duration_ms INTEGER,
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sync_log table: %w", err)
	}

	// Initialize Neo4j sync procedures
	neo4jSession := s.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jSession.Close(ctx)

	// Create procedure for syncing CI nodes
	_, err = neo4jSession.Run(ctx, `
		CREATE OR REPLACE PROCEDURE syncCI(ciId STRING, ciName STRING, ciType STRING, 
			ciAttributes MAP, ciTags LIST<STRING>, action STRING)
		YIELD node
		CASE action
			WHEN 'CREATE' THEN
				MERGE (n:ConfigurationItem {id: ciId})
				SET n.name = ciName,
				    n.type = ciType,
				    n.attributes = ciAttributes,
				    n.tags = ciTags,
				    n.synced_at = datetime()
				RETURN n
			WHEN 'UPDATE' THEN
				MATCH (n:ConfigurationItem {id: ciId})
				SET n.name = ciName,
				    n.type = ciType,
				    n.attributes = ciAttributes,
				    n.tags = ciTags,
				    n.synced_at = datetime()
				RETURN n
			WHEN 'DELETE' THEN
				MATCH (n:ConfigurationItem {id: ciId})
				DELETE n
				RETURN null
		END CASE
	`)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j syncCI procedure: %w", err)
	}

	// Create procedure for syncing relationships
	_, err = neo4jSession.Run(ctx, `
		CREATE OR REPLACE PROCEDURE syncRelationship(relId STRING, sourceId STRING, targetId STRING, 
			relType STRING, relAttributes MAP, action STRING)
		YIELD relationship
		CASE action
			WHEN 'CREATE' THEN
				MATCH (source:ConfigurationItem {id: sourceId})
				MATCH (target:ConfigurationItem {id: targetId})
				MERGE (source)-[r:RELATIONSHIP {id: relId}]->(target)
				SET r.type = relType,
				    r.attributes = relAttributes,
				    r.synced_at = datetime()
				RETURN r
			WHEN 'UPDATE' THEN
				MATCH (source:ConfigurationItem {id: sourceId})-[r:RELATIONSHIP {id: relId}]->(target:ConfigurationItem {id: targetId})
				SET r.type = relType,
				    r.attributes = relAttributes,
				    r.synced_at = datetime()
				RETURN r
			WHEN 'DELETE' THEN
				MATCH (source:ConfigurationItem {id: sourceId})-[r:RELATIONSHIP {id: relId}]->(target:ConfigurationItem {id: targetId})
				DELETE r
				RETURN null
		END CASE
	`)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j syncRelationship procedure: %w", err)
	}

	s.logger.Info("Sync infrastructure initialized successfully")
	return nil
}

// RecordEvent records a synchronization event
func (s *SyncService) RecordEvent(ctx context.Context, entityType, entityID, action string, data map[string]interface{}) error {
	event := SyncEvent{
		ID:         generateEventID(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Data:       data,
		Timestamp:  time.Now(),
		Status:     "PENDING",
		RetryCount: 0,
	}

	// Store in PostgreSQL
	_, err := s.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.ID, event.EntityType, event.EntityID, event.Action, event.Data, event.Status, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to record sync event: %w", err)
	}

	// Store in Redis for real-time processing
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal sync event: %w", err)
	}

	err = s.redisClient.SetWithTTL(ctx, fmt.Sprintf("sync:event:%s", event.ID), string(eventJSON), 24*time.Hour)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to store sync event in Redis")
	}

	// Send to event channel for immediate processing
	select {
	case s.eventChan <- event:
	default:
		s.logger.Warn().Msg("Event channel full, event will be processed by batch processor")
	}

	s.logger.Debug().Str("event_id", event.ID).Str("entity_type", entityType).Str("action", action).Msg("Sync event recorded")
	return nil
}

// ProcessEvent processes a single synchronization event
func (s *SyncService) ProcessEvent(ctx context.Context, event SyncEvent) error {
	startTime := time.Now()
	
	// Update status to processing
	err := s.updateEventStatus(ctx, event.ID, "PROCESSING", "")
	if err != nil {
		return fmt.Errorf("failed to update event status to processing: %w", err)
	}

	var syncErr error

	switch event.EntityType {
	case "configuration_item":
		syncErr = s.syncConfigurationItem(ctx, event)
	case "relationship":
		syncErr = s.syncRelationship(ctx, event)
	default:
		syncErr = fmt.Errorf("unsupported entity type: %s", event.EntityType)
	}

	duration := time.Since(startTime)
	status := "COMPLETED"
	errorMsg := ""

	if syncErr != nil {
		status = "FAILED"
		errorMsg = syncErr.Error()
		
		// Send to error channel for retry processing
		s.errorChan <- SyncError{
			EventID:    event.ID,
			Error:      syncErr,
			Timestamp:  time.Now(),
			RetryCount: event.RetryCount,
		}
	}

	// Update event status
	err = s.updateEventStatus(ctx, event.ID, status, errorMsg)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update final event status")
	}

	// Log the sync attempt
	_, err = s.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_log (event_id, entity_type, entity_id, action, status, duration_ms, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.EntityType, event.EntityID, event.Action, status, duration.Milliseconds(), errorMsg, time.Now())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to log sync attempt")
	}

	if syncErr != nil {
		return syncErr
	}

	s.logger.Debug().Str("event_id", event.ID).Str("status", status).Dur("duration", duration).Msg("Event processed")
	return nil
}

// syncConfigurationItem synchronizes a configuration item between PostgreSQL and Neo4j
func (s *SyncService) syncConfigurationItem(ctx context.Context, event SyncEvent) error {
	neo4jSession := s.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jSession.Close(ctx)

	// Extract CI data from event
	ciName, _ := event.Data["name"].(string)
	ciType, _ := event.Data["type"].(string)
	ciAttributes, _ := event.Data["attributes"].(map[string]interface{})
	ciTags, _ := event.Data["tags"].([]interface{})
	
	// Convert tags to string slice
	var tags []string
	for _, tag := range ciTags {
		if str, ok := tag.(string); ok {
			tags = append(tags, str)
		}
	}

	// Call Neo4j procedure
	_, err := neo4jSession.Run(ctx, `
		CALL syncCI($ciId, $ciName, $ciType, $ciAttributes, $ciTags, $action)
	`, map[string]interface{}{
		"ciId":        event.EntityID,
		"ciName":      ciName,
		"ciType":      ciType,
		"ciAttributes": ciAttributes,
		"ciTags":      tags,
		"action":      event.Action,
	})

	if err != nil {
		return fmt.Errorf("failed to sync CI to Neo4j: %w", err)
	}

	return nil
}

// syncRelationship synchronizes a relationship between PostgreSQL and Neo4j
func (s *SyncService) syncRelationship(ctx context.Context, event SyncEvent) error {
	neo4jSession := s.dbManager.Neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jSession.Close(ctx)

	// Extract relationship data from event
	sourceID, _ := event.Data["source_id"].(string)
	targetID, _ := event.Data["target_id"].(string)
	relType, _ := event.Data["type"].(string)
	relAttributes, _ := event.Data["attributes"].(map[string]interface{})

	// Call Neo4j procedure
	_, err := neo4jSession.Run(ctx, `
		CALL syncRelationship($relId, $sourceId, $targetId, $relType, $relAttributes, $action)
	`, map[string]interface{}{
		"relId":        event.EntityID,
		"sourceId":     sourceID,
		"targetId":     targetID,
		"relType":      relType,
		"relAttributes": relAttributes,
		"action":       event.Action,
	})

	if err != nil {
		return fmt.Errorf("failed to sync relationship to Neo4j: %w", err)
	}

	return nil
}

// updateEventStatus updates the status of a sync event
func (s *SyncService) updateEventStatus(ctx context.Context, eventID, status, errorMsg string) error {
	_, err := s.dbManager.Postgres.Exec(ctx, `
		UPDATE sync_events 
		SET status = $1, error_message = $2, updated_at = NOW(), processed_at = CASE WHEN $1 IN ('COMPLETED', 'FAILED') THEN NOW() ELSE NULL END
		WHERE id = $3
	`, status, errorMsg, eventID)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	// Update Redis cache
	eventJSON, err := s.redisClient.Get(ctx, fmt.Sprintf("sync:event:%s", eventID))
	if err == nil {
		var event SyncEvent
		if json.Unmarshal([]byte(eventJSON), &event) == nil {
			event.Status = status
			event.Error = errorMsg
			updatedJSON, _ := json.Marshal(event)
			s.redisClient.SetWithTTL(ctx, fmt.Sprintf("sync:event:%s", eventID), string(updatedJSON), 24*time.Hour)
		}
	}

	return nil
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("sync_%d", time.Now().UnixNano())
}

// startEventProcessor processes sync events from the channel and database
func (s *SyncService) startEventProcessor() {
	s.logger.Info("Starting sync event processor")
	
	for {
		select {
		case event := <-s.eventChan:
			// Process individual event from channel
			go func(e SyncEvent) {
				ctx := context.Background()
				if err := s.ProcessEvent(ctx, e); err != nil {
					s.logger.Error().Err(err).Str("event_id", e.ID).Msg("Failed to process sync event")
				}
			}(event)
			
		default:
			// Process batch events from database
			go s.processBatchEvents()
			time.Sleep(1 * time.Second)
		}
	}
}

// startErrorProcessor handles retry logic for failed sync events
func (s *SyncService) startErrorProcessor() {
	s.logger.Info("Starting sync error processor")
	
	for err := range s.errorChan {
		go func(syncErr SyncError) {
			if syncErr.RetryCount < 3 { // Max 3 retries
				s.logger.Warn(). 
					Str("event_id", syncErr.EventID).
					Int("retry_count", syncErr.RetryCount).
					Err(syncErr.Error).
					Msg("Retrying failed sync event")
				
				// Wait before retry
				time.Sleep(time.Duration(syncErr.RetryCount+1) * 5 * time.Second)
				
				// Get the event from database
				event, err := s.getEventByID(context.Background(), syncErr.EventID)
				if err != nil {
					s.logger.Error().Err(err).Str("event_id", syncErr.EventID).Msg("Failed to get event for retry")
					return
				}
				
				// Update retry count and process again
				event.RetryCount = syncErr.RetryCount + 1
				if processErr := s.ProcessEvent(context.Background(), *event); processErr != nil {
					s.logger.Error().Err(processErr).Str("event_id", event.ID).Msg("Retry failed")
				}
			} else {
				s.logger.Error().
					Str("event_id", syncErr.EventID).
					Int("retry_count", syncErr.RetryCount).
					Err(syncErr.Error).
					Msg("Sync event failed after maximum retries")
			}
		}(err)
	}
}

// startCleanupWorker periodically cleans up old sync events and logs
func (s *SyncService) startCleanupWorker() {
	s.logger.Info("Starting sync cleanup worker")
	
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		
		// Clean up old sync events
		_, err := s.dbManager.Postgres.Exec(ctx, "SELECT cleanup_old_sync_events(30)")
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to cleanup old sync events")
		}
		
		// Clean up Redis cache
		keys, err := s.redisClient.Keys(ctx, "sync:event:*")
		if err == nil && len(keys) > 0 {
			for _, key := range keys {
				ttl, err := s.redisClient.TTL(ctx, key)
				if err == nil && ttl <= 0 {
					s.redisClient.Delete(ctx, key)
				}
			}
		}
		
		s.logger.Info("Sync cleanup completed")
	}
}

// startStatsCollector periodically collects and updates sync statistics
func (s *SyncService) startStatsCollector() {
	s.logger.Info("Starting sync stats collector")
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		stats, err := s.getSyncStats(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to collect sync stats")
			continue
		}
		
		// Update internal stats
		s.stats = stats
		
		// Store stats in database
		_, err = s.dbManager.Postgres.Exec(ctx, `
			INSERT INTO sync_stats (total_events, successful_events, failed_events, pending_events, 
				last_sync_time, average_sync_time, last_error, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				total_events = EXCLUDED.total_events,
				successful_events = EXCLUDED.successful_events,
				failed_events = EXCLUDED.failed_events,
				pending_events = EXCLUDED.pending_events,
				last_sync_time = EXCLUDED.last_sync_time,
				average_sync_time = EXCLUDED.average_sync_time,
				last_error = EXCLUDED.last_error,
				updated_at = EXCLUDED.updated_at
		`, stats.TotalEvents, stats.SuccessfulEvents, stats.FailedEvents, stats.PendingEvents,
			stats.LastSyncTime, stats.AverageSyncTime, func() *string {
				if stats.LastError != nil {
					errMsg := stats.LastError.Error.Error()
					return &errMsg
				}
				return nil
			}(), time.Now())
		
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to update sync stats in database")
		}
		
		s.logger.Debug().
			Int64("total_events", stats.TotalEvents).
			Int64("successful_events", stats.SuccessfulEvents).
			Int64("failed_events", stats.FailedEvents).
			Int64("pending_events", stats.PendingEvents).
			Msg("Sync stats updated")
	}
}

// processBatchEvents processes pending sync events from database in batches
func (s *SyncService) processBatchEvents() {
	ctx := context.Background()
	
	// Get pending events
	rows, err := s.dbManager.Postgres.Query(ctx, `
		SELECT id, entity_type, entity_id, action, data, status, retry_count, created_at
		FROM sync_events 
		WHERE status = 'PENDING' 
		ORDER BY created_at ASC 
		LIMIT 100
	`)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch pending sync events")
		return
	}
	defer rows.Close()
	
	var events []SyncEvent
	for rows.Next() {
		var event SyncEvent
		var dataJSON []byte
		
		err := rows.Scan(&event.ID, &event.EntityType, &event.EntityID, &event.Action, 
			&dataJSON, &event.Status, &event.RetryCount, &event.Timestamp)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan sync event")
			continue
		}
		
		// Unmarshal JSON data
		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			s.logger.Error().Err(err).Str("event_id", event.ID).Msg("Failed to unmarshal event data")
			continue
		}
		
		events = append(events, event)
	}
	
	if len(events) == 0 {
		return
	}
	
	s.logger.Info().Int("event_count", len(events)).Msg("Processing batch sync events")
	
	// Process events concurrently with limited concurrency
	sem := make(chan struct{}, 10) // Limit to 10 concurrent processes
	var wg sync.WaitGroup
	
	for _, event := range events {
		wg.Add(1)
		go func(e SyncEvent) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			if err := s.ProcessEvent(ctx, e); err != nil {
				s.logger.Error().Err(err).Str("event_id", e.ID).Msg("Failed to process batch sync event")
			}
		}(event)
	}
	
	wg.Wait()
	s.logger.Info().Int("event_count", len(events)).Msg("Batch sync events processing completed")
}

// getEventByID retrieves a sync event by ID from database
func (s *SyncService) getEventByID(ctx context.Context, eventID string) (*SyncEvent, error) {
	var event SyncEvent
	var dataJSON []byte
	
	err := s.dbManager.Postgres.QueryRow(ctx, `
		SELECT id, entity_type, entity_id, action, data, status, retry_count, created_at
		FROM sync_events 
		WHERE id = $1
	`, eventID).Scan(&event.ID, &event.EntityType, &event.EntityID, &event.Action, 
		&dataJSON, &event.Status, &event.RetryCount, &event.Timestamp)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get event by ID: %w", err)
	}
	
	// Unmarshal JSON data
	if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}
	
	return &event, nil
}

// getSyncStats retrieves current synchronization statistics
func (s *SyncService) getSyncStats(ctx context.Context) (*SyncStats, error) {
	stats := &SyncStats{}
	
	// Get basic counts
	err := s.dbManager.Postgres.QueryRow(ctx, `
		SELECT 
			COUNT(*) as total_events,
			COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END) as successful_events,
			COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed_events,
			COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending_events,
			MAX(created_at) as last_sync_time,
			AVG(CASE WHEN processed_at IS NOT NULL AND created_at IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (processed_at - created_at)) * 1000 ELSE NULL END) as avg_sync_time_ms
		FROM sync_events
	`).Scan(&stats.TotalEvents, &stats.SuccessfulEvents, &stats.FailedEvents, 
		&stats.PendingEvents, &stats.LastSyncTime, &stats.AverageSyncTime)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get sync stats: %w", err)
	}
	
	// Convert average sync time from milliseconds to duration
	if stats.AverageSyncTime > 0 {
		stats.AverageSyncTime = time.Duration(stats.AverageSyncTime) * time.Millisecond
	}
	
	// Get last error
	var lastErrorStr *string
	err = s.dbManager.Postgres.QueryRow(ctx, `
		SELECT error_message 
		FROM sync_events 
		WHERE status = 'FAILED' AND error_message IS NOT NULL 
		ORDER BY created_at DESC 
		LIMIT 1
	`).Scan(&lastErrorStr)
	
	if err == nil && lastErrorStr != nil {
		stats.LastError = &SyncError{
			Error:     fmt.Errorf(*lastErrorStr),
			Timestamp: time.Now(),
		}
	}
	
	return stats, nil
}

// GetStats returns current synchronization statistics
func (s *SyncService) GetStats(ctx context.Context) (*SyncStats, error) {
	return s.getSyncStats(ctx)
}

// GetPendingEventsCount returns the number of pending sync events
func (s *SyncService) GetPendingEventsCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.dbManager.Postgres.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM sync_events 
		WHERE status = 'PENDING'
	`).Scan(&count)
	
	if err != nil {
		return 0, fmt.Errorf("failed to get pending events count: %w", err)
	}
	
	return count, nil
}

// GetRecentErrors returns recent sync errors
func (s *SyncService) GetRecentErrors(ctx context.Context, limit int) ([]SyncError, error) {
	rows, err := s.dbManager.Postgres.Query(ctx, `
		SELECT id, error_message, created_at, retry_count
		FROM sync_events 
		WHERE status = 'FAILED' AND error_message IS NOT NULL 
		ORDER BY created_at DESC 
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent errors: %w", err)
	}
	defer rows.Close()
	
	var errors []SyncError
	for rows.Next() {
		var error SyncError
		var errorMsg string
		
		err := rows.Scan(&error.EventID, &errorMsg, &error.Timestamp, &error.RetryCount)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan sync error")
			continue
		}
		
		error.Error = fmt.Errorf(errorMsg)
		errors = append(errors, error)
	}
	
	return errors, nil
}

// ForceSync forces synchronization of a specific entity
func (s *SyncService) ForceSync(ctx context.Context, entityType, entityID string) error {
	// Get the latest data from PostgreSQL
	var data map[string]interface{}
	var query string
	var args []interface{}
	
	switch entityType {
	case "configuration_item":
		query = `
			SELECT jsonb_build_object(
				'id', id, 'name', name, 'type', type, 'description', description,
				'status', status, 'attributes', attributes, 'tags', tags,
				'created_by', created_by, 'created_at', created_at, 'updated_at', updated_at
			) as data
			FROM configuration_items WHERE id = $1
		`
		args = []interface{}{entityID}
		
	case "relationship":
		query = `
			SELECT jsonb_build_object(
				'id', id, 'source_id', source_id, 'target_id', target_id,
				'type', type, 'description', description, 'attributes', attributes,
				'strength', strength, 'created_by', created_by, 'created_at', created_at
			) as data
			FROM relationships WHERE id = $1
		`
		args = []interface{}{entityID}
		
	default:
		return fmt.Errorf("unsupported entity type: %s", entityType)
	}
	
	err := s.dbManager.Postgres.QueryRow(ctx, query, args...).Scan(&data)
	if err != nil {
		return fmt.Errorf("failed to get entity data: %w", err)
	}
	
	// Create sync event
	return s.RecordEvent(ctx, entityType, entityID, "UPDATE", data)
}

// Close gracefully shuts down the sync service
func (s *SyncService) Close() error {
	s.logger.Info("Shutting down sync service")
	
	// Close channels
	close(s.eventChan)
	close(s.errorChan)
	
	// Wait for pending operations to complete
	time.Sleep(2 * time.Second)
	
	s.logger.Info("Sync service shutdown completed")
	return nil
}
