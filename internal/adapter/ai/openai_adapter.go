package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fixora/fixora/internal/ports"
)

// OpenAIAdapter implements AI services using OpenAI APIs
type OpenAIAdapter struct {
	apiKey         string
	baseURL        string
	model          string
	embeddingModel string
	embeddings     ports.EmbeddingProvider
	client         *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(config ports.AIConfig) ports.AIProviderFactory {
	client := &http.Client{
		Timeout: time.Duration(config.TimeoutMs) * time.Millisecond,
	}

	baseURL := "https://api.openai.com/v1"
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "text-embedding-ada-002"
	}
	if config.SuggestionModel == "" {
		config.SuggestionModel = "gpt-3.5-turbo"
	}

	adapter := &OpenAIAdapter{
		apiKey:         config.APIKey,
		baseURL:        baseURL,
		model:          config.SuggestionModel,
		embeddingModel: config.EmbeddingModel,
		client:         client,
	}

	// Create embedding provider
	adapter.embeddings = &OpenAIEmbeddingProvider{
		apiKey:     config.APIKey,
		model:      config.EmbeddingModel,
		dimension:  config.EmbeddingDim,
		baseURL:    baseURL,
		httpClient: client,
	}

	return adapter
}

// Suggestion returns an AI suggestion service
func (o *OpenAIAdapter) Suggestion() ports.AISuggestionService {
	return &OpenAISuggestionService{
		apiKey:     o.apiKey,
		model:      o.model,
		baseURL:    o.baseURL,
		httpClient: o.client,
	}
}

// Embeddings returns an embedding provider
func (o *OpenAIAdapter) Embeddings() ports.EmbeddingProvider {
	return o.embeddings
}

// Training returns an AI training service
func (o *OpenAIAdapter) Training() ports.AITrainingService {
	return &OpenAITrainingService{
		apiKey:     o.apiKey,
		baseURL:    o.baseURL,
		httpClient: o.client,
	}
}

// Provider returns the current provider type
func (o *OpenAIAdapter) Provider() string {
	return "openai"
}

// IsHealthy checks if the OpenAI services are healthy
func (o *OpenAIAdapter) IsHealthy(ctx context.Context) error {
	// Simple health check by testing API connectivity
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("OpenAI API health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI API returned status: %d", resp.StatusCode)
	}

	return nil
}

// OpenAISuggestionService implements AISuggestionService for OpenAI
type OpenAISuggestionService struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// SuggestMitigation provides AI-powered mitigation suggestions using OpenAI
func (s *OpenAISuggestionService) SuggestMitigation(ctx context.Context, description string) (ports.SuggestionResult, error) {
	prompt := fmt.Sprintf(`
As an IT support assistant, analyze the following issue and provide specific, actionable suggestions for mitigation:

Issue: %s

Please provide:
1. A brief analysis of the potential cause
2. 2-3 specific steps the user can take to mitigate the issue
3. A recommendation on whether this should be escalated to IT support

Be concise and practical. Focus on common IT issues and solutions.
`, description)

	requestBody := map[string]interface{}{
		"model": s.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an experienced IT support assistant providing helpful technical guidance."},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  300,
		"temperature": 0.7,
		"top_p":       1,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return ports.SuggestionResult{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return ports.SuggestionResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ports.SuggestionResult{}, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ports.SuggestionResult{}, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return ports.SuggestionResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return ports.SuggestionResult{}, fmt.Errorf("no choices in response")
	}

	// Calculate confidence based on response quality and token usage
	confidence := calculateConfidence(response.Choices[0].Message.Content, response.Usage.TotalTokens)

	// Determine category from content
	category := determineCategory(response.Choices[0].Message.Content, description)

	return ports.SuggestionResult{
		Suggestion: response.Choices[0].Message.Content,
		Confidence: confidence,
		Category:   category,
		Source:     "openai",
		UsedCache:  false,
	}, nil
}

// PredictAttributes predicts title, category, and priority based on description using heuristics.
func (s *OpenAISuggestionService) PredictAttributes(ctx context.Context, description string) (ports.PredictedAttributes, error) {
	title := oaGenerateTitle(description)
	category := determineCategory(title, description)
	priority := oaEstimatePriority(description)

	return ports.PredictedAttributes{
		Title: ports.FieldPrediction{
			Value:      title,
			Confidence: 0.68,
			Source:     "openai-heuristic",
		},
		Category: ports.FieldPrediction{
			Value:      category,
			Confidence: 0.72,
			Source:     "openai-heuristic",
		},
		Priority: ports.FieldPrediction{
			Value:      priority,
			Confidence: 0.70,
			Source:     "openai-heuristic",
		},
	}, nil
}

// StreamSuggestionMitigation provides streaming AI suggestions (simplified implementation)
func (s *OpenAISuggestionService) StreamSuggestionMitigation(ctx context.Context, description string) (<-chan ports.SuggestionEvent, error) {
	// For simplicity, this implementation uses the non-streaming API and simulates streaming
	// In a production environment, you would use OpenAI's streaming API
	eventChan := make(chan ports.SuggestionEvent, 10)
	queryID := fmt.Sprintf("openai_%d", time.Now().UnixNano())

	go func() {
		defer close(eventChan)

		// Send init event
		eventChan <- ports.SuggestionEvent{
			Type:    "init",
			QueryID: queryID,
			Data: map[string]interface{}{
				"query":     description,
				"startTime": time.Now().Unix(),
			},
		}

		// Get suggestion using non-streaming API
		result, err := s.SuggestMitigation(ctx, description)
		if err != nil {
			eventChan <- ports.SuggestionEvent{
				Type:    "error",
				QueryID: queryID,
				Error:   err.Error(),
			}
			return
		}

		// Send as single candidate (in real streaming, this would be multiple candidates)
		candidate := ports.CandidateData{
			Rank:       1,
			Score:      result.Confidence,
			Suggestion: result.Suggestion,
			Category:   result.Category,
		}

		eventChan <- ports.SuggestionEvent{
			Type:    "candidate",
			QueryID: queryID,
			Data:    candidate,
		}

		// Send end event
		eventChan <- ports.SuggestionEvent{
			Type:    "end",
			QueryID: queryID,
			Data: ports.EndData{
				TotalCandidates: 1,
				ElapsedMs:       1000, // Mock timing
			},
		}
	}()

	return eventChan, nil
}

// ValidateProvider checks if the OpenAI API is accessible
func (s *OpenAISuggestionService) ValidateProvider(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI API validation failed: %d", resp.StatusCode)
	}

	return nil
}

// OpenAIEmbeddingProvider implements EmbeddingProvider for OpenAI
type OpenAIEmbeddingProvider struct {
	apiKey     string
	model      string
	dimension  int
	baseURL    string
	httpClient *http.Client
}

// Embed generates embedding vector for a single text
func (e *OpenAIEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// EmbedBatch generates embedding vectors for multiple texts
func (e *OpenAIEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	requestBody := map[string]interface{}{
		"model": e.model,
		"input": texts,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI embeddings API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI embeddings API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode embeddings response: %w", err)
	}

	if len(response.Data) != len(texts) {
		return nil, fmt.Errorf("unexpected number of embeddings: got %d, want %d", len(response.Data), len(texts))
	}

	embeddings := make([][]float32, len(texts))
	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// Dimension returns the dimension of the embedding vectors
func (e *OpenAIEmbeddingProvider) Dimension() int {
	return e.dimension
}

// ValidateEmbedding checks if embedding dimension is correct
func (e *OpenAIEmbeddingProvider) ValidateEmbedding(embedding []float32) bool {
	return len(embedding) == e.dimension
}

// OpenAITrainingService implements AITrainingService for OpenAI
type OpenAITrainingService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// LearnFromResolved trains the AI model with resolved ticket data
func (t *OpenAITrainingService) LearnFromResolved(ctx context.Context, ticket *ports.TicketTrainingData) error {
	// This would typically involve:
	// 1. Creating fine-tuning examples from resolved tickets
	// 2. Submitting to OpenAI's fine-tuning API
	// 3. Managing training jobs and model updates

	// For now, we'll just log the training data
	_ = ticket // In production, process this data

	return nil
}

// LearnFromKnowledge trains the AI with knowledge base updates
func (t *OpenAITrainingService) LearnFromKnowledge(ctx context.Context, entry *ports.KnowledgeTrainingData) error {
	// Similar to LearnFromResolved, but for knowledge base entries
	_ = entry // In production, process this data

	return nil
}

// ValidateTraining checks if training data is valid
func (t *OpenAITrainingService) ValidateTraining(ctx context.Context, data interface{}) error {
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

// Helper functions

func calculateConfidence(content string, tokenCount int) float64 {
	// Simple confidence calculation based on content length and token usage
	baseConfidence := 0.7

	if len(content) > 100 {
		baseConfidence += 0.1
	}

	if tokenCount > 50 && tokenCount < 200 {
		baseConfidence += 0.1
	}

	if baseConfidence > 0.95 {
		baseConfidence = 0.95
	}

	return baseConfidence
}

func determineCategory(content, description string) string {
	descLower := strings.ToLower(description)

	if strings.Contains(descLower, "wifi") || strings.Contains(descLower, "network") {
		return "Network"
	}
	if strings.Contains(descLower, "software") || strings.Contains(descLower, "application") {
		return "Software"
	}
	if strings.Contains(descLower, "hardware") || strings.Contains(descLower, "device") {
		return "Hardware"
	}
	if strings.Contains(descLower, "account") || strings.Contains(descLower, "login") {
		return "Account"
	}

	return "General"
}

// Helper: generate concise title from description
func oaGenerateTitle(desc string) string {
	t := strings.TrimSpace(desc)
	if t == "" {
		return "Issue reported"
	}
	words := strings.Fields(t)
	if len(words) <= 6 {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:6], " ") + "..."
}

// Helper: estimate priority via keywords
func oaEstimatePriority(desc string) string {
	d := strings.ToLower(desc)
	critical := []string{"down", "outage", "cannot access", "security breach", "data loss"}
	for _, k := range critical {
		if strings.Contains(d, k) {
			return "CRITICAL"
		}
	}
	high := []string{"urgent", "crash", "error", "failed", "not working"}
	for _, k := range high {
		if strings.Contains(d, k) {
			return "HIGH"
		}
	}
	low := []string{"suggestion", "feature request", "minor", "typo"}
	for _, k := range low {
		if strings.Contains(d, k) {
			return "LOW"
		}
	}
	return "MEDIUM"
}
