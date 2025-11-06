package http

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/usecase"

	"github.com/gorilla/mux"
)

// CommentUseCase defines the behavior the handler depends on.
// Using an interface here makes the handler easily testable with mocks.
type CommentUseCase interface {
    CreateComment(ctx context.Context, req usecase.CreateCommentRequest) (*usecase.CommentResponse, error)
    GetCommentsByTicket(ctx context.Context, ticketID string, page, perPage int) (*usecase.ListCommentsResponse, error)
    UpdateComment(ctx context.Context, commentID string, req usecase.UpdateCommentRequest) (*usecase.CommentResponse, error)
    DeleteComment(ctx context.Context, commentID string) error
}

// CommentHandler handles HTTP requests for comments
type CommentHandler struct {
    commentUseCase CommentUseCase
}

// NewCommentHandler creates a new comment handler
func NewCommentHandler(commentUseCase CommentUseCase) *CommentHandler {
    return &CommentHandler{
        commentUseCase: commentUseCase,
    }
}

// RegisterRoutes registers comment routes
func (h *CommentHandler) RegisterRoutes(router *mux.Router) {
	// Comment routes for specific tickets
	router.HandleFunc("/api/v1/tickets/{id}/comments", h.CreateComment).Methods("POST")
	router.HandleFunc("/api/v1/tickets/{id}/comments", h.GetCommentsByTicket).Methods("GET")

	// Comment routes for individual comments
	router.HandleFunc("/api/v1/comments/{id}", h.UpdateComment).Methods("PATCH")
	router.HandleFunc("/api/v1/comments/{id}", h.DeleteComment).Methods("DELETE")
}

// CreateComment handles comment creation
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "ticket_id", "Ticket ID is required")
		return
	}

	var req usecase.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Set ticket ID from URL
	req.TicketID = ticketID

	// Get user ID from context (in real implementation, from auth middleware)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user" // Fallback for development
	}
	req.AuthorID = userID

	// Set role based on user type (in real implementation, from auth context)
	role := r.Header.Get("X-User-Role")
	if role == "" {
		role = "EMPLOYEE" // Default role
	}
	req.Role = domain.CommentRole(role)

	response, err := h.commentUseCase.CreateComment(r.Context(), req)
	if err != nil {
		switch err.Error() {
		case "ticket not found":
			h.writeErrorResponse(w, http.StatusNotFound, "ticket_not_found", "Ticket not found")
		case "validation failed: comment body is required":
			h.writeErrorResponse(w, http.StatusBadRequest, "empty_comment_body", "Comment body is required")
		case "validation failed: comment body must not exceed 5000 characters":
			h.writeErrorResponse(w, http.StatusBadRequest, "comment_too_long", "Comment body must not exceed 5000 characters")
		case "validation failed: invalid comment role":
			h.writeErrorResponse(w, http.StatusBadRequest, "invalid_role", "Invalid comment role")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to create comment")
		}
		return
	}

	h.writeSuccessResponse(w, http.StatusCreated, "Comment created successfully", response.Comment)
}

// GetCommentsByTicket handles retrieving comments for a ticket
func (h *CommentHandler) GetCommentsByTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID := vars["id"]

	if ticketID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "ticket_id", "Ticket ID is required")
		return
	}

	// Parse pagination parameters
	page := 1
	perPage := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			if pp > 100 {
				pp = 100
			}
			perPage = pp
		}
	}

	response, err := h.commentUseCase.GetCommentsByTicket(r.Context(), ticketID, page, perPage)
	if err != nil {
		switch err.Error() {
		case "ticket not found":
			h.writeErrorResponse(w, http.StatusNotFound, "ticket_not_found", "Ticket not found")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve comments")
		}
		return
	}

	h.writeSuccessResponse(w, http.StatusOK, "Comments retrieved successfully", response)
}

// UpdateComment handles comment updates
func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commentID := vars["id"]

	if commentID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "comment_id", "Comment ID is required")
		return
	}

	var req usecase.UpdateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	response, err := h.commentUseCase.UpdateComment(r.Context(), commentID, req)
	if err != nil {
		switch err.Error() {
		case "comment not found":
			h.writeErrorResponse(w, http.StatusNotFound, "comment_not_found", "Comment not found")
		case "validation failed: comment body is required":
			h.writeErrorResponse(w, http.StatusBadRequest, "empty_comment_body", "Comment body is required")
		case "validation failed: comment body must not exceed 5000 characters":
			h.writeErrorResponse(w, http.StatusBadRequest, "comment_too_long", "Comment body must not exceed 5000 characters")
		case "comment can only be edited within 15 minutes of creation":
			h.writeErrorResponse(w, http.StatusForbidden, "edit_window_expired", "Comment can only be edited within 15 minutes of creation")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to update comment")
		}
		return
	}

	h.writeSuccessResponse(w, http.StatusOK, "Comment updated successfully", response.Comment)
}

// DeleteComment handles comment deletion
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commentID := vars["id"]

	if commentID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "comment_id", "Comment ID is required")
		return
	}

	err := h.commentUseCase.DeleteComment(r.Context(), commentID)
	if err != nil {
		switch err.Error() {
		case "comment not found":
			h.writeErrorResponse(w, http.StatusNotFound, "comment_not_found", "Comment not found")
		case "comment can only be deleted within 15 minutes of creation":
			h.writeErrorResponse(w, http.StatusForbidden, "delete_window_expired", "Comment can only be deleted within 15 minutes of creation")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to delete comment")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods for writing responses

func (h *CommentHandler) writeSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"status":  true,
		"message": message,
		"data":    data,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *CommentHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"status":  false,
		"message": message,
		"data":    nil,
		"code":    code,
	}

	json.NewEncoder(w).Encode(response)
}