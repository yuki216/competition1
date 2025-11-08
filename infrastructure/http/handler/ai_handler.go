package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/fixora/fixora/application/usecase"
	"github.com/fixora/fixora/domain"
)

// AIHandler handles HTTP requests for AI services
type AIHandler struct{ aiUseCase *usecase.AIUseCase }

func NewAIHandler(aiUseCase *usecase.AIUseCase) *AIHandler { return &AIHandler{aiUseCase: aiUseCase} }

func (h *AIHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/v1/ai/suggest", h.GetSuggestion).Methods("POST")
	router.HandleFunc("/v1/ai/suggest/stream", h.StreamSuggestion).Methods("GET")
	router.HandleFunc("/v1/ai/kb/search", h.SearchKnowledgeBase).Methods("POST")
	router.HandleFunc("/v1/ai/embedding", h.GenerateEmbedding).Methods("POST")
	router.HandleFunc("/v1/ai/analyze", h.AnalyzeTicketContent).Methods("POST")
	router.HandleFunc("/v1/ai/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/v1/ai/info", h.GetProviderInfo).Methods("GET")
	router.HandleFunc("/v1/tickets/ai-intake", h.AIIntakeCreateTicket).Methods("POST")
}

func (h *AIHandler) GetSuggestion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description"`
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

func (h *AIHandler) StreamSuggestion(w http.ResponseWriter, r *http.Request) {
	description := r.URL.Query().Get("query")
	if description == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	eventChan, err := h.aiUseCase.StreamSuggestion(r.Context(), description)
	if err != nil {
		h.sendSSEError(w, "stream_error", err.Error())
		return
	}
	for event := range eventChan {
		if err := h.sendSSEEvent(w, event); err != nil {
			return
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (h *AIHandler) SearchKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query   string               `json:"query"`
		Filters domain.KBChunkFilter `json:"filters"`
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
	response := map[string]interface{}{"query": req.Query, "results": chunks, "count": len(chunks)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AIHandler) GenerateEmbedding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text  string   `json:"text"`
		Texts []string `json:"texts,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var response interface{}
	if len(req.Texts) > 0 {
		embeddings, err := h.aiUseCase.GenerateBatchEmbeddings(r.Context(), req.Texts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{"embeddings": embeddings, "count": len(embeddings)}
	} else {
		if req.Text == "" {
			http.Error(w, "Text is required", http.StatusBadRequest)
			return
		}
		embedding, err := h.aiUseCase.GenerateEmbedding(r.Context(), req.Text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{"embedding": embedding, "dimension": len(embedding)}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AIHandler) AnalyzeTicketContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
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

func (h *AIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	err := h.aiUseCase.ValidateAIProvider(r.Context())
	response := map[string]interface{}{"healthy": err == nil, "timestamp": time.Now().Unix()}
	if err != nil {
		response["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AIHandler) GetProviderInfo(w http.ResponseWriter, r *http.Request) {
	info := h.aiUseCase.GetAIProviderInfo(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *AIHandler) AIIntakeCreateTicket(w http.ResponseWriter, r *http.Request) {
	var req usecase.AITicketIntakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		req.AutoTitleFromAI = true
	}
	if string(req.Category) == "" {
		req.AutoCategorize = true
	}
	if string(req.Priority) == "" {
		req.AutoPrioritize = true
	}
	createdBy := r.Header.Get("X-User-ID")
	if createdBy == "" {
		createdBy = "default-user"
	}
	res, err := h.aiUseCase.IntakeCreateTicket(r.Context(), req, createdBy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

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
	if _, err := w.Write([]byte(eventStr)); err != nil {
		return err
	}
	return nil
}

func (h *AIHandler) sendSSEError(w http.ResponseWriter, errorCode, message string) {
	data, _ := json.Marshal(map[string]string{"code": errorCode, "message": message})
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(data))
}

const (
	SSEEventTypeInit      = "init"
	SSEEventTypeCandidate = "candidate"
	SSEEventTypeProgress  = "progress"
	SSEEventTypeEnd       = "end"
	SSEEventTypeError     = "error"
)
