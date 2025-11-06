package ai

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/fixora/fixora/internal/ports"
)

// MockAIService provides a mock implementation of AI services for testing
type MockAIService struct {
	enabled   bool
	latency   time.Duration
	errorRate float64
	cache     sync.Map
}

// NewMockAIService creates a new mock AI service
func NewMockAIService(config ports.AIConfig) *MockAIService {
	return &MockAIService{
		enabled:   true,
		latency:   time.Duration(config.TimeoutMs) * time.Millisecond,
		errorRate: 0.1, // 10% error rate for testing
	}
}

// SuggestMitigation provides mock AI-powered mitigation suggestions
func (m *MockAIService) SuggestMitigation(ctx context.Context, description string) (ports.SuggestionResult, error) {
	if !m.enabled {
		return ports.SuggestionResult{}, fmt.Errorf("AI service disabled")
	}

	// Simulate network latency
	select {
	case <-time.After(m.latency):
	case <-ctx.Done():
		return ports.SuggestionResult{}, ctx.Err()
	}

	// Simulate random errors
	if rand.Float64() < m.errorRate {
		return ports.SuggestionResult{}, fmt.Errorf("mock AI service error")
	}

	// Check cache first
	if cached, ok := m.cache.Load(description); ok {
		if result, ok := cached.(ports.SuggestionResult); ok {
			result.UsedCache = true
			return result, nil
		}
	}

	// Generate mock suggestion based on keywords in description
	suggestion, confidence, category := m.generateMockSuggestion(description)

	result := ports.SuggestionResult{
		Suggestion: suggestion,
		Confidence: confidence,
		Category:   category,
		Source:     "mock",
		UsedCache:  false,
	}

	// Cache the result
	m.cache.Store(description, result)

	return result, nil
}

// StreamSuggestionMitigation provides streaming mock AI suggestions
func (m *MockAIService) StreamSuggestionMitigation(ctx context.Context, description string) (<-chan ports.SuggestionEvent, error) {
	if !m.enabled {
		return nil, fmt.Errorf("AI service disabled")
	}

	eventChan := make(chan ports.SuggestionEvent, 10)
	queryID := fmt.Sprintf("mock_%d", time.Now().UnixNano())

	// Start streaming in a goroutine
	go func() {
		defer close(eventChan)

		// Send init event
		select {
		case eventChan <- ports.SuggestionEvent{
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

		// Simulate search delay
		time.Sleep(100 * time.Millisecond)

		// Generate mock candidates
		candidates := m.generateMockCandidates(description)

		for i, candidate := range candidates {
			select {
			case eventChan <- ports.SuggestionEvent{
				Type:    "candidate",
				QueryID: queryID,
				Data:    candidate,
			}:
				// Small delay between candidates
				time.Sleep(50 * time.Millisecond)
			case <-ctx.Done():
				return
			}

			// Send progress updates
			if i%2 == 0 {
				select {
				case eventChan <- ports.SuggestionEvent{
					Type:    "progress",
					QueryID: queryID,
					Data: ports.ProgressData{
						RetrievedCount: i + 1,
						ElapsedMs:      int64(time.Since(time.Now()).Milliseconds()),
					},
				}:
				case <-ctx.Done():
					return
				}
			}
		}

		// Send end event
		select {
		case eventChan <- ports.SuggestionEvent{
			Type:    "end",
			QueryID: queryID,
			Data: ports.EndData{
				TotalCandidates: len(candidates),
				ElapsedMs:       int64(m.latency.Milliseconds()),
			},
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
	return &MockEmbeddingProvider{
		dimension: dimension,
		enabled:   true,
	}
}

// Embed generates mock embedding vector for a single text
func (m *MockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if !m.enabled {
		return nil, fmt.Errorf("embedding provider disabled")
	}

	// Generate deterministic but pseudo-random embeddings based on text hash
	embedding := make([]float32, m.dimension)
	hash := simpleHash(text)

	rand.Seed(int64(hash))
	for i := range embedding {
		embedding[i] = rand.Float32()*2 - 1 // Range: -1 to 1
	}

	return embedding, nil
}

// EmbedBatch generates mock embedding vectors for multiple texts
func (m *MockEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if !m.enabled {
		return nil, fmt.Errorf("embedding provider disabled")
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := m.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// Dimension returns the dimension of the embedding vectors
func (m *MockEmbeddingProvider) Dimension() int {
	return m.dimension
}

// ValidateEmbedding checks if embedding dimension is correct
func (m *MockEmbeddingProvider) ValidateEmbedding(embedding []float32) bool {
	return len(embedding) == m.dimension
}

// MockTrainingService provides mock AI training capabilities
type MockTrainingService struct {
	enabled bool
}

// NewMockTrainingService creates a new mock training service
func NewMockTrainingService() *MockTrainingService {
	return &MockTrainingService{
		enabled: true,
	}
}

// LearnFromResolved trains the AI model with resolved ticket data (mock)
func (m *MockTrainingService) LearnFromResolved(ctx context.Context, ticket *ports.TicketTrainingData) error {
	if !m.enabled {
		return fmt.Errorf("training service disabled")
	}

	// Simulate training time
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	// In a real implementation, this would:
	// 1. Extract features from the ticket data
	// 2. Update the AI model or knowledge base
	// 3. Store the training data for future reference
	// 4. Log the training process

	return nil
}

// LearnFromKnowledge trains the AI with knowledge base updates (mock)
func (m *MockTrainingService) LearnFromKnowledge(ctx context.Context, entry *ports.KnowledgeTrainingData) error {
	if !m.enabled {
		return fmt.Errorf("training service disabled")
	}

	// Simulate training time
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// ValidateTraining checks if training data is valid
func (m *MockTrainingService) ValidateTraining(ctx context.Context, data interface{}) error {
	if !m.enabled {
		return fmt.Errorf("training service disabled")
	}

	switch d := data.(type) {
	case *ports.TicketTrainingData:
		if d.TicketID == "" || d.Description == "" {
			return fmt.Errorf("invalid ticket training data")
		}
	case *ports.KnowledgeTrainingData:
		if d.EntryID == "" || d.Content == "" {
			return fmt.Errorf("invalid knowledge training data")
		}
	default:
		return fmt.Errorf("unknown training data type")
	}

	return nil
}

// MockAIProviderFactory creates mock AI service instances
type MockAIProviderFactory struct {
	aiConfig ports.AIConfig
}

// NewMockAIProviderFactory creates a new mock AI provider factory
func NewMockAIProviderFactory(config ports.AIConfig) ports.AIProviderFactory {
	return &MockAIProviderFactory{
		aiConfig: config,
	}
}

// Suggestion returns a mock AI suggestion service
func (f *MockAIProviderFactory) Suggestion() ports.AISuggestionService {
	return NewMockAIService(f.aiConfig)
}

// Embeddings returns a mock embedding provider
func (f *MockAIProviderFactory) Embeddings() ports.EmbeddingProvider {
	return NewMockEmbeddingProvider(f.aiConfig.EmbeddingDim)
}

// Training returns a mock AI training service
func (f *MockAIProviderFactory) Training() ports.AITrainingService {
	return NewMockTrainingService()
}

// Provider returns the current provider type
func (f *MockAIProviderFactory) Provider() string {
	return "mock"
}

// IsHealthy checks if the mock AI services are healthy
func (f *MockAIProviderFactory) IsHealthy(ctx context.Context) error {
	// Mock implementation always returns healthy
	return nil
}

// Helper functions

func (m *MockAIService) generateMockSuggestion(description string) (string, float64, string) {
	descLower := strings.ToLower(description)

	// Network issues
	if strings.Contains(descLower, "wifi") || strings.Contains(descLower, "network") || strings.Contains(descLower, "connection") {
		suggestions := []string{
			"Try restarting your router and modem",
			"Check if other devices can connect to the network",
			"Move closer to the WiFi router",
			"Update your network adapter drivers",
		}
		return suggestions[rand.Intn(len(suggestions))], 0.75 + rand.Float64()*0.2, "Network"
	}

	// Software issues
	if strings.Contains(descLower, "software") || strings.Contains(descLower, "application") || strings.Contains(descLower, "program") {
		suggestions := []string{
			"Try restarting the application",
			"Check for software updates",
			"Clear application cache and temporary files",
			"Reinstall the application if issues persist",
		}
		return suggestions[rand.Intn(len(suggestions))], 0.70 + rand.Float64()*0.2, "Software"
	}

	// Hardware issues
	if strings.Contains(descLower, "hardware") || strings.Contains(descLower, "device") || strings.Contains(descLower, "computer") {
		suggestions := []string{
			"Check if the device is properly connected",
			"Update device drivers",
			"Test the device on another computer",
			"Contact IT support for hardware diagnosis",
		}
		return suggestions[rand.Intn(len(suggestions))], 0.65 + rand.Float64()*0.2, "Hardware"
	}

	// Account issues
	if strings.Contains(descLower, "account") || strings.Contains(descLower, "login") || strings.Contains(descLower, "password") {
		suggestions := []string{
			"Try resetting your password",
			"Clear browser cache and cookies",
			"Try using a different browser",
			"Contact IT support for account assistance",
		}
		return suggestions[rand.Intn(len(suggestions))], 0.80 + rand.Float64()*0.15, "Account"
	}

	// Generic suggestion
	suggestions := []string{
		"Please provide more details about the issue you're experiencing",
		"Try restarting your computer",
		"Check if the issue is reproducible",
		"Contact IT support for further assistance",
	}
	return suggestions[rand.Intn(len(suggestions))], 0.50 + rand.Float64()*0.2, "General"
}

func (m *MockAIService) generateMockCandidates(description string) []ports.CandidateData {
	candidates := []ports.CandidateData{}
	suggestion, confidence, category := m.generateMockSuggestion(description)

	// Generate 3-5 candidates with varying scores
	numCandidates := 3 + rand.Intn(3)
	for i := 0; i < numCandidates; i++ {
		candidate := ports.CandidateData{
			Rank:       i + 1,
			Score:      confidence - float64(i)*0.1, // Decreasing scores
			Suggestion: suggestion,
			Category:   category,
			EntryID:    fmt.Sprintf("mock_entry_%d", i+1),
			ChunkIndex: i,
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

func simpleHash(s string) uint32 {
	hash := uint32(2166136261)
	for _, c := range s {
		hash ^= uint32(c)
		hash *= 16777619
	}
	return hash
}