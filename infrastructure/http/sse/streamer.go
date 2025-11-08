package sse

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/fixora/fixora/application/port/outbound"
)

// Streamer manages Server-Sent Events streaming
type Streamer struct {
    clients    map[string]*Client
    mu         sync.RWMutex
    register   chan *Client
    unregister chan *Client
    broadcast  chan []byte
}

// Client represents an SSE client connection
type Client struct {
    ID        string
    Channel   chan []byte
    Context   context.Context
    CloseFunc func()
    LastPing  time.Time
    mu        sync.Mutex
}

// NewStreamer creates a new SSE streamer
func NewStreamer() *Streamer {
    return &Streamer{
        clients:    make(map[string]*Client),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan []byte, 256),
    }
}

// Start starts the SSE streamer
func (s *Streamer) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(15 * time.Second) // Heartbeat every 15 seconds
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return

            case client := <-s.register:
                s.mu.Lock()
                s.clients[client.ID] = client
                s.mu.Unlock()

            case client := <-s.unregister:
                s.mu.Lock()
                if _, ok := s.clients[client.ID]; ok {
                    delete(s.clients, client.ID)
                    close(client.Channel)
                }
                s.mu.Unlock()

            case message := <-s.broadcast:
                s.mu.RLock()
                for _, client := range s.clients {
                    select {
                    case client.Channel <- message:
                    default:
                        // Client channel is full, remove it
                        go s.removeClient(client.ID)
                    }
                }
                s.mu.RUnlock()

            case <-ticker.C:
                // Send heartbeat to all clients
                s.sendHeartbeat()

            }
        }
    }()
}

// AddClient adds a new SSE client
func (s *Streamer) AddClient(clientID string) *Client {
    ctx, cancel := context.WithCancel(context.Background())

    client := &Client{
        ID:       clientID,
        Channel:  make(chan []byte, 256),
        Context:  ctx,
        CloseFunc: cancel,
        LastPing: time.Now(),
    }

    s.register <- client
    return client
}

// RemoveClient removes an SSE client
func (s *Streamer) RemoveClient(clientID string) {
    s.removeClient(clientID)
}

// Broadcast broadcasts a message to all clients
func (s *Streamer) Broadcast(eventType, queryID string, data interface{}) error {
    event := SSEEvent{
        Type:    eventType,
        QueryID: queryID,
        Data:    data,
        Time:    time.Now().Unix(),
    }

    message, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }

    select {
    case s.broadcast <- message:
        return nil
    default:
        return fmt.Errorf("broadcast channel is full")
    }
}

// SendEvent sends an event to a specific client
func (s *Streamer) SendEvent(clientID string, eventType string, data interface{}) error {
    event := SSEEvent{
        Type: eventType,
        Data: data,
        Time: time.Now().Unix(),
    }

    message, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }

    s.mu.RLock()
    client, ok := s.clients[clientID]
    s.mu.RUnlock()

    if !ok {
        return fmt.Errorf("client not found")
    }

    select {
    case client.Channel <- message:
        return nil
    default:
        return fmt.Errorf("client channel is full")
    }
}

// GetClientCount returns the number of connected clients
func (s *Streamer) GetClientCount() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.clients)
}

// HandleSSE handles SSE HTTP requests
func (s *Streamer) HandleSSE(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

    // Get client ID from query or generate one
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = fmt.Sprintf("client_%d", time.Now().UnixNano())
    }

    // Create client
    client := s.AddClient(clientID)
    defer s.RemoveClient(clientID)

    // Flush headers
    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }

    // Send initial connection event
    initEvent := map[string]interface{}{
        "client_id": clientID,
        "connected": true,
        "timestamp": time.Now().Unix(),
    }

    if err := s.writeSSEEvent(w, "connected", initEvent); err != nil {
        return
    }

    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }

    // Handle client messages
    for {
        select {
        case <-r.Context().Done():
            return

        case <-client.Context.Done():
            return

        case message := <-client.Channel:
            if err := s.writeSSEMessage(w, message); err != nil {
                return
            }

            if flusher, ok := w.(http.Flusher); ok {
                flusher.Flush()
            }

        // Timeout after 30 seconds of inactivity
        case <-time.After(30 * time.Second):
            if err := s.writeSSEComment(w, "timeout"); err != nil {
                return
            }
            if flusher, ok := w.(http.Flusher); ok {
                flusher.Flush()
            }
        }
    }
}

// StreamSuggestions streams AI suggestions to a client
func (s *Streamer) StreamSuggestions(w http.ResponseWriter, r *http.Request, suggestionChan <-chan outbound.SuggestionEvent) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // Get client ID
    clientID := r.URL.Query().Get("client_id")
    if clientID == "" {
        clientID = fmt.Sprintf("suggestion_%d", time.Now().UnixNano())
    }

    // Flush headers
    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }

    // Process suggestion events
    for event := range suggestionChan {
        if err := s.writeSSEEvent(w, event.Type, event); err != nil {
            return
        }

        if flusher, ok := w.(http.Flusher); ok {
            flusher.Flush()
        }
    }

    // Send end event
    endEvent := map[string]interface{}{
        "type": "stream_end",
        "timestamp": time.Now().Unix(),
    }

    s.writeSSEEvent(w, "end", endEvent)
    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }
}

func (s *Streamer) removeClient(clientID string) {
    s.mu.Lock()
    if client, ok := s.clients[clientID]; ok {
        if client.CloseFunc != nil {
            client.CloseFunc()
        }
        delete(s.clients, clientID)
        close(client.Channel)
    }
    s.mu.Unlock()
}

func (s *Streamer) sendHeartbeat() {
    s.mu.RLock()
    for _, client := range s.clients {
        client.mu.Lock()
        client.LastPing = time.Now()
        client.mu.Unlock()
    }
    s.mu.RUnlock()
}

func (s *Streamer) writeSSEEvent(w http.ResponseWriter, eventType string, data interface{}) error {
    payload := SSEEvent{Type: eventType, Data: data, Time: time.Now().Unix()}
    message, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    _, err = w.Write([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(message))))
    return err
}

func (s *Streamer) writeSSEMessage(w http.ResponseWriter, message []byte) error {
    _, err := w.Write([]byte(fmt.Sprintf("data: %s\n\n", string(message))))
    return err
}

func (s *Streamer) writeSSEComment(w http.ResponseWriter, comment string) error {
    _, err := w.Write([]byte(fmt.Sprintf(":%s\n\n", comment)))
    return err
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
    Type    string      `json:"type"`
    QueryID string      `json:"query_id,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Time    int64       `json:"time"`
}

// StreamingMetrics tracks streaming statistics
type StreamingMetrics struct {
    TotalConnections   int64     `json:"total_connections"`
    ActiveConnections  int64     `json:"active_connections"`
    MessagesSent       int64     `json:"messages_sent"`
    AverageLatency     int64     `json:"average_latency_ms"`
    LastConnectionTime time.Time `json:"last_connection_time"`
}

func (s *Streamer) GetMetrics() StreamingMetrics {
    // Placeholder metrics; enhance with atomic counters if needed
    return StreamingMetrics{
        TotalConnections:   int64(len(s.clients)),
        ActiveConnections:  int64(len(s.clients)),
        MessagesSent:       0,
        AverageLatency:     0,
        LastConnectionTime: time.Now(),
    }
}