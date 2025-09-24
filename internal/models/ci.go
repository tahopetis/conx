package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CI represents a Configuration Item with FSD-compliant flexible attributes
type CI struct {
	// Core Identification
	ID           uuid.UUID  `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Type         string     `json:"type" db:"type"`
	Description  string     `json:"description" db:"description"`
	
	// Status and Classification
	Status         string     `json:"status" db:"status"`
	Criticality    string     `json:"criticality" db:"criticality"`
	
	// Ownership and Location
	Owner          string     `json:"owner" db:"owner"`
	Location       string     `json:"location" db:"location"`
	
	// FSD-Compliant Flexible Attributes
	Attributes     json.RawMessage `json:"attributes" db:"attributes"`  // JSONB for user-defined schema
	Tags           []string        `json:"tags" db:"tags"`              // String array for flexible tagging
	
	// Date Tracking
	InstallDate    *time.Time `json:"install_date" db:"install_date"`
	WarrantyExpiry *time.Time `json:"warranty_expiry" db:"warranty_expiry"`
	LastUpdated    *time.Time `json:"last_updated" db:"last_updated"`
	LastScanned    *time.Time `json:"last_scanned" db:"last_scanned"`
	
	// Lifecycle Management
	IsActive       bool       `json:"is_active" db:"is_active"`
	IsDeleted      bool       `json:"is_deleted" db:"is_deleted"`
	
	// Audit Trail
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy      uuid.UUID  `json:"created_by" db:"created_by"`
	UpdatedBy      uuid.UUID  `json:"updated_by" db:"updated_by"`
}

// CITypeSchema represents a user-defined CI type schema
type CITypeSchema struct {
	ID          uuid.UUID             `json:"id" db:"id"`
	Name        string               `json:"name" db:"name"`
	Description string               `json:"description" db:"description"`
	Attributes  []CITypeAttribute    `json:"attributes" db:"attributes"`
	IsActive    bool                 `json:"is_active" db:"is_active"`
	CreatedAt   time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID            `json:"created_by" db:"created_by"`
	UpdatedBy   uuid.UUID            `json:"updated_by" db:"updated_by"`
}

// CITypeAttribute represents an attribute definition in a CI type schema
type CITypeAttribute struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`        // string, number, boolean, date, array, object
	Required    bool                   `json:"required"`
	Description string                 `json:"description"`
	Default     interface{}            `json:"default,omitempty"`
	Validation  map[string]interface{} `json:"validation,omitempty"`
}

// CIRelationship represents a relationship between CIs with FSD-compliant flexible attributes
type CIRelationship struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	SourceCIID   uuid.UUID      `json:"source_ci_id" db:"source_ci_id"`
	TargetCIID   uuid.UUID      `json:"target_ci_id" db:"target_ci_id"`
	Type         string         `json:"type" db:"type"`
	// FSD-Compliant Flexible Attributes
	Attributes   json.RawMessage `json:"attributes" db:"attributes"`  // JSONB for user-defined relationship attributes
	Description  string         `json:"description" db:"description"`
	IsActive     bool           `json:"is_active" db:"is_active"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	CreatedBy    uuid.UUID      `json:"created_by" db:"created_by"`
	UpdatedBy    uuid.UUID      `json:"updated_by" db:"updated_by"`
}

// RelationshipTypeSchema represents a user-defined relationship type schema
type RelationshipTypeSchema struct {
	ID          uuid.UUID             `json:"id" db:"id"`
	Name        string               `json:"name" db:"name"`
	Description string               `json:"description" db:"description"`
	Attributes  []CITypeAttribute    `json:"attributes" db:"attributes"`
	IsActive    bool                 `json:"is_active" db:"is_active"`
	CreatedAt   time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID            `json:"created_by" db:"created_by"`
	UpdatedBy   uuid.UUID            `json:"updated_by" db:"updated_by"`
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
	Rule    string      `json:"rule,omitempty"`
}

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	IsValid   bool            `json:"is_valid"`
	Errors    []ValidationError `json:"errors,omitempty"`
	Warnings  []ValidationError `json:"warnings,omitempty"`
}

// Request/Response structures

// CreateCIRequest represents a request to create a CI
type CreateCIRequest struct {
	Name         string                 `json:"name" validate:"required"`
	Type         string                 `json:"type" validate:"required"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	Criticality  string                 `json:"criticality"`
	Owner        string                 `json:"owner"`
	Location     string                 `json:"location"`
	Attributes   json.RawMessage        `json:"attributes"`
	Tags         []string               `json:"tags"`
	InstallDate  *time.Time            `json:"install_date"`
	WarrantyExpiry *time.Time          `json:"warranty_expiry"`
}

// UpdateCIRequest represents a request to update a CI
type UpdateCIRequest struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	Criticality  string                 `json:"criticality"`
	Owner        string                 `json:"owner"`
	Location     string                 `json:"location"`
	Attributes   json.RawMessage        `json:"attributes"`
	Tags         []string               `json:"tags"`
	InstallDate  *time.Time            `json:"install_date"`
	WarrantyExpiry *time.Time          `json:"warranty_expiry"`
	LastUpdated  *time.Time            `json:"last_updated"`
	LastScanned  *time.Time            `json:"last_scanned"`
	IsActive     *bool                  `json:"is_active"`
}

// ListCIsRequest represents a request to list CIs
type ListCIsRequest struct {
	Page         int      `json:"page" validate:"min=1"`
	PageSize     int      `json:"page_size" validate:"min=1,max=100"`
	Search       string   `json:"search"`
	Type         string   `json:"type"`
	Status       string   `json:"status"`
	Criticality  string   `json:"criticality"`
	Owner        string   `json:"owner"`
	Location     string   `json:"location"`
	Tags         []string `json:"tags"`
	SortBy       string   `json:"sort_by"`
	SortOrder    string   `json:"sort_order" validate:"oneof=asc desc"`
}

// ListCIsResponse represents a response for listing CIs
type ListCIsResponse struct {
	CIs         []CI       `json:"cis"`
	TotalCount  int64      `json:"total_count"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
	TotalPages  int        `json:"total_pages"`
}

// CreateRelationshipRequest represents a request to create a relationship
type CreateRelationshipRequest struct {
	SourceCIID   uuid.UUID      `json:"source_ci_id" validate:"required"`
	TargetCIID   uuid.UUID      `json:"target_ci_id" validate:"required"`
	Type         string         `json:"type" validate:"required"`
	Attributes   json.RawMessage `json:"attributes"`
	Description  string         `json:"description"`
}

// UpdateRelationshipRequest represents a request to update a relationship
type UpdateRelationshipRequest struct {
	Type         string         `json:"type"`
	Attributes   json.RawMessage `json:"attributes"`
	Description  string         `json:"description"`
	IsActive     *bool           `json:"is_active"`
}

// CreateCITypeSchemaRequest represents a request to create a CI type schema
type CreateCITypeSchemaRequest struct {
	Name        string               `json:"name" validate:"required"`
	Description string               `json:"description"`
	Attributes  []CITypeAttribute    `json:"attributes" validate:"required"`
}

// UpdateCITypeSchemaRequest represents a request to update a CI type schema
type UpdateCITypeSchemaRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Attributes  []CITypeAttribute    `json:"attributes"`
	IsActive    *bool                 `json:"is_active"`
}

// CreateRelationshipTypeSchemaRequest represents a request to create a relationship type schema
type CreateRelationshipTypeSchemaRequest struct {
	Name        string               `json:"name" validate:"required"`
	Description string               `json:"description"`
	Attributes  []CITypeAttribute    `json:"attributes" validate:"required"`
}

// UpdateRelationshipTypeSchemaRequest represents a request to update a relationship type schema
type UpdateRelationshipTypeSchemaRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Attributes  []CITypeAttribute    `json:"attributes"`
	IsActive    *bool                 `json:"is_active"`
}

// ValidateCIAgainstSchemaRequest represents a request to validate CI data against schema
type ValidateCIAgainstSchemaRequest struct {
	CI        CI           `json:"ci"`
	Schema    CITypeSchema `json:"schema"`
}

// ValidateRelationshipAgainstSchemaRequest represents a request to validate relationship data against schema
type ValidateRelationshipAgainstSchemaRequest struct {
	Relationship CIRelationship        `json:"relationship"`
	Schema       RelationshipTypeSchema `json:"schema"`
}

// Constants for default values
const (
	// CI Status values
	CIStatusActive      = "active"
	CIStatusInactive    = "inactive"
	CIStatusMaintenance = "maintenance"
	CIStatusRetired     = "retired"
	CIStatusFixRequired = "fix_required"

	// CI Criticality values
	CICriticalityLow    = "low"
	CICriticalityMedium = "medium"
	CICriticalityHigh   = "high"
	CICriticalityCritical = "critical"

	// Attribute types
	AttributeTypeString  = "string"
	AttributeTypeNumber  = "number"
	AttributeTypeBoolean = "boolean"
	AttributeTypeDate    = "date"
	AttributeTypeArray   = "array"
	AttributeTypeObject  = "object"
)

// Helper functions

// GetDefaultCISchemas returns predefined CI type schemas
func GetDefaultCISchemas() []CITypeSchema {
	return []CITypeSchema{
		{
			Name:        "server",
			Description: "Physical or virtual server",
			Attributes: []CITypeAttribute{
				{Name: "ip_address", Type: AttributeTypeString, Required: true, Description: "Primary IP address", Validation: map[string]interface{}{"format": "ipv4"}},
				{Name: "cpu_cores", Type: AttributeTypeNumber, Required: true, Description: "Number of CPU cores", Validation: map[string]interface{}{"min": 1}},
				{Name: "memory_gb", Type: AttributeTypeNumber, Required: true, Description: "Memory in GB", Validation: map[string]interface{}{"min": 1}},
				{Name: "os_version", Type: AttributeTypeString, Required: false, Description: "Operating system version"},
				{Name: "hostname", Type: AttributeTypeString, Required: false, Description: "Server hostname"},
				{Name: "environment", Type: AttributeTypeString, Required: false, Description: "Environment", Validation: map[string]interface{}{"enum": []string{"development", "staging", "production"}}},
			},
		},
		{
			Name:        "application",
			Description: "Software application",
			Attributes: []CITypeAttribute{
				{Name: "version", Type: AttributeTypeString, Required: true, Description: "Application version"},
				{Name: "framework", Type: AttributeTypeString, Required: false, Description: "Development framework"},
				{Name: "dependencies", Type: AttributeTypeArray, Required: false, Description: "Application dependencies"},
				{Name: "environment", Type: AttributeTypeString, Required: true, Description: "Runtime environment", Validation: map[string]interface{}{"enum": []string{"development", "staging", "production"}}},
				{Name: "language", Type: AttributeTypeString, Required: false, Description: "Programming language"},
				{Name: "port", Type: AttributeTypeNumber, Required: false, Description: "Application port", Validation: map[string]interface{}{"min": 1, "max": 65535}},
			},
		},
		{
			Name:        "database",
			Description: "Database system",
			Attributes: []CITypeAttribute{
				{Name: "engine", Type: AttributeTypeString, Required: true, Description: "Database engine", Validation: map[string]interface{}{"enum": []string{"postgresql", "mysql", "mongodb", "redis", "oracle"}}},
				{Name: "version", Type: AttributeTypeString, Required: true, Description: "Database version"},
				{Name: "size_gb", Type: AttributeTypeNumber, Required: false, Description: "Database size in GB"},
				{Name: "tables_count", Type: AttributeTypeNumber, Required: false, Description: "Number of tables"},
				{Name: "connection_string", Type: AttributeTypeString, Required: false, Description: "Database connection string"},
			},
		},
		{
			Name:        "network_device",
			Description: "Network device",
			Attributes: []CITypeAttribute{
				{Name: "device_type", Type: AttributeTypeString, Required: true, Description: "Device type", Validation: map[string]interface{}{"enum": []string{"router", "switch", "firewall", "access_point"}}},
				{Name: "management_ip", Type: AttributeTypeString, Required: true, Description: "Management IP address", Validation: map[string]interface{}{"format": "ipv4"}},
				{Name: "ports_count", Type: AttributeTypeNumber, Required: false, Description: "Number of ports"},
				{Name: "vlan", Type: AttributeTypeNumber, Required: false, Description: "VLAN ID"},
				{Name: "model", Type: AttributeTypeString, Required: false, Description: "Device model"},
			},
		},
	}
}

// GetDefaultRelationshipSchemas returns predefined relationship type schemas
func GetDefaultRelationshipSchemas() []RelationshipTypeSchema {
	return []RelationshipTypeSchema{
		{
			Name:        "depends_on",
			Description: "Dependency relationship",
			Attributes: []CITypeAttribute{
				{Name: "dependency_type", Type: AttributeTypeString, Required: false, Description: "Type of dependency"},
				{Name: "is_critical", Type: AttributeTypeBoolean, Required: false, Description: "Whether this is a critical dependency"},
			},
		},
		{
			Name:        "hosts",
			Description: "Hosting relationship",
			Attributes: []CITypeAttribute{
				{Name: "virtualization_type", Type: AttributeTypeString, Required: false, Description: "Virtualization type"},
				{Name: "resource_allocation", Type: AttributeTypeObject, Required: false, Description: "Resource allocation details"},
			},
		},
		{
			Name:        "connected_to",
			Description: "Network connection",
			Attributes: []CITypeAttribute{
				{Name: "connection_type", Type: AttributeTypeString, Required: false, Description: "Connection type"},
				{Name: "bandwidth", Type: AttributeTypeString, Required: false, Description: "Connection bandwidth"},
				{Name: "port", Type: AttributeTypeString, Required: false, Description: "Connected port"},
			},
		},
		{
			Name:        "runs_on",
			Description: "Application runtime relationship",
			Attributes: []CITypeAttribute{
				{Name: "runtime_environment", Type: AttributeTypeString, Required: false, Description: "Runtime environment"},
				{Name: "configuration", Type: AttributeTypeObject, Required: false, Description: "Runtime configuration"},
			},
		},
	}
}
