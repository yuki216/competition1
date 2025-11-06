package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vobe/auth-service/application/port/inbound"
	"github.com/vobe/auth-service/infrastructure/http/middleware"
	"github.com/vobe/auth-service/infrastructure/http/response"
	"github.com/vobe/auth-service/infrastructure/http/validator"
)

type UserManagementHandler struct {
	userManagementUseCase inbound.UserManagementUseCase
	authMiddleware       *middleware.AuthMiddleware
}

func NewUserManagementHandler(
	userManagementUseCase inbound.UserManagementUseCase,
	authMiddleware *middleware.AuthMiddleware,
) *UserManagementHandler {
	return &UserManagementHandler{
		userManagementUseCase: userManagementUseCase,
		authMiddleware:       authMiddleware,
	}
}

// CreateUser creates a new user
func (h *UserManagementHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req inbound.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Generate UUID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Set default status if not provided
	if req.Status == "" {
		req.Status = "active"
	}

	// Validate input
	if !validator.ValidateRequired(req.Name) {
		response.UnprocessableEntity(w, "Name is required")
		return
	}

	if !validator.ValidateEmail(req.Email) {
		response.UnprocessableEntity(w, "Invalid email format")
		return
	}

	if !validator.ValidateRequired(req.Password) {
		response.UnprocessableEntity(w, "Password is required")
		return
	}

	if len(req.Password) < 8 {
		response.UnprocessableEntity(w, "Password must be at least 8 characters")
		return
	}

	if !validator.ValidateRequired(req.Role) {
		response.UnprocessableEntity(w, "Role is required")
		return
	}

	// Call use case
	err := h.userManagementUseCase.CreateUser(r.Context(), req)
	if err != nil {
		switch err.Error() {
		case "invalid name format":
			response.UnprocessableEntity(w, "Invalid name format")
		case "invalid email format":
			response.UnprocessableEntity(w, "Invalid email format")
		case "password must be at least 8 characters":
			response.UnprocessableEntity(w, "Password must be at least 8 characters")
		case "invalid role":
			response.UnprocessableEntity(w, "Invalid role")
		case "invalid status":
			response.UnprocessableEntity(w, "Invalid status")
		case "email already exists":
			response.Conflict(w, "Email already exists")
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	response.Success(w, http.StatusCreated, "User created successfully", nil)
}

// UpdateUser updates an existing user
func (h *UserManagementHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract user ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	userID := strings.TrimSuffix(path, "/")
	if userID == "" {
		response.BadRequest(w, "User ID is required")
		return
	}

	var req inbound.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Call use case
	err := h.userManagementUseCase.UpdateUser(r.Context(), userID, req)
	if err != nil {
		switch err.Error() {
		case "user not found":
			response.NotFound(w, "User not found")
		case "invalid name format":
			response.UnprocessableEntity(w, "Invalid name format")
		case "invalid role":
			response.UnprocessableEntity(w, "Invalid role")
		case "invalid status":
			response.UnprocessableEntity(w, "Invalid status")
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	response.Success(w, http.StatusOK, "User updated successfully", nil)
}

// DeleteUser soft deletes a user
func (h *UserManagementHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract user ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	userID := strings.TrimSuffix(path, "/")
	if userID == "" {
		response.BadRequest(w, "User ID is required")
		return
	}

	// Call use case
	err := h.userManagementUseCase.DeleteUser(r.Context(), userID)
	if err != nil {
		switch err.Error() {
		case "user not found":
			response.NotFound(w, "User not found")
		case "user ID cannot be empty":
			response.BadRequest(w, "User ID is required")
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	response.Success(w, http.StatusOK, "User deleted successfully", nil)
}

// GetUserDetail retrieves user details
func (h *UserManagementHandler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract user ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	userID := strings.TrimSuffix(path, "/")
	if userID == "" {
		response.BadRequest(w, "User ID is required")
		return
	}

	// Call use case
	userDetail, err := h.userManagementUseCase.GetUserDetail(r.Context(), userID)
	if err != nil {
		switch err.Error() {
		case "user not found":
			response.NotFound(w, "User not found")
		case "user ID cannot be empty":
			response.BadRequest(w, "User ID is required")
		default:
			response.InternalServerError(w, "Internal server error")
		}
		return
	}

	response.Success(w, http.StatusOK, "success", userDetail)
}

// ListUsers retrieves a list of users with pagination and filters
func (h *UserManagementHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse query parameters
	req := inbound.ListUsersRequest{}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Parse filters
	req.Filter.Name = r.URL.Query().Get("name")
	req.Filter.Role = r.URL.Query().Get("role")
	req.Filter.Status = r.URL.Query().Get("status")

	// Call use case
	result, err := h.userManagementUseCase.ListUsers(r.Context(), req)
	if err != nil {
		response.InternalServerError(w, "Internal server error")
		return
	}

	response.Success(w, http.StatusOK, "success", result)
}

// RegisterRoutes registers user management routes with admin middleware
func (h *UserManagementHandler) RegisterRoutes(mux *http.ServeMux) {
	// All routes require admin authentication
	mux.HandleFunc("/v1/admin/users", h.authMiddleware.RequireAdmin(h.handleUsers))
	mux.HandleFunc("/v1/admin/users/", h.authMiddleware.RequireAdmin(h.handleUserByID))
}

// handleUsers handles the /v1/admin/users endpoint
func (h *UserManagementHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListUsers(w, r)
	case http.MethodPost:
		h.CreateUser(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleUserByID handles the /v1/admin/users/{id} endpoint
func (h *UserManagementHandler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetUserDetail(w, r)
	case http.MethodPut:
		h.UpdateUser(w, r)
	case http.MethodDelete:
		h.DeleteUser(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}