# Functional Specification Document (FSD)

Product Name: conx
Version: 1.0
Date: 2025-09-23
Author: Tahopetis

## 🎯 Executive Summary

conx is a lightweight, modern Configuration Management Database (CMDB) designed for small to mid-sized organizations. It provides intuitive management of Configuration Items (CIs), their relationships, and comprehensive visualization capabilities. The system features a hybrid database architecture combining PostgreSQL for structured data and Neo4j for graph relationships, ensuring optimal performance even at scale (50k+ CIs).

## 📋 Table of Contents

1. [Product Overview](#product-overview)
2. [User Personas](#user-personas)
3. [Functional Requirements](#functional-requirements)
4. [User Interface Design](#user-interface-design)
5. [Data Model](#data-model)
6. [Security Requirements](#security-requirements)
7. [Performance Requirements](#performance-requirements)
8. [Integration Requirements](#integration-requirements)
9. [Use Cases](#use-cases)
10. [Success Metrics](#success-metrics)
11. [Out of Scope](#out-of-scope)

## 🎯 Product Overview

### Vision
To provide a simple, powerful, and scalable CMDB solution that helps IT teams gain visibility into their infrastructure without the complexity of enterprise ITSM suites.

### Key Features
- **CI Management**: Create, read, update, delete configuration items with flexible attributes
- **Relationship Management**: Define and visualize dependencies between CIs
- **Graph Visualization**: Interactive graph visualization using Neo4j for optimal performance
- **CSV Import/Export**: Bulk data import and export capabilities
- **Audit Logging**: Complete audit trail of all changes
- **Role-Based Access Control (RBAC)**: Granular permissions system
- **REST API**: Comprehensive API for automation and integration
- **Search & Filter**: Powerful search and filtering capabilities

## 👥 User Personas

### 1. IT Operations Manager
- **Role**: Manages infrastructure components and teams
- **Needs**: 
  - Overview of all infrastructure components
  - Understanding of dependencies and impact analysis
  - Capacity planning and resource allocation
- **Key Features**: Dashboard, relationship graph, CI management

### 2. DevOps/SRE Engineer
- **Role**: Manages service deployments and reliability
- **Needs**:
  - Service dependency mapping
  - Change impact analysis
  - Performance monitoring integration
- **Key Features**: Relationship graph, API access, search functionality

### 3. Security/Auditor
- **Role**: Ensures compliance and security standards
- **Needs**:
  - Complete audit trails
  - Access control monitoring
  - Configuration compliance tracking
- **Key Features**: Audit logs, RBAC management, compliance reporting

### 4. System Administrator
- **Role**: Day-to-day system management
- **Needs**:
  - Easy CI creation and management
  - Bulk operations
  - Integration with existing tools
- **Key Features**: CI CRUD operations, CSV import, API access

## 🔧 Functional Requirements

### FR1: Configuration Item Management

#### FR1.1: CI Creation
- **FR1.1.1**: Users can create CIs with the following mandatory fields:
  - Name (string, unique within type)
  - Type (string, predefined or custom)
  - Attributes (JSONB, user-defined schema)
  - Tags (array of strings)
- **FR1.1.2**: System shall auto-generate unique ID for each CI
- **FR1.1.3**: System shall record creation timestamp and user
- **FR1.1.4**: Users can define custom attribute schemas per CI type

#### FR1.2: CI Retrieval
- **FR1.2.1**: Users can view list of all CIs with pagination
- **FR1.2.2**: Users can filter CIs by type, tags, and attributes
- **FR1.2.3**: Users can search CIs by name and description
- **FR1.2.4**: Users can view detailed information for a specific CI
- **FR1.2.5**: System shall display last modified timestamp and user

#### FR1.3: CI Update
- **FR1.3.1**: Users with appropriate permissions can update CI fields
- **FR1.3.2**: System shall maintain version history of all changes
- **FR1.3.3**: System shall validate updates against CI type schema
- **FR1.3.4**: System shall record update timestamp and user

#### FR1.4: CI Deletion
- **FR1.4.1**: Users with appropriate permissions can delete CIs
- **FR1.4.2**: System shall prevent deletion of CIs with active relationships
- **FR1.4.3**: System shall offer option to cascade delete relationships
- **FR1.4.4**: System shall record deletion in audit log

### FR2: Relationship Management

#### FR2.1: Relationship Creation
- **FR2.1.1**: Users can create relationships between CIs
- **FR2.1.2**: System shall support relationship types:
  - DEPENDS_ON
  - HOSTS
  - CONNECTS_TO
  - Custom user-defined types
- **FR2.1.3**: System shall prevent circular dependencies
- **FR2.1.4**: System shall record relationship creation in audit log

#### FR2.2: Relationship Visualization
- **FR2.2.1**: Users can view interactive graph of CI relationships
- **FR2.2.2**: System shall support subgraph exploration (click to expand)
- **FR2.2.3**: Users can filter graph by CI types and relationship types
- **FR2.2.4**: System shall provide different layout algorithms
- **FR2.2.5**: System shall handle large graphs (50k+ nodes) with clustering

#### FR2.3: Relationship Management
- **FR2.3.1**: Users can update relationship types and attributes
- **FR2.3.2**: Users can delete relationships with confirmation
- **FR2.3.3**: System shall show impact analysis before relationship deletion

### FR3: Data Import/Export

#### FR3.1: CSV Import
- **FR3.1.1**: Users can import CIs from CSV files
- **FR3.1.2**: System shall provide column mapping interface
- **FR3.1.3**: System shall validate data during import
- **FR3.1.4**: System shall handle duplicates (skip, update, or create)
- **FR3.1.5**: System shall provide import progress and error reporting

#### FR3.2: CSV Export
- **FR3.2.1**: Users can export CIs to CSV format
- **FR3.2.2**: Users can select which fields to include
- **FR3.2.3**: Users can filter data before export
- **FR3.2.4**: System shall export relationships where possible

### FR4: Security and Access Control

#### FR4.1: Authentication
- **FR4.1.1**: System shall support username/password authentication
- **FR4.1.2**: System shall use JWT tokens for API authentication
- **FR4.1.3**: System shall support token expiration (24 hours)
- **FR4.1.4**: System shall implement secure password hashing (Argon2id)

#### FR4.2: Authorization
- **FR4.2.1**: System shall implement RBAC with following roles:
  - Admin: Full system access
  - CI Manager: CI CRUD, relationship management
  - Viewer: Read-only access
  - Auditor: Read access + audit logs
- **FR4.2.2**: System shall support granular permissions:
  - ci:create, ci:read, ci:update, ci:delete
  - relationship:manage
  - audit_log:read
  - user:manage
  - import:csv
- **FR4.2.3**: System shall support CI type-specific permissions
- **FR4.2.4**: System shall enforce permissions at API and UI levels

### FR5: Audit and Compliance

#### FR5.1: Audit Logging
- **FR5.1.1**: System shall log all CRUD operations on CIs
- **FR5.1.2**: System shall log all relationship changes
- **FR5.1.3**: System shall log all user management actions
- **FR5.1.4**: System shall log authentication events
- **FR5.1.5**: Audit logs shall include: timestamp, user, action, entity, details

#### FR5.2: Audit Reporting
- **FR5.2.1**: Users can view and filter audit logs
- **FR5.2.2**: Users can export audit logs
- **FR5.2.3**: System shall provide audit trail for any entity
- **FR5.2.4**: System shall prevent audit log tampering

### FR6: Search and Filtering

#### FR6.1: Search Functionality
- **FR6.1.1**: Users can search CIs by name, type, and tags
- **FR6.1.2**: Users can search within CI attributes
- **FR6.1.3**: System shall provide search suggestions
- **FR6.1.4**: System shall support full-text search

#### FR6.2: Advanced Filtering
- **FR6.2.1**: Users can filter by CI type
- **FR6.2.2**: Users can filter by tags
- **FR6.2.3**: Users can filter by attribute values
- **FR6.2.4**: Users can save and load filter presets
- **FR6.2.5**: Users can combine multiple filters

### FR7: System Administration

#### FR7.1: User Management
- **FR7.1.1**: Admins can create, update, and delete users
- **FR7.1.2**: Admins can assign roles to users
- **FR7.1.3**: System shall support user activation/deactivation
- **FR7.1.4**: System shall enforce password policies

#### FR7.2: System Configuration
- **FR7.2.1**: Admins can configure CI type schemas
- **FR7.2.2**: Admins can configure relationship types
- **FR7.2.3**: Admins can configure system settings
- **FR7.2.4**: System shall provide configuration backup/restore

### FR8: API Access

#### FR8.1: REST API
- **FR8.1.1**: System shall provide comprehensive REST API
- **FR8.1.2**: API shall support all UI functionality
- **FR8.1.3**: API shall return JSON responses
- **FR8.1.4**: API shall include proper error handling
- **FR8.1.5**: API shall include rate limiting

#### FR8.2: API Documentation
- **FR8.2.1**: System shall provide interactive API documentation
- **FR8.2.2**: Documentation shall include examples
- **FR8.2.3**: Documentation shall be versioned

## 🎨 User Interface Design

### UI1: Layout and Navigation

#### UI1.1: Main Layout
- **UI1.1.1**: Application shall use responsive design
- **UI1.1.2**: Layout shall include:
  - Top navigation bar with search and user menu
  - Sidebar with main navigation
  - Main content area
  - Optional detail drawer/panel
- **UI1.1.3**: Navigation shall be consistent across all pages

#### UI1.2: Dashboard
- **UI1.2.1**: Dashboard shall show system overview
- **UI1.2.2**: Dashboard shall include:
  - Total CI count by type
  - Recent changes
  - System health status
  - Quick actions
- **UI1.2.3**: Dashboard shall be customizable

### UI2: CI Management Interface

#### UI2.1: CI List View
- **UI2.1.1**: Display CIs in table format
- **UI2.1.2**: Include columns: Name, Type, Tags, Last Modified
- **UI2.1.3**: Support sorting and pagination
- **UI2.1.4**: Include bulk actions (edit, delete, export)

#### UI2.2: CI Detail View
- **UI2.2.1**: Show comprehensive CI information
- **UI2.2.2**: Display relationships graph
- **UI2.2.3**: Show audit history
- **UI2.2.4**: Include action buttons (edit, delete, export)

#### UI2.3: CI Form
- **UI2.3.1**: Dynamic form based on CI type schema
- **UI2.3.2**: Include validation and error messages
- **UI2.3.3**: Support auto-save for forms
- **UI2.3.4**: Include help text and examples

### UI3: Graph Visualization

#### UI3.1: Graph Viewer
- **UI3.1.1**: Interactive graph visualization
- **UI3.1.2**: Support zoom and pan
- **UI3.1.3**: Show node details on hover/click
- **UI3.1.4**: Support different layout algorithms
- **UI3.1.5**: Handle large graphs with clustering

#### UI3.2: Graph Controls
- **UI3.2.1**: Filter by CI type and relationship type
- **UI3.2.2**: Control graph layout and appearance
- **UI3.2.3**: Export graph as image
- **UI3.2.4**: Show graph statistics

### UI4: Import/Export Interface

#### UI4.1: Import Wizard
- **UI4.1.1**: Step-by-step import process
- **UI4.1.2**: File upload with drag-and-drop
- **UI4.1.3**: Column mapping interface
- **UI4.1.4**: Preview and validation
- **UI4.1.5**: Progress tracking and error reporting

#### UI4.2: Export Interface
- **UI4.2.1**: Export configuration options
- **UI4.2.2**: Field selection
- **UI4.2.3**: Format selection (CSV, JSON)
- **UI4.2.4**: Download management

## 🗃️ Data Model

### DM1: Core Entities

#### DM1.1: Configuration Item
- **id**: UUID (Primary Key)
- **name**: String (Required, Unique within type)
- **type**: String (Required)
- **attributes**: JSONB (User-defined schema)
- **tags**: String Array
- **created_at**: Timestamp
- **updated_at**: Timestamp
- **created_by**: UUID (Foreign Key to Users)
- **updated_by**: UUID (Foreign Key to Users)

#### DM1.2: Relationship
- **id**: UUID (Primary Key)
- **source_id**: UUID (Foreign Key to CIs)
- **target_id**: UUID (Foreign Key to CIs)
- **type**: String (Required)
- **attributes**: JSONB (Optional)
- **created_at**: Timestamp
- **created_by**: UUID (Foreign Key to Users)

#### DM1.3: User
- **id**: UUID (Primary Key)
- **username**: String (Unique, Required)
- **email**: String (Unique, Required)
- **password_hash**: String (Required)
- **is_active**: Boolean (Default: true)
- **created_at**: Timestamp
- **updated_at**: Timestamp

#### DM1.4: Role
- **id**: UUID (Primary Key)
- **name**: String (Unique, Required)
- **description**: String
- **created_at**: Timestamp

#### DM1.5: Permission
- **id**: UUID (Primary Key)
- **name**: String (Unique, Required)
- **description**: String
- **resource_type**: String

#### DM1.6: Audit Log
- **id**: UUID (Primary Key)
- **entity_type**: String
- **entity_id**: UUID
- **action**: String
- **changed_by**: UUID (Foreign Key to Users)
- **changed_at**: Timestamp
- **details**: JSONB

### DM2: Graph Data Model (Neo4j)

#### DM2.1: Nodes
- **ConfigurationItem**: {id, name, type, attributes}
- **User**: {id, username, email}

#### DM2.2: Relationships
- **DEPENDS_ON**: {type, created_at}
- **HOSTS**: {type, created_at}
- **CONNECTS_TO**: {type, created_at}
- **MODIFIED**: {timestamp, action}

## 🔒 Security Requirements

### SR1: Authentication
- **SR1.1**: System shall use secure password hashing (Argon2id)
- **SR1.2**: System shall implement JWT with secure signing
- **SR1.3**: System shall support token expiration (24 hours)
- **SR1.4**: System shall implement secure session management

### SR2: Authorization
- **SR2.1**: System shall implement principle of least privilege
- **SR2.2**: System shall validate permissions on every request
- **SR2.3**: System shall support role hierarchy
- **SR2.4**: System shall audit all authorization decisions

### SR3: Data Protection
- **SR3.1**: System shall encrypt sensitive data at rest
- **SR3.2**: System shall use HTTPS in production
- **SR3.3**: System shall implement input validation
- **SR3.4**: System shall protect against SQL injection and XSS

### SR4: Audit and Compliance
- **SR4.1**: System shall maintain immutable audit logs
- **SR4.2**: System shall log all security events
- **SR4.3**: System shall support audit log export
- **SR4.4**: System shall provide compliance reporting

## ⚡ Performance Requirements

### PR1: Response Time
- **PR1.1**: API response time < 200ms for simple queries
- **PR1.2**: API response time < 1s for complex graph queries
- **PR1.3**: Page load time < 2s for all pages
- **PR1.4**: Graph rendering time < 3s for 1000 nodes

### PR2: Scalability
- **PR2.1**: System shall support 50k+ CIs
- **PR2.2**: System shall support 200k+ relationships
- **PR2.3**: System shall support 100+ concurrent users
- **PR2.4**: System shall handle 1000+ API requests per minute

### PR3: Database Performance
- **PR3.1**: PostgreSQL query time < 100ms for indexed queries
- **PR3.2**: Neo4j query time < 500ms for graph traversals
- **PR3.3**: Database connection pool efficiency > 90%
- **PR3.4**: Cache hit ratio > 80%

### PR4: Resource Usage
- **PR4.1**: Memory usage < 4GB for application
- **PR4.2**: Neo4j heap memory < 8GB
- **PR4.3**: PostgreSQL memory < 4GB
- **PR4.4**: CPU usage < 70% under normal load

## 🔌 Integration Requirements

### IR1: API Integration
- **IR1.1**: System shall provide RESTful API
- **IR1.2**: API shall support authentication and authorization
- **IR1.3**: API shall include rate limiting
- **IR1.4**: API shall provide comprehensive documentation

### IR2: Data Import/Export
- **IR2.1**: System shall support CSV import/export
- **IR2.2**: System shall support JSON import/export
- **IR2.3**: System shall provide data validation
- **IR2.4**: System shall handle large files efficiently

### IR3: Third-party Integration
- **IR3.1**: System shall provide webhooks for events
- **IR3.2**: System shall support API key authentication
- **IR3.3**: System shall include integration examples
- **IR3.4**: System shall maintain backward compatibility

## 📋 Use Cases

### UC1: CI Management
**User**: System Administrator
**Goal**: Create and manage configuration items

**Steps**:
1. User logs into the system
2. User navigates to CI management
3. User clicks "Create New CI"
4. User fills CI form with name, type, and attributes
5. User saves the CI
6. System validates and stores the CI
7. System records the action in audit log

### UC2: Relationship Visualization
**User**: DevOps Engineer
**Goal**: Visualize service dependencies

**Steps**:
1. User logs into the system
2. User navigates to graph visualization
3. User selects a service CI
4. System displays the service and its dependencies
5. User explores related CIs by clicking nodes
6. User filters by specific relationship types
7. User exports the graph as image

### UC3: Bulk Data Import
**User**: IT Operations Manager
**Goal**: Import existing CIs from CSV

**Steps**:
1. User prepares CSV file with CI data
2. User logs into the system
3. User navigates to import functionality
4. User uploads CSV file
5. User maps columns to CI fields
6. User previews and validates data
7. User starts import process
8. System processes import and shows progress
9. User reviews import results and errors

### UC4: Audit Compliance
**User**: Security Auditor
**Goal**: Review changes to critical CIs

**Steps**:
1. User logs into the system
2. User navigates to audit logs
3. User filters logs by date and entity type
4. User reviews changes made to critical CIs
5. User exports audit report
6. User verifies compliance with policies

## 📊 Success Metrics

### SM1: User Adoption
- **SM1.1**: 80%+ of target users create >5 CIs in first session
- **SM1.2**: 50%+ of users use graph visualization weekly
- **SM1.3**: 90%+ of users rate system as "easy to use"

### SM2: Performance
- **SM2.1**: 99% of API responses < 200ms
- **SM2.2**: 95% of page loads < 2s
- **SM2.3**: System uptime > 99.5%

### SM3: Data Quality
- **SM3.1**: 95%+ of CIs have complete information
- **SM3.2**: 90%+ of relationships are accurate
- **SM3.3**: Duplicate CI rate < 5%

### SM4: Operational Efficiency
- **SM4.1**: Average CI creation time < 2 minutes
- **SM4.2**: Impact analysis time reduced by 70%
- **SM4.3**: Audit preparation time reduced by 80%

## 🚫 Out of Scope

### OS1: ITIL Process Management
- Incident management
- Change management
- Problem management
- Service level management

### OS2: Automated Discovery
- Network scanning
- Agent-based discovery
- Automatic CI synchronization
- Third-party CMDB integration

### OS3: Advanced Workflow
- Approval workflows
- Multi-step processes
- Complex business rules
- Custom scripting

### OS4: Enterprise Features
- Multi-tenancy
- Advanced reporting
- Machine learning analytics
- Predictive maintenance

---

*This document is subject to change based on project requirements and stakeholder feedback.*
