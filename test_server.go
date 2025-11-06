package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/fixora/fixora/internal/adapter/ai"
	httpadapter "github.com/fixora/fixora/internal/adapter/http"
	"github.com/fixora/fixora/internal/config"
	"github.com/fixora/fixora/internal/infra/sse"
	"github.com/fixora/fixora/internal/usecase"
)

func main() {
	fmt.Println("🧪 Testing Fixora Server Structure...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		return
	}

	// Initialize mock dependencies
	aiFactory := ai.NewMockAIProviderFactory(cfg.ToAIConfig())
	_ = sse.NewStreamer() // Initialize but don't use for this test

	// Initialize minimal use cases (with nil repositories for basic structure testing)
	ticketUseCase := usecase.NewTicketUseCase(nil, nil, aiFactory.Suggestion(), nil, nil)
	aiUseCase := usecase.NewAIUseCase(aiFactory.Suggestion(), aiFactory.Embeddings(), nil, nil, aiFactory.Training())
	kbUseCase := usecase.NewKnowledgeUseCase(nil, aiFactory.Embeddings(), nil)

	// Initialize HTTP server with config
	serverConfig := httpadapter.ServerConfig{
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	server := httpadapter.NewServer(serverConfig, ticketUseCase, aiUseCase, kbUseCase)

	fmt.Println("✅ Server initialized successfully")

	// Test basic health endpoint
	testHealthEndpoint(server)

	// Test AI endpoints (which work with mocks)
	testAIEndpoints(server)
}

func testHealthEndpoint(server *httpadapter.Server) {
	fmt.Println("\n🏥 Testing Health Endpoint...")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := server.GetHandler()
	handler.ServeHTTP(w, req)

	fmt.Printf("Status: %d %s\n", w.Code, http.StatusText(w.Code))
	if w.Code == 200 {
		fmt.Printf("✅ Health check passed\n")
		fmt.Printf("Response: %s\n", w.Body.String())
	} else {
		fmt.Printf("❌ Health check failed\n")
	}
}

func testAIEndpoints(server *httpadapter.Server) {
	fmt.Println("\n🤖 Testing AI Endpoints...")

	endpoints := []struct {
		name   string
		method string
		path   string
		body   string
	}{
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
	}

	for _, endpoint := range endpoints {
		testEndpoint(server, endpoint)
	}
}

func testEndpoint(server *httpadapter.Server, endpoint struct {
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

	handler := server.GetHandler()
	handler.ServeHTTP(w, req)

	fmt.Printf("Status: %d %s\n", w.Code, http.StatusText(w.Code))

	if w.Code >= 200 && w.Code < 300 {
		fmt.Printf("✅ Success\n")
	} else if w.Code >= 400 {
		fmt.Printf("❌ Error Response\n")
	}

	if w.Body.Len() > 0 {
		fmt.Printf("Response: %s\n", w.Body.String())
	}
}