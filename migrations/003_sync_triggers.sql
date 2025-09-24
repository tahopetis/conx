-- Migration: Synchronization Triggers
-- Description: Create triggers for detecting changes in PostgreSQL and generating sync events

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create function to generate sync event for CI changes
CREATE OR REPLACE FUNCTION generate_ci_sync_event()
RETURNS TRIGGER AS $$
DECLARE
	event_data JSONB;
	action_type TEXT;
BEGIN
	-- Determine action type
	IF TG_OP = 'INSERT' THEN
		action_type := 'CREATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'name', NEW.name,
			'type', NEW.type,
			'description', COALESCE(NEW.description, ''),
			'status', COALESCE(NEW.status, 'active'),
			'attributes', COALESCE(NEW.attributes, '{}'::jsonb),
			'tags', COALESCE(NEW.tags, '{}'::text[]),
			'created_by', COALESCE(NEW.created_by, 'system'),
			'created_at', COALESCE(NEW.created_at, NOW()),
			'updated_at', COALESCE(NEW.updated_at, NOW())
		);
	ELSIF TG_OP = 'UPDATE' THEN
		action_type := 'UPDATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'name', NEW.name,
			'type', NEW.type,
			'description', COALESCE(NEW.description, ''),
			'status', COALESCE(NEW.status, 'active'),
			'attributes', COALESCE(NEW.attributes, '{}'::jsonb),
			'tags', COALESCE(NEW.tags, '{}'::text[]),
			'created_by', COALESCE(NEW.created_by, 'system'),
			'updated_by', COALESCE(NEW.updated_by, 'system'),
			'created_at', COALESCE(NEW.created_at, NOW()),
			'updated_at', COALESCE(NEW.updated_at, NOW())
		);
	ELSIF TG_OP = 'DELETE' THEN
		action_type := 'DELETE';
		event_data := jsonb_build_object(
			'id', OLD.id,
			'name', OLD.name,
			'type', OLD.type,
			'description', COALESCE(OLD.description, ''),
			'status', COALESCE(OLD.status, 'active'),
			'attributes', COALESCE(OLD.attributes, '{}'::jsonb),
			'tags', COALESCE(OLD.tags, '{}'::text[]),
			'created_by', COALESCE(OLD.created_by, 'system'),
			'created_at', COALESCE(OLD.created_at, NOW()),
			'updated_at', COALESCE(OLD.updated_at, NOW())
		);
	END IF;

	-- Insert sync event
	INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
	VALUES (
		uuid_generate_v4(),
		'configuration_item',
		CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END,
		action_type,
		event_data,
		'PENDING',
		NOW()
	);

	-- Return appropriate value based on operation
	IF TG_OP = 'DELETE' THEN
		RETURN OLD;
	ELSE
		RETURN NEW;
	END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to generate sync event for relationship changes
CREATE OR REPLACE FUNCTION generate_relationship_sync_event()
RETURNS TRIGGER AS $$
DECLARE
	event_data JSONB;
	action_type TEXT;
BEGIN
	-- Determine action type
	IF TG_OP = 'INSERT' THEN
		action_type := 'CREATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'source_id', NEW.source_id,
			'target_id', NEW.target_id,
			'type', NEW.type,
			'description', COALESCE(NEW.description, ''),
			'attributes', COALESCE(NEW.attributes, '{}'::jsonb),
			'strength', COALESCE(NEW.strength, 1),
			'created_by', COALESCE(NEW.created_by, 'system'),
			'created_at', COALESCE(NEW.created_at, NOW())
		);
	ELSIF TG_OP = 'UPDATE' THEN
		action_type := 'UPDATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'source_id', NEW.source_id,
			'target_id', NEW.target_id,
			'type', NEW.type,
			'description', COALESCE(NEW.description, ''),
			'attributes', COALESCE(NEW.attributes, '{}'::jsonb),
			'strength', COALESCE(NEW.strength, 1),
			'created_by', COALESCE(NEW.created_by, 'system'),
			'created_at', COALESCE(NEW.created_at, NOW()),
			'updated_at', COALESCE(NEW.updated_at, NOW())
		);
	ELSIF TG_OP = 'DELETE' THEN
		action_type := 'DELETE';
		event_data := jsonb_build_object(
			'id', OLD.id,
			'source_id', OLD.source_id,
			'target_id', OLD.target_id,
			'type', OLD.type,
			'description', COALESCE(OLD.description, ''),
			'attributes', COALESCE(OLD.attributes, '{}'::jsonb),
			'strength', COALESCE(OLD.strength, 1),
			'created_by', COALESCE(OLD.created_by, 'system'),
			'created_at', COALESCE(OLD.created_at, NOW())
		);
	END IF;

	-- Insert sync event
	INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
	VALUES (
		uuid_generate_v4(),
		'relationship',
		CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END,
		action_type,
		event_data,
		'PENDING',
		NOW()
	);

	-- Return appropriate value based on operation
	IF TG_OP = 'DELETE' THEN
		RETURN OLD;
	ELSE
		RETURN NEW;
	END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to generate sync event for user changes (for user node sync in Neo4j)
CREATE OR REPLACE FUNCTION generate_user_sync_event()
RETURNS TRIGGER AS $$
DECLARE
	event_data JSONB;
	action_type TEXT;
BEGIN
	-- Determine action type
	IF TG_OP = 'INSERT' THEN
		action_type := 'CREATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'username', NEW.username,
			'email', NEW.email,
			'is_active', NEW.is_active,
			'created_at', COALESCE(NEW.created_at, NOW())
		);
	ELSIF TG_OP = 'UPDATE' THEN
		action_type := 'UPDATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'username', NEW.username,
			'email', NEW.email,
			'is_active', NEW.is_active,
			'created_at', COALESCE(NEW.created_at, NOW()),
			'updated_at', COALESCE(NEW.updated_at, NOW())
		);
	ELSIF TG_OP = 'DELETE' THEN
		action_type := 'DELETE';
		event_data := jsonb_build_object(
			'id', OLD.id,
			'username', OLD.username,
			'email', OLD.email,
			'is_active', OLD.is_active,
			'created_at', COALESCE(OLD.created_at, NOW())
		);
	END IF;

	-- Insert sync event
	INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
	VALUES (
		uuid_generate_v4(),
		'user',
		CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END,
		action_type,
		event_data,
		'PENDING',
		NOW()
	);

	-- Return appropriate value based on operation
	IF TG_OP = 'DELETE' THEN
		RETURN OLD;
	ELSE
		RETURN NEW;
	END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to generate sync event for role changes (for role-based access sync)
CREATE OR REPLACE FUNCTION generate_role_sync_event()
RETURNS TRIGGER AS $$
DECLARE
	event_data JSONB;
	action_type TEXT;
BEGIN
	-- Determine action type
	IF TG_OP = 'INSERT' THEN
		action_type := 'CREATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'name', NEW.name,
			'description', COALESCE(NEW.description, ''),
			'created_at', COALESCE(NEW.created_at, NOW())
		);
	ELSIF TG_OP = 'UPDATE' THEN
		action_type := 'UPDATE';
		event_data := jsonb_build_object(
			'id', NEW.id,
			'name', NEW.name,
			'description', COALESCE(NEW.description, ''),
			'created_at', COALESCE(NEW.created_at, NOW())
		);
	ELSIF TG_OP = 'DELETE' THEN
		action_type := 'DELETE';
		event_data := jsonb_build_object(
			'id', OLD.id,
			'name', OLD.name,
			'description', COALESCE(OLD.description, ''),
			'created_at', COALESCE(OLD.created_at, NOW())
		);
	END IF;

	-- Insert sync event
	INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
	VALUES (
		uuid_generate_v4(),
		'role',
		CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END,
		action_type,
		event_data,
		'PENDING',
		NOW()
	);

	-- Return appropriate value based on operation
	IF TG_OP = 'DELETE' THEN
		RETURN OLD;
	ELSE
		RETURN NEW;
	END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to generate sync event for user-role assignment changes
CREATE OR REPLACE FUNCTION generate_user_role_sync_event()
RETURNS TRIGGER AS $$
DECLARE
	event_data JSONB;
	action_type TEXT;
BEGIN
	-- Determine action type
	IF TG_OP = 'INSERT' THEN
		action_type := 'ASSIGN';
		event_data := jsonb_build_object(
			'user_id', NEW.user_id,
			'role_id', NEW.role_id,
			'assigned_by', COALESCE(NEW.assigned_by, 'system'),
			'assigned_at', COALESCE(NEW.assigned_at, NOW())
		);
	ELSIF TG_OP = 'DELETE' THEN
		action_type := 'UNASSIGN';
		event_data := jsonb_build_object(
			'user_id', OLD.user_id,
			'role_id', OLD.role_id,
			'assigned_by', COALESCE(OLD.assigned_by, 'system'),
			'assigned_at', COALESCE(OLD.assigned_at, NOW())
		);
	END IF;

	-- Insert sync event
	INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
	VALUES (
		uuid_generate_v4(),
		'user_role',
		CASE WHEN TG_OP = 'DELETE' THEN OLD.user_id ELSE NEW.user_id END,
		action_type,
		event_data,
		'PENDING',
		NOW()
	);

	-- Return appropriate value based on operation
	IF TG_OP = 'DELETE' THEN
		RETURN OLD;
	ELSE
		RETURN NEW;
	END IF;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for configuration_items table
DROP TRIGGER IF EXISTS ci_sync_trigger ON configuration_items;
CREATE TRIGGER ci_sync_trigger
AFTER INSERT OR UPDATE OR DELETE ON configuration_items
FOR EACH ROW
EXECUTE FUNCTION generate_ci_sync_event();

-- Create triggers for relationships table
DROP TRIGGER IF EXISTS relationship_sync_trigger ON relationships;
CREATE TRIGGER relationship_sync_trigger
AFTER INSERT OR UPDATE OR DELETE ON relationships
FOR EACH ROW
EXECUTE FUNCTION generate_relationship_sync_event();

-- Create triggers for users table
DROP TRIGGER IF EXISTS user_sync_trigger ON users;
CREATE TRIGGER user_sync_trigger
AFTER INSERT OR UPDATE OR DELETE ON users
FOR EACH ROW
EXECUTE FUNCTION generate_user_sync_event();

-- Create triggers for roles table
DROP TRIGGER IF EXISTS role_sync_trigger ON roles;
CREATE TRIGGER role_sync_trigger
AFTER INSERT OR UPDATE OR DELETE ON roles
FOR EACH ROW
EXECUTE FUNCTION generate_role_sync_event();

-- Create triggers for user_roles table
DROP TRIGGER IF EXISTS user_role_sync_trigger ON user_roles;
CREATE TRIGGER user_role_sync_trigger
AFTER INSERT OR DELETE ON user_roles
FOR EACH ROW
EXECUTE FUNCTION generate_user_role_sync_event();

-- Create function to handle batch sync events (for bulk operations)
CREATE OR REPLACE FUNCTION generate_batch_sync_events()
RETURNS TRIGGER AS $$
DECLARE
	event_data JSONB;
	action_type TEXT;
	entity_id UUID;
BEGIN
	-- This function can be used for bulk operations that need to generate multiple sync events
	-- For now, we'll handle it as a single event for the batch operation
	
	action_type := 'BATCH_UPDATE';
	entity_id := uuid_generate_v4(); -- Generate a unique ID for the batch event
	
	event_data := jsonb_build_object(
		'operation', TG_OP,
		'table_name', TG_TABLE_NAME,
		'timestamp', NOW(),
		'statement', current_query()
	);

	-- Insert sync event for the batch operation
	INSERT INTO sync_events (id, entity_type, entity_id, action, data, status, created_at)
	VALUES (
		uuid_generate_v4(),
		'batch_operation',
		entity_id,
		action_type,
		event_data,
		'PENDING',
		NOW()
	);

	RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create function to cleanup old sync events (can be called by a scheduled job)
CREATE OR REPLACE FUNCTION cleanup_old_sync_events(retention_days INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
	deleted_count INTEGER;
BEGIN
	-- Delete old sync events
	DELETE FROM sync_log 
	WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
	
	GET DIAGNOSTICS deleted_count = ROW_COUNT;
	
	-- Also delete old completed sync events from main table
	DELETE FROM sync_events 
	WHERE status IN ('COMPLETED', 'FAILED') 
	AND created_at < NOW() - INTERVAL '1 day' * retention_days;
	
	GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
	
	RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create sync_conflicts table for conflict resolution
CREATE TABLE IF NOT EXISTS sync_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    conflict_type VARCHAR(50) NOT NULL,
    postgres_data JSONB NOT NULL,
    neo4j_data JSONB NOT NULL,
    resolution VARCHAR(20) NOT NULL DEFAULT 'postgres_wins',
    resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT valid_conflict_type CHECK (conflict_type IN ('data_mismatch', 'missing_entity', 'relationship_conflict', 'timestamp_conflict', 'version_conflict')),
    CONSTRAINT valid_resolution CHECK (resolution IN ('postgres_wins', 'neo4j_wins', 'merge', 'manual', 'timestamp'))
);

-- Create indexes for sync_conflicts
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_resolved ON sync_conflicts(resolved);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_entity ON sync_conflicts(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_created_at ON sync_conflicts(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_conflict_type ON sync_conflicts(conflict_type);

-- Create sync_alerts table for monitoring alerts
CREATE TABLE IF NOT EXISTS sync_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    severity VARCHAR(20) NOT NULL,
    type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    data JSONB DEFAULT '{}'::jsonb,
    resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    CONSTRAINT valid_severity CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    CONSTRAINT valid_expires_at CHECK (expires_at > created_at)
);

-- Create indexes for sync_alerts
CREATE INDEX IF NOT EXISTS idx_sync_alerts_resolved ON sync_alerts(resolved);
CREATE INDEX IF NOT EXISTS idx_sync_alerts_severity ON sync_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_sync_alerts_created_at ON sync_alerts(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_alerts_expires_at ON sync_alerts(expires_at);

-- Create sync_fallback_operations table for fallback sync procedures
CREATE TABLE IF NOT EXISTS sync_fallback_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_event_id UUID NOT NULL,
    strategy VARCHAR(30) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL,
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    retry_count INTEGER DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT valid_strategy CHECK (strategy IN ('retry', 'manual', 'skip', 'queue', 'full_resync', 'selective_resync')),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'completed_with_errors'))
);

-- Create indexes for sync_fallback_operations
CREATE INDEX IF NOT EXISTS idx_sync_fallback_operations_status ON sync_fallback_operations(status);
CREATE INDEX IF NOT EXISTS idx_sync_fallback_operations_strategy ON sync_fallback_operations(strategy);
CREATE INDEX IF NOT EXISTS idx_sync_fallback_operations_entity ON sync_fallback_operations(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_sync_fallback_operations_created_at ON sync_fallback_operations(created_at);

-- Create sync_fallback_log table for fallback operation logging
CREATE TABLE IF NOT EXISTS sync_fallback_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL,
    strategy VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for sync_fallback_log
CREATE INDEX IF NOT EXISTS idx_sync_fallback_log_created_at ON sync_fallback_log(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_fallback_log_event_id ON sync_fallback_log(event_id);

-- Create sync_full_resync_status table for tracking full resync operations
CREATE TABLE IF NOT EXISTS sync_full_resync_status (
    id SERIAL PRIMARY KEY,
    in_progress BOOLEAN NOT NULL DEFAULT false,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_resync_started_at TIMESTAMP WITH TIME ZONE,
    last_resync_completed_at TIMESTAMP WITH TIME ZONE,
    last_resync_duration INTERVAL,
    last_resync_status VARCHAR(50),
    last_resync_summary TEXT
);

-- Create function to get sync statistics
CREATE OR REPLACE FUNCTION get_sync_statistics()
RETURNS TABLE (
	total_events BIGINT,
	pending_events BIGINT,
	processing_events BIGINT,
	completed_events BIGINT,
	failed_events BIGINT,
	avg_processing_time INTERVAL,
	last_event_time TIMESTAMP WITH TIME ZONE,
	error_rate NUMERIC
) AS $$
BEGIN
	RETURN QUERY
	SELECT 
		COUNT(*) as total_events,
		COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending_events,
		COUNT(CASE WHEN status = 'PROCESSING' THEN 1 END) as processing_events,
		COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END) as completed_events,
		COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed_events,
		AVG(CASE WHEN processed_at IS NOT NULL AND created_at IS NOT NULL 
			THEN processed_at - created_at ELSE NULL END) as avg_processing_time,
		MAX(created_at) as last_event_time,
		CASE 
			WHEN COUNT(*) > 0 
			THEN COUNT(CASE WHEN status = 'FAILED' THEN 1 END)::NUMERIC / COUNT(*) * 100 
			ELSE 0 
		END as error_rate
	FROM sync_events;
END;
$$ LANGUAGE plpgsql;

-- Create indexes for better performance of sync operations
CREATE INDEX IF NOT EXISTS idx_sync_events_action_created ON sync_events(action, created_at);
CREATE INDEX IF NOT EXISTS idx_sync_events_entity_action ON sync_events(entity_type, action, created_at);
CREATE INDEX IF NOT EXISTS idx_sync_log_created_at ON sync_log(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_log_event_id ON sync_log(event_id);

-- Grant necessary permissions (adjust based on your database user)
-- GRANT EXECUTE ON FUNCTION generate_ci_sync_event() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION generate_relationship_sync_event() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION generate_user_sync_event() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION generate_role_sync_event() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION generate_user_role_sync_event() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION cleanup_old_sync_events(INTEGER) TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION get_sync_statistics() TO cmdb_user;

-- Migration completion comment
-- Migration 003: Synchronization Triggers completed successfully
-- Triggers created for automatic sync event generation
-- Functions created for batch operations and cleanup
-- Indexes created for optimal performance
