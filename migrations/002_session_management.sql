-- Migration: Session Management
-- Description: Create tables for user session management and activity tracking

-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    refresh_token TEXT NOT NULL UNIQUE,
    ip_address INET NOT NULL,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_active_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Constraints
    CONSTRAINT sessions_token_check CHECK (token IS NOT NULL AND length(token) > 0),
    CONSTRAINT sessions_refresh_token_check CHECK (refresh_token IS NOT NULL AND length(refresh_token) > 0),
    CONSTRAINT sessions_expires_at_check CHECK (expires_at > created_at),
    CONSTRAINT sessions_last_active_at_check CHECK (last_active_at >= created_at),
    CONSTRAINT sessions_revoked_at_check CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

-- Create session_activities table
CREATE TABLE IF NOT EXISTS session_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    details TEXT,
    ip_address INET NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT session_activities_action_check CHECK (action IS NOT NULL AND length(action) > 0),
    CONSTRAINT session_activities_ip_address_check CHECK (ip_address IS NOT NULL)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_sessions_is_active ON sessions(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_sessions_revoked_at ON sessions(revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_user_active ON sessions(user_id, is_active) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_session_activities_session_id ON session_activities(session_id);
CREATE INDEX IF NOT EXISTS idx_session_activities_action ON session_activities(action);
CREATE INDEX IF NOT EXISTS idx_session_activities_created_at ON session_activities(created_at);
CREATE INDEX IF NOT EXISTS idx_session_activities_session_created ON session_activities(session_id, created_at);

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_sessions_user_expires ON sessions(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_active_expires ON sessions(is_active, expires_at) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_session_activities_session_action ON session_activities(session_id, action);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for sessions table
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to automatically mark expired sessions as inactive
CREATE OR REPLACE FUNCTION mark_expired_sessions_inactive()
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE sessions
    SET is_active = false,
        updated_at = NOW()
    WHERE expires_at <= NOW()
    AND is_active = true;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Create function to clean up old session activities (keep last 90 days)
CREATE OR REPLACE FUNCTION cleanup_old_session_activities()
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    DELETE FROM session_activities
    WHERE created_at < NOW() - INTERVAL '90 days';
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Create function to get session statistics
CREATE OR REPLACE FUNCTION get_session_statistics(
    OUT total_sessions INTEGER,
    OUT active_sessions INTEGER,
    OUT expired_sessions INTEGER,
    OUT revoked_sessions INTEGER,
    OUT sessions_today INTEGER,
    OUT sessions_this_week INTEGER,
    OUT sessions_this_month INTEGER,
    OUT average_session_time_hours FLOAT
)
AS $$
BEGIN
    -- Total sessions
    SELECT COUNT(*) INTO total_sessions FROM sessions;
    
    -- Active sessions
    SELECT COUNT(*) INTO active_sessions 
    FROM sessions 
    WHERE is_active = true 
    AND revoked_at IS NULL 
    AND expires_at > NOW();
    
    -- Expired sessions
    SELECT COUNT(*) INTO expired_sessions 
    FROM sessions 
    WHERE expires_at <= NOW();
    
    -- Revoked sessions
    SELECT COUNT(*) INTO revoked_sessions 
    FROM sessions 
    WHERE revoked_at IS NOT NULL;
    
    -- Sessions created today
    SELECT COUNT(*) INTO sessions_today 
    FROM sessions 
    WHERE DATE(created_at) = CURRENT_DATE;
    
    -- Sessions created this week
    SELECT COUNT(*) INTO sessions_this_week 
    FROM sessions 
    WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE);
    
    -- Sessions created this month
    SELECT COUNT(*) INTO sessions_this_month 
    FROM sessions 
    WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE);
    
    -- Average session time in hours
    SELECT AVG(EXTRACT(EPOCH FROM (COALESCE(revoked_at, expires_at) - created_at)) / 3600)
    INTO average_session_time_hours
    FROM sessions 
    WHERE revoked_at IS NOT NULL OR expires_at <= NOW();
    
    RETURN;
END;
$$ LANGUAGE plpgsql;

-- Create view for active sessions with user information
CREATE OR REPLACE VIEW active_sessions_with_users AS
SELECT 
    s.id,
    s.user_id,
    u.username,
    u.email,
    s.token,
    s.refresh_token,
    s.ip_address,
    s.user_agent,
    s.expires_at,
    s.last_active_at,
    s.created_at,
    s.is_active
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.is_active = true 
AND s.revoked_at IS NULL 
AND s.expires_at > NOW();

-- Create view for session activities with user information
CREATE OR REPLACE VIEW session_activities_with_users AS
SELECT 
    sa.id,
    sa.session_id,
    sa.action,
    sa.details,
    sa.ip_address,
    sa.user_agent,
    sa.created_at,
    s.user_id,
    u.username,
    u.email
FROM session_activities sa
JOIN sessions s ON sa.session_id = s.id
JOIN users u ON s.user_id = u.id
ORDER BY sa.created_at DESC;

-- Grant permissions (adjust based on your database user)
-- GRANT ALL PRIVILEGES ON sessions TO cmdb_user;
-- GRANT ALL PRIVILEGES ON session_activities TO cmdb_user;
-- GRANT ALL PRIVILEGES ON active_sessions_with_users TO cmdb_user;
-- GRANT ALL PRIVILEGES ON session_activities_with_users TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION mark_expired_sessions_inactive() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION cleanup_old_session_activities() TO cmdb_user;
-- GRANT EXECUTE ON FUNCTION get_session_statistics() TO cmdb_user;

-- Insert default session settings into configuration if needed
-- This would depend on your configuration system
INSERT INTO configuration (key, value, description, created_at, updated_at)
VALUES 
    ('session.max_concurrent_sessions', '5', 'Maximum number of concurrent sessions per user', NOW(), NOW()),
    ('session.default_ttl_hours', '24', 'Default session time-to-live in hours', NOW(), NOW()),
    ('session.max_ttl_hours', '720', 'Maximum session time-to-live in hours', NOW(), NOW()),
    ('session.min_ttl_minutes', '15', 'Minimum session time-to-live in minutes', NOW(), NOW()),
    ('session.cleanup_interval_hours', '1', 'Interval for session cleanup in hours', NOW(), NOW()),
    ('session.activity_retention_days', '90', 'Number of days to retain session activities', NOW(), NOW())
ON CONFLICT (key) DO UPDATE SET 
    value = EXCLUDED.value,
    updated_at = NOW();

-- Migration completion comment
-- Migration 002: Session Management completed successfully
-- Tables created: sessions, session_activities
-- Indexes created for optimal performance
-- Functions created for maintenance and statistics
-- Views created for common queries
-- Default configuration values inserted
