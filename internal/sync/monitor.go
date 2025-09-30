package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connect/internal/database"
	"github.com/rs/zerolog/log"
)

// Monitor handles synchronization monitoring and health checks
type Monitor struct {
	dbManager   *database.Manager
	syncService *SyncService
	resolver    *ConflictResolver
	logger      *log.Logger
}

// SyncHealth represents the health status of the synchronization system
type SyncHealth struct {
	OverallStatus    string            `json:"overall_status"`
	PostgresStatus   DatabaseStatus    `json:"postgres_status"`
	Neo4jStatus      DatabaseStatus    `json:"neo4j_status"`
	RedisStatus      CacheStatus       `json:"redis_status"`
	EventQueue       EventQueueStatus  `json:"event_queue"`
	ConflictStatus   ConflictStatus    `json:"conflict_status"`
	Performance      PerformanceMetrics `json:"performance"`
	LastCheck        time.Time         `json:"last_check"`
	Issues           []string          `json:"issues"`
}

// DatabaseStatus represents the status of a database
type DatabaseStatus struct {
	Connected    bool      `json:"connected"`
	ResponseTime int64     `json:"response_time_ms"`
	LastChecked  time.Time `json:"last_checked"`
	Error        string    `json:"error,omitempty"`
}

// CacheStatus represents the status of the cache
type CacheStatus struct {
	Connected    bool      `json:"connected"`
	ResponseTime int64     `json:"response_time_ms"`
	LastChecked  time.Time `json:"last_checked"`
	Error        string    `json:"error,omitempty"`
}

// EventQueueStatus represents the status of the event queue
type EventQueueStatus struct {
	PendingEvents   int64     `json:"pending_events"`
	ProcessingEvents int64     `json:"processing_events"`
	FailedEvents    int64     `json:"failed_events"`
	AvgWaitTime     float64   `json:"avg_wait_time_seconds"`
	LastProcessed   time.Time `json:"last_processed"`
}

// ConflictStatus represents the status of conflicts
type ConflictStatus struct {
	UnresolvedConflicts int64     `json:"unresolved_conflicts"`
	TotalConflicts      int64     `json:"total_conflicts"`
	LastConflict        time.Time `json:"last_conflict,omitempty"`
	ResolutionRate      float64   `json:"resolution_rate_percent"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	AvgSyncTime      float64 `json:"avg_sync_time_ms"`
	Throughput       float64 `json:"throughput_events_per_minute"`
	ErrorRate        float64 `json:"error_rate_percent"`
	LastHourEvents   int64   `json:"last_hour_events"`
	LastDayEvents    int64   `json:"last_day_events"`
}

// SyncAlert represents a synchronization alert
type SyncAlert struct {
	ID          string                 `json:"id"`
	Severity    string                 `json:"severity"` // "info", "warning", "error", "critical"
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

// AlertRule represents a rule for generating alerts
type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Enabled     bool          `json:"enabled"`
	Condition   string        `json:"condition"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Severity    string        `json:"severity"`
	Message     string        `json:"message"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// NewMonitor creates a new synchronization monitor
func NewMonitor(dbManager *database.Manager, syncService *SyncService, resolver *ConflictResolver, logger *log.Logger) *Monitor {
	return &Monitor{
		dbManager:   dbManager,
		syncService: syncService,
		resolver:    resolver,
		logger:      logger,
	}
}

// CheckHealth performs a comprehensive health check of the synchronization system
func (m *Monitor) CheckHealth(ctx context.Context) (*SyncHealth, error) {
	health := &SyncHealth{
		LastCheck: time.Now(),
		Issues:    make([]string, 0),
	}

	// Check database connections
	m.checkDatabaseConnections(ctx, health)

	// Check event queue status
	m.checkEventQueue(ctx, health)

	// Check conflict status
	m.checkConflictStatus(ctx, health)

	// Check performance metrics
	m.checkPerformance(ctx, health)

	// Determine overall status
	m.determineOverallStatus(health)

	return health, nil
}

// checkDatabaseConnections checks the status of database connections
func (m *Monitor) checkDatabaseConnections(ctx context.Context, health *SyncHealth) {
	// Check PostgreSQL
	start := time.Now()
	err := m.dbManager.Postgres.Ping(ctx)
	responseTime := time.Since(start).Milliseconds()

	health.PostgresStatus = DatabaseStatus{
		Connected:    err == nil,
		ResponseTime: responseTime,
		LastChecked: time.Now(),
	}

	if err != nil {
		health.PostgresStatus.Error = err.Error()
		health.Issues = append(health.Issues, fmt.Sprintf("PostgreSQL connection failed: %s", err.Error()))
	} else if responseTime > 1000 { // More than 1 second
		health.Issues = append(health.Issues, fmt.Sprintf("PostgreSQL response time is high: %dms", responseTime))
	}

	// Check Neo4j
	start = time.Now()
	err = m.dbManager.Neo4j.VerifyConnectivity(ctx)
	responseTime = time.Since(start).Milliseconds()

	health.Neo4jStatus = DatabaseStatus{
		Connected:    err == nil,
		ResponseTime: responseTime,
		LastChecked: time.Now(),
	}

	if err != nil {
		health.Neo4jStatus.Error = err.Error()
		health.Issues = append(health.Issues, fmt.Sprintf("Neo4j connection failed: %s", err.Error()))
	} else if responseTime > 1000 { // More than 1 second
		health.Issues = append(health.Issues, fmt.Sprintf("Neo4j response time is high: %dms", responseTime))
	}

	// Check Redis
	start = time.Now()
	err = m.dbManager.Redis.Ping(ctx)
	responseTime = time.Since(start).Milliseconds()

	health.RedisStatus = CacheStatus{
		Connected:    err == nil,
		ResponseTime: responseTime,
		LastChecked: time.Now(),
	}

	if err != nil {
		health.RedisStatus.Error = err.Error()
		health.Issues = append(health.Issues, fmt.Sprintf("Redis connection failed: %s", err.Error()))
	} else if responseTime > 500 { // More than 500ms
		health.Issues = append(health.Issues, fmt.Sprintf("Redis response time is high: %dms", responseTime))
	}
}

// checkEventQueue checks the status of the event queue
func (m *Monitor) checkEventQueue(ctx context.Context, health *SyncHealth) {
	// Get event counts
	var pending, processing, failed int64
	var lastProcessed time.Time

	err := m.dbManager.Postgres.QueryRow(ctx, `
		SELECT 
			COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'PROCESSING' THEN 1 END) as processing,
			COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed,
			COALESCE(MAX(processed_at), NOW()) as last_processed
		FROM sync_events
	`).Scan(&pending, &processing, &failed, &lastProcessed)

	if err != nil {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed to get event queue status: %s", err.Error()))
		return
	}

	health.EventQueue = EventQueueStatus{
		PendingEvents:    pending,
		ProcessingEvents: processing,
		FailedEvents:     failed,
		LastProcessed:    lastProcessed,
	}

	// Calculate average wait time
	if pending > 0 {
		var avgWaitTime float64
		err := m.dbManager.Postgres.QueryRow(ctx, `
			SELECT AVG(EXTRACT(EPOCH FROM (NOW() - created_at))) 
			FROM sync_events 
			WHERE status = 'PENDING'
		`).Scan(&avgWaitTime)

		if err == nil {
			health.EventQueue.AvgWaitTime = avgWaitTime
		}
	}

	// Check for issues
	if pending > 1000 {
		health.Issues = append(health.Issues, fmt.Sprintf("High number of pending events: %d", pending))
	}
	if processing > 50 {
		health.Issues = append(health.Issues, fmt.Sprintf("High number of processing events: %d", processing))
	}
	if failed > 100 {
		health.Issues = append(health.Issues, fmt.Sprintf("High number of failed events: %d", failed))
	}
	if health.EventQueue.AvgWaitTime > 300 { // More than 5 minutes
		health.Issues = append(health.Issues, fmt.Sprintf("High average wait time: %.2f seconds", health.EventQueue.AvgWaitTime))
	}
}

// checkConflictStatus checks the status of conflicts
func (m *Monitor) checkConflictStatus(ctx context.Context, health *SyncHealth) {
	var unresolved, total int64
	var lastConflict time.Time

	err := m.dbManager.Postgres.QueryRow(ctx, `
		SELECT 
			COUNT(CASE WHEN resolved = false THEN 1 END) as unresolved,
			COUNT(*) as total,
			COALESCE(MAX(created_at), NOW()) as last_conflict
		FROM sync_conflicts
	`).Scan(&unresolved, &total, &lastConflict)

	if err != nil {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed to get conflict status: %s", err.Error()))
		return
	}

	health.ConflictStatus = ConflictStatus{
		UnresolvedConflicts: unresolved,
		TotalConflicts:      total,
		LastConflict:        lastConflict,
	}

	// Calculate resolution rate
	if total > 0 {
		health.ConflictStatus.ResolutionRate = float64(total-unresolved) / float64(total) * 100
	}

	// Check for issues
	if unresolved > 50 {
		health.Issues = append(health.Issues, fmt.Sprintf("High number of unresolved conflicts: %d", unresolved))
	}
	if health.ConflictStatus.ResolutionRate < 90 {
		health.Issues = append(health.Issues, fmt.Sprintf("Low conflict resolution rate: %.2f%%", health.ConflictStatus.ResolutionRate))
	}
}

// checkPerformance checks performance metrics
func (m *Monitor) checkPerformance(ctx context.Context, health *SyncHealth) {
	// Get average sync time
	var avgSyncTime float64
	err := m.dbManager.Postgres.QueryRow(ctx, `
		SELECT AVG(duration_ms) 
		FROM sync_log 
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&avgSyncTime)

	if err != nil {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed to get average sync time: %s", err.Error()))
	} else {
		health.Performance.AvgSyncTime = avgSyncTime
	}

	// Get throughput (events per minute)
	var lastHourEvents, lastDayEvents int64
	err = m.dbManager.Postgres.QueryRow(ctx, `
		SELECT 
			COUNT(CASE WHEN created_at > NOW() - INTERVAL '1 hour' THEN 1 END) as last_hour,
			COUNT(CASE WHEN created_at > NOW() - INTERVAL '1 day' THEN 1 END) as last_day
		FROM sync_events
	`).Scan(&lastHourEvents, &lastDayEvents)

	if err != nil {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed to get event counts: %s", err.Error()))
	} else {
		health.Performance.LastHourEvents = lastHourEvents
		health.Performance.LastDayEvents = lastDayEvents
		health.Performance.Throughput = float64(lastHourEvents) / 60.0 // events per minute
	}

	// Get error rate
	var totalEvents, failedEvents int64
	err = m.dbManager.Postgres.QueryRow(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed
		FROM sync_events 
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&totalEvents, &failedEvents)

	if err == nil && totalEvents > 0 {
		health.Performance.ErrorRate = float64(failedEvents) / float64(totalEvents) * 100
	}

	// Check for performance issues
	if health.Performance.AvgSyncTime > 5000 { // More than 5 seconds
		health.Issues = append(health.Issues, fmt.Sprintf("High average sync time: %.2fms", health.Performance.AvgSyncTime))
	}
	if health.Performance.ErrorRate > 10 { // More than 10%
		health.Issues = append(health.Issues, fmt.Sprintf("High error rate: %.2f%%", health.Performance.ErrorRate))
	}
}

// determineOverallStatus determines the overall health status
func (m *Monitor) determineOverallStatus(health *SyncHealth) {
	criticalIssues := 0
	warningIssues := 0

	for _, issue := range health.Issues {
		if contains(issue, []string{"failed", "disconnected", "high"}) {
			criticalIssues++
		} else {
			warningIssues++
		}
	}

	if criticalIssues > 0 {
		health.OverallStatus = "critical"
	} else if warningIssues > 2 {
		health.OverallStatus = "warning"
	} else if warningIssues > 0 {
		health.OverallStatus = "degraded"
	} else {
		health.OverallStatus = "healthy"
	}
}

// contains checks if a string contains any of the substrings
func contains(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) > 0 && len(substr) > 0 {
			// Simple substring check
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// GetMetrics returns detailed synchronization metrics
func (m *Monitor) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get basic stats
	stats, err := m.syncService.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync stats: %w", err)
	}
	metrics["stats"] = stats

	// Get health status
	health, err := m.CheckHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}
	metrics["health"] = health

	// Get recent events (last 24 hours)
	recentEvents, err := m.getRecentEvents(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to get recent events")
	} else {
		metrics["recent_events"] = recentEvents
	}

	// Get recent errors
	recentErrors, err := m.syncService.GetRecentErrors(ctx, 10)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to get recent errors")
	} else {
		metrics["recent_errors"] = recentErrors
	}

	// Get conflicts
	conflicts, err := m.resolver.GetConflicts(ctx, 10)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to get conflicts")
	} else {
		metrics["conflicts"] = conflicts
	}

	return metrics, nil
}

// getRecentEvents gets recent sync events
func (m *Monitor) getRecentEvents(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := m.dbManager.Postgres.Query(ctx, `
		SELECT id, entity_type, entity_id, action, status, created_at, duration_ms
		FROM sync_log 
		WHERE created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC 
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		event := make(map[string]interface{})
		var duration *int64

		err := rows.Scan(&event["id"], &event["entity_type"], &event["entity_id"], 
			&event["action"], &event["status"], &event["created_at"], &duration)
		if err != nil {
			return nil, err
		}

		if duration != nil {
			event["duration_ms"] = *duration
		}

		events = append(events, event)
	}

	return events, nil
}

// CreateAlert creates a synchronization alert
func (m *Monitor) CreateAlert(ctx context.Context, severity, alertType, message string, data map[string]interface{}) error {
	alert := &SyncAlert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Severity:  severity,
		Type:      alertType,
		Message:   message,
		Data:      data,
		Resolved:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Alerts expire after 24 hours
	}

	alertJSON, _ := json.Marshal(alert)

	_, err := m.dbManager.Postgres.Exec(ctx, `
		INSERT INTO sync_alerts (id, severity, type, message, data, resolved, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, alert.ID, alert.Severity, alert.Type, alert.Message, string(alertJSON), 
		alert.Resolved, alert.CreatedAt, alert.ExpiresAt)

	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	m.logger.Warn().
		Str("alert_id", alert.ID).
		Str("severity", alert.Severity).
		Str("type", alert.Type).
		Str("message", alert.Message).
		Msg("Sync alert created")

	return nil
}

// GetActiveAlerts returns active (unresolved and unexpired) alerts
func (m *Monitor) GetActiveAlerts(ctx context.Context) ([]*SyncAlert, error) {
	rows, err := m.dbManager.Postgres.Query(ctx, `
		SELECT id, severity, type, message, data, resolved, resolved_at, created_at, expires_at
		FROM sync_alerts 
		WHERE resolved = false AND expires_at > NOW()
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*SyncAlert
	for rows.Next() {
		alert := &SyncAlert{}
		var dataJSON []byte

		err := rows.Scan(&alert.ID, &alert.Severity, &alert.Type, &alert.Message,
			&dataJSON, &alert.Resolved, &alert.ResolvedAt, &alert.CreatedAt, &alert.ExpiresAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal data
		if len(dataJSON) > 0 {
			json.Unmarshal(dataJSON, &alert.Data)
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// ResolveAlert marks an alert as resolved
func (m *Monitor) ResolveAlert(ctx context.Context, alertID string) error {
	now := time.Now()
	_, err := m.dbManager.Postgres.Exec(ctx, `
		UPDATE sync_alerts 
		SET resolved = true, resolved_at = $1
		WHERE id = $2
	`, now, alertID)

	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	m.logger.Info().Str("alert_id", alertID).Msg("Alert resolved")
	return nil
}

// StartMonitoring starts continuous monitoring
func (m *Monitor) StartMonitoring(ctx context.Context) {
	m.logger.Info("Starting sync monitoring")

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Stopping sync monitoring")
			return
		case <-ticker.C:
			go m.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs a health check and generates alerts if needed
func (m *Monitor) performHealthCheck(ctx context.Context) {
	health, err := m.CheckHealth(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to perform health check")
		return
	}

	// Generate alerts based on health status
	switch health.OverallStatus {
	case "critical":
		m.CreateAlert(ctx, "critical", "system_health", 
			"Critical synchronization system issues detected", 
			map[string]interface{}{"health": health})
	case "warning":
		m.CreateAlert(ctx, "warning", "system_health", 
			"Warning: synchronization system issues detected", 
			map[string]interface{}{"health": health})
	}

	// Check specific metrics for alerts
	if health.EventQueue.PendingEvents > 500 {
		m.CreateAlert(ctx, "warning", "event_queue", 
			fmt.Sprintf("High number of pending sync events: %d", health.EventQueue.PendingEvents),
			map[string]interface{}{"pending_events": health.EventQueue.PendingEvents})
	}

	if health.Performance.ErrorRate > 20 {
		m.CreateAlert(ctx, "error", "performance", 
			fmt.Sprintf("High sync error rate: %.2f%%", health.Performance.ErrorRate),
			map[string]interface{}{"error_rate": health.Performance.ErrorRate})
	}

	if health.ConflictStatus.UnresolvedConflicts > 20 {
		m.CreateAlert(ctx, "warning", "conflicts", 
			fmt.Sprintf("High number of unresolved conflicts: %d", health.ConflictStatus.UnresolvedConflicts),
			map[string]interface{}{"unresolved_conflicts": health.ConflictStatus.UnresolvedConflicts})
	}
}

// CleanupExpiredAlerts cleans up expired alerts
func (m *Monitor) CleanupExpiredAlerts(ctx context.Context) error {
	_, err := m.dbManager.Postgres.Exec(ctx, `
		DELETE FROM sync_alerts 
		WHERE expires_at < NOW() OR (resolved = true AND resolved_at < NOW() - INTERVAL '7 days')
	`)

	if err != nil {
		return fmt.Errorf("failed to cleanup expired alerts: %w", err)
	}

	m.logger.Info().Msg("Expired alerts cleaned up")
	return nil
}
