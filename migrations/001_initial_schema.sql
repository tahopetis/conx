-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create permissions table
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource_type VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create role-permission mapping table
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Create user-role mapping table
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- Create CI type definitions table
CREATE TABLE ci_type_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type_name VARCHAR(100) UNIQUE NOT NULL,
    field_schema JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create configuration items table
CREATE TABLE configuration_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    
    -- Constraints
    CONSTRAINT unique_name_per_type UNIQUE (name, type)
);

-- Create relationships table
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id UUID NOT NULL REFERENCES configuration_items(id),
    target_id UUID NOT NULL REFERENCES configuration_items(id),
    type VARCHAR(50) NOT NULL,
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    
    -- Constraints
    CONSTRAINT no_self_relationship CHECK (source_id != target_id)
);

-- Create audit logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    action VARCHAR(50) NOT NULL,
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    details JSONB NOT NULL DEFAULT '{}'
);

-- Create import jobs table
CREATE TABLE import_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT,
    status VARCHAR(20) DEFAULT 'pending',
    total_records INTEGER DEFAULT 0,
    successful_records INTEGER DEFAULT 0,
    failed_records INTEGER DEFAULT 0,
    error_details JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX idx_cis_type ON configuration_items(type);
CREATE INDEX idx_cis_tags ON configuration_items USING GIN(tags);
CREATE INDEX idx_cis_attributes ON configuration_items USING GIN(attributes);
CREATE INDEX idx_cis_created_at ON configuration_items(created_at);

CREATE INDEX idx_relationships_source ON relationships(source_id);
CREATE INDEX idx_relationships_target ON relationships(target_id);
CREATE INDEX idx_relationships_type ON relationships(type);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_changed_at ON audit_logs(changed_at);
CREATE INDEX idx_audit_changed_by ON audit_logs(changed_by);

CREATE INDEX idx_import_status ON import_jobs(status);
CREATE INDEX idx_import_created_by ON import_jobs(created_by);

-- Insert default roles
INSERT INTO roles (id, name, description, created_at) VALUES
('00000000-0000-0000-0000-000000000001', 'admin', 'System administrator with full access', NOW()),
('00000000-0000-0000-0000-000000000002', 'ci_manager', 'Can manage configuration items and relationships', NOW()),
('00000000-0000-0000-0000-000000000003', 'viewer', 'Read-only access to the system', NOW()),
('00000000-0000-0000-0000-000000000004', 'auditor', 'Read access plus audit logs', NOW());

-- Insert default permissions
INSERT INTO permissions (id, name, description, resource_type, created_at) VALUES
('00000000-0000-0000-0000-000000000010', 'ci:create', 'Create configuration items', 'ci', NOW()),
('00000000-0000-0000-0000-000000000011', 'ci:read', 'Read configuration items', 'ci', NOW()),
('00000000-0000-0000-0000-000000000012', 'ci:update', 'Update configuration items', 'ci', NOW()),
('00000000-0000-0000-0000-000000000013', 'ci:delete', 'Delete configuration items', 'ci', NOW()),
('00000000-0000-0000-0000-000000000020', 'relationship:manage', 'Manage relationships between CIs', 'relationship', NOW()),
('00000000-0000-0000-0000-000000000030', 'audit_log:read', 'Read audit logs', 'audit_log', NOW()),
('00000000-0000-0000-0000-000000000040', 'user:manage', 'Manage users and roles', 'user', NOW()),
('00000000-0000-0000-0000-000000000050', 'import:csv', 'Import data from CSV files', 'import', NOW());

-- Assign permissions to roles
-- Admin role gets all permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000011'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000012'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000013'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000020'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000030'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000040'),
('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000050');

-- CI Manager role permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000010'),
('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000011'),
('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000012'),
('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000013'),
('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000020'),
('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000050');

-- Viewer role permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000011');

-- Auditor role permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000011'),
('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000030');

-- Insert default CI type definitions
INSERT INTO ci_type_definitions (id, type_name, field_schema, created_at, updated_at) VALUES
('00000000-0000-0000-0000-000000000100', 'server', '{"type": "object", "properties": {"ip_address": {"type": "string"}, "hostname": {"type": "string"}, "os": {"type": "string"}, "cpu_cores": {"type": "integer"}, "memory_gb": {"type": "integer"}}}', NOW(), NOW()),
('00000000-0000-0000-0000-000000000101', 'application', '{"type": "object", "properties": {"version": {"type": "string"}, "language": {"type": "string"}, "framework": {"type": "string"}, "port": {"type": "integer"}}}', NOW(), NOW()),
('00000000-0000-0000-0000-000000000102', 'database', '{"type": "object", "properties": {"engine": {"type": "string"}, "version": {"type": "string"}, "port": {"type": "integer"}, "data_directory": {"type": "string"}}}', NOW(), NOW()),
('00000000-0000-0000-0000-000000000103', 'network_device', '{"type": "object", "properties": {"device_type": {"type": "string"}, "ip_address": {"type": "string"}, "mac_address": {"type": "string"}, "model": {"type": "string"}}}', NOW(), NOW());

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop tables in reverse order to avoid foreign key constraint violations
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS relationships;
DROP TABLE IF EXISTS configuration_items;
DROP TABLE IF EXISTS ci_type_definitions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;

-- Drop UUID extension
DROP EXTENSION IF EXISTS "uuid-ossp";
