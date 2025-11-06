package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/fixora/fixora/internal/adapter/ai"
	"github.com/fixora/fixora/internal/adapter/http"
	"github.com/fixora/fixora/internal/adapter/persistence"
	"github.com/fixora/fixora/internal/config"
	"github.com/fixora/fixora/internal/infra/sse"
	"github.com/fixora/fixora/internal/usecase"
)

func main() {
	fmt.Println("🧪 Testing Fixora API Structure...")

	// Initialize mock dependencies
	aiFactory := ai.NewMockAIProviderFactory(config.DefaultAIConfig())
	streamer := sse.NewStreamer()

	// Initialize mock repositories (these would normally connect to database)
	ticketRepo := &persistence.MockTicketRepository{}
	commentRepo := &persistence.MockCommentRepository{}
	kbRepo := &persistence.MockKnowledgeRepository{}

	// Initialize use cases
	ticketUseCase := usecase.NewTicketUseCase(ticketRepo, commentRepo, aiFactory.Suggestion(), nil, nil)
	aiUseCase := usecase.NewAIUseCase(aiFactory.Suggestion(), aiFactory.Embeddings(), kbRepo, ticketRepo, aiFactory.Training())
	kbUseCase := usecase.NewKnowledgeUseCase(kbRepo, aiFactory.Embeddings(), nil)

	// Initialize HTTP server with test config
	serverConfig := http.ServerConfig{
		Port:         "8080",
		ReadTimeout:  config.DefaultReadTimeout,
		WriteTimeout: config.DefaultWriteTimeout,
		IdleTimeout:  config.DefaultIdleTimeout,
	}

	server := http.NewServer(serverConfig, ticketUseCase, aiUseCase, kbUseCase)

	fmt.Println("✅ Server initialized successfully")

	// Test basic endpoints
	testEndpoints(server)
}

func testEndpoints(server *http.Server) {
	fmt.Println("\n📡 Testing API Endpoints...")

	endpoints := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "Health Check",
			method: "GET",
			path:   "/health",
			body:   "",
		},
		{
			name:   "AI Health Check",
			method: "GET",
			path:   "/api/v1/ai/health",
			body:   "",
		},
		{
			name:   "AI Provider Info",
			method: "GET",
			path:   "/api/v1/ai/info",
			body:   "",
		},
		{
			name:   "AI Suggestion",
			method: "POST",
			path:   "/api/v1/ai/suggest",
			body:   `{"description": "My computer won't connect to WiFi"}`,
		},
		{
			name:   "Create Ticket",
			method: "POST",
			path:   "/api/v1/tickets",
			body:   `{"title":"WiFi Issue","description":"Cannot connect to office WiFi network","category":"Network","priority":"Medium","created_by":"test-user"}`,
		},
		{
			name:   "List Tickets",
			method: "GET",
			path:   "/api/v1/tickets",
			body:   "",
		},
		{
			name:   "Get Ticket Stats",
			method: "GET",
			path:   "/api/v1/tickets/stats",
			body:   "",
		},
		{
			name:   "Create Knowledge Entry",
			method: "POST",
			path:   "/api/v1/kb/entries",
			body:   `{"title":"WiFi Troubleshooting Guide","content":"Step 1: Restart router...\nStep 2: Check password...\n","category":"Network","tags":["wifi","network"]}`,
		},
		{
			name:   "List Knowledge Entries",
			method: "GET",
			path:   "/api/v1/kb/entries",
			body:   "",
		},
	}

	for _, endpoint := range endpoints {
		testEndpoint(server, endpoint)
	}
}

func testEndpoint(server *http.Server, endpoint struct {
	name   string
	method string
	path   string
	body   string
}) {
	fmt.Printf("\n🔍 Testing: %s [%s %s]\n", endpoint.name, endpoint.method, endpoint.path)

	var bodyReader *strings.Reader
	if endpoint.body != "" {
		bodyReader = strings.NewReader(endpoint.body)
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(endpoint.method, endpoint.path, bodyReader)
	if endpoint.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-User-ID", "test-user")

	w := httptest.NewRecorder()

	// Get the underlying http.Server's handler
	handler := getServerHandler(server)
	handler.ServeHTTP(w, req)

	fmt.Printf("Status: %d %s\n", w.Code, http.StatusText(w.Code))

	if w.Code >= 200 && w.Code < 300 {
		fmt.Printf("✅ Success\n")
	} else if w.Code >= 400 {
		fmt.Printf("❌ Error Response\n")
	}

	// Pretty print response body if it's JSON
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") && w.Body.Len() > 0 {
		var prettyJSON interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &prettyJSON); err == nil {
			indented, _ := json.MarshalIndent(prettyJSON, "  ", "  ")
			fmt.Printf("Response: %s\n", string(indented))
		} else {
			fmt.Printf("Response: %s\n", w.Body.String())
		}
	} else if w.Body.Len() > 0 {
		fmt.Printf("Response: %s\n", w.Body.String())
	}
}

func getServerHandler(server *http.Server) http.Handler {
	// This is a bit of a hack to extract the handler from the server
	// In a real test, you'd probably access this more directly
	return server.Handler
}