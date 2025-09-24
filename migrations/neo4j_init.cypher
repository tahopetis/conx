-- Neo4j initialization script for conx CMDB
-- This script sets up the graph database schema and indexes

-- Create constraints for uniqueness and data integrity
CREATE CONSTRAINT ci_id_unique IF NOT EXISTS FOR (n:ConfigurationItem) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT user_id_unique IF NOT EXISTS FOR (n:User) REQUIRE n.id IS UNIQUE;

-- Create indexes for performance
CREATE INDEX ci_type_idx IF NOT EXISTS FOR (n:ConfigurationItem) ON (n.type);
CREATE INDEX ci_name_idx IF NOT EXISTS FOR (n:ConfigurationItem) ON (n.name);
CREATE INDEX ci_created_at_idx IF NOT EXISTS FOR (n:ConfigurationItem) ON (n.created_at);
CREATE INDEX user_username_idx IF NOT EXISTS FOR (n:User) ON (n.username);
CREATE INDEX user_email_idx IF NOT EXISTS FOR (n:User) ON (n.email);

-- Create full-text search index for Configuration Items
CREATE FULLTEXT INDEX ci_search_idx IF NOT EXISTS 
FOR (n:ConfigurationItem) 
ON EACH [n.name, n.type]
OPTIONS {indexConfig: {
    `fulltext.analyzer`: 'standard'
}};

-- Create full-text search index for Users
CREATE FULLTEXT INDEX user_search_idx IF NOT EXISTS 
FOR (n:User) 
ON EACH [n.username, n.email]
OPTIONS {indexConfig: {
    `fulltext.analyzer`: 'standard'
}};

-- Create relationship type indexes for performance
CREATE INDEX rel_type_idx IF NOT EXISTS FOR ()-[r]->() ON (type(r));

-- Create a unique constraint for relationship types to ensure data consistency
-- This is a virtual constraint since Neo4j doesn't support direct relationship type constraints

-- Create procedures for common graph operations
-- Procedure to create or update a Configuration Item node
CREATE OR REPLACE PROCEDURE createOrUpdateCI(
    ciId STRING,
    ciName STRING,
    ciType STRING,
    ciAttributes MAP,
    ciTags LIST<STRING>,
    createdAt DATETIME,
    updatedAt DATETIME,
    createdBy STRING
)
YIELD node
MERGE (n:ConfigurationItem {id: ciId})
SET n.name = ciName,
    n.type = ciType,
    n.attributes = ciAttributes,
    n.tags = ciTags,
    n.created_at = createdAt,
    n.updated_at = updatedAt,
    n.created_by = createdBy
RETURN n;

-- Procedure to create or update a User node
CREATE OR REPLACE PROCEDURE createOrUpdateUser(
    userId STRING,
    username STRING,
    email STRING,
    createdAt DATETIME
)
YIELD node
MERGE (n:User {id: userId})
SET n.username = username,
    n.email = email,
    n.created_at = createdAt
RETURN n;

-- Procedure to create a relationship between two CIs
CREATE OR REPLACE PROCEDURE createRelationship(
    sourceId STRING,
    targetId STRING,
    relType STRING,
    relAttributes MAP,
    createdAt DATETIME,
    createdBy STRING
)
YIELD relationship
MATCH (source:ConfigurationItem {id: sourceId})
MATCH (target:ConfigurationItem {id: targetId})
WHERE source.id <> target.id  // Prevent self-relationships
CREATE (source)-[r:RELATIONSHIP]->(target)
SET r.type = relType,
    r.attributes = relAttributes,
    r.created_at = createdAt,
    r.created_by = createdBy
RETURN r;

-- Procedure to get a subgraph starting from a specific node
CREATE OR REPLACE PROCEDURE getSubgraph(
    nodeId STRING,
    maxDepth INTEGER,
    maxNodes INTEGER
)
YIELD nodes, relationships
MATCH (start:ConfigurationItem {id: nodeId})
CALL apoc.path.subgraphAll(start, {
    maxLevel: maxDepth,
    maxNodes: maxNodes,
    relationshipFilter: "RELATIONSHIP>"
}) YIELD nodes, relationships
RETURN nodes, relationships;

-- Procedure to search Configuration Items
CREATE OR REPLACE PROCEDURE searchCIs(
    searchTerm STRING,
    limit INTEGER
)
YIELD nodes
CALL db.index.fulltext.queryNodes("ci_search_idx", searchTerm)
YIELD node, score
WHERE node:ConfigurationItem
RETURN node
LIMIT limit;

-- Procedure to get neighbors of a node with filtering
CREATE OR REPLACE PROCEDURE getNeighbors(
    nodeId STRING,
    relTypes LIST<STRING],
    limit INTEGER
)
YIELD nodes, relationships
MATCH (start:ConfigurationItem {id: nodeId})-[r]->(neighbor:ConfigurationItem)
WHERE (size(relTypes) = 0 OR r.type IN relTypes)
RETURN collect(neighbor) as nodes, collect(r) as relationships
LIMIT limit;

-- Procedure to get graph statistics
CREATE OR REPLACE PROCEDURE getGraphStatistics()
YIELD stats
MATCH (n:ConfigurationItem)
WITH count(n) as totalCis
MATCH ()-[r]->()
WITH totalCis, count(r) as totalRels
MATCH (n:ConfigurationItem)
WITH totalCis, totalRels, count(DISTINCT n.type) as uniqueTypes
RETURN {
    total_nodes: totalCis,
    total_relationships: totalRels,
    unique_types: uniqueTypes,
    density: toFloat(totalRels) / (toFloat(totalCis) * (toFloat(totalCis) - 1))
} as stats;

-- Procedure to delete a CI and all its relationships
CREATE OR REPLACE PROCEDURE deleteCI(ciId STRING)
YIELD deletedNodes, deletedRelationships
MATCH (n:ConfigurationItem {id: ciId})
DETACH DELETE n
RETURN count(n) as deletedNodes, 0 as deletedRelationships;

-- Procedure to create a modified relationship for audit trail
CREATE OR REPLACE PROCEDURE createModifiedRelationship(
    userId STRING,
    entityId STRING,
    action STRING,
    timestamp DATETIME,
    details MAP
)
YIELD relationship
MATCH (user:User {id: userId})
MATCH (entity:ConfigurationItem {id: entityId})
CREATE (user)-[r:MODIFIED]->(entity)
SET r.action = action,
    r.timestamp = timestamp,
    r.details = details
RETURN r;

-- Create a sample data for testing (optional - can be commented out in production)
-- Create sample users
CALL createOrUpdateUser(
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'admin@conx.local',
    datetime()
);

CALL createOrUpdateUser(
    '00000000-0000-0000-0000-000000000002',
    'ci_manager',
    'manager@conx.local',
    datetime()
);

-- Create sample CIs
CALL createOrUpdateCI(
    '00000000-0000-0000-0000-000000000100',
    'Web Server 1',
    'server',
    {ip_address: '192.168.1.10', hostname: 'webserver1', os: 'Ubuntu 20.04', cpu_cores: 4, memory_gb: 16},
    ['production', 'web'],
    datetime(),
    datetime(),
    '00000000-0000-0000-0000-000000000001'
);

CALL createOrUpdateCI(
    '00000000-0000-0000-0000-000000000101',
    'Database Server',
    'server',
    {ip_address: '192.168.1.20', hostname: 'dbserver', os: 'Ubuntu 20.04', cpu_cores: 8, memory_gb: 32},
    ['production', 'database'],
    datetime(),
    datetime(),
    '00000000-0000-0000-0000-000000000001'
);

CALL createOrUpdateCI(
    '00000000-0000-0000-0000-000000000102',
    'Web Application',
    'application',
    {version: '1.0.0', language: 'Java', framework: 'Spring Boot', port: 8080},
    ['production', 'web'],
    datetime(),
    datetime(),
    '00000000-0000-0000-0000-000000000001'
);

-- Create sample relationships
CALL createRelationship(
    '00000000-0000-0000-0000-000000000102',
    '00000000-0000-0000-0000-000000000100',
    'HOSTS',
    {},
    datetime(),
    '00000000-0000-0000-0000-000000000001'
);

CALL createRelationship(
    '00000000-0000-0000-0000-000000000102',
    '00000000-0000-0000-0000-000000000101',
    'DEPENDS_ON',
    {},
    datetime(),
    '00000000-0000-0000-0000-000000000001'
);

-- Show initialization summary
CALL getGraphStatistics() YIELD stats
RETURN 'Neo4j initialization completed successfully' as message, stats;
