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

// FallbackService handles fallback synchronization procedures when the main sync fails
type FallbackService struct {
	dbManager   *database.Manager
	syncService *SyncService
	resolver    *ConflictResolver
	monitor     *Monitor
	logger      *log.Logger
}

// FallbackStrategy represents different fallback strategies
type FallbackStrategy string

const (
	StrategyRetry           FallbackStrategy = "retry"
	StrategyManual          FallbackStrategy = "manual"
	StrategySkip           FallbackStrategy = "skip"
	StrategyQueue          FallbackStrategy = "queue"
	StrategyFullResync     FallbackStrategy = "full_resync"
	StrategySelectiveResync FallbackStrategy = "selective_resync"
)

// FallbackConfig represents fallback configuration
type FallbackConfig struct {
	Enabled              bool          `yaml:"enabled"`
	MaxRetries           int           `yaml:"max_retries"`
	RetryDelay           time.Duration `yaml:"retry_delay"`
	Strategy             FallbackStrategy `yaml:"strategy"`
	QueueThreshold       int           `yaml:"queue_threshold"`
	ManualIntervention   bool          `yaml:"manual_intervention"`
	FullResyncInterval   time.Duration `yaml:"full_resync_interval"`
	SelectiveResyncLimit int           `yaml:"selective_resync_limit"`
}

// FallbackOperation represents a fallback operation
type FallbackOperation struct {
	ID             string                 `json:"id"`
	OriginalEventID string                 `json:"original_event_id"`
	Strategy       FallbackStrategy       `json:"strategy"`
	EntityType     string                 `json:"entity_type"`
	EntityID       string                 `json:"entity_id"`
	Action         string                 `json:"action"`
	Data           map[string]interface{} `json:"data"`
	RetryCount     int                    `json:"retry_count"`
	Status         string                 `json:"status"` // pending, processing, completed, failed
	Error          string                 `json:"error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

// FallbackReport represents a report of fallback operations
type FallbackReport struct {
	TotalOperations      int64               `json:"total_operations"`
	PendingOperations    int64               `json:"pending_operations"`
	CompletedOperations  int64               `json:"completed_operations"`
	FailedOperations     int64               `json:"failed_operations"`
	AverageRetryCount   float64             `json:"average_retry_count"`
	SuccessRate          float64             `json:"success_rate_percent"`
	LastOperationTime   time.Time           `json:"last_operation_time"`
	OperationBreakdown  map[string]int64    `json:"operation_breakdown"`
	ErrorSummary        map[string]int64    `json:"error_summary"`
}

// NewFallbackService creates a new fallback service
func NewFallbackService(dbManager *database.Manager, syncService *SyncService, resolver *ConflictResolver, monitor *Monitor, logger *log.Logger) *FallbackService {
	return &FallbackService{
		dbManager:   dbManager,
		syncService: syncService,
		resolver:    resolver,
		monitor:     monitor,
		logger:      logger,
	}
}

// HandleSyncFailure handles synchronization failures with fallback strategies
func (fs *FallbackService) HandleSyncFailure(ctx context.Context, event SyncEvent, err error) error {
	fs.logger.Error().
		Str("event_id", event.ID).
		Str("entity_type", event.EntityType).
		Str("action", event.Action).
		Err(err).
		Msg("Sync failure detected, applying fallback strategy")

	// Get fallback configuration
	config := fs.getFallbackConfig()

	if !config.Enabled {
		fs.logger.Warn().Msg("Fallback sync is disabled, skipping fallback handling")
		return err
	}

	// Determine fallback strategy
	strategy := config.Strategy
	if event.RetryCount >= config.MaxRetries {
		// If we've exceeded max retries, use a different strategy
		switch strategy {
		case StrategyRetry:
			strategy = StrategyQueue
		case StrategyQueue:
			strategy = StrategyManual
		}
	}

	// Apply fallback strategy
	switch strategy {
	case StrategyRetry:
		return fs.handleRetryStrategy(ctx, event, err, config)
	case StrategyManual:
		return fs.handleManualStrategy(ctx, event, err, config)
	case StrategySkip:
		return fs.handleSkipStrategy(ctx, event, err, config)
	case StrategyQueue:
		return fs.handleQueueStrategy(ctx, event, err, config)
	case StrategyFullResync:
		return fs.handleFullResyncStrategy(ctx, event, err, config)
	case StrategySelectiveResync:
		return fs.handleSelectiveResyncStrategy(ctx, event, err, config)
	default:
		fs.logger.Error().Str("strategy", string(strategy)).Msg("Unknown fallback strategy")
		return err
	}
}

// handleRetryStrategy handles retry fallback strategy
func (fs *FallbackService) handleRetryStrategy(ctx context.Context, event SyncEvent, err error, config FallbackConfig) error {
	fs.logger.Info().
		Str("event_id", event.ID).
		Int("retry_count", event.RetryCount).
		Msg("Applying retry fallback strategy")

	// Wait before retry
	time.Sleep(config.RetryDelay)

	// Retry the sync operation
	retryErr := fs.syncService.ProcessEvent(ctx, event)
	if retryErr != nil {
		fs.logger.Error().
			Str("event_id", event.ID).
			Err(retryErr).
			Msg("Retry failed")
		return retryErr
	}

	fs.logger.Info().
		Str("event_id", event.ID).
		Msg("Retry succeeded")
	return nil
}

// handleManualStrategy handles manual intervention fallback strategy
func (fs *FallbackService) handleManualStrategy(ctx context.Context, event SyncEvent, err error, config FallbackConfig) error {
	fs.logger.Warn().
		Str("event_id", event.ID).
		Msg("Applying manual intervention fallback strategy")

	// Create fallback operation record
	operation := &FallbackOperation{
		ID:             fmt.Sprintf("fallback_%d", time.Now().UnixNano()),
		OriginalEventID: event.ID,
		Strategy:       StrategyManual,
		EntityType:     event.EntityType,
		EntityID:       event.EntityID,
		Action:         event.Action,
		Data:           event.Data,
		RetryCount:     event.RetryCount,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	if err := fs.createFallbackOperation(ctx, operation); err != nil {
		fs.logger.Error().Err(err).Msg("Failed to create fallback operation record")
		return err
	}

	// Create alert for manual intervention
	alertData := map[string]interface{}{
		"event_id":      event.ID,
		"entity_type":  event.EntityType,
		"entity_id":    event.EntityID,
		"operation_id": operation.ID,
		"error":        err.Error(),
	}

	alertErr := fs.monitor.CreateAlert(ctx, "error", "manual_intervention_required",
		fmt.Sprintf("Manual intervention required for sync event %s", event.ID), alertData)
	if alertErr != nil {
		fs.logger.Error().Err(alertErr).Msg("Failed to create manual intervention alert")
	}

	return fmt.Errorf("manual intervention required - fallback operation created: %s", operation.ID)
}

// handleSkipStrategy handles skip fallback strategy
func (fs *FallbackService) handleSkipStrategy(ctx context.Context, event SyncEvent, err error, config FallbackConfig) error {
	fs.logger.Warn().
		Str("event_id", event.ID).
		Msg("Applying skip fallback strategy")

	// Log the skip event
	_, logErr := fs.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_fallback_log (event_id, entity_type, entity_id, action, strategy, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.EntityType, event.EntityID, event.Action, "skip", "skipped", err.Error(), time.Now())

	if logErr != nil {
		fs.logger.Error().Err(logErr).Msg("Failed to log skip event")
	}

	// Create low-priority alert
	alertData := map[string]interface{}{
		"event_id":     event.ID,
		"entity_type": event.EntityType,
		"entity_id":   event.EntityID,
		"error":       err.Error(),
	}

	fs.monitor.CreateAlert(ctx, "info", "sync_skipped",
		fmt.Sprintf("Sync event %s skipped due to failure", event.ID), alertData)

	return nil
}

// handleQueueStrategy handles queue fallback strategy
func (fs *FallbackService) handleQueueStrategy(ctx context.Context, event SyncEvent, err error, config FallbackConfig) error {
	fs.logger.Info().
		Str("event_id", event.ID).
		Msg("Applying queue fallback strategy")

	// Create fallback operation record
	operation := &FallbackOperation{
		ID:             fmt.Sprintf("fallback_%d", time.Now().UnixNano()),
		OriginalEventID: event.ID,
		Strategy:       StrategyQueue,
		EntityType:     event.EntityType,
		EntityID:       event.EntityID,
		Action:         event.Action,
		Data:           event.Data,
		RetryCount:     event.RetryCount,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	if err := fs.createFallbackOperation(ctx, operation); err != nil {
		fs.logger.Error().Err(err).Msg("Failed to create fallback operation record")
		return err
	}

	// Check if queue threshold is exceeded
	pendingCount, err := fs.getPendingFallbackOperationsCount(ctx)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to get pending fallback operations count")
		return err
	}

	if pendingCount > int64(config.QueueThreshold) {
		fs.logger.Warn().
			Int64("pending_count", pendingCount).
			Int("threshold", config.QueueThreshold).
			Msg("Fallback queue threshold exceeded")

		// Create alert for queue threshold exceeded
		alertData := map[string]interface{}{
			"pending_count": pendingCount,
			"threshold":     config.QueueThreshold,
		}

		fs.monitor.CreateAlert(ctx, "warning", "fallback_queue_threshold_exceeded",
			fmt.Sprintf("Fallback queue threshold exceeded: %d operations pending", pendingCount), alertData)
	}

	return fmt.Errorf("event queued for fallback processing: %s", operation.ID)
}

// handleFullResyncStrategy handles full resync fallback strategy
func (fs *FallbackService) handleFullResyncStrategy(ctx context.Context, event SyncEvent, err error, config FallbackConfig) error {
	fs.logger.Info().
		Str("event_id", event.ID).
		Msg("Applying full resync fallback strategy")

	// Check if full resync is already in progress
	inProgress, err := fs.isFullResyncInProgress(ctx)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to check full resync status")
		return err
	}

	if inProgress {
		fs.logger.Info().Msg("Full resync already in progress, skipping")
		return fmt.Errorf("full resync already in progress")
	}

	// Create fallback operation record
	operation := &FallbackOperation{
		ID:             fmt.Sprintf("fallback_%d", time.Now().UnixNano()),
		OriginalEventID: event.ID,
		Strategy:       StrategyFullResync,
		EntityType:     event.EntityType,
		EntityID:       event.EntityID,
		Action:         event.Action,
		Data:           event.Data,
		RetryCount:     event.RetryCount,
		Status:         "processing",
		CreatedAt:      time.Now(),
		StartedAt:      func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := fs.createFallbackOperation(ctx, operation); err != nil {
		fs.logger.Error().Err(err).Msg("Failed to create fallback operation record")
		return err
	}

	// Start full resync in background
	go fs.performFullResync(ctx, operation)

	return fmt.Errorf("full resync initiated: %s", operation.ID)
}

// handleSelectiveResyncStrategy handles selective resync fallback strategy
func (fs *FallbackService) handleSelectiveResyncStrategy(ctx context.Context, event SyncEvent, err error, config FallbackConfig) error {
	fs.logger.Info().
		Str("event_id", event.ID).
		Msg("Applying selective resync fallback strategy")

	// Create fallback operation record
	operation := &FallbackOperation{
		ID:             fmt.Sprintf("fallback_%d", time.Now().UnixNano()),
		OriginalEventID: event.ID,
		Strategy:       StrategySelectiveResync,
		EntityType:     event.EntityType,
		EntityID:       event.EntityID,
		Action:         event.Action,
		Data:           event.Data,
		RetryCount:     event.RetryCount,
		Status:         "processing",
		CreatedAt:      time.Now(),
		StartedAt:      func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := fs.createFallbackOperation(ctx, operation); err != nil {
		fs.logger.Error().Err(err).Msg("Failed to create fallback operation record")
		return err
	}

	// Start selective resync in background
	go fs.performSelectiveResync(ctx, operation, config)

	return fmt.Errorf("selective resync initiated: %s", operation.ID)
}

// createFallbackOperation creates a fallback operation record
func (fs *FallbackService) createFallbackOperation(ctx context.Context, operation *FallbackOperation) error {
	dataJSON, _ := json.Marshal(operation.Data)

	_, err := fs.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_fallback_operations (
			id, original_event_id, strategy, entity_type, entity_id, action, 
			data, retry_count, status, created_at, started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, operation.ID, operation.OriginalEventID, operation.Strategy, operation.EntityType,
		operation.EntityID, operation.Action, string(dataJSON), operation.RetryCount,
		operation.Status, operation.CreatedAt, operation.StartedAt, operation.CompletedAt)

	return err
}

// performFullResync performs a full resync of all data
func (fs *FallbackService) performFullResync(ctx context.Context, operation *FallbackOperation) {
	fs.logger.Info().Str("operation_id", operation.ID).Msg("Starting full resync")

	startTime := time.Now()
	var successCount, failureCount int64

	// Mark full resync as in progress
	fs.setFullResyncStatus(ctx, true)

	defer func() {
		// Mark full resync as completed
		fs.setFullResyncStatus(ctx, false)
		
		// Update operation status
		duration := time.Since(startTime)
		status := "completed"
		if failureCount > 0 {
			status = "completed_with_errors"
		}

		fs.updateFallbackOperationStatus(ctx, operation.ID, status, 
			fmt.Sprintf("Full resync completed: %d succeeded, %d failed, duration: %v", 
				successCount, failureCount, duration))

		fs.logger.Info().
			Str("operation_id", operation.ID).
			Int64("success_count", successCount).
			Int64("failure_count", failureCount).
			Dur("duration", duration).
			Msg("Full resync completed")
	}()

	// Resync configuration items
	ciSuccess, ciFailure := fs.resyncConfigurationItems(ctx)
	successCount += ciSuccess
	failureCount += ciFailure

	// Resync relationships
	relSuccess, relFailure := fs.resyncRelationships(ctx)
	successCount += relSuccess
	failureCount += relFailure

	// Resync users
	userSuccess, userFailure := fs.resyncUsers(ctx)
	successCount += userSuccess
	failureCount += userFailure

	// Resync roles
	roleSuccess, roleFailure := fs.resyncRoles(ctx)
	successCount += roleSuccess
	failureCount += roleFailure
}

// performSelectiveResync performs a selective resync of recent data
func (fs *FallbackService) performSelectiveResync(ctx context.Context, operation *FallbackOperation, config FallbackConfig) {
	fs.logger.Info().Str("operation_id", operation.ID).Msg("Starting selective resync")

	startTime := time.Now()
	var successCount, failureCount int64

	defer func() {
		// Update operation status
		duration := time.Since(startTime)
		status := "completed"
		if failureCount > 0 {
			status = "completed_with_errors"
		}

		fs.updateFallbackOperationStatus(ctx, operation.ID, status, 
			fmt.Sprintf("Selective resync completed: %d succeeded, %d failed, duration: %v", 
				successCount, failureCount, duration))

		fs.logger.Info().
			Str("operation_id", operation.ID).
			Int64("success_count", successCount).
			Int64("failure_count", failureCount).
			Dur("duration", duration).
			Msg("Selective resync completed")
	}()

	// Get recent failed events
	recentEvents, err := fs.getRecentFailedEvents(ctx, config.SelectiveResyncLimit)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to get recent failed events")
		return
	}

	// Process each failed event
	for _, event := range recentEvents {
		if err := fs.syncService.ProcessEvent(ctx, event); err != nil {
			failureCount++
			fs.logger.Error().
				Str("event_id", event.ID).
				Err(err).
				Msg("Failed to process event during selective resync")
		} else {
			successCount++
		}
	}
}

// resyncConfigurationItems resyncs all configuration items
func (fs *FallbackService) resyncConfigurationItems(ctx context.Context) (successCount, failureCount int64) {
	fs.logger.Info().Msg("Resyncing configuration items")

	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT id, name, type, description, status, attributes, tags, created_by, created_at, updated_at
		FROM configuration_items
	`)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to query configuration items")
		return 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var ci struct {
			ID          string
			Name        string
			Type        string
			Description string
			Status      string
			Attributes  map[string]interface{}
			Tags        []string
			CreatedBy   string
			CreatedAt   time.Time
			UpdatedAt   time.Time
		}

		err := rows.Scan(&ci.ID, &ci.Name, &ci.Type, &ci.Description, &ci.Status,
			&ci.Attributes, &ci.Tags, &ci.CreatedBy, &ci.CreatedAt, &ci.UpdatedAt)
		if err != nil {
			failureCount++
			continue
		}

		// Create sync event
		event := SyncEvent{
			ID:         generateEventID(),
			EntityType: "configuration_item",
			EntityID:   ci.ID,
			Action:     "UPDATE",
			Data: map[string]interface{}{
				"id":          ci.ID,
				"name":        ci.Name,
				"type":        ci.Type,
				"description": ci.Description,
				"status":      ci.Status,
				"attributes":  ci.Attributes,
				"tags":        ci.Tags,
				"created_by":  ci.CreatedBy,
				"created_at":  ci.CreatedAt,
				"updated_at":  ci.UpdatedAt,
			},
			Timestamp: time.Now(),
		}

		if err := fs.syncService.ProcessEvent(ctx, event); err != nil {
			failureCount++
			fs.logger.Error().Str("ci_id", ci.ID).Err(err).Msg("Failed to resync configuration item")
		} else {
			successCount++
		}
	}

	return successCount, failureCount
}

// resyncRelationships resyncs all relationships
func (fs *FallbackService) resyncRelationships(ctx context.Context) (successCount, failureCount int64) {
	fs.logger.Info().Msg("Resyncing relationships")

	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT id, source_id, target_id, type, description, attributes, strength, created_by, created_at
		FROM relationships
	`)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to query relationships")
		return 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var rel struct {
			ID          string
			SourceID    string
			TargetID    string
			Type        string
			Description string
			Attributes  map[string]interface{}
			Strength    int
			CreatedBy   string
			CreatedAt   time.Time
		}

		err := rows.Scan(&rel.ID, &rel.SourceID, &rel.TargetID, &rel.Type, &rel.Description,
			&rel.Attributes, &rel.Strength, &rel.CreatedBy, &rel.CreatedAt)
		if err != nil {
			failureCount++
			continue
		}

		// Create sync event
		event := SyncEvent{
			ID:         generateEventID(),
			EntityType: "relationship",
			EntityID:   rel.ID,
			Action:     "UPDATE",
			Data: map[string]interface{}{
				"id":           rel.ID,
				"source_id":    rel.SourceID,
				"target_id":    rel.TargetID,
				"type":         rel.Type,
				"description":  rel.Description,
				"attributes":   rel.Attributes,
				"strength":     rel.Strength,
				"created_by":   rel.CreatedBy,
				"created_at":   rel.CreatedAt,
			},
			Timestamp: time.Now(),
		}

		if err := fs.syncService.ProcessEvent(ctx, event); err != nil {
			failureCount++
			fs.logger.Error().Str("rel_id", rel.ID).Err(err).Msg("Failed to resync relationship")
		} else {
			successCount++
		}
	}

	return successCount, failureCount
}

// resyncUsers resyncs all users
func (fs *FallbackService) resyncUsers(ctx context.Context) (successCount, failureCount int64) {
	fs.logger.Info().Msg("Resyncing users")

	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT id, username, email, is_active, created_at, updated_at
		FROM users
	`)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to query users")
		return 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var user struct {
			ID        string
			Username  string
			Email     string
			IsActive  bool
			CreatedAt time.Time
			UpdatedAt time.Time
		}

		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			failureCount++
			continue
		}

		// Create sync event
		event := SyncEvent{
			ID:         generateEventID(),
			EntityType: "user",
			EntityID:   user.ID,
			Action:     "UPDATE",
			Data: map[string]interface{}{
				"id":         user.ID,
				"username":   user.Username,
				"email":      user.Email,
				"is_active":  user.IsActive,
				"created_at": user.CreatedAt,
				"updated_at": user.UpdatedAt,
			},
			Timestamp: time.Now(),
		}

		if err := fs.syncService.ProcessEvent(ctx, event); err != nil {
			failureCount++
			fs.logger.Error().Str("user_id", user.ID).Err(err).Msg("Failed to resync user")
		} else {
			successCount++
		}
	}

	return successCount, failureCount
}

// resyncRoles resyncs all roles
func (fs *FallbackService) resyncRoles(ctx context.Context) (successCount, failureCount int64) {
	fs.logger.Info().Msg("Resyncing roles")

	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT id, name, description, created_at
		FROM roles
	`)
	if err != nil {
		fs.logger.Error().Err(err).Msg("Failed to query roles")
		return 0, 1
	}
	defer rows.Close()

	for rows.Next() {
		var role struct {
			ID          string
			Name        string
			Description string
			CreatedAt   time.Time
		}

		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt)
		if err != nil {
			failureCount++
			continue
		}

		// Create sync event
		event := SyncEvent{
			ID:         generateEventID(),
			EntityType: "role",
			EntityID:   role.ID,
			Action:     "UPDATE",
			Data: map[string]interface{}{
				"id":          role.ID,
				"name":        role.Name,
				"description": role.Description,
				"created_at":  role.CreatedAt,
			},
			Timestamp: time.Now(),
		}

		if err := fs.syncService.ProcessEvent(ctx, event); err != nil {
			failureCount++
			fs.logger.Error().Str("role_id", role.ID).Err(err).Msg("Failed to resync role")
		} else {
			successCount++
		}
	}

	return successCount, failureCount
}

// getFallbackConfig returns the fallback configuration
func (fs *FallbackService) getFallbackConfig() FallbackConfig {
	return FallbackConfig{
		Enabled:              true,
		MaxRetries:           3,
		RetryDelay:           5 * time.Second,
		Strategy:             StrategyQueue,
		QueueThreshold:       100,
		ManualIntervention:   true,
		FullResyncInterval:   24 * time.Hour,
		SelectiveResyncLimit:  50,
	}
}

// getPendingFallbackOperationsCount returns the count of pending fallback operations
func (fs *FallbackService) getPendingFallbackOperationsCount(ctx context.Context) (int64, error) {
	var count int64
	err := fs.dbManager.Postgres.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM sync_fallback_operations 
		WHERE status = 'pending'
	`).Scan(&count)

	return count, err
}

// isFullResyncInProgress checks if a full resync is currently in progress
func (fs *FallbackService) isFullResyncInProgress(ctx context.Context) (bool, error) {
	var inProgress bool
	err := fs.dbManager.Postgres.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sync_fallback_operations 
			WHERE strategy = 'full_resync' AND status = 'processing'
		)
	`).Scan(&inProgress)

	return inProgress, err
}

// setFullResyncStatus sets the full resync status
func (fs *FallbackService) setFullResyncStatus(ctx context.Context, inProgress bool) error {
	_, err := fs.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_full_resync_status (in_progress, last_updated)
		VALUES ($1, NOW())
		ON CONFLICT (id) DO UPDATE SET
			in_progress = EXCLUDED.in_progress,
			last_updated = EXCLUDED.last_updated
	`, inProgress)

	return err
}

// updateFallbackOperationStatus updates the status of a fallback operation
func (fs *FallbackService) updateFallbackOperationStatus(ctx context.Context, operationID, status, message string) error {
	now := time.Now()
	_, err := fs.dbManager.Postgres.Exec(ctx, `
		UPDATE sync_fallback_operations 
		SET status = $1, error_message = $2, completed_at = $3, updated_at = $4
		WHERE id = $5
	`, status, message, &now, now, operationID)

	return err
}

// getRecentFailedEvents returns recent failed sync events
func (fs *FallbackService) getRecentFailedEvents(ctx context.Context, limit int) ([]SyncEvent, error) {
	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT id, entity_type, entity_id, action, data, status, retry_count, created_at
		FROM sync_events 
		WHERE status = 'FAILED' 
		ORDER BY created_at DESC 
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SyncEvent
	for rows.Next() {
		var event SyncEvent
		var dataJSON []byte

		err := rows.Scan(&event.ID, &event.EntityType, &event.EntityID, &event.Action,
			&dataJSON, &event.Status, &event.RetryCount, &event.Timestamp)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON data
		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// GetFallbackReport returns a report of fallback operations
func (fs *FallbackService) GetFallbackReport(ctx context.Context) (*FallbackReport, error) {
	report := &FallbackReport{
		OperationBreakdown: make(map[string]int64),
		ErrorSummary:       make(map[string]int64),
	}

	// Get basic counts
	err := fs.dbManager.Postgres.QueryRow(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			AVG(retry_count) as avg_retries,
			COALESCE(MAX(completed_at), NOW()) as last_operation
		FROM sync_fallback_operations
	`).Scan(&report.TotalOperations, &report.PendingOperations, &report.CompletedOperations,
		&report.FailedOperations, &report.AverageRetryCount, &report.LastOperationTime)

	if err != nil {
		return nil, fmt.Errorf("failed to get fallback report: %w", err)
	}

	// Calculate success rate
	if report.TotalOperations > 0 {
		report.SuccessRate = float64(report.CompletedOperations) / float64(report.TotalOperations) * 100
	}

	// Get operation breakdown
	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT strategy, COUNT(*) 
		FROM sync_fallback_operations 
		GROUP BY strategy
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var strategy string
			var count int64
			if err := rows.Scan(&strategy, &count); err == nil {
				report.OperationBreakdown[strategy] = count
			}
		}
	}

	// Get error summary
	rows, err = fs.dbManager.Postgres.Query(ctx, `
		SELECT error_message, COUNT(*) 
		FROM sync_fallback_operations 
		WHERE status = 'failed' AND error_message IS NOT NULL 
		GROUP BY error_message 
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var errorMsg string
			var count int64
			if err := rows.Scan(&errorMsg, &count); err == nil {
				report.ErrorSummary[errorMsg] = count
			}
		}
	}

	return report, nil
}

// ProcessQueuedOperations processes queued fallback operations
func (fs *FallbackService) ProcessQueuedOperations(ctx context.Context, batchSize int) error {
	fs.logger.Info().Int("batch_size", batchSize).Msg("Processing queued fallback operations")

	// Get pending operations
	rows, err := fs.dbManager.Postgres.Query(ctx, `
		SELECT id, original_event_id, strategy, entity_type, entity_id, action, data, retry_count
		FROM sync_fallback_operations 
		WHERE status = 'pending' 
		ORDER BY created_at ASC 
		LIMIT $1
	`, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get queued operations: %w", err)
	}
	defer rows.Close()

	var operations []FallbackOperation
	for rows.Next() {
		var op FallbackOperation
		var dataJSON []byte

		err := rows.Scan(&op.ID, &op.OriginalEventID, &op.Strategy, &op.EntityType,
			&op.EntityID, &op.Action, &dataJSON, &op.RetryCount)
		if err != nil {
			fs.logger.Error().Err(err).Msg("Failed to scan fallback operation")
			continue
		}

		// Unmarshal JSON data
		if err := json.Unmarshal(dataJSON, &op.Data); err != nil {
			fs.logger.Error().Err(err).Str("operation_id", op.ID).Msg("Failed to unmarshal operation data")
			continue
		}

		operations = append(operations, op)
	}

	if len(operations) == 0 {
		fs.logger.Info().Msg("No queued operations to process")
		return nil
	}

	fs.logger.Info().Int("operation_count", len(operations)).Msg("Processing queued fallback operations")

	// Process operations
	for _, op := range operations {
		// Create sync event from operation
		event := SyncEvent{
			ID:         op.OriginalEventID,
			EntityType: op.EntityType,
			EntityID:   op.EntityID,
			Action:     op.Action,
			Data:       op.Data,
			Timestamp:  time.Now(),
			RetryCount: op.RetryCount,
		}

		// Update operation status to processing
		now := time.Now()
		fs.updateFallbackOperationStatus(ctx, op.ID, "processing", "")
		startedAt := now

		// Process the event
		if err := fs.syncService.ProcessEvent(ctx, event); err != nil {
			// Update operation status to failed
			fs.updateFallbackOperationStatus(ctx, op.ID, "failed", err.Error())
			fs.logger.Error().
				Str("operation_id", op.ID).
				Err(err).
				Msg("Failed to process queued operation")
		} else {
			// Update operation status to completed
			fs.updateFallbackOperationStatus(ctx, op.ID, "completed", "")
			fs.logger.Info().
				Str("operation_id", op.ID).
				Dur("duration", time.Since(startedAt)).
				Msg("Queued operation processed successfully")
		}
	}

	return nil
}

// StartFallbackProcessor starts the background fallback processor
func (fs *FallbackService) StartFallbackProcessor(ctx context.Context) {
	fs.logger.Info("Starting fallback processor")

	ticker := time.NewTicker(5 * time.Minute) // Process every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fs.logger.Info("Stopping fallback processor")
			return
		case <-ticker.C:
			go func() {
				if err := fs.ProcessQueuedOperations(ctx, 10); err != nil {
					fs.logger.Error().Err(err).Msg("Failed to process queued fallback operations")
				}
			}()
		}
	}
}
