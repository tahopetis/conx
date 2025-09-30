package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"connect/internal/models"
	"connect/internal/repositories"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// CIHandler handles CI-related endpoints
type CIHandler struct {
	ciRepo *repositories.CIRepository
}

// NewCIHandler creates a new CIHandler
func NewCIHandler(ciRepo *repositories.CIRepository) *CIHandler {
	return &CIHandler{ciRepo: ciRepo}
}

// RegisterRoutes registers CI-related routes
func (h *CIHandler) RegisterRoutes(router *mux.Router) {
	// CI CRUD routes
	router.HandleFunc("/api/v1/cis", h.authMiddleware(h.handleListCIs)).Methods("GET")
	router.HandleFunc("/api/v1/cis", h.authMiddleware(h.handleCreateCI)).Methods("POST")
	router.HandleFunc("/api/v1/cis/{id}", h.authMiddleware(h.handleGetCI)).Methods("GET")
	router.HandleFunc("/api/v1/cis/{id}", h.authMiddleware(h.handleUpdateCI)).Methods("PUT")
	router.HandleFunc("/api/v1/cis/{id}", h.authMiddleware(h.handleDeleteCI)).Methods("DELETE")

	// CI relationship routes
	router.HandleFunc("/api/v1/cis/{id}/relationships", h.authMiddleware(h.handleGetRelationships)).Methods("GET")
	router.HandleFunc("/api/v1/relationships", h.authMiddleware(h.handleCreateRelationship)).Methods("POST")
	router.HandleFunc("/api/v1/relationships/{id}", h.authMiddleware(h.handleDeleteRelationship)).Methods("DELETE")
}

// CI CRUD Handlers

// handleListCIs handles listing CIs with pagination and filtering
func (h *CIHandler) handleListCIs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	req := &models.ListCIsRequest{
		Page:     1,
		PageSize: 20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			req.PageSize = pageSize
		}
	}

	req.Search = r.URL.Query().Get("search")
	req.Type = r.URL.Query().Get("type")
	req.Status = r.URL.Query().Get("status")
	req.Criticality = r.URL.Query().Get("criticality")
	req.Owner = r.URL.Query().Get("owner")
	req.Location = r.URL.Query().Get("location")
	req.SortBy = r.URL.Query().Get("sort_by")
	req.SortOrder = r.URL.Query().Get("sort_order")

	// Parse tags
	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
	}

	// Get CIs
	response, err := h.ciRepo.ListCIs(ctx, req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list CIs", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, response)
}

// handleCreateCI handles creating a new CI
func (h *CIHandler) handleCreateCI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)

	var req models.CreateCIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Create CI object
	ci := &models.CI{
		ID:           uuid.New(),
		Name:         req.Name,
		Type:         req.Type,
		Description:  req.Description,
		Status:       req.Status,
		Criticality:  req.Criticality,
		Owner:        req.Owner,
		Location:     req.Location,
		Attributes:   req.Attributes,
		Tags:         req.Tags,
		InstallDate:  req.InstallDate,
		WarrantyExpiry: req.WarrantyExpiry,
		CreatedBy:    userID,
		UpdatedBy:    userID,
	}

	// Try to get schema for CI type validation
	schema, err := h.ciRepo.GetCISchemaByType(ctx, req.Type)
	if err == nil {
		// Schema found, create with validation
		createdCI, err := h.ciRepo.CreateCIWithValidation(ctx, ci, schema)
		if err != nil {
			h.respondWithError(w, http.StatusInternalServerError, "Failed to create CI with validation", err)
			return
		}
		h.respondWithJSON(w, http.StatusCreated, createdCI)
		return
	}

	// No schema found, create without validation
	createdCI, err := h.ciRepo.CreateCI(ctx, ci)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create CI", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, createdCI)
}

// handleGetCI handles retrieving a CI by ID
func (h *CIHandler) handleGetCI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	ciID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid CI ID", err)
		return
	}

	ci, err := h.ciRepo.GetCI(ctx, ciID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "CI not found", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, ci)
}

// handleUpdateCI handles updating an existing CI
func (h *CIHandler) handleUpdateCI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)
	vars := mux.Vars(r)

	ciID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid CI ID", err)
		return
	}

	// Get existing CI
	existingCI, err := h.ciRepo.GetCI(ctx, ciID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "CI not found", err)
		return
	}

	var req models.UpdateCIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update CI fields
	if req.Name != "" {
		existingCI.Name = req.Name
	}
	if req.Type != "" {
		existingCI.Type = req.Type
	}
	if req.Description != "" {
		existingCI.Description = req.Description
	}
	if req.Status != "" {
		existingCI.Status = req.Status
	}
	if req.Criticality != "" {
		existingCI.Criticality = req.Criticality
	}
	if req.Owner != "" {
		existingCI.Owner = req.Owner
	}
	if req.Location != "" {
		existingCI.Location = req.Location
	}
	if len(req.Attributes) > 0 {
		existingCI.Attributes = req.Attributes
	}
	if len(req.Tags) > 0 {
		existingCI.Tags = req.Tags
	}
	if req.InstallDate != nil {
		existingCI.InstallDate = req.InstallDate
	}
	if req.WarrantyExpiry != nil {
		existingCI.WarrantyExpiry = req.WarrantyExpiry
	}
	if req.LastUpdated != nil {
		existingCI.LastUpdated = req.LastUpdated
	}
	if req.LastScanned != nil {
		existingCI.LastScanned = req.LastScanned
	}
	if req.IsActive != nil {
		existingCI.IsActive = *req.IsActive
	}
	existingCI.UpdatedBy = userID

	// Try to get schema for CI type validation
	schema, err := h.ciRepo.GetCISchemaByType(ctx, existingCI.Type)
	if err == nil {
		// Schema found, update with validation
		updatedCI, err := h.ciRepo.UpdateCIWithValidation(ctx, existingCI, schema)
		if err != nil {
			h.respondWithError(w, http.StatusInternalServerError, "Failed to update CI with validation", err)
			return
		}
		h.respondWithJSON(w, http.StatusOK, updatedCI)
		return
	}

	// No schema found, update without validation
	updatedCI, err := h.ciRepo.UpdateCI(ctx, existingCI)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update CI", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, updatedCI)
}

// handleDeleteCI handles deleting a CI
func (h *CIHandler) handleDeleteCI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	ciID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid CI ID", err)
		return
	}

	if err := h.ciRepo.DeleteCI(ctx, ciID); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete CI", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "CI deleted successfully"})
}

// Relationship Handlers

// handleGetRelationships handles retrieving relationships for a CI
func (h *CIHandler) handleGetRelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	ciID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid CI ID", err)
		return
	}

	// Check if CI exists
	_, err = h.ciRepo.GetCI(ctx, ciID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, "CI not found", err)
		return
	}

	relationships, err := h.ciRepo.GetRelationshipsByCI(ctx, ciID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get relationships", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, relationships)
}

// handleCreateRelationship handles creating a new relationship
func (h *CIHandler) handleCreateRelationship(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := h.getUserIDFromContext(ctx)

	var req models.CreateRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Check for circular dependency
	hasCircular, err := h.ciRepo.CheckCircularDependency(ctx, req.SourceCIID, req.TargetCIID, req.Type)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to check circular dependency", err)
		return
	}

	if hasCircular {
		h.respondWithError(w, http.StatusBadRequest, "Circular dependency detected", nil)
		return
	}

	// Create relationship object
	relationship := &models.CIRelationship{
		ID:           uuid.New(),
		SourceCIID:   req.SourceCIID,
		TargetCIID:   req.TargetCIID,
		Type:         req.Type,
		Attributes:   req.Attributes,
		Description:  req.Description,
		CreatedBy:    userID,
		UpdatedBy:    userID,
	}

	// Try to get schema for relationship type validation
	schema, err := h.ciRepo.GetRelationshipSchemaByType(ctx, req.Type)
	if err == nil {
		// Schema found, create with validation
		createdRelationship, err := h.ciRepo.CreateRelationshipWithValidation(ctx, relationship, schema)
		if err != nil {
			h.respondWithError(w, http.StatusInternalServerError, "Failed to create relationship with validation", err)
			return
		}
		h.respondWithJSON(w, http.StatusCreated, createdRelationship)
		return
	}

	// No schema found, create without validation
	createdRelationship, err := h.ciRepo.CreateRelationship(ctx, relationship)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create relationship", err)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, createdRelationship)
}

// handleDeleteRelationship handles deleting a relationship
func (h *CIHandler) handleDeleteRelationship(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	relationshipID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid relationship ID", err)
		return
	}

	if err := h.ciRepo.DeleteRelationship(ctx, relationshipID); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete relationship", err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Relationship deleted successfully"})
}

// Helper methods

// authMiddleware is a placeholder for authentication middleware
func (h *CIHandler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In a real implementation, this would validate JWT tokens
		// For now, we'll just pass through
		next(w, r)
	}
}

// getUserIDFromContext extracts user ID from context
func (h *CIHandler) getUserIDFromContext(ctx context.Context) uuid.UUID {
	// In a real implementation, this would extract user ID from JWT token
	// For now, we'll return a placeholder
	return uuid.New()
}

// respondWithError sends an error response
func (h *CIHandler) respondWithError(w http.ResponseWriter, code int, message string, err error) {
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
func (h *CIHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Failed to marshal response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
