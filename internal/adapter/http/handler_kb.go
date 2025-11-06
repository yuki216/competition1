package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"fixora/internal/domain"
	"fixora/internal/usecase"

	"github.com/gorilla/mux"
)

// KBHandler handles HTTP requests for knowledge base
type KBHandler struct {
	kbUseCase *usecase.KnowledgeUseCase
}

// NewKBHandler creates a new knowledge base handler
func NewKBHandler(kbUseCase *usecase.KnowledgeUseCase) *KBHandler {
	return &KBHandler{
		kbUseCase: kbUseCase,
	}
}

// RegisterRoutes registers knowledge base routes
func (h *KBHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/kb/entries", h.CreateEntry).Methods("POST")
	router.HandleFunc("/api/v1/kb/entries", h.ListEntries).Methods("GET")
	router.HandleFunc("/api/v1/kb/entries/{id}", h.GetEntry).Methods("GET")
	router.HandleFunc("/api/v1/kb/entries/{id}", h.UpdateEntry).Methods("PATCH")
	router.HandleFunc("/api/v1/kb/entries/{id}/publish", h.PublishEntry).Methods("POST")
	router.HandleFunc("/api/v1/kb/entries/{id}", h.DeleteEntry).Methods("DELETE")
	router.HandleFunc("/api/v1/kb/search", h.SearchEntries).Methods("POST")
	router.HandleFunc("/api/v1/kb/upload-text", h.UploadText).Methods("POST")
}

// CreateEntry handles knowledge base entry creation
func (h *KBHandler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateKnowledgeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from context (in real implementation, from auth middleware)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user" // Fallback for development
	}
	req.CreatedBy = userID

	entry, err := h.kbUseCase.CreateEntry(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

// GetEntry handles retrieving a single knowledge base entry
func (h *KBHandler) GetEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entryID := vars["id"]

	if entryID == "" {
		http.Error(w, "Entry ID is required", http.StatusBadRequest)
		return
	}

	entry, err := h.kbUseCase.GetEntry(r.Context(), entryID)
	if err != nil {
		if err.Error() == "knowledge base entry not found" {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// ListEntries handles listing knowledge base entries
func (h *KBHandler) ListEntries(w http.ResponseWriter, r *http.Request) {
	filter := domain.KBChunkFilter{}

	// Parse query parameters
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = category
	}

	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		// Simple comma-separated tag parsing
		tags := splitTags(tagsStr)
		filter.Tags = tags
	}

	if topKStr := r.URL.Query().Get("top_k"); topKStr != "" {
		if topK, err := strconv.Atoi(topKStr); err == nil {
			filter.TopK = topK
		}
	}

	entries, err := h.kbUseCase.ListEntries(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateEntry handles knowledge base entry updates
func (h *KBHandler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entryID := vars["id"]

	if entryID == "" {
		http.Error(w, "Entry ID is required", http.StatusBadRequest)
		return
	}

	var req usecase.UpdateKnowledgeEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entry, err := h.kbUseCase.UpdateEntry(r.Context(), entryID, req)
	if err != nil {
		if err.Error() == "knowledge base entry not found" {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// PublishEntry handles publishing a knowledge base entry
func (h *KBHandler) PublishEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entryID := vars["id"]

	if entryID == "" {
		http.Error(w, "Entry ID is required", http.StatusBadRequest)
		return
	}

	if err := h.kbUseCase.PublishEntry(r.Context(), entryID); err != nil {
		if err.Error() == "knowledge base entry not found" {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Entry published successfully",
		"entry_id": entryID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteEntry handles knowledge base entry deletion
func (h *KBHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entryID := vars["id"]

	if entryID == "" {
		http.Error(w, "Entry ID is required", http.StatusBadRequest)
		return
	}

	if err := h.kbUseCase.DeleteEntry(r.Context(), entryID); err != nil {
		if err.Error() == "knowledge base entry not found" {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchEntries handles knowledge base search
func (h *KBHandler) SearchEntries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query   string                 `json:"query" validate:"required"`
		Filters domain.KBChunkFilter   `json:"filters,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	chunks, err := h.kbUseCase.SearchEntries(r.Context(), req.Query, req.Filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"query":   req.Query,
		"results": chunks,
		"count":   len(chunks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UploadText handles text upload for knowledge base entries
func (h *KBHandler) UploadText(w http.ResponseWriter, r *http.Request) {
	// Get form data
	title := r.FormValue("title")
	content := r.FormValue("content")
	category := r.FormValue("category")
	tagsStr := r.FormValue("tags")
	publishStr := r.FormValue("publish")

	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Parse tags
	var tags []string
	if tagsStr != "" {
		tags = splitTags(tagsStr)
	}

	// Parse publish flag
	publish := false
	if publishStr == "true" {
		publish = true
	}

	// Get user ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	// Create entry
	req := usecase.CreateKnowledgeEntryRequest{
		Title:     title,
		Content:   content,
		Category:  category,
		Tags:      tags,
		CreatedBy: userID,
	}

	entry, err := h.kbUseCase.CreateEntry(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish if requested
	if publish {
		if err := h.kbUseCase.PublishEntry(r.Context(), entry.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"entry_id": entry.ID,
		"title":    entry.Title,
		"status":   entry.Status,
		"published": publish,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func splitTags(tagsStr string) []string {
	// Simple comma-separated tag parsing
	if tagsStr == "" {
		return nil
	}

	tags := make([]string, 0)
	for _, tag := range strings.Split(tagsStr, ",") {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}

	return tags
}