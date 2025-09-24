# Technical Specification Document (TSD)

Project: conx
Version: 1.0
Date: 2025-09-23
Author: Tahopetis

## 🎯 Purpose

conx is a lightweight, open-source Configuration Management Database (CMDB) designed to manage configuration items (CIs), their attributes, and relationships. This document describes the technical design of the system, covering architecture, data model, API contracts, security, testing, and deployment with a focus on performance at scale (50k+ CIs).

## 📋 Table of Contents

1. [System Architecture](#system-architecture)
2. [Data Model](#data-model)
3. [API Specification](#api-specification)
4. [Security Design](#security-design)
5. [Performance Design](#performance-design)
6. [Testing Strategy](#testing-strategy)
7. [Deployment Architecture](#deployment-architecture)
8. [Monitoring & Logging](#monitoring--logging)
9. [Development Workflow](#development-workflow)
10. [Technology Stack](#technology-stack)

## 🏗️ System Architecture

### 2.1 High-Level Architecture

```mermaid
flowchart TB
    subgraph "Client Layer"
        A[Vue 3 SPA]
        B[Mobile/Web Clients]
    end
    
    subgraph "API Gateway Layer"
        C[Go REST API]
        D[Authentication Middleware]
        E[RBAC Middleware]
    end
    
    subgraph "Service Layer"
        F[Core Services]
        G[Graph Service]
        H[Import Service]
        I[Audit Service]
        J[Auth Service]
    end
    
    subgraph "Data Layer"
        K[(PostgreSQL)]
        L[(Neo4j OSS)]
        M[(Redis Cache)]
    end
    
    subgraph "External Services"
        N[Monitoring]
        O[Logging]
    end
    
    A --> C
    B --> C
    C --> D
    D --> E
    E --> F
    E --> G
    E --> H
    E --> I
    E --> J
    F --> K
    F --> L
    G --> L
    H --> K
    I --> K
    J --> K
    F --> M
    G --> M
    C --> N
    C --> O
```

### 2.2 Component Overview

#### 2.2.1 Frontend (Vue 3 SPA)
- **Framework**: Vue 3 with Composition API
- **State Management**: Pinia
- **UI Library**: TailwindCSS
- **Graph Visualization**: D3.js + Vis.js
- **Build Tool**: Vite
- **Testing**: Vitest + Vue Testing Library

#### 2.2.2 Backend (Go API)
- **Framework**: Chi router
- **Database**: PostgreSQL + Neo4j
- **ORM**: sqlx for PostgreSQL, neo4j-go-driver for Neo4j
- **Authentication**: JWT with Argon2id password hashing
- **Caching**: Redis
- **Logging**: Zerolog

#### 2.2.3 Graph Service
- **Database**: Neo4j OSS Community Edition
- **Query Language**: Cypher with APOC procedures
- **Driver**: neo4j-go-driver
- **Caching**: Redis for frequent queries
- **Performance**: Connection pooling and query optimization

#### 2.2.4 Data Synchronization
- **Strategy**: Event-driven architecture
- **PostgreSQL → Neo4j**: Change Data Capture (CDC) via triggers
- **Consistency**: Eventual consistency with conflict resolution
- **Fallback**: Periodic batch sync

## 🗃️ Data Model

### 3.1 PostgreSQL Schema

#### 3.1.1 Core Tables

```sql
-- Configuration Items
CREATE TABLE configuration_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Relationships (for audit and backup)
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES configuration_items(id),
    target_id UUID NOT NULL REFERENCES configuration_items(id),
    type VARCHAR(50) NOT NULL,
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    
    -- Constraints
    CONSTRAINT no_self_relationship CHECK (source_id != target_id)
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Roles
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Permissions
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource_type VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Role-Permission Mapping
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- User-Role Mapping
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- CI Type Definitions
CREATE TABLE ci_type_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type_name VARCHAR(100) UNIQUE NOT NULL,
    field_schema JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Audit Logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    action VARCHAR(50) NOT NULL,
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    details JSONB NOT NULL DEFAULT '{}'
);

-- Import Jobs
CREATE TABLE import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
```

#### 3.1.2 Indexes

```sql
-- Configuration Items Indexes
CREATE INDEX idx_cis_type ON configuration_items(type);
CREATE INDEX idx_cis_tags ON configuration_items USING GIN(tags);
CREATE INDEX idx_cis_attributes ON configuration_items USING GIN(attributes);
CREATE INDEX idx_cis_created_at ON configuration_items(created_at);

-- Relationships Indexes
CREATE INDEX idx_relationships_source ON relationships(source_id);
CREATE INDEX idx_relationships_target ON relationships(target_id);
CREATE INDEX idx_relationships_type ON relationships(type);

-- Users Indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Audit Logs Indexes
CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_changed_at ON audit_logs(changed_at);
CREATE INDEX idx_audit_changed_by ON audit_logs(changed_by);

-- Import Jobs Indexes
CREATE INDEX idx_import_status ON import_jobs(status);
CREATE INDEX idx_import_created_by ON import_jobs(created_by);
```

### 3.2 Neo4j Graph Schema

#### 3.2.1 Node Labels

```cypher
// Configuration Item Nodes
(:ConfigurationItem {
    id: string,
    name: string,
    type: string,
    attributes: map,
    tags: string[],
    created_at: datetime,
    updated_at: datetime
})

// User Nodes
(:User {
    id: string,
    username: string,
    email: string,
    created_at: datetime
})
```

#### 3.2.2 Relationship Types

```cypher
// Standard Relationships
(:ConfigurationItem)-[:DEPENDS_ON {
    type: string,
    created_at: datetime,
    created_by: string
}]->(:ConfigurationItem)

(:ConfigurationItem)-[:HOSTS {
    type: string,
    created_at: datetime,
    created_by: string
}]->(:ConfigurationItem)

(:ConfigurationItem)-[:CONNECTS_TO {
    type: string,
    created_at: datetime,
    created_by: string
}]->(:ConfigurationItem)

// Audit Relationships
(:User)-[:MODIFIED {
    action: string,
    timestamp: datetime,
    details: map
}]->(:ConfigurationItem)
```

#### 3.2.3 Neo4j Indexes

```cypher
// Create indexes for performance
CREATE INDEX ci_id_idx IF NOT EXISTS FOR (n:ConfigurationItem) ON (n.id);
CREATE INDEX ci_type_idx IF NOT EXISTS FOR (n:ConfigurationItem) ON (n.type);
CREATE INDEX ci_name_idx IF NOT EXISTS FOR (n:ConfigurationItem) ON (n.name);

// Full-text search index
CREATE FULLTEXT INDEX ci_search_idx IF NOT EXISTS 
FOR (n:ConfigurationItem) 
ON EACH [n.name, n.type]
OPTIONS {indexConfig: {
    `fulltext.analyzer`: 'standard'
}};

// User indexes
CREATE INDEX user_id_idx IF NOT EXISTS FOR (n:User) ON (n.id);
CREATE INDEX user_username_idx IF NOT EXISTS FOR (n:User) ON (n.username);
```

## 📡 API Specification

### 4.1 Authentication Endpoints

#### POST /auth/login
```json
Request:
{
  "username": "string",
  "password": "string"
}

Response:
{
  "access_token": "string",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": {
    "id": "uuid",
    "username": "string",
    "email": "string",
    "roles": ["string"],
    "permissions": ["string"]
  }
}
```

#### POST /auth/refresh
```json
Request:
{
  "refresh_token": "string"
}

Response:
{
  "access_token": "string",
  "token_type": "Bearer",
  "expires_in": 86400
}
```

### 4.2 Configuration Item Endpoints

#### GET /cis
```json
Response:
{
  "data": [
    {
      "id": "uuid",
      "name": "string",
      "type": "string",
      "attributes": {},
      "tags": ["string"],
      "created_at": "timestamp",
      "updated_at": "timestamp",
      "created_by": "uuid",
      "updated_by": "uuid"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

#### POST /cis
```json
Request:
{
  "name": "string",
  "type": "string",
  "attributes": {},
  "tags": ["string"]
}

Response:
{
  "id": "uuid",
  "name": "string",
  "type": "string",
  "attributes": {},
  "tags": ["string"],
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "created_by": "uuid"
}
```

#### GET /cis/{id}
```json
Response:
{
  "id": "uuid",
  "name": "string",
  "type": "string",
  "attributes": {},
  "tags": ["string"],
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "created_by": "uuid",
  "updated_by": "uuid",
  "relationships": [
    {
      "id": "uuid",
      "type": "string",
      "target_id": "uuid",
      "target_name": "string"
    }
  ]
}
```

#### PUT /cis/{id}
```json
Request:
{
  "name": "string",
  "type": "string",
  "attributes": {},
  "tags": ["string"]
}

Response:
{
  "id": "uuid",
  "name": "string",
  "type": "string",
  "attributes": {},
  "tags": ["string"],
  "updated_at": "timestamp",
  "updated_by": "uuid"
}
```

#### DELETE /cis/{id}
```json
Response:
{
  "message": "Configuration item deleted successfully"
}
```

### 4.3 Graph Endpoints

#### GET /graph/nodes/{id}
```json
Response:
{
  "nodes": [
    {
      "id": "uuid",
      "name": "string",
      "type": "string",
      "attributes": {},
      "x": 0,
      "y": 0
    }
  ],
  "edges": [
    {
      "source": "uuid",
      "target": "uuid",
      "type": "string"
    }
  ]
}
```

#### GET /graph/explore
```json
Response:
{
  "nodes": [...],
  "edges": [...],
  "stats": {
    "total_nodes": 1000,
    "total_edges": 2500,
    "density": 0.005
  }
}
```

#### GET /graph/cluster
```json
Response:
{
  "clusters": [
    {
      "id": "string",
      "label": "string",
      "nodes": ["uuid"],
      "size": 10
    }
  ]
}
```

### 4.4 Relationship Endpoints

#### POST /relationships
```json
Request:
{
  "source_id": "uuid",
  "target_id": "uuid",
  "type": "string",
  "attributes": {}
}

Response:
{
  "id": "uuid",
  "source_id": "uuid",
  "target_id": "uuid",
  "type": "string",
  "created_at": "timestamp"
}
```

#### DELETE /relationships/{id}
```json
Response:
{
  "message": "Relationship deleted successfully"
}
```

### 4.5 Import/Export Endpoints

#### POST /import/csv
```json
Request:
{
  "file": "multipart/form-data",
  "mapping": {
    "name": "column_name",
    "type": "column_name",
    "attributes": {
      "field1": "column_name",
      "field2": "column_name"
    }
  },
  "options": {
    "delimiter": ",",
    "header": true,
    "skip_duplicates": false
  }
}

Response:
{
  "job_id": "uuid",
  "status": "processing",
  "message": "Import job started"
}
```

#### GET /import/jobs/{id}
```json
Response:
{
  "id": "uuid",
  "status": "completed",
  "total_records": 1000,
  "successful_records": 950,
  "failed_records": 50,
  "error_details": [...],
  "created_at": "timestamp",
  "completed_at": "timestamp"
}
```

#### GET /export/csv
```json
Response:
{
  "download_url": "string",
  "expires_at": "timestamp"
}
```

### 4.6 User Management Endpoints

#### GET /users
```json
Response:
{
  "data": [
    {
      "id": "uuid",
      "username": "string",
      "email": "string",
      "is_active": true,
      "roles": ["string"],
      "created_at": "timestamp"
    }
  ]
}
```

#### POST /users
```json
Request:
{
  "username": "string",
  "email": "string",
  "password": "string",
  "roles": ["string"]
}

Response:
{
  "id": "uuid",
  "username": "string",
  "email": "string",
  "is_active": true,
  "roles": ["string"],
  "created_at": "timestamp"
}
```

### 4.7 Audit Log Endpoints

#### GET /audit-logs
```json
Response:
{
  "data": [
    {
      "id": "uuid",
      "entity_type": "string",
      "entity_id": "uuid",
      "action": "string",
      "changed_by": "uuid",
      "changed_by_username": "string",
      "changed_at": "timestamp",
      "details": {}
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1000
  }
}
```

## 🔒 Security Design

### 5.1 Authentication Architecture

#### 5.1.1 JWT Implementation
```go
type JWTService struct {
    secretKey     string
    accessTokenTTL time.Duration
}

func (j *JWTService) GenerateToken(user *User) (string, error) {
    claims := jwt.MapClaims{
        "user_id": user.ID,
        "username": user.Username,
        "roles": user.Roles,
        "permissions": user.Permissions,
        "exp": time.Now().Add(j.accessTokenTTL).Unix(),
        "iat": time.Now().Unix(),
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.secretKey))
}

func (j *JWTService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(j.secretKey), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return &claims, nil
    }
    
    return nil, fmt.Errorf("invalid token")
}
```

#### 5.1.2 Password Hashing
```go
type PasswordService struct {
    cost int
}

func (p *PasswordService) HashPassword(password string) (string, error) {
    hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
    if err != nil {
        return "", err
    }
    return hash, nil
}

func (p *PasswordService) VerifyPassword(password, hash string) (bool, error) {
    match, err := argon2id.ComparePasswordAndHash(password, hash)
    if err != nil {
        return false, err
    }
    return match, nil
}
```

### 5.2 RBAC Implementation

#### 5.2.1 Permission Middleware
```go
func RBACMiddleware(permissions ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := GetUserFromContext(r.Context())
            if user == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            for _, requiredPermission := range permissions {
                if !user.HasPermission(requiredPermission) {
                    http.Error(w, "Forbidden", http.StatusForbidden)
                    return
                }
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

#### 5.2.2 User-Permission Model
```go
type User struct {
    ID          uuid.UUID
    Username    string
    Email       string
    IsActive    bool
    Roles       []*Role
    Permissions []string
}

func (u *User) HasPermission(permission string) bool {
    for _, p := range u.Permissions {
        if p == permission {
            return true
        }
    }
    return false
}

type Role struct {
    ID          uuid.UUID
    Name        string
    Description string
    Permissions []*Permission
}

type Permission struct {
    ID           uuid.UUID
    Name         string
    Description  string
    ResourceType string
}
```

### 5.3 Data Protection

#### 5.3.1 Input Validation
```go
type CIValidator struct {
    typeDefinitions map[string]*CITypeDefinition
}

func (v *CIValidator) ValidateCI(ci *ConfigurationItem) error {
    // Validate required fields
    if ci.Name == "" {
        return errors.New("name is required")
    }
    
    if ci.Type == "" {
        return errors.New("type is required")
    }
    
    // Validate against type schema
    if typeDef, exists := v.typeDefinitions[ci.Type]; exists {
        return v.validateAgainstSchema(ci.Attributes, typeDef.FieldSchema)
    }
    
    return nil
}
```

#### 5.3.2 SQL Injection Protection
```go
func (r *CIRepository) GetByID(ctx context.Context, id uuid.UUID) (*ConfigurationItem, error) {
    query := `
        SELECT id, name, type, attributes, tags, created_at, updated_at, created_by, updated_by
        FROM configuration_items
        WHERE id = $1
    `
    
    var ci ConfigurationItem
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &ci.ID, &ci.Name, &ci.Type, &ci.Attributes, &ci.Tags,
        &ci.CreatedAt, &ci.UpdatedAt, &ci.CreatedBy, &ci.UpdatedBy,
    )
    
    if err != nil {
        return nil, err
    }
    
    return &ci, nil
}
```

## ⚡ Performance Design

### 6.1 Database Optimization

#### 6.1.1 PostgreSQL Optimization
```sql
-- Connection pool configuration
SET max_connections = 200;
SET shared_buffers = '1GB';
SET effective_cache_size = '3GB';
SET maintenance_work_mem = '256MB';
SET checkpoint_completion_target = 0.9;
SET wal_buffers = '16MB';
SET default_statistics_target = 100;

-- Query optimization example
EXPLAIN ANALYZE
SELECT ci.id, ci.name, ci.type, ci.attributes
FROM configuration_items ci
WHERE ci.type = $1
AND ci.created_at > $2
ORDER BY ci.created_at DESC
LIMIT $3;
```

#### 6.1.2 Neo4j Optimization
```cypher
// Memory configuration
dbms.memory.heap.initial_size=2g
dbms.memory.heap.max_size=4g
dbms.memory.pagecache.size=2g

// Query optimization
PROFILE
MATCH (n:ConfigurationItem {type: $type})
MATCH (n)-[r*1..3]-(related:ConfigurationItem)
WHERE related.type IN $relatedTypes
RETURN n, r, related
LIMIT $limit;

// Index usage
EXPLAIN
MATCH (n:ConfigurationItem)
WHERE n.name STARTS WITH $prefix
RETURN n;
```

### 6.2 Caching Strategy

#### 6.2.1 Redis Cache Implementation
```go
type CacheService struct {
    client *redis.Client
    ttl    time.Duration
}

func (c *CacheService) GetCI(ctx context.Context, id uuid.UUID) (*ConfigurationItem, error) {
    key := fmt.Sprintf("ci:%s", id)
    data, err := c.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, nil // Cache miss
    }
    if err != nil {
        return nil, err
    }
    
    var ci ConfigurationItem
    if err := json.Unmarshal(data, &ci); err != nil {
        return nil, err
    }
    
    return &ci, nil
}

func (c *CacheService) SetCI(ctx context.Context, ci *ConfigurationItem) error {
    key := fmt.Sprintf("ci:%s", ci.ID)
    data, err := json.Marshal(ci)
    if err != nil {
        return err
    }
    
    return c.client.Set(ctx, key, data, c.ttl).Err()
}
```

#### 6.2.2 Graph Query Caching
```go
func (g *GraphService) GetSubgraph(ctx context.Context, nodeID string, depth int) (*Graph, error) {
    cacheKey := fmt.Sprintf("subgraph:%s:%d", nodeID, depth)
    
    // Try cache first
    if cached, err := g.cache.Get(ctx, cacheKey); err == nil {
        var graph Graph
        if err := json.Unmarshal(cached, &graph); err == nil {
            return &graph, nil
        }
    }
    
    // Cache miss - query Neo4j
    graph, err := g.querySubgraph(ctx, nodeID, depth)
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    data, _ := json.Marshal(graph)
    g.cache.Set(ctx, cacheKey, data, 5*time.Minute)
    
    return graph, nil
}
```

### 6.3 API Performance

#### 6.3.1 Response Time Optimization
```go
func (h *CIHandler) ListCIs(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page == 0 {
        page = 1
    }
    
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 20
    }
    
    // Use context with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    // Get data with pagination
    cis, total, err := h.ciRepo.List(ctx, page, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Response with pagination metadata
    response := map[string]interface{}{
        "data": cis,
        "pagination": map[string]interface{}{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

#### 6.3.2 Connection Pooling
```go
func NewDatabaseConnection(config *Config) (*sql.DB, error) {
    db, err := sql.Open("postgres", config.DatabaseURL)
    if err != nil {
        return nil, err
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Test connection
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

## 🧪 Testing Strategy

### 7.1 Unit Testing

#### 7.1.1 Go Unit Tests
```go
func TestCIRepository_Create(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    repo := NewCIRepository(db)
    
    // Test data
    ci := &ConfigurationItem{
        Name:      "Test Server",
        Type:      "server",
        Attributes: map[string]interface{}{"ip": "192.168.1.1"},
        Tags:      []string{"production", "web"},
    }
    
    // Execute
    created, err := repo.Create(context.Background(), ci)
    
    // Assert
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, created.ID)
    assert.Equal(t, ci.Name, created.Name)
    assert.Equal(t, ci.Type, created.Type)
}

func TestAuthService_Login(t *testing.T) {
    auth := NewAuthService(testDB, testPasswordService)
    
    // Create test user
    user := &User{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "hashedpassword",
    }
    
    // Test login
    token, err := auth.Login(context.Background(), "testuser", "password")
    
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

#### 7.1.2 Frontend Unit Tests
```typescript
import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import CIList from '@/components/CIList.vue'

describe('CIList', () => {
  it('renders CI items correctly', () => {
    const cis = [
      { id: '1', name: 'Server 1', type: 'server' },
      { id: '2', name: 'App 1', type: 'application' }
    ]
    
    const wrapper = mount(CIList, {
      props: { cis }
    })
    
    expect(wrapper.findAll('.ci-item')).toHaveLength(2)
    expect(wrapper.text()).toContain('Server 1')
    expect(wrapper.text()).toContain('App 1')
  })
  
  it('emits edit event when edit button is clicked', async () => {
    const wrapper = mount(CIList, {
      props: { cis: [{ id: '1', name: 'Test', type: 'server' }] }
    })
    
    await wrapper.find('.edit-btn').trigger('click')
    expect(wrapper.emitted('edit')).toBeTruthy()
    expect(wrapper.emitted('edit')[0][0]).toEqual({ id: '1', name: 'Test', type: 'server' })
  })
})
```

### 7.2 Integration Testing

#### 7.2.1 Database Integration Tests
```go
func TestCIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup test containers
    pgContainer := setupPostgresContainer(t)
    neo4jContainer := setupNeo4jContainer(t)
    
    // Initialize repositories
    pgDB := setupPostgresDB(t, pgContainer)
    neo4jDriver := setupNeo4jDriver(t, neo4jContainer)
    
    ciRepo := NewCIRepository(pgDB)
    graphRepo := NewGraphRepository(neo4jDriver)
    
    // Test CI creation
    ci := &ConfigurationItem{
        Name: "Integration Test CI",
        Type: "server",
    }
    
    created, err := ciRepo.Create(context.Background(), ci)
    assert.NoError(t, err)
    
    // Test graph sync
    err = graphRepo.SyncCI(context.Background(), created)
    assert.NoError(t, err)
    
    // Test graph query
    graph, err := graphRepo.GetSubgraph(context.Background(), created.ID.String(), 1)
    assert.NoError(t, err)
    assert.Len(t, graph.Nodes, 1)
}
```

#### 7.2.2 API Integration Tests
```go
func TestAPIIntegration(t *testing.T) {
    // Setup test server
    app := setupTestServer(t)
    
    // Test authentication
    loginResp := testRequest(t, app, "POST", "/auth/login", map[string]string{
        "username": "admin",
        "password": "password",
    })
    
    var loginResult map[string]interface{}
    json.Unmarshal(loginResp.Body.Bytes(), &loginResult)
    token := loginResult["access_token"].(string)
    
    // Test CI creation with auth
    ciData := map[string]interface{}{
        "name": "API Test CI",
        "type": "server",
        "attributes": map[string]string{"ip": "192.168.1.1"},
    }
    
    createResp := testRequestWithAuth(t, app, "POST", "/cis", ciData, token)
    assert.Equal(t, http.StatusCreated, createResp.Code)
    
    var createdCI map[string]interface{}
    json.Unmarshal(createResp.Body.Bytes(), &createdCI)
    assert.Equal(t, "API Test CI", createdCI["name"])
}
```

### 7.3 E2E Testing

#### 7.3.1 Cypress E2E Tests
```javascript
describe('CMDB Lite E2E Tests', () => {
  beforeEach(() => {
    cy.login('admin', 'password')
  })
  
  it('should create a new CI', () => {
    cy.visit('/cis')
    cy.get('[data-testid="create-ci-btn"]').click()
    
    cy.get('[data-testid="ci-name-input"]').type('Test Server')
    cy.get('[data-testid="ci-type-select"]').select('server')
    cy.get('[data-testid="ci-attributes-json"]').type('{"ip": "192.168.1.1"}')
    cy.get('[data-testid="save-ci-btn"]').click()
    
    cy.url().should('include', '/cis')
    cy.contains('Test Server').should('be.visible')
  })
  
  it('should display relationship graph', () => {
    cy.visit('/graph')
    cy.get('[data-testid="graph-container"]').should('be.visible')
    cy.get('[data-testid="graph-node"]').should('have.length.greaterThan', 0)
  })
  
  it('should import CSV file', () => {
    cy.visit('/import')
    cy.get('[data-testid="csv-file-input"]').attachFile('test-cis.csv')
    cy.get('[data-testid="import-btn"]').click()
    
    cy.get('[data-testid="import-status"]').should('contain', 'Processing')
    cy.get('[data-testid="import-status"]', { timeout: 30000 })
      .should('contain', 'Completed')
  })
})
```

## 🚀 Deployment Architecture

### 8.1 Docker Compose Configuration

#### 8.1.1 docker-compose.yml
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: cmdb
      POSTGRES_USER: cmdb_user
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cmdb_user -d cmdb"]
      interval: 30s
      timeout: 10s
      retries: 3

  neo4j:
    image: neo4j:5-community
    environment:
      NEO4J_AUTH: neo4j/${NEO4J_PASSWORD}
      NEO4J_PLUGINS: '["apoc"]'
      NEO4J_dbms_memory_heap_initial__size: 2g
      NEO4J_dbms_memory_heap_max__size: 4g
      NEO4J_dbms_memory_pagecache_size: 2g
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
    ports:
      - "7474:7474"
      - "7687:7687"
    healthcheck:
      test: ["CMD", "cypher-shell", "-u", "neo4j", "-p", "${NEO4J_PASSWORD}", "RETURN 1"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    depends_on:
      postgres:
        condition: service_healthy
      neo4j:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://cmdb_user:${POSTGRES_PASSWORD}@postgres:5432/cmdb
      NEO4J_URL: bolt://neo4j:${NEO4J_PASSWORD}@neo4j:7687
      REDIS_URL: redis://redis:6379
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  frontend:
    build:
      context: ./web
      dockerfile: Dockerfile
    depends_on:
      - api
    environment:
      VITE_API_URL: http://localhost:8080
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
  neo4j_data:
  neo4j_logs:
  redis_data:
```

### 8.2 Production Deployment

#### 8.2.1 Kubernetes Deployment
```yaml
# postgres-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: cmdb
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

#### 8.2.2 Neo4j Deployment
```yaml
# neo4j-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neo4j
spec:
  replicas: 1
  selector:
    matchLabels:
      app: neo4j
  template:
    metadata:
      labels:
        app: neo4j
    spec:
      containers:
      - name: neo4j
        image: neo4j:5-community
        env:
        - name: NEO4J_AUTH
          value: "neo4j/$(NEO4J_PASSWORD)"
        - name: NEO4J_PLUGINS
          value: '["apoc"]'
        - name: NEO4J_dbms_memory_heap_initial__size
          value: "2g"
        - name: NEO4J_dbms_memory_heap_max__size
          value: "4g"
        envFrom:
        - secretRef:
            name: neo4j-secret
        ports:
        - containerPort: 7474
        - containerPort: 7687
        volumeMounts:
        - name: neo4j-data
          mountPath: /data
        - name: neo4j-logs
          mountPath: /logs
      volumes:
      - name: neo4j-data
        persistentVolumeClaim:
          claimName: neo4j-pvc
      - name: neo4j-logs
        persistentVolumeClaim:
          claimName: neo4j-logs-pvc
```

### 8.3 CI/CD Pipeline

#### 8.3.1 GitHub Actions Workflow
```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      neo4j:
        image: neo4j:5-community
        env:
          NEO4J_AUTH: neo4j/neo4j
        options: >-
          --health-cmd cypher-shell -u neo4j -p neo4j "RETURN 1"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Set up Node.js
      uses: actions/setup-node@v3
      with:
        node-version: 18
    
    - name: Install dependencies
      run: |
        go mod download
        cd web && npm install
    
    - name: Run backend tests
      run: go test -v ./...
      env:
        DATABASE_URL: postgres://postgres:postgres@localhost:5432/cmdb_test
        NEO4J_URL: bolt://neo4j:neo4j@localhost:7687
        REDIS_URL: redis://localhost:6379
    
    - name: Run frontend tests
      run: cd web && npm run test
    
    - name: Build application
      run: |
        go build -o bin/api ./cmd/api
        cd web && npm run build

  deploy:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2
    
    - name: Login to DockerHub
      uses: docker/login-action@v2
      with:
        username: ${{ secrets.DOCKERHUB_USERNAME }}
        password: ${{ secrets.DOCKERHUB_TOKEN }}
    
    - name: Build and push API
      uses: docker/build-push-action@v4
      with:
        context: .
        file: Dockerfile.api
        push: true
        tags: ${{ secrets.DOCKERHUB_USERNAME }}/cmdb-api:latest
    
    - name: Build and push frontend
      uses: docker/build-push-action@v4
      with:
        context: ./web
        push: true
        tags: ${{ secrets.DOCKERHUB_USERNAME }}/cmdb-frontend:latest
    
    - name: Deploy to production
      run: |
        # Add deployment commands here
        echo "Deploying to production..."
```

## 📊 Monitoring & Logging

### 9.1 Logging Architecture

#### 9.1.1 Structured Logging
```go
type Logger struct {
    logger *zerolog.Logger
}

func NewLogger(service string) *Logger {
    logger := zerolog.New(os.Stdout).
        With().
        Timestamp().
        Str("service", service).
        Str("version", version).
        Logger()
    
    return &Logger{logger: &logger}
}

func (l *Logger) InfoRequest(r *http.Request, message string, fields ...map[string]interface{}) {
    event := l.logger.Info().
        Str("method", r.Method).
        Str("path", r.URL.Path).
        Str("ip", r.RemoteAddr).
        Str("user_agent", r.UserAgent())
    
    for _, field := range fields {
        for k, v := range field {
            event = event.Interface(k, v)
        }
    }
    
    event.Msg(message)
}

func (l *Logger) ErrorRequest(r *http.Request, err error, message string, fields ...map[string]interface{}) {
    event := l.logger.Error().
        Err(err).
        Str("method", r.Method).
        Str("path", r.URL.Path).
        Str("ip", r.RemoteAddr)
    
    for _, field := range fields {
        for k, v := range field {
            event = event.Interface(k, v)
        }
    }
    
    event.Msg(message)
}
```

#### 9.1.2 Request Logging Middleware
```go
func LoggingMiddleware(logger *Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap response writer to capture status code
            ww := &responseWriter{w, http.StatusOK}
            
            // Add request ID
            requestID := r.Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = uuid.New().String()
            }
            
            ctx := context.WithValue(r.Context(), "requestID", requestID)
            r = r.WithContext(ctx)
            
            // Process request
            defer func() {
                duration := time.Since(start)
                
                logger.InfoRequest(r, "Request processed", map[string]interface{}{
                    "request_id": requestID,
                    "status_code": ww.status,
                    "duration_ms": duration.Milliseconds(),
                    "duration": duration.String(),
                })
            }()
            
            next.ServeHTTP(ww, r)
        })
    }
}
```

### 9.2 Metrics Collection

#### 9.2.1 Prometheus Metrics
```go
type Metrics struct {
    requestsTotal    *prometheus.CounterVec
    requestDuration  *prometheus.HistogramVec
    activeConnections *prometheus.Gauge
    databaseQueries  *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
    return &Metrics{
        requestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "cmdb_requests_total",
                Help: "Total number of requests",
            },
            []string{"method", "endpoint", "status"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "cmdb_request_duration_seconds",
                Help:    "Request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "endpoint"},
        ),
        activeConnections: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "cmdb_active_connections",
                Help: "Number of active connections",
            },
        ),
        databaseQueries: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "cmdb_database_query_duration_seconds",
                Help:    "Database query duration in seconds",
                Buckets: []float64{0.001, 0.01, 0.1, 1, 10},
            },
            []string{"database", "operation"},
        ),
    }
}

func (m *Metrics) Register() {
    prometheus.MustRegister(m.requestsTotal)
    prometheus.MustRegister(m.requestDuration)
    prometheus.MustRegister(m.activeConnections)
    prometheus.MustRegister(m.databaseQueries)
}
```

#### 9.2.2 Metrics Middleware
```go
func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap response writer
            ww := &responseWriter{w, http.StatusOK}
            
            // Increment active connections
            metrics.activeConnections.Inc()
            defer metrics.activeConnections.Dec()
            
            // Process request
            next.ServeHTTP(ww, r)
            
            // Record metrics
            duration := time.Since(start)
            metrics.requestsTotal.WithLabelValues(
                r.Method,
                r.URL.Path,
                strconv.Itoa(ww.status),
            ).Inc()
            
            metrics.requestDuration.WithLabelValues(
                r.Method,
                r.URL.Path,
            ).Observe(duration.Seconds())
        })
    }
}
```

### 9.3 Health Checks

#### 9.3.1 Health Check Endpoints
```go
type HealthChecker struct {
    pgDB    *sql.DB
    neo4jDB neo4j.Driver
    redis   *redis.Client
}

func (h *HealthChecker) Check(ctx context.Context) HealthStatus {
    status := HealthStatus{
        Status: "healthy",
        Checks: make(map[string]CheckResult),
    }
    
    // Check PostgreSQL
    if err := h.checkPostgreSQL(ctx); err != nil {
        status.Status = "unhealthy"
        status.Checks["postgresql"] = CheckResult{
            Status:  "unhealthy",
            Message: err.Error(),
        }
    } else {
        status.Checks["postgresql"] = CheckResult{
            Status: "healthy",
        }
    }
    
    // Check Neo4j
    if err := h.checkNeo4j(ctx); err != nil {
        status.Status = "unhealthy"
        status.Checks["neo4j"] = CheckResult{
            Status:  "unhealthy",
            Message: err.Error(),
        }
    } else {
        status.Checks["neo4j"] = CheckResult{
            Status: "healthy",
        }
    }
    
    // Check Redis
    if err := h.checkRedis(ctx); err != nil {
        status.Status = "unhealthy"
        status.Checks["redis"] = CheckResult{
            Status:  "unhealthy",
            Message: err.Error(),
        }
    } else {
        status.Checks["redis"] = CheckResult{
            Status: "healthy",
        }
    }
    
    return status
}

func (h *HealthChecker) checkPostgreSQL(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    return h.pgDB.PingContext(ctx)
}

func (h *HealthChecker) checkNeo4j(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    session := h.neo4jDB.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
    defer session.Close()
    
    _, err := session.Run(ctx, "RETURN 1", nil)
    return err
}

func (h *HealthChecker) checkRedis(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    return h.redis.Ping(ctx).Err()
}
```

## 🔧 Development Workflow

### 10.1 Project Structure

```
cmdb-lite/
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── migrations/
│       ├── 001_initial_schema.sql
│       └── 002_add_rbac.sql
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes.go
│   ├── auth/
│   │   ├── jwt.go
│   │   ├── password.go
│   │   └── rbac.go
│   ├── ci/
│   │   ├── repository.go
│   │   ├── service.go
│   │   └── validator.go
│   ├── graph/
│   │   ├── repository.go
│   │   ├── service.go
│   │   └── sync.go
│   ├── database/
│   │   ├── postgres.go
│   │   └── neo4j.go
│   └── config/
│       └── config.go
├── pkg/
│   ├── logger/
│   ├── metrics/
│   └── cache/
├── web/
│   ├── src/
│   │   ├── components/
│   │   ├── views/
│   │   ├── stores/
│   │   └── utils/
│   ├── public/
│   └── package.json
├── scripts/
│   ├── build.sh
│   ├── deploy.sh
│   └── test.sh
├── docker/
│   ├── Dockerfile.api
│   └── Dockerfile.frontend
├── configs/
│   ├── docker-compose.yml
│   └── k8s/
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── docs/
│   ├── api/
│   └── deployment/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 10.2 Development Environment Setup

#### 10.2.1 Makefile
```makefile
.PHONY: build test clean run docker-build docker-run migrate-up migrate-down

# Build the application
build:
	go build -o bin/api ./cmd/api
	cd web && npm run build

# Run tests
test:
	go test -v ./...
	cd web && npm run test

# Run integration tests
test-integration:
	go test -tags=integration -v ./...

# Run E2E tests
test-e2e:
	cd web && npm run test:e2e

# Clean build artifacts
clean:
	rm -rf bin/
	cd web && rm -rf dist/

# Run the application
run:
	go run ./cmd/api

# Run with hot reload
dev:
	air -c .air.toml

# Database migrations
migrate-up:
	migrate -path ./cmd/migrations -database "postgres://cmdb_user:password@localhost:5432/cmdb?sslmode=disable" up

migrate-down:
	migrate -path ./cmd/migrations -database "postgres://cmdb_user:password@localhost:5432/cmdb?sslmode=disable" down

# Docker commands
docker-build:
	docker build -f docker/Dockerfile.api -t cmdb-api:latest .
	docker build -f docker/Dockerfile.frontend -t cmdb-frontend:latest .

docker-run:
	docker-compose up -d

docker-stop:
	docker-compose down

# Development setup
setup:
	go mod download
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/air-verse/air@latest
	cd web && npm install

# Code quality
lint:
	golangci-lint run
	cd web && npm run lint

format:
	go fmt ./...
	cd web && npm run format

# Security scan
security:
	gosec ./...
	cd web && npm audit
```

#### 10.2.2 Development Configuration
```yaml
# configs/dev.yaml
database:
  url: "postgres://cmdb_user:dev_password@localhost:5432/cmdb_dev?sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 25
  conn_max_lifetime: "5m"

neo4j:
  url: "bolt://localhost:7687"
  username: "neo4j"
  password: "dev_password"
  max_pool_size: 50

redis:
  url: "redis://localhost:6379"
  pool_size: 10

jwt:
  secret: "dev-secret-key-change-in-production"
  access_token_ttl: "24h"
  refresh_token_ttl: "168h"

server:
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"

logging:
  level: "debug"
  format: "json"

metrics:
  enabled: true
  path: "/metrics"

cors:
  allowed_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "OPTIONS"
  allowed_headers:
    - "Origin"
    - "Content-Type"
    - "Accept"
    - "Authorization"
```

## 🛠️ Technology Stack

### 11.1 Backend Technologies

#### 11.1.1 Go Frameworks and Libraries
```go
// go.mod
module github.com/yourorg/cmdb-lite

go 1.21

require (
    github.com/go-chi/chi/v5 v5.0.8
    github.com/jackc/pgx/v5 v5.4.3
    github.com/neo4j/neo4j-go-driver/v5 v5.12.0
    github.com/redis/go-redis/v9 v9.2.1
    github.com/golang-jwt/jwt/v5 v5.0.0
    github.com/argon2id/argon2id v0.0.0-20230301181536-e8183c1c4a5a
    github.com/rs/zerolog v1.30.0
    github.com/prometheus/client_golang v1.16.0
    github.com/spf13/viper v1.16.0
    github.com/go-playground/validator/v10 v10.14.0
    github.com/golang-migrate/migrate/v4 v4.15.2
    github.com/testcontainers/testcontainers-go v0.20.1
)

require (
    // Additional dependencies
)
```

#### 11.1.2 Database Drivers
```go
// PostgreSQL
import (
    "database/sql"
    "github.com/jackc/pgx/v5/stdlib"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Neo4j
import (
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Redis
import (
    "github.com/redis/go-redis/v9"
)
```

### 11.2 Frontend Technologies

#### 11.2.1 Package.json Dependencies
```json
{
  "name": "cmdb-lite-frontend",
  "version": "2.0.0",
  "dependencies": {
    "vue": "^3.3.4",
    "vue-router": "^4.2.4",
    "pinia": "^2.1.6",
    "axios": "^1.5.0",
    "d3": "^7.8.5",
    "vis-network": "^9.1.6",
    "tailwindcss": "^3.3.3",
    "@headlessui/vue": "^1.7.16",
    "@heroicons/vue": "^2.0.18",
    "vue-toastification": "^2.0.0-rc.5",
    "vue-virtual-scroller": "^2.0.0-beta.8"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^4.3.4",
    "vite": "^4.4.9",
    "vitest": "^0.34.4",
    "@vue/test-utils": "^2.4.1",
    "jsdom": "^22.1.0",
    "cypress": "^13.1.0",
    "@types/d3": "^7.4.0",
    "typescript": "^5.2.2",
    "eslint": "^8.48.0",
    "prettier": "^3.0.3"
  }
}
```

### 11.3 Infrastructure Technologies

#### 11.3.1 Docker and Kubernetes
```yaml
# Docker versions
docker: "24.0.0"
docker-compose: "2.21.0"

# Kubernetes versions
kubernetes: "1.28.0"
helm: "3.12.0"

# Monitoring
prometheus: "2.45.0"
grafana: "10.1.0"
elasticsearch: "8.9.0"
kibana: "8.9.0"

# CI/CD
github-actions: "latest"
jenkins: "2.414.1"
```

### 11.4 Development Tools

#### 11.4.1 Development Environment
```bash
# Go tools
go: 1.21.0
golangci-lint: 1.54.0
gosec: 2.17.0
air: 1.49.0

# Node.js tools
node: 18.17.0
npm: 9.6.7
yarn: 1.22.19

# Database tools
psql: 15.4
neo4j-admin: 5.10.0
redis-cli: 7.0.12

# Testing tools
testcontainers: 0.20.1
cypress: 13.1.0
postman: 10.17.0
```

---

*This document provides the technical foundation for conx development. All technical decisions should be made in consultation with this specification.*
