package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/usecase"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommentUseCase is a mock implementation of CommentUseCase
type MockCommentUseCase struct {
	mock.Mock
}

func (m *MockCommentUseCase) CreateComment(ctx context.Context, req usecase.CreateCommentRequest) (*usecase.CommentResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*usecase.CommentResponse), args.Error(1)
}

func (m *MockCommentUseCase) GetCommentsByTicket(ctx context.Context, ticketID string, page, perPage int) (*usecase.ListCommentsResponse, error) {
	args := m.Called(ctx, ticketID, page, perPage)
	return args.Get(0).(*usecase.ListCommentsResponse), args.Error(1)
}

func (m *MockCommentUseCase) UpdateComment(ctx context.Context, commentID string, req usecase.UpdateCommentRequest) (*usecase.CommentResponse, error) {
	args := m.Called(ctx, commentID, req)
	return args.Get(0).(*usecase.CommentResponse), args.Error(1)
}

func (m *MockCommentUseCase) DeleteComment(ctx context.Context, commentID string) error {
	args := m.Called(ctx, commentID)
	return args.Error(0)
}

func TestCommentHandler_CreateComment(t *testing.T) {
	tests := []struct {
		name           string
		ticketID       string
		requestBody    string
		headers        map[string]string
		mockResponse   *usecase.CommentResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "successful comment creation",
			ticketID: "ticket-123",
			requestBody: `{
				"ticket_id": "ticket-123",
				"author_id": "user-123",
				"role": "EMPLOYEE",
				"body": "This is a test comment"
			}`,
			headers: map[string]string{
				"X-User-ID":   "user-123",
				"X-User-Role": "EMPLOYEE",
			},
			mockResponse: &usecase.CommentResponse{
				Comment: &domain.Comment{
					ID:       "comment-123",
					TicketID: "ticket-123",
					AuthorID: "user-123",
					Role:     domain.CommentRoleEmployee,
					Body:     "This is a test comment",
				},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"status":true,"message":"Comment created successfully","data":{"id":"comment-123","ticket_id":"ticket-123","author_id":"user-123","role":"EMPLOYEE","body":"This is a test comment","created_at":"0001-01-01T00:00:00Z"}}`,
		},
		{
			name:           "missing ticket ID",
			ticketID:       "",
			requestBody:    `{"body": "test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Ticket ID is required","data":null,"code":"ticket_id"}`,
		},
		{
			name:           "invalid request body",
			ticketID:       "ticket-123",
			requestBody:    `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Invalid request body","data":null,"code":"invalid_request"}`,
		},
		{
			name:     "ticket not found",
			ticketID: "ticket-123",
			requestBody: `{
				"ticket_id": "ticket-123",
				"author_id": "user-123",
				"role": "EMPLOYEE",
				"body": "This is a test comment"
			}`,
			headers: map[string]string{
				"X-User-ID":   "user-123",
				"X-User-Role": "EMPLOYEE",
			},
			mockError:      assert.AnError,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":false,"message":"Ticket not found","data":null,"code":"ticket_not_found"}`,
		},
		{
			name:     "empty comment body validation error",
			ticketID: "ticket-123",
			requestBody: `{
				"ticket_id": "ticket-123",
				"author_id": "user-123",
				"role": "EMPLOYEE",
				"body": ""
			}`,
			headers: map[string]string{
				"X-User-ID":   "user-123",
				"X-User-Role": "EMPLOYEE",
			},
			mockError:      assert.AnError,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Comment body is required","data":null,"code":"empty_comment_body"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUseCase := &MockCommentUseCase{}
			handler := NewCommentHandler(mockUseCase)

			// Setup expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				mockUseCase.On("CreateComment", mock.Anything, mock.AnythingOfType("usecase.CreateCommentRequest")).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("POST", "/api/v1/tickets/"+tt.ticketID+"/comments", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Setup router with proper route
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/tickets/{id}/comments", handler.CreateComment).Methods("POST")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())

			// Verify mock expectations
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCommentHandler_GetCommentsByTicket(t *testing.T) {
	tests := []struct {
		name           string
		ticketID       string
		queryParams    string
		mockResponse   *usecase.ListCommentsResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "successful retrieval",
			ticketID: "ticket-123",
			mockResponse: &usecase.ListCommentsResponse{
				Comments: []*domain.Comment{
					{ID: "comment-1", TicketID: "ticket-123", Body: "Test comment 1"},
					{ID: "comment-2", TicketID: "ticket-123", Body: "Test comment 2"},
				},
				Total:   2,
				Page:    1,
				PerPage: 20,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":true,"message":"Comments retrieved successfully","data":{"comments":[{"id":"comment-1","ticket_id":"ticket-123","author_id":"","role":"","body":"Test comment 1","created_at":"0001-01-01T00:00:00Z"},{"id":"comment-2","ticket_id":"ticket-123","author_id":"","role":"","body":"Test comment 2","created_at":"0001-01-01T00:00:00Z"}],"total":2,"page":1,"per_page":20}}`,
		},
		{
			name:           "missing ticket ID",
			ticketID:       "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Ticket ID is required","data":null,"code":"ticket_id"}`,
		},
		{
			name:           "ticket not found",
			ticketID:       "ticket-123",
			mockError:      assert.AnError,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":false,"message":"Ticket not found","data":null,"code":"ticket_not_found"}`,
		},
		{
			name:        "with pagination parameters",
			ticketID:    "ticket-123",
			queryParams: "?page=2&per_page=10",
			mockResponse: &usecase.ListCommentsResponse{
				Comments: []*domain.Comment{},
				Total:    25,
				Page:     2,
				PerPage:  10,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":true,"message":"Comments retrieved successfully","data":{"comments":[],"total":25,"page":2,"per_page":10}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUseCase := &MockCommentUseCase{}
			handler := NewCommentHandler(mockUseCase)

			// Setup expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				// Default page=1, perPage=20 for test cases without query params
				page, perPage := 1, 20
				if tt.queryParams != "" {
					if tt.queryParams == "?page=2&per_page=10" {
						page, perPage = 2, 10
					}
				}
				mockUseCase.On("GetCommentsByTicket", mock.Anything, tt.ticketID, page, perPage).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			url := "/api/v1/tickets/" + tt.ticketID + "/comments" + tt.queryParams
			req := httptest.NewRequest("GET", url, nil)

			// Setup router with proper route
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/tickets/{id}/comments", handler.GetCommentsByTicket).Methods("GET")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())

			// Verify mock expectations
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCommentHandler_UpdateComment(t *testing.T) {
	tests := []struct {
		name           string
		commentID      string
		requestBody    string
		mockResponse   *usecase.CommentResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "successful comment update",
			commentID: "comment-123",
			requestBody: `{
				"body": "Updated comment content"
			}`,
			mockResponse: &usecase.CommentResponse{
				Comment: &domain.Comment{
					ID:       "comment-123",
					TicketID: "ticket-123",
					AuthorID: "user-123",
					Role:     domain.CommentRoleEmployee,
					Body:     "Updated comment content",
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":true,"message":"Comment updated successfully","data":{"id":"comment-123","ticket_id":"ticket-123","author_id":"user-123","role":"EMPLOYEE","body":"Updated comment content","created_at":"0001-01-01T00:00:00Z"}}`,
		},
		{
			name:           "missing comment ID",
			commentID:      "",
			requestBody:    `{"body": "test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Comment ID is required","data":null,"code":"comment_id"}`,
		},
		{
			name:           "invalid request body",
			commentID:      "comment-123",
			requestBody:    `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Invalid request body","data":null,"code":"invalid_request"}`,
		},
		{
			name:      "comment not found",
			commentID: "comment-123",
			requestBody: `{
				"body": "Updated comment content"
			}`,
			mockError:      assert.AnError,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":false,"message":"Comment not found","data":null,"code":"comment_not_found"}`,
		},
		{
			name:      "edit window expired",
			commentID: "comment-123",
			requestBody: `{
				"body": "Updated comment content"
			}`,
			mockError:      assert.AnError,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"status":false,"message":"Comment can only be edited within 15 minutes of creation","data":null,"code":"edit_window_expired"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUseCase := &MockCommentUseCase{}
			handler := NewCommentHandler(mockUseCase)

			// Setup expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				mockUseCase.On("UpdateComment", mock.Anything, tt.commentID, mock.AnythingOfType("usecase.UpdateCommentRequest")).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("PATCH", "/api/v1/comments/"+tt.commentID, bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Setup router with proper route
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/comments/{id}", handler.UpdateComment).Methods("PATCH")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())

			// Verify mock expectations
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCommentHandler_DeleteComment(t *testing.T) {
	tests := []struct {
		name           string
		commentID      string
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful comment deletion",
			commentID:      "comment-123",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing comment ID",
			commentID:      "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":false,"message":"Comment ID is required","data":null,"code":"comment_id"}`,
		},
		{
			name:           "comment not found",
			commentID:      "comment-123",
			mockError:      assert.AnError,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":false,"message":"Comment not found","data":null,"code":"comment_not_found"}`,
		},
		{
			name:           "delete window expired",
			commentID:      "comment-123",
			mockError:      assert.AnError,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"status":false,"message":"Comment can only be deleted within 15 minutes of creation","data":null,"code":"delete_window_expired"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUseCase := &MockCommentUseCase{}
			handler := NewCommentHandler(mockUseCase)

			// Setup expectations
			if tt.commentID != "" {
				mockUseCase.On("DeleteComment", mock.Anything, tt.commentID).Return(tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("DELETE", "/api/v1/comments/"+tt.commentID, nil)

			// Setup router with proper route
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/comments/{id}", handler.DeleteComment).Methods("DELETE")

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			}

			// Verify mock expectations
			mockUseCase.AssertExpectations(t)
		})
	}
}

func TestCommentHandler_RegisterRoutes(t *testing.T) {
	// Setup mocks
	mockUseCase := &MockCommentUseCase{}
	handler := NewCommentHandler(mockUseCase)

	// Create router
	router := mux.NewRouter()

	// Register routes
	handler.RegisterRoutes(router)

	// Test that routes are registered by checking the router
	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/tickets/{id}/comments"},
		{"GET", "/api/v1/tickets/{id}/comments"},
		{"PATCH", "/api/v1/comments/{id}"},
		{"DELETE", "/api/v1/comments/{id}"},
	}

	for _, route := range routes {
		// Create a test request to verify the route exists
		req := httptest.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// We expect either a 404 (if route exists but other validation fails)
		// or a 405 (if route exists but method is wrong for this specific path)
		// The important thing is that we don't get a 404 for all routes, which would indicate
		// that the route wasn't registered at all
		assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be registered", route.method, route.path)
	}
}
