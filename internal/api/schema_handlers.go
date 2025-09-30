package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"connect/internal/models"
	"connect/internal/repositories"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// SchemaHandler handles schema management endpoints
type SchemaHandler struct {
	ciRepo *repositories.CIRepository
}

// NewSchemaHandler creates a new SchemaHandler
func NewSchemaHandler(ciRepo *repositories.CIRepository) *SchemaHandler {
	return &SchemaHandler{ciRepo: ciRepo}
}

// RegisterRoutes registers schema management routes
func (h *SchemaHandler) RegisterRoutes(router *mux.Router) {
	// CI Type Schema routes
	router.HandleFunc("/api/v1/schemas/ci-types", h.authMiddleware(h.handleListCITypeSchemas)).Methods("GET")
	router.HandleFunc("/api/v1/schemas/ci-types", h.authMiddleware(h.handleCreateCITypeSchema)).Methods("POST")
	router.HandleFunc("/api/v1/schemas/ci-types/{id}", h.authMiddleware(h.handleGetCITypeSchema)).Methods("GET")
	router.HandleFunc("/api/v1/schemas/ci-types/{id}", h.authMiddleware(h.handleUpdateCITypeSchema)).Methods("PUT")
	router.HandleFunc("/api/v1/schemas/ci-types/{id}", h.authMiddleware(h.handleDeleteCITypeSchema)).Methods("DELETE")

	// Relationship Type Schema routes
	router.HandleFunc("/api/v1/schemas/relationship-types", h.authMiddleware(h.handleListRelationshipTypeSchemas)).Methods("GET")
	router.HandleFunc("/api/v1/schemas/relationship-types", h.authMiddleware(h.handleCreateRelationshipTypeSchema)).Methods("POST")
	router.HandleFunc("/api/v1/schemas/relationship-types/{id}", h.authMiddleware(h.handleGetRelationshipTypeSchema)).Methods("GET")
	router.HandleFunc("/api/v1/schemas/relationship-types/{id}", h.authMiddleware(h.handleUpdateRelationshipTypeSchema)).Methods("PUT")
	router.HandleFunc("/api/v1/schemas/relationship-types/{id}", h.authMiddleware(h.handleDeleteRelationshipTypeSchema)).Methods("DELETE")

	// Schema validation routes
	router.HandleFunc("/api/v1/schemas/validate/ci", h.authMiddleware(h.handleValidateCIAgainstSchema)).Methods("POST")
	router.HandleFunc("/api/v1/schemas/validate/relationship", h.authMiddleware(h.handleValidateRelationshipAgainstSchema)).Methods("POST")

	// Default schema templates routes
	router.HandleFunc("/api/v1/schemas/templates/ci", h.authMiddleware(h.handleGetDefaultCISchemas)).Methods("GET")
	router.HandleFunc("/api/v1/schemas/templates/relationship", h.authMiddleware(h.handleGetDefaultRelationshipSchemas)).Methods("GET")
	router.HandleFunc("/api/v1/schemas/templates/ci/{name}", h.authMiddleware(h.handleCreateSchemaFromTemplate)).Methods("POST")
}

// CI Type Schema Handlers

// handleListCITypeSchemas handles listing CI type schemas
func (h *SchemaHandler) handleListCITypeSchemas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	schemas, totalCount, err := h.ciRepo.ListCITypeSchemas(ctx, page, pageSize)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list CI type schemas", err)
		return
	}

	response := map[string]interface{}{
		"schemas":     schemas,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// handleCreateCITypeSchema handles creating a new CI type schema
func (h *SchemaHandler) handleCreateCITypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)

	var req models.CreateCITypeSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Validate schema definition
	validator := models.NewSchemaValidator()
	schemaDef := models.CITypeSchema{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Attributes:  req.Attributes,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	validationResult := validator.ValidateSchemaDefinition(schemaDef)
	if !validationResult.IsValid {
		h.respondWithError(w, http.StatusBadRequest, "Schema definition validation failed", nil)
		h.respondWithJSON(w, http.StatusBadRequest, validationResult)
		return
	}

	// Create schema
	schema, err := h.ciRepo.CreateCITypeSchema(ctx, &schemaDef)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create CI type schema", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, schema)
}

// handleGetCITypeSchema handles retrieving a CI type schema by ID
func (h *SchemaHandler) handleGetCITypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	schemaID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid schema ID", err)
		return
	}

	schema, err := h.ciRepo.GetCITypeSchema(ctx, schemaID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "CI type schema not found", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, schema)
}

// handleUpdateCITypeSchema handles updating a CI type schema
func (h *SchemaHandler) handleUpdateCITypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)
	vars := mux.Vars(r)

	schemaID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid schema ID", err)
		return
	}

	var req models.UpdateCITypeSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get existing schema
	existingSchema, err := h.ciRepo.GetCITypeSchema(ctx, schemaID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "CI type schema not found", err)
		return
	}

	// Update schema fields
	if req.Name != "" {
		existingSchema.Name = req.Name
	}
	if req.Description != "" {
		existingSchema.Description = req.Description
	}
	if len(req.Attributes) > 0 {
		existingSchema.Attributes = req.Attributes
	}
	if req.IsActive != nil {
		existingSchema.IsActive = *req.IsActive
	}
	existingSchema.UpdatedBy = userID

	// Validate updated schema definition
	validator := models.NewSchemaValidator()
	validationResult := validator.ValidateSchemaDefinition(*existingSchema)
	if !validationResult.IsValid {
		h.respondWithError(w, http.StatusBadRequest, "Schema definition validation failed", nil)
		h.respondWithJSON(w, http.StatusBadRequest, validationResult)
		return
	}

	// Update schema
	updatedSchema, err := h.ciRepo.UpdateCITypeSchema(ctx, existingSchema)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update CI type schema", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, updatedSchema)
}

// handleDeleteCITypeSchema handles deleting a CI type schema
func (h *SchemaHandler) handleDeleteCITypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	schemaID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid schema ID", err)
		return
	}

	// Check if schema exists
	_, err = h.ciRepo.GetCITypeSchema(ctx, schemaID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "CI type schema not found", err)
		return
	}

	// Delete schema
	if err := h.ciRepo.DeleteCITypeSchema(ctx, schemaID); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete CI type schema", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "CI type schema deleted successfully"})
}

// Relationship Type Schema Handlers

// handleListRelationshipTypeSchemas handles listing relationship type schemas
func (h *SchemaHandler) handleListRelationshipTypeSchemas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	schemas, totalCount, err := h.ciRepo.ListRelationshipTypeSchemas(ctx, page, pageSize)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list relationship type schemas", err)
		return
	}

	response := map[string]interface{}{
		"schemas":     schemas,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// handleCreateRelationshipTypeSchema handles creating a new relationship type schema
func (h *SchemaHandler) handleCreateRelationshipTypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)

	var req models.CreateRelationshipTypeSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Create schema
	schemaDef := models.RelationshipTypeSchema{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Attributes:  req.Attributes,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	// Validate schema definition
	validator := models.NewSchemaValidator()
	validationResult := validator.ValidateSchemaDefinition(models.CITypeSchema{
		Name:       schemaDef.Name,
		Attributes: schemaDef.Attributes,
	})
	if !validationResult.IsValid {
		h.respondWithError(w, http.StatusBadRequest, "Schema definition validation failed", nil)
		h.respondWithJSON(w, http.StatusBadRequest, validationResult)
		return
	}

	// Create schema
	schema, err := h.ciRepo.CreateRelationshipTypeSchema(ctx, &schemaDef)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create relationship type schema", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, schema)
}

// handleGetRelationshipTypeSchema handles retrieving a relationship type schema by ID
func (h *SchemaHandler) handleGetRelationshipTypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	schemaID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid schema ID", err)
		return
	}

	schema, err := h.ciRepo.GetRelationshipTypeSchema(ctx, schemaID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Relationship type schema not found", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, schema)
}

// handleUpdateRelationshipTypeSchema handles updating a relationship type schema
func (h *SchemaHandler) handleUpdateRelationshipTypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)
	vars := mux.Vars(r)

	schemaID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid schema ID", err)
		return
	}

	var req models.UpdateRelationshipTypeSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get existing schema
	existingSchema, err := h.ciRepo.GetRelationshipTypeSchema(ctx, schemaID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Relationship type schema not found", err)
		return
	}

	// Update schema fields
	if req.Name != "" {
		existingSchema.Name = req.Name
	}
	if req.Description != "" {
		existingSchema.Description = req.Description
	}
	if len(req.Attributes) > 0 {
		existingSchema.Attributes = req.Attributes
	}
	if req.IsActive != nil {
		existingSchema.IsActive = *req.IsActive
	}
	existingSchema.UpdatedBy = userID

	// Validate updated schema definition
	validator := models.NewSchemaValidator()
	validationResult := validator.ValidateSchemaDefinition(models.CITypeSchema{
		Name:       existingSchema.Name,
		Attributes: existingSchema.Attributes,
	})
	if !validationResult.IsValid {
		h.respondWithError(w, http.StatusBadRequest, "Schema definition validation failed", nil)
		h.respondWithJSON(w, http.StatusBadRequest, validationResult)
		return
	}

	// Update schema
	updatedSchema, err := h.ciRepo.UpdateRelationshipTypeSchema(ctx, existingSchema)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update relationship type schema", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, updatedSchema)
}

// handleDeleteRelationshipTypeSchema handles deleting a relationship type schema
func (h *SchemaHandler) handleDeleteRelationshipTypeSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	schemaID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid schema ID", err)
		return
	}

	// Check if schema exists
	_, err = h.ciRepo.GetRelationshipTypeSchema(ctx, schemaID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "Relationship type schema not found", err)
		return
	}

	// Delete schema
	if err := h.ciRepo.DeleteRelationshipTypeSchema(ctx, schemaID); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete relationship type schema", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Relationship type schema deleted successfully"})
}

// Schema Validation Handlers

// handleValidateCIAgainstSchema handles validating CI data against a schema
func (h *SchemaHandler) handleValidateCIAgainstSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.ValidateCIAgainstSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate CI against schema
	result, err := h.ciRepo.ValidateCIAgainstSchema(ctx, &req.CI, &req.Schema)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to validate CI", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// handleValidateRelationshipAgainstSchema handles validating relationship data against a schema
func (h *SchemaHandler) handleValidateRelationshipAgainstSchema(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.ValidateRelationshipAgainstSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate relationship against schema
	result, err := h.ciRepo.ValidateRelationshipAgainstSchema(ctx, &req.Relationship, &req.Schema)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to validate relationship", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// Default Schema Template Handlers

// handleGetDefaultCISchemas handles getting default CI type schemas
func (h *SchemaHandler) handleGetDefaultCISchemas(w http.ResponseWriter, r *http.Request) {
	defaultSchemas := models.GetDefaultCISchemas()
	h.respondWithJSON(w, http.StatusOK, defaultSchemas)
}

// handleGetDefaultRelationshipSchemas handles getting default relationship type schemas
func (h *SchemaHandler) handleGetDefaultRelationshipSchemas(w http.ResponseWriter, r *http.Request) {
	defaultSchemas := models.GetDefaultRelationshipSchemas()
	h.respondWithJSON(w, http.StatusOK, defaultSchemas)
}

// handleCreateSchemaFromTemplate handles creating a schema from a template
func (h *SchemaHandler) handleCreateSchemaFromTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)
	vars := mux.Vars(r)

	templateName := vars["name"]

	// Get template schemas
	var templateSchema models.CITypeSchema
	defaultSchemas := models.GetDefaultCISchemas()
	found := false

	for _, schema := range defaultSchemas {
		if schema.Name == templateName {
			templateSchema = schema
			found = true
			break
		}
	}

	if !found {
		h.respondWithError(w, http.StatusNotFound, "Template not found", nil)
		return
	}

	// Create new schema from template
	newSchema := models.CITypeSchema{
		ID:          uuid.New(),
		Name:        templateSchema.Name,
		Description: templateSchema.Description,
		Attributes:  templateSchema.Attributes,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}

	// Create schema
	schema, err := h.ciRepo.CreateCITypeSchema(ctx, &newSchema)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create schema from template", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, schema)
}

// Helper methods

// authMiddleware is a placeholder for authentication middleware
func (h *SchemaHandler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In a real implementation, this would validate JWT tokens
		// For now, we'll just pass through
		next(w, r)
	}
}

// getUserIDFromContext extracts user ID from context
func (h *SchemaHandler) getUserIDFromContext(ctx context.Context) uuid.UUID {
	// In a real implementation, this would extract user ID from JWT token
	// For now, we'll return a placeholder
	return uuid.New()
}

// validateRequest validates request struct
func (h *SchemaHandler) validateRequest(req interface{}) error {
	// In a real implementation, this would use a validation library
	// For now, we'll just return nil
	return nil
}

// respondWithError sends an error response
func (h *SchemaHandler) respondWithError(w http.ResponseWriter, code int, message string, err error) {
	response := map[string]interface{}{
		"error":   message,
		"success": false,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	h.respondWithJSON(w, code, response)
}

// respondWithJSON sends a JSON response
func (h *SchemaHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to marshal response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
