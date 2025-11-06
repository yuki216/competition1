package http

import (
	"context"
	"log"
	"net/http"
	"time"

	"fixora/internal/usecase"

	"github.com/gorilla/mux"
)

// Server represents the HTTP server
type Server struct {
	addr         string
	ticketHandler *TicketHandler
	aiHandler    *AIHandler
	kbHandler    *KBHandler
	server       *http.Server
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewServer creates a new HTTP server
func NewServer(
	config ServerConfig,
	ticketUseCase *usecase.TicketUseCase,
	aiUseCase *usecase.AIUseCase,
	kbUseCase *usecase.KnowledgeUseCase, // Assuming you have this
) *Server {
	// Create handlers
	ticketHandler := NewTicketHandler(ticketUseCase)
	aiHandler := NewAIHandler(aiUseCase)
	kbHandler := NewKBHandler(kbUseCase)

	// Create router
	router := mux.NewRouter()

	// Register routes
	ticketHandler.RegisterRoutes(router)
	aiHandler.RegisterRoutes(router)
	kbHandler.RegisterRoutes(router)

	// Add middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)
	router.Use(recoveryMiddleware)

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	return &Server{
		addr:          ":" + config.Port,
		ticketHandler: ticketHandler,
		aiHandler:    aiHandler,
		kbHandler:    kbHandler,
		server: &http.Server{
			Addr:         ":" + config.Port,
			Handler:      router,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// Middleware

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}