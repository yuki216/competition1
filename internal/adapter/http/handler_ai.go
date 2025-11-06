package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/usecase"

	"github.com/gorilla/mux"
)

// AIHandler handles HTTP requests for AI services
type AIHandler struct {
	aiUseCase *usecase.AIUseCase
}

// NewAIHandler creates a new AI handler
func NewAIHandler(aiUseCase *usecase.AIUseCase) *AIHandler {
	return &AIHandler{
		aiUseCase: aiUseCase,
	}
}

// RegisterRoutes registers AI routes
func (h *AIHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/ai/suggest", h.GetSuggestion).Methods("POST")
	router.HandleFunc("/api/v1/ai/suggest/stream", h.StreamSuggestion).Methods("GET")
	router.HandleFunc("/api/v1/ai/kb/search", h.SearchKnowledgeBase).Methods("POST")
	router.HandleFunc("/api/v1/ai/embedding", h.GenerateEmbedding).Methods("POST")
	router.HandleFunc("/api/v1/ai/analyze", h.AnalyzeTicketContent).Methods("POST")
	router.HandleFunc("/api/v1/ai/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/api/v1/ai/info", h.GetProviderInfo).Methods("GET")
}

// GetSuggestion handles AI suggestion requests
func (h *AIHandler) GetSuggestion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Description == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
		return
	}

	suggestion, err := h.aiUseCase.GetSuggestion(r.Context(), req.Description)
	if err != nil {
		if err.Error() == "AI confidence too low" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestion)
}

// StreamSuggestion handles streaming AI suggestion requests
func (h *AIHandler) StreamSuggestion(w http.ResponseWriter, r *http.Request) {
	description := r.URL.Query().Get("query")
	if description == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Flush headers
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Get streaming suggestion
	eventChan, err := h.aiUseCase.StreamSuggestion(r.Context(), description)
	if err != nil {
		h.sendSSEError(w, "stream_error", err.Error())
		return
	}

	// Send events to client
	for event := range eventChan {
		if err := h.sendSSEEvent(w, event); err != nil {
			// Client disconnected or other error
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Small delay to prevent overwhelming the client
		time.Sleep(50 * time.Millisecond)
	}
}

// SearchKnowledgeBase handles knowledge base search requests
func (h *AIHandler) SearchKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query   string                 `json:"query" validate:"required"`
		Filters domain.KBChunkFilter   `json:"filters"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	chunks, err := h.aiUseCase.SearchKnowledgeBase(r.Context(), req.Query, req.Filters)
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

// GenerateEmbedding handles embedding generation requests
func (h *AIHandler) GenerateEmbedding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text  string   `json:"text" validate:"required"`
		Texts []string `json:"texts,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var response interface{}

	if len(req.Texts) > 0 {
		// Batch embedding generation
		embeddings, err := h.aiUseCase.GenerateBatchEmbeddings(r.Context(), req.Texts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"embeddings": embeddings,
			"count":      len(embeddings),
		}
	} else {
		// Single embedding generation
		if req.Text == "" {
			http.Error(w, "Text is required", http.StatusBadRequest)
			return
		}

		embedding, err := h.aiUseCase.GenerateEmbedding(r.Context(), req.Text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"embedding": embedding,
			"dimension": len(embedding),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyzeTicketContent handles ticket content analysis requests
func (h *AIHandler) AnalyzeTicketContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title" validate:"required"`
		Description string `json:"description" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Description == "" {
		http.Error(w, "Title and description are required", http.StatusBadRequest)
		return
	}

	analysis, err := h.aiUseCase.AnalyzeTicketContent(r.Context(), req.Title, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// HealthCheck handles AI service health check
func (h *AIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	err := h.aiUseCase.ValidateAIProvider(r.Context())

	response := map[string]interface{}{
		"healthy": err == nil,
		"timestamp": time.Now().Unix(),
	}

	if err != nil {
		response["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProviderInfo returns AI provider information
func (h *AIHandler) GetProviderInfo(w http.ResponseWriter, r *http.Request) {
	info := h.aiUseCase.GetAIProviderInfo(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Helper methods for Server-Sent Events

func (h *AIHandler) sendSSEEvent(w http.ResponseWriter, event interface{}) error {
	var eventStr string

	switch e := event.(type) {
	case map[string]interface{}:
		eventType, _ := e["type"].(string)
		data, _ := json.Marshal(e["data"])
		queryID, _ := e["query_id"].(string)

		eventStr = fmt.Sprintf("event: %s\nid: %s\ndata: %s\n\n", eventType, queryID, string(data))

	default:
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		eventStr = fmt.Sprintf("data: %s\n\n", string(data))
	}

	_, err := w.Write([]byte(eventStr))
	return err
}

func (h *AIHandler) sendSSEError(w http.ResponseWriter, errorCode, message string) {
	errorEvent := map[string]interface{}{
		"type": "error",
		"data": map[string]interface{}{
			"code":    errorCode,
			"message": message,
		},
	}

	data, _ := json.Marshal(errorEvent["data"])
	eventStr := fmt.Sprintf("event: error\ndata: %s\n\n", string(data))

	w.Write([]byte(eventStr))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// SSE Event Types
const (
	SSEEventTypeInit     = "init"
	SSEEventTypeCandidate = "candidate"
	SSEEventTypeProgress = "progress"
	SSEEventTypeEnd      = "end"
	SSEEventTypeError    = "error"
)