package ai

import (
    "context"
    "fmt"
    "math/rand"
    "strings"
    "sync"
    "time"

    outbound "github.com/fixora/fixora/application/port/outbound"
)

// MockAIService provides a mock implementation of AI services for testing
type MockAIService struct {
    enabled   bool
    latency   time.Duration
    errorRate float64
    cache     sync.Map
}

// NewMockAIService creates a new mock AI service
func NewMockAIService(config outbound.AIConfig) *MockAIService {
    return &MockAIService{
        enabled:   true,
        latency:   time.Duration(config.TimeoutMs) * time.Millisecond,
        errorRate: 0.1,
    }
}

// SuggestMitigation provides mock AI-powered mitigation suggestions
func (m *MockAIService) SuggestMitigation(ctx context.Context, description string) (outbound.SuggestionResult, error) {
    if !m.enabled {
        return outbound.SuggestionResult{}, fmt.Errorf("AI service disabled")
    }

    select {
    case <-time.After(m.latency):
    case <-ctx.Done():
        return outbound.SuggestionResult{}, ctx.Err()
    }

    if rand.Float64() < m.errorRate {
        return outbound.SuggestionResult{}, fmt.Errorf("mock AI service error")
    }

    if cached, ok := m.cache.Load(description); ok {
        if result, ok := cached.(outbound.SuggestionResult); ok {
            result.UsedCache = true
            return result, nil
        }
    }

    suggestion, confidence, category := m.generateMockSuggestion(description)

    result := outbound.SuggestionResult{
        Suggestion: suggestion,
        Confidence: confidence,
        Category:   category,
        Source:     "mock",
        UsedCache:  false,
    }

    m.cache.Store(description, result)
    return result, nil
}

// StreamSuggestionMitigation provides streaming mock AI suggestions
func (m *MockAIService) StreamSuggestionMitigation(ctx context.Context, description string) (<-chan outbound.SuggestionEvent, error) {
    if !m.enabled {
        return nil, fmt.Errorf("AI service disabled")
    }

    eventChan := make(chan outbound.SuggestionEvent, 10)
    queryID := fmt.Sprintf("mock_%d", time.Now().UnixNano())

    go func() {
        defer close(eventChan)

        select {
        case eventChan <- outbound.SuggestionEvent{
            Type:    "init",
            QueryID: queryID,
            Data: map[string]interface{}{
                "query":     description,
                "startTime": time.Now().Unix(),
            },
        }:
        case <-ctx.Done():
            return
        }

        time.Sleep(100 * time.Millisecond)
        candidates := m.generateMockCandidates(description)

        start := time.Now()
        for i, candidate := range candidates {
            select {
            case eventChan <- outbound.SuggestionEvent{Type: "candidate", QueryID: queryID, Data: candidate}:
                time.Sleep(50 * time.Millisecond)
            case <-ctx.Done():
                return
            }

            if i%2 == 0 {
                select {
                case eventChan <- outbound.SuggestionEvent{
                    Type:    "progress",
                    QueryID: queryID,
                    Data: outbound.ProgressData{RetrievedCount: i + 1, ElapsedMs: int64(time.Since(start).Milliseconds())},
                }:
                case <-ctx.Done():
                    return
                }
            }
        }

        select {
        case eventChan <- outbound.SuggestionEvent{
            Type:    "end",
            QueryID: queryID,
            Data: outbound.EndData{TotalCandidates: len(candidates), ElapsedMs: int64(m.latency.Milliseconds())},
        }:
        case <-ctx.Done():
            return
        }
    }()

    return eventChan, nil
}

// ValidateProvider checks if the mock AI service is available
func (m *MockAIService) ValidateProvider(ctx context.Context) error {
    if m.enabled {
        return nil
    }
    return fmt.Errorf("mock AI service not available")
}

// MockEmbeddingProvider provides mock embedding generation
type MockEmbeddingProvider struct {
    dimension int
    enabled   bool
}

// NewMockEmbeddingProvider creates a new mock embedding provider
func NewMockEmbeddingProvider(dimension int) *MockEmbeddingProvider {
    return &MockEmbeddingProvider{dimension: dimension, enabled: true}
}

func (m *MockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    if !m.enabled {
        return nil, fmt.Errorf("embedding provider disabled")
    }
    embedding := make([]float32, m.dimension)
    hash := simpleHash(text)
    rand.Seed(int64(hash))
    for i := range embedding {
        embedding[i] = rand.Float32()*2 - 1
    }
    return embedding, nil
}

func (m *MockEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    if !m.enabled {
        return nil, fmt.Errorf("embedding provider disabled")
    }
    embeddings := make([][]float32, len(texts))
    for i, text := range texts {
        e, err := m.Embed(ctx, text)
        if err != nil {
            return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
        }
        embeddings[i] = e
    }
    return embeddings, nil
}

func (m *MockEmbeddingProvider) Dimension() int { return m.dimension }
func (m *MockEmbeddingProvider) ValidateEmbedding(embedding []float32) bool { return len(embedding) == m.dimension }

// MockTrainingService provides mock AI training capabilities
type MockTrainingService struct{ enabled bool }

func NewMockTrainingService() *MockTrainingService { return &MockTrainingService{enabled: true} }

func (m *MockTrainingService) LearnFromResolved(ctx context.Context, ticket *outbound.TicketTrainingData) error {
    if !m.enabled { return fmt.Errorf("training service disabled") }
    select { case <-time.After(100 * time.Millisecond): case <-ctx.Done(): return ctx.Err() }
    return nil
}

func (m *MockTrainingService) LearnFromKnowledge(ctx context.Context, entry *outbound.KnowledgeTrainingData) error {
    if !m.enabled { return fmt.Errorf("training service disabled") }
    select { case <-time.After(100 * time.Millisecond): case <-ctx.Done(): return ctx.Err() }
    return nil
}

func (m *MockTrainingService) ValidateTraining(ctx context.Context, data interface{}) error {
    if data == nil { return fmt.Errorf("training data cannot be nil") }
    return nil
}

// MockAIProviderFactory wires mock services
type MockAIProviderFactory struct { aiConfig outbound.AIConfig }

func NewMockAIProviderFactory(config outbound.AIConfig) outbound.AIProviderFactory { return &MockAIProviderFactory{aiConfig: config} }
func (f *MockAIProviderFactory) Suggestion() outbound.AISuggestionService { return NewMockAIService(f.aiConfig) }
func (f *MockAIProviderFactory) Embeddings() outbound.EmbeddingProvider { return NewMockEmbeddingProvider(f.aiConfig.EmbeddingDim) }
func (f *MockAIProviderFactory) Training() outbound.AITrainingService { return NewMockTrainingService() }
func (f *MockAIProviderFactory) Provider() string { return f.aiConfig.Provider }
func (f *MockAIProviderFactory) IsHealthy(ctx context.Context) error { return nil }

func (m *MockAIService) generateMockSuggestion(description string) (string, float64, string) {
    desc := strings.ToLower(description)
    switch {
    case strings.Contains(desc, "network"):
        return "Periksa koneksi dan restart router.", 0.82, "NETWORK"
    case strings.Contains(desc, "password") || strings.Contains(desc, "login"):
        return "Reset password dan cek kebijakan SSO.", 0.78, "ACCOUNT"
    case strings.Contains(desc, "printer"):
        return "Instal ulang driver dan cek kabel.", 0.75, "HARDWARE"
    default:
        return "Kumpulkan log dan coba restart aplikasi.", 0.65, "SOFTWARE"
    }
}

func (m *MockAIService) generateMockCandidates(description string) []outbound.CandidateData {
    _ = strings.ToLower(description)
    candidates := []outbound.CandidateData{
        {Rank: 1, Score: 0.82, Suggestion: "Cek konektivitas dan DNS", Category: "NETWORK"},
        {Rank: 2, Score: 0.76, Suggestion: "Perbarui driver terkait", Category: "SOFTWARE"},
        {Rank: 3, Score: 0.68, Suggestion: "Restart service terkait", Category: "OTHER"},
    }
    return candidates
}

func generateTitleFromDescription(desc string) string {
    d := strings.TrimSpace(desc)
    if len(d) > 60 { d = d[:60] }
    return d
}

func estimatePriority(desc string) string {
    d := strings.ToLower(desc)
    switch {
    case strings.Contains(d, "urgent") || strings.Contains(d, "critical"):
        return "CRITICAL"
    case strings.Contains(d, "slow") || strings.Contains(d, "error"):
        return "HIGH"
    default:
        return "MEDIUM"
    }
}

func simpleHash(s string) uint32 {
    var h uint32
    for i := 0; i < len(s); i++ { h = h*31 + uint32(s[i]) }
    return h
}

func (m *MockAIService) PredictAttributes(ctx context.Context, description string) (outbound.PredictedAttributes, error) {
    title := generateTitleFromDescription(description)
    priority := estimatePriority(description)
    category := "OTHER"
    return outbound.PredictedAttributes{
        Title: outbound.FieldPrediction{Value: title, Confidence: 0.6, Source: "mock-heuristic"},
        Category: outbound.FieldPrediction{Value: category, Confidence: 0.5, Source: "mock-heuristic"},
        Priority: outbound.FieldPrediction{Value: priority, Confidence: 0.55, Source: "mock-heuristic"},
    }, nil
}