package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fixora/fixora/internal/adapter/ai"
	httpadapter "github.com/fixora/fixora/internal/adapter/http"
	"github.com/fixora/fixora/internal/adapter/persistence"
	"github.com/fixora/fixora/internal/config"
	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/infra/sse"
	"github.com/fixora/fixora/internal/ports"
	"github.com/fixora/fixora/internal/usecase"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

// FixoraIntegrationTestSuite represents the test suite for Fixora API integration tests
type FixoraIntegrationTestSuite struct {
	db                *sql.DB
	server            *httpadapter.Server
	config            *config.Config
	ticketUseCase     *usecase.TicketUseCase
	aiUseCase         *usecase.AIUseCase
	knowledgeUseCase  *usecase.KnowledgeUseCase
	streamer          *sse.Streamer
	ctx               context.Context
}

// setupFixoraIntegrationTest initializes the test suite with database and mock services
func setupFixoraIntegrationTest(t *testing.T) *FixoraIntegrationTestSuite {
	ctx := context.Background()

	// Load test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         "8081", // Different port for testing
			Host:         "localhost",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			Environment:  "test",
			Debug:        false,
		},
		Database: config.DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			User:           "postgres",
			Password:       "postgres",
			DBName:         "fixora_test",
			SSLMode:        "disable",
			MaxConnections: 10,
			MaxIdleTime:    5 * time.Minute,
			ConnectTimeout: 5 * time.Second,
			QueryTimeout:   10 * time.Second,
		},
		AI: config.AIConfig{
			Provider:          "mock",
			MockMode:         true,
			EmbeddingModel:   "text-embedding-ada-002",
			SuggestionModel:  "gpt-3.5-turbo",
			EmbeddingDim:     1536,
			TopK:             10,
			MinConfidence:    0.4,
			TimeoutMs:        5000,
			EnableCache:      false,
			CacheTTLMin:      60,
			ChunkSize:        800,
			ChunkOverlap:     150,
			BatchSize:        32,
			MaxConcurrency:   5,
		},
		SSE: config.SSEConfig{
			Enabled:           true,
			FlushInterval:     250 * time.Millisecond,
			HeartbeatInterval: 15 * time.Second,
			MaxConnections:    1000,
			MessageBufferSize: 256,
			ClientTimeout:     30 * time.Second,
		},
	}

	// Ensure test database exists and has schema
	db, err := ensureTestDatabaseAndSchemaForFixora(cfg)
	if err != nil {
		t.Fatalf("Failed to prepare test database: %v", err)
	}

	// Clean up database before tests
	if err := cleanupFixoraDatabase(db); err != nil {
		t.Fatalf("Failed to cleanup database: %v", err)
	}

	// Initialize repositories
	repos := initFixoraRepositories(db, cfg)

	// Initialize AI services with mock mode
	aiFactory := initFixoraAIServices(cfg)

	// Initialize SSE streamer
	streamer := sse.NewStreamer()
	streamer.Start(ctx)

	// Initialize use cases
	useCases := initFixoraUseCases(repos, aiFactory, streamer)

	// Initialize HTTP server
	serverConfig := httpadapter.ServerConfig{
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	server := httpadapter.NewServer(serverConfig, useCases.Ticket, useCases.AI, useCases.Knowledge)

	return &FixoraIntegrationTestSuite{
		db:               db,
		server:           server,
		config:           cfg,
		ticketUseCase:    useCases.Ticket,
		aiUseCase:        useCases.AI,
		knowledgeUseCase: useCases.Knowledge,
		streamer:         streamer,
		ctx:              ctx,
	}
}

// cleanup performs cleanup after tests
func (s *FixoraIntegrationTestSuite) cleanup(t *testing.T) {
	if err := cleanupFixoraDatabase(s.db); err != nil {
		t.Logf("Failed to cleanup database: %v", err)
	}
	if err := s.db.Close(); err != nil {
		t.Logf("Failed to close database connection: %v", err)
	}
	// Note: Streamer doesn't have explicit Stop method, it's context-based
	// The context is cancelled when the test completes
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	suite := setupFixoraIntegrationTest(t)
	defer suite.cleanup(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := suite.server.GetHandler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", response["status"])
	}
}

// TestTicketAPI tests all ticket-related endpoints
func TestTicketAPI(t *testing.T) {
	suite := setupFixoraIntegrationTest(t)
	defer suite.cleanup(t)

	handler := suite.server.GetHandler()

	t.Run("CreateTicket", func(t *testing.T) {
		createReq := map[string]interface{}{
			"title":       "Test Ticket",
			"description": "This is a test ticket for integration testing",
			"category":    "SOFTWARE",
			"priority":    "MEDIUM",
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "test-user-123")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			// Read response body to understand the error
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", http.StatusCreated, resp.StatusCode, string(body))
		}

		// Read the response body first
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("Failed to read response body: %v", readErr)
		}

		// The response is wrapped in a "ticket" field
		var response struct {
			Ticket domain.Ticket `json:"ticket"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("Failed to decode response: %v\nResponse body: %s", err, string(body))
		}
		ticket := response.Ticket

		if ticket.Title != "Test Ticket" {
			t.Errorf("Expected title 'Test Ticket', got '%s'", ticket.Title)
		}

		if ticket.Status != domain.TicketStatusOpen {
			t.Errorf("Expected status %s, got %s", domain.TicketStatusOpen, ticket.Status)
		}

		if ticket.CreatedBy != "test-user-123" {
			t.Errorf("Expected created_by 'test-user-123', got '%s'", ticket.CreatedBy)
		}
	})

	t.Run("GetTicket", func(t *testing.T) {
		// First create a ticket with UUID to avoid collisions
		ticketID := uuid.New().String()
		ticket := domain.Ticket{
			ID:          ticketID,
			Title:       "Test Get",
			Description: "Test description",
			Status:      domain.TicketStatusOpen,
			Category:    domain.TicketCategoryHardware,
			Priority:    domain.TicketPriorityHigh,
			CreatedBy:   "test-user-456",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Insert ticket directly into database for testing
		_, err := suite.db.ExecContext(suite.ctx, `
			INSERT INTO tickets (id, title, description, status, category, priority, created_by, ai_insight, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9)
		`, ticket.ID, ticket.Title, ticket.Description, ticket.Status, ticket.Category, ticket.Priority, ticket.CreatedBy, ticket.CreatedAt, ticket.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to insert test ticket: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/tickets/"+ticket.ID, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var retrievedTicket domain.Ticket
		if err := json.NewDecoder(resp.Body).Decode(&retrievedTicket); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if retrievedTicket.ID != ticket.ID {
			t.Errorf("Expected ticket ID %s, got %s", ticket.ID, retrievedTicket.ID)
		}
	})

	t.Run("ListTickets", func(t *testing.T) {
		// Create multiple tickets with UUIDs to avoid collisions
		tickets := []domain.Ticket{
			{
				ID:          uuid.New().String(),
				Title:       "Ticket 1",
				Description: "Description 1",
				Status:      domain.TicketStatusOpen,
				Category:    domain.TicketCategoryNetwork,
				Priority:    domain.TicketPriorityLow,
				CreatedBy:   "user1",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New().String(),
				Title:       "Ticket 2",
				Description: "Description 2",
				Status:      domain.TicketStatusOpen,
				Category:    domain.TicketCategorySoftware,
				Priority:    domain.TicketPriorityMedium,
				CreatedBy:   "user2",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		for _, ticket := range tickets {
			_, err := suite.db.ExecContext(suite.ctx, `
				INSERT INTO tickets (id, title, description, status, category, priority, created_by, ai_insight, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9)
			`, ticket.ID, ticket.Title, ticket.Description, ticket.Status, ticket.Category, ticket.Priority, ticket.CreatedBy, ticket.CreatedAt, ticket.UpdatedAt)
			if err != nil {
				t.Fatalf("Failed to insert test ticket: %v", err)
			}
		}

		req := httptest.NewRequest("GET", "/api/v1/tickets", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		ticketsData, ok := response["tickets"].([]interface{})
		if !ok {
			t.Fatal("Expected tickets array in response")
		}

		if len(ticketsData) < 2 {
			t.Errorf("Expected at least 2 tickets, got %d", len(ticketsData))
		}
	})

	t.Run("AssignTicket", func(t *testing.T) {
		// Create a ticket with UUID
		ticketID := uuid.New().String()
		ticket := domain.Ticket{
			ID:          ticketID,
			Title:       "Test Assignment",
			Description: "Test assignment description",
			Status:      domain.TicketStatusOpen,
			Category:    domain.TicketCategoryAccount,
			Priority:    domain.TicketPriorityHigh,
			CreatedBy:   "user1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		_, err := suite.db.ExecContext(suite.ctx, `
			INSERT INTO tickets (id, title, description, status, category, priority, created_by, ai_insight, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9)
		`, ticket.ID, ticket.Title, ticket.Description, ticket.Status, ticket.Category, ticket.Priority, ticket.CreatedBy, ticket.CreatedAt, ticket.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to insert test ticket: %v", err)
		}

		assignReq := map[string]string{
			"assigned_to": "admin-123",
		}

		body, _ := json.Marshal(assignReq)
		req := httptest.NewRequest("POST", "/api/v1/tickets/"+ticket.ID+"/assign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Read response body to understand the error
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, resp.StatusCode, string(body))
		}

		// Read the response body first
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("Failed to read response body: %v", readErr)
		}

		
		// For Update operations (Assign/Resolve), the response is the ticket directly (not wrapped)
		var updatedTicket domain.Ticket
		if err := json.Unmarshal(body, &updatedTicket); err != nil {
			t.Fatalf("Failed to decode response: %v\nResponse body: %s", err, string(body))
		}

		if updatedTicket.AssignedTo == nil || *updatedTicket.AssignedTo != "admin-123" {
			t.Errorf("Expected assigned_to 'admin-123', got '%v'", updatedTicket.AssignedTo)
		}

		if updatedTicket.Status != domain.TicketStatusInProgress {
			t.Errorf("Expected status %s, got %s", domain.TicketStatusInProgress, updatedTicket.Status)
		}
	})

	t.Run("ResolveTicket", func(t *testing.T) {
		// Create and assign a ticket first
		ticketID := uuid.New().String()
		assignedTo := "admin-123"
		ticket := domain.Ticket{
			ID:          ticketID,
			Title:       "Test Resolution",
			Description: "Test resolution description",
			Status:      domain.TicketStatusInProgress,
			Category:    domain.TicketCategoryOther,
			Priority:    domain.TicketPriorityMedium,
			CreatedBy:   "user1",
			AssignedTo:  &assignedTo,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		_, err := suite.db.ExecContext(suite.ctx, `
			INSERT INTO tickets (id, title, description, status, category, priority, created_by, assigned_to, ai_insight, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9, $10)
		`, ticket.ID, ticket.Title, ticket.Description, ticket.Status, ticket.Category, ticket.Priority, ticket.CreatedBy, ticket.AssignedTo, ticket.CreatedAt, ticket.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to insert test ticket: %v", err)
		}

		resolveReq := map[string]string{
			"resolution": "Issue has been resolved by updating the software configuration",
		}

		body, _ := json.Marshal(resolveReq)
		req := httptest.NewRequest("POST", "/api/v1/tickets/"+ticket.ID+"/resolve", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Read response body to understand the error
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, resp.StatusCode, string(body))
		}

		// Read the response body first
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("Failed to read response body: %v", readErr)
		}

		
		// For Update operations (Assign/Resolve), the response is the ticket directly (not wrapped)
		var resolvedTicket domain.Ticket
		if err := json.Unmarshal(body, &resolvedTicket); err != nil {
			t.Fatalf("Failed to decode response: %v\nResponse body: %s", err, string(body))
		}

		if resolvedTicket.Status != domain.TicketStatusResolved {
			t.Errorf("Expected status %s, got %s", domain.TicketStatusResolved, resolvedTicket.Status)
		}
	})
}

// TestAIAPI tests all AI-related endpoints
func TestAIAPI(t *testing.T) {
	suite := setupFixoraIntegrationTest(t)
	defer suite.cleanup(t)

	handler := suite.server.GetHandler()

	t.Run("GetSuggestion", func(t *testing.T) {
		suggestionReq := map[string]string{
			"description": "Computer is running very slow and applications take too long to open",
		}

		body, _ := json.Marshal(suggestionReq)
		req := httptest.NewRequest("POST", "/api/v1/ai/suggest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		// Should return 200 or 204 (No Content) based on confidence
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			// Read response body to understand the error
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200 or 204, got %d. Response: %s", resp.StatusCode, string(body))
		}

		if resp.StatusCode == http.StatusOK {
			// Read response body
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				t.Fatalf("Failed to read response body: %v", readErr)
			}

			var suggestion map[string]interface{}
			if err := json.Unmarshal(body, &suggestion); err != nil {
				t.Fatalf("Failed to decode response: %v\nResponse body: %s", err, string(body))
			}

			// Check that suggestion contains expected fields based on actual response format
			if _, ok := suggestion["category"]; !ok {
				t.Error("Expected 'category' field in suggestion response")
			}
			if _, ok := suggestion["suggestion"]; !ok {
				t.Error("Expected 'suggestion' field in suggestion response")
			}
			if _, ok := suggestion["confidence"]; !ok {
				t.Error("Expected 'confidence' field in suggestion response")
			}
			if _, ok := suggestion["source"]; !ok {
				t.Error("Expected 'source' field in suggestion response")
			}
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ai/health", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := health["healthy"]; !ok {
			t.Error("Expected 'healthy' field in health response")
		}
	})

	t.Run("GetProviderInfo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ai/info", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Read response body to see actual format
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("Failed to read response body: %v", readErr)
		}
		t.Logf("AI Info response: %s", string(body))

		var info map[string]interface{}
		if err := json.Unmarshal(body, &info); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check for available AI services instead of "provider" field
		if _, ok := info["embedding_service"]; !ok {
			t.Errorf("Expected 'embedding_service' field in info response. Got: %+v", info)
		}
		if _, ok := info["suggestion_service"]; !ok {
			t.Errorf("Expected 'suggestion_service' field in info response. Got: %+v", info)
		}
	})
}

// TestKnowledgeBaseAPI tests all knowledge base related endpoints
func TestKnowledgeBaseAPI(t *testing.T) {
	suite := setupFixoraIntegrationTest(t)
	defer suite.cleanup(t)

	handler := suite.server.GetHandler()

	t.Run("CreateEntry", func(t *testing.T) {
		createReq := map[string]interface{}{
			"title":    "How to Reset Password",
			"content":  "To reset your password, go to the login page and click 'Forgot Password'",
			"category": "ACCOUNT",
			"tags":     []string{"password", "reset", "account"},
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/kb/entries", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "admin-user")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}

		var entry map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if entry["title"] != "How to Reset Password" {
			t.Errorf("Expected title 'How to Reset Password', got '%v'", entry["title"])
		}
	})

	t.Run("SearchEntries", func(t *testing.T) {
		// First create an entry
		createReq := map[string]interface{}{
			"title":    "Network Connectivity Issues",
			"content":  "Troubleshooting steps for network connectivity problems including Wi-Fi and Ethernet",
			"category": "NETWORK",
			"tags":     []string{"network", "connectivity", "troubleshooting"},
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/kb/entries", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "admin-user")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		w.Result().Body.Close()

		// Now search
		searchReq := map[string]interface{}{
			"query": "network troubleshooting",
		}

		body, _ = json.Marshal(searchReq)
		req = httptest.NewRequest("POST", "/api/v1/kb/search", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var searchResult map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := searchResult["results"]; !ok {
			t.Error("Expected 'results' field in search response")
		}

		if _, ok := searchResult["count"]; !ok {
			t.Error("Expected 'count' field in search response")
		}
	})
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	suite := setupFixoraIntegrationTest(t)
	defer suite.cleanup(t)

	handler := suite.server.GetHandler()

	t.Run("InvalidTicketID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tickets/invalid-id", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		createReq := map[string]interface{}{
			"title": "", // Empty title should cause validation error
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/tickets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "test-user")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		// Should return bad request for missing required fields
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
}

// Helper functions

func initFixoraRepositories(db *sql.DB, cfg *config.Config) struct {
	Ticket    ports.TicketRepository
	Comment   ports.CommentRepository
	Knowledge ports.KnowledgeRepository
} {
	return struct {
		Ticket    ports.TicketRepository
		Comment   ports.CommentRepository
		Knowledge ports.KnowledgeRepository
	}{
		Ticket:    persistence.NewPostgresTicketRepository(db),
		Comment:   persistence.NewPostgresCommentRepository(db),
		Knowledge: persistence.NewPostgresKnowledgeRepository(db, nil),
	}
}

func initFixoraAIServices(cfg *config.Config) ports.AIProviderFactory {
	aiConfig := cfg.ToAIConfig()
	return ai.NewMockAIProviderFactory(aiConfig)
}

func initFixoraUseCases(repos struct {
	Ticket    ports.TicketRepository
	Comment   ports.CommentRepository
	Knowledge ports.KnowledgeRepository
}, aiFactory ports.AIProviderFactory, streamer *sse.Streamer) struct {
	Ticket    *usecase.TicketUseCase
	AI        *usecase.AIUseCase
	Knowledge *usecase.KnowledgeUseCase
} {
	ticketUseCase := usecase.NewTicketUseCase(
		repos.Ticket,
		repos.Comment,
		aiFactory.Suggestion(),
		nil, // Event publisher
		nil, // Notification service
	)

	aiUseCase := usecase.NewAIUseCase(
		aiFactory.Suggestion(),
		aiFactory.Embeddings(),
		repos.Knowledge,
		repos.Ticket,
		aiFactory.Training(),
	)

	knowledgeUseCase := usecase.NewKnowledgeUseCase(
		repos.Knowledge,
		aiFactory.Embeddings(),
		nil, // Event publisher
	)

	return struct {
		Ticket    *usecase.TicketUseCase
		AI        *usecase.AIUseCase
		Knowledge *usecase.KnowledgeUseCase
	}{
		Ticket:    ticketUseCase,
		AI:        aiUseCase,
		Knowledge: knowledgeUseCase,
	}
}

func ensureTestDatabaseAndSchemaForFixora(cfg *config.Config) (*sql.DB, error) {
	dbURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// First connect to postgres database to create test database if it doesn't exist
	adminURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.SSLMode,
	)

	adminDB, err := sql.Open("postgres", adminURL)
	if err != nil {
		return nil, fmt.Errorf("connect admin db: %w", err)
	}
	defer adminDB.Close()

	// Check if test database exists
	var exists int
	err = adminDB.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", cfg.Database.DBName).Scan(&exists)
	if err != nil {
		// Database doesn't exist, create it
		createStmt := fmt.Sprintf("CREATE DATABASE \"%s\"", cfg.Database.DBName)
		if _, err := adminDB.Exec(createStmt); err != nil {
			return nil, fmt.Errorf("create database: %w", err)
		}
	}

	// Connect to test database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect test db: %w", err)
	}

	// Create schema
	if err := ensureFixoraSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func ensureFixoraSchema(db *sql.DB) error {
	// Create vector extension if not exists
	if _, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		// Log warning but continue - vector extension might not be available for testing
		fmt.Printf("Warning: Could not create vector extension: %v\n", err)
	}

	// Drop existing tables for clean test environment
	tables := []string{"comments", "tickets", "knowledge_chunks", "knowledge_entries"}
	for _, table := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}

	// Create knowledge_entries table
	kbEntries := `
	CREATE TABLE knowledge_entries (
		id VARCHAR(255) PRIMARY KEY,
		title VARCHAR(500) NOT NULL,
		content TEXT NOT NULL,
		category VARCHAR(100),
		tags TEXT[],
		status VARCHAR(50) NOT NULL DEFAULT 'draft',
		created_by VARCHAR(255) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(kbEntries); err != nil {
		return fmt.Errorf("create knowledge_entries table: %w", err)
	}

	// Create knowledge_chunks table (without vector initially for compatibility)
	kbChunks := `
	CREATE TABLE knowledge_chunks (
		id VARCHAR(255) PRIMARY KEY,
		entry_id VARCHAR(255) NOT NULL,
		chunk_index INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (entry_id) REFERENCES knowledge_entries(id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(kbChunks); err != nil {
		return fmt.Errorf("create knowledge_chunks table: %w", err)
	}

	// Try to add embedding column if vector extension is available
	if _, err := db.Exec("ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS embedding vector(1536)"); err != nil {
		// If vector extension is not available, continue without embedding column
		fmt.Printf("Warning: Could not add embedding column: %v\n", err)
	}

	// Create tickets table
	tickets := `
	CREATE TABLE tickets (
		id VARCHAR(255) PRIMARY KEY,
		title VARCHAR(500) NOT NULL,
		description TEXT NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
		category VARCHAR(100) NOT NULL,
		priority VARCHAR(50) NOT NULL DEFAULT 'MEDIUM',
		created_by VARCHAR(255) NOT NULL,
		assigned_to VARCHAR(255),
		ai_insight JSONB DEFAULT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(tickets); err != nil {
		return fmt.Errorf("create tickets table: %w", err)
	}

	// Create comments table
	comments := `
	CREATE TABLE comments (
		id VARCHAR(255) PRIMARY KEY,
		ticket_id VARCHAR(255) NOT NULL,
		content TEXT NOT NULL,
		created_by VARCHAR(255) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(comments); err != nil {
		return fmt.Errorf("create comments table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX idx_knowledge_entries_status ON knowledge_entries(status);",
		"CREATE INDEX idx_knowledge_entries_category ON knowledge_entries(category);",
		"CREATE INDEX idx_knowledge_chunks_entry_id ON knowledge_chunks(entry_id);",
		"CREATE INDEX idx_tickets_status ON tickets(status);",
		"CREATE INDEX idx_tickets_category ON tickets(category);",
		"CREATE INDEX idx_tickets_priority ON tickets(priority);",
		"CREATE INDEX idx_tickets_created_by ON tickets(created_by);",
		"CREATE INDEX idx_tickets_assigned_to ON tickets(assigned_to);",
		"CREATE INDEX idx_comments_ticket_id ON comments(ticket_id);",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func cleanupFixoraDatabase(db *sql.DB) error {
	tables := []string{"comments", "tickets", "knowledge_chunks", "knowledge_entries"}

	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("truncate table %s: %w", table, err)
		}
	}

	return nil
}