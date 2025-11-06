package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"fixora/internal/domain"
	"fixora/internal/usecase"

	"github.com/gorilla/mux"
)

// TicketHandler handles HTTP requests for tickets
type TicketHandler struct {
	ticketUseCase *usecase.TicketUseCase
}

// NewTicketHandler creates a new ticket handler
func NewTicketHandler(ticketUseCase *usecase.TicketUseCase) *TicketHandler {
	return &TicketHandler{
		ticketUseCase: ticketUseCase,
	}
}

// RegisterRoutes registers ticket routes
func (h *TicketHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/tickets", h.CreateTicket).Methods("POST")
	router.HandleFunc("/api/v1/tickets", h.ListTickets).Methods("GET")
	router.HandleFunc("/api/v1/tickets/{id}", h.GetTicket).Methods("GET")
	router.HandleFunc("/api/v1/tickets/{id}", h.UpdateTicket).Methods("PATCH")
	router.HandleFunc("/api/v1/tickets/{id}/assign", h.AssignTicket).Methods("POST")
	router.HandleFunc("/api/v1/tickets/{id}/resolve", h.ResolveTicket).Methods("POST")
	router.HandleFunc("/api/v1/tickets/{id}/close", h.CloseTicket).Methods("POST")
	router.HandleFunc("/api/v1/tickets/stats", h.GetTicketStats).Methods("GET")
}

// CreateTicket handles ticket creation
func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateTicketRequest
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

	response, err := h.ticketUseCase.CreateTicket(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetTicket handles retrieving a single ticket
func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		http.Error(w, "Ticket ID is required", http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketUseCase.GetTicket(r.Context(), ticketID)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// ListTickets handles listing tickets with filters
func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	filter := domain.TicketFilter{}

	// Parse query parameters
	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.TicketStatus(status)
		filter.Status = &s
	}

	if category := r.URL.Query().Get("category"); category != "" {
		c := domain.TicketCategory(category)
		filter.Category = &c
	}

	if priority := r.URL.Query().Get("priority"); priority != "" {
		p := domain.TicketPriority(priority)
		filter.Priority = &p
	}

	if createdBy := r.URL.Query().Get("created_by"); createdBy != "" {
		filter.CreatedBy = &createdBy
	}

	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filter.AssignedTo = &assignedTo
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	tickets, total, err := h.ticketUseCase.ListTickets(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"tickets": tickets,
		"total":   total,
		"limit":   filter.Limit,
		"offset":  filter.Offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateTicket handles ticket updates
func (h *TicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		http.Error(w, "Ticket ID is required", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketUseCase.UpdateTicket(r.Context(), ticketID, updates)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// AssignTicket handles ticket assignment
func (h *TicketHandler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		http.Error(w, "Ticket ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		AssignedTo string `json:"assigned_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AssignedTo == "" {
		http.Error(w, "Assigned to is required", http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketUseCase.AssignTicket(r.Context(), ticketID, req.AssignedTo)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// ResolveTicket handles ticket resolution
func (h *TicketHandler) ResolveTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		http.Error(w, "Ticket ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Resolution string `json:"resolution"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Resolution == "" {
		http.Error(w, "Resolution is required", http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketUseCase.ResolveTicket(r.Context(), ticketID, req.Resolution)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// CloseTicket handles ticket closure
func (h *TicketHandler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		http.Error(w, "Ticket ID is required", http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketUseCase.CloseTicket(r.Context(), ticketID)
	if err != nil {
		if err.Error() == "ticket not found" {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// GetTicketStats handles ticket statistics
func (h *TicketHandler) GetTicketStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.ticketUseCase.GetTicketStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}