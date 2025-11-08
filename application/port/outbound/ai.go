package outbound

import (
    "context"
)

// AISuggestionService defines the interface for AI suggestion services
type AISuggestionService interface {
    // SuggestMitigation provides AI-powered mitigation suggestions
    SuggestMitigation(ctx context.Context, description string) (SuggestionResult, error)

    // StreamSuggestionMitigation provides streaming AI suggestions
    StreamSuggestionMitigation(ctx context.Context, description string) (<-chan SuggestionEvent, error)

    // ValidateProvider checks if the AI service is available
    ValidateProvider(ctx context.Context) error

    // PredictAttributes predicts ticket title, category, and priority from description
    PredictAttributes(ctx context.Context, description string) (PredictedAttributes, error)
}

// AITrainingService defines the interface for AI training services
type AITrainingService interface {
    // LearnFromResolved trains the AI model with resolved ticket data
    LearnFromResolved(ctx context.Context, ticket *TicketTrainingData) error

    // LearnFromKnowledge trains the AI with knowledge base updates
    LearnFromKnowledge(ctx context.Context, entry *KnowledgeTrainingData) error

    // ValidateTraining checks if training data is valid
    ValidateTraining(ctx context.Context, data interface{}) error
}

// EmbeddingProvider defines the interface for text embedding services
type EmbeddingProvider interface {
    // Embed generates embedding vector for a single text
    Embed(ctx context.Context, text string) ([]float32, error)

    // EmbedBatch generates embedding vectors for multiple texts
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

    // Dimension returns the dimension of the embedding vectors
    Dimension() int

    // ValidateEmbedding checks if embedding dimension is correct
    ValidateEmbedding(embedding []float32) bool
}

// AIProviderFactory creates AI service instances based on provider type
type AIProviderFactory interface {
    // Suggestion returns an AI suggestion service
    Suggestion() AISuggestionService

    // Embeddings returns an embedding provider
    Embeddings() EmbeddingProvider

    // Training returns an AI training service
    Training() AITrainingService

    // Provider returns the current provider type
    Provider() string

    // IsHealthy checks if the AI services are healthy
    IsHealthy(ctx context.Context) error
}

// Data structures for AI services

// SuggestionResult represents the result of an AI suggestion
type SuggestionResult struct {
    Suggestion string  `json:"suggestion"`
    Confidence float64 `json:"confidence"`
    Category   string  `json:"category,omitempty"`
    Source     string  `json:"source"`
    UsedCache  bool    `json:"used_cache"`
}

// SuggestionEvent represents a streaming suggestion event
type SuggestionEvent struct {
    Type    string      `json:"type"`    // init, candidate, progress, end, error
    QueryID string      `json:"query_id"`
    Data    interface{} `json:"data"`
    Error   string      `json:"error,omitempty"`
}

// CandidateData represents a suggestion candidate
type CandidateData struct {
    Rank        int     `json:"rank"`
    Score       float64 `json:"score"`
    Suggestion  string  `json:"suggestion"`
    Category    string  `json:"category,omitempty"`
    EntryID     string  `json:"entry_id,omitempty"`
    ChunkIndex  int     `json:"chunk_index,omitempty"`
}

// ProgressData represents search progress
type ProgressData struct {
    RetrievedCount int   `json:"retrieved_count"`
    ElapsedMs      int64 `json:"elapsed_ms"`
}

// EndData represents stream completion
type EndData struct {
    TotalCandidates int   `json:"total_candidates"`
    ElapsedMs       int64 `json:"elapsed_ms"`
}

// PredictedAttributes represents AI predictions for ticket fields
type PredictedAttributes struct {
    Title    FieldPrediction `json:"title"`
    Category FieldPrediction `json:"category"`
    Priority FieldPrediction `json:"priority"`
}

// FieldPrediction represents a predicted value with confidence
type FieldPrediction struct {
    Value      string  `json:"value"`
    Confidence float64 `json:"confidence"`
    Source     string  `json:"source,omitempty"` // e.g., provider name
}

// TicketTrainingData represents training data from resolved tickets
type TicketTrainingData struct {
    TicketID    string   `json:"ticket_id"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Category    string   `json:"category"`
    Solution    string   `json:"solution"`
    Comments    []string `json:"comments"`
    Resolution  string   `json:"resolution"`
    Rating      *int     `json:"rating,omitempty"` // User satisfaction rating
}

// KnowledgeTrainingData represents training data from knowledge base
type KnowledgeTrainingData struct {
    EntryID   string   `json:"entry_id"`
    Title     string   `json:"title"`
    Content   string   `json:"content"`
    Category  string   `json:"category"`
    Tags      []string `json:"tags"`
    Solution  string   `json:"solution"`
}

// AI Configuration
type AIConfig struct {
    Provider          string  `json:"provider"`
    APIKey           string  `json:"api_key"`
    EmbeddingModel   string  `json:"embedding_model"`
    SuggestionModel  string  `json:"suggestion_model"`
    EmbeddingDim     int     `json:"embedding_dim"`
    TopK             int     `json:"top_k"`
    MinConfidence    float64 `json:"min_confidence"`
    TimeoutMs        int     `json:"timeout_ms"`
    EnableCache      bool    `json:"enable_cache"`
    CacheTTLMin      int     `json:"cache_ttl_min"`
}

// Default AI configuration
func DefaultAIConfig() AIConfig {
    return AIConfig{
        Provider:         "openai",
        EmbeddingModel:   "text-embedding-ada-002",
        SuggestionModel:  "gpt-3.5-turbo",
        EmbeddingDim:     1536,
        TopK:            10,
        MinConfidence:   0.4,
        TimeoutMs:       5000,
        EnableCache:     true,
        CacheTTLMin:     60,
    }
}

// AI Service errors
const (
    ErrAIUnavailable      = "AI service unavailable"
    ErrInvalidAPIKey     = "invalid API key"
    ErrInsufficientCredits = "insufficient API credits"
    ErrModelNotFound     = "model not found"
    ErrRequestTimeout    = "request timeout"
    ErrRateLimitExceeded = "rate limit exceeded"
    ErrInvalidEmbedding  = "invalid embedding dimension"
    ErrTrainingFailed    = "training failed"
)