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

    outbound "github.com/fixora/fixora/application/port/outbound"
)

// OpenAIAdapter implements AI services using OpenAI APIs
type OpenAIAdapter struct {
    apiKey         string
    baseURL        string
    model          string
    embeddingModel string
    embeddings     outbound.EmbeddingProvider
    client         *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(config outbound.AIConfig) outbound.AIProviderFactory {
    client := &http.Client{Timeout: time.Duration(config.TimeoutMs) * time.Millisecond}
    baseURL := "https://api.openai.com/v1"
    if config.EmbeddingModel == "" { config.EmbeddingModel = "text-embedding-ada-002" }
    if config.SuggestionModel == "" { config.SuggestionModel = "gpt-3.5-turbo" }

    adapter := &OpenAIAdapter{
        apiKey:         config.APIKey,
        baseURL:        baseURL,
        model:          config.SuggestionModel,
        embeddingModel: config.EmbeddingModel,
        client:         client,
    }

    adapter.embeddings = &OpenAIEmbeddingProvider{
        apiKey:     config.APIKey,
        model:      config.EmbeddingModel,
        dimension:  config.EmbeddingDim,
        baseURL:    baseURL,
        httpClient: client,
    }
    return adapter
}

func (o *OpenAIAdapter) Suggestion() outbound.AISuggestionService {
    return &OpenAISuggestionService{apiKey: o.apiKey, model: o.model, baseURL: o.baseURL, httpClient: o.client}
}
func (o *OpenAIAdapter) Embeddings() outbound.EmbeddingProvider { return o.embeddings }
func (o *OpenAIAdapter) Training() outbound.AITrainingService { return &OpenAITrainingService{apiKey: o.apiKey, baseURL: o.baseURL, httpClient: o.client} }
func (o *OpenAIAdapter) Provider() string { return "openai" }
func (o *OpenAIAdapter) IsHealthy(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/models", nil)
    if err != nil { return fmt.Errorf("failed to create health check request: %w", err) }
    req.Header.Set("Authorization", "Bearer "+o.apiKey)
    resp, err := o.client.Do(req)
    if err != nil { return fmt.Errorf("OpenAI API health check failed: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { return fmt.Errorf("OpenAI API returned status: %d", resp.StatusCode) }
    return nil
}

type OpenAISuggestionService struct { apiKey, model, baseURL string; httpClient *http.Client }

func (s *OpenAISuggestionService) SuggestMitigation(ctx context.Context, description string) (outbound.SuggestionResult, error) {
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
        "max_tokens": 300,
        "temperature": 0.7,
        "top_p": 1,
    }
    jsonBody, err := json.Marshal(requestBody)
    if err != nil { return outbound.SuggestionResult{}, fmt.Errorf("failed to marshal request: %w", err) }

    req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
    if err != nil { return outbound.SuggestionResult{}, fmt.Errorf("failed to create request: %w", err) }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+s.apiKey)

    resp, err := s.httpClient.Do(req)
    if err != nil { return outbound.SuggestionResult{}, fmt.Errorf("failed to call OpenAI API: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return outbound.SuggestionResult{}, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
    }

    var response struct {
        Choices []struct { Message struct { Content string `json:"content"` } `json:"message"` } `json:"choices"`
        Usage   struct { TotalTokens int `json:"total_tokens"` } `json:"usage"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil { return outbound.SuggestionResult{}, fmt.Errorf("failed to decode response: %w", err) }
    if len(response.Choices) == 0 { return outbound.SuggestionResult{}, fmt.Errorf("no choices in response") }

    confidence := calculateConfidence(response.Choices[0].Message.Content, response.Usage.TotalTokens)
    category := determineCategory(response.Choices[0].Message.Content, description)
    return outbound.SuggestionResult{Suggestion: response.Choices[0].Message.Content, Confidence: confidence, Category: category, Source: "openai", UsedCache: false}, nil
}

func (s *OpenAISuggestionService) PredictAttributes(ctx context.Context, description string) (outbound.PredictedAttributes, error) {
    title := oaGenerateTitle(description)
    category := determineCategory(title, description)
    priority := oaEstimatePriority(description)
    return outbound.PredictedAttributes{
        Title:    outbound.FieldPrediction{Value: title, Confidence: 0.68, Source: "openai-heuristic"},
        Category: outbound.FieldPrediction{Value: category, Confidence: 0.72, Source: "openai-heuristic"},
        Priority: outbound.FieldPrediction{Value: priority, Confidence: 0.70, Source: "openai-heuristic"},
    }, nil
}

func (s *OpenAISuggestionService) StreamSuggestionMitigation(ctx context.Context, description string) (<-chan outbound.SuggestionEvent, error) {
    eventChan := make(chan outbound.SuggestionEvent, 10)
    queryID := fmt.Sprintf("openai_%d", time.Now().UnixNano())
    go func() {
        defer close(eventChan)
        eventChan <- outbound.SuggestionEvent{Type: "init", QueryID: queryID, Data: map[string]interface{}{"query": description, "startTime": time.Now().Unix()}}
        result, err := s.SuggestMitigation(ctx, description)
        if err != nil { eventChan <- outbound.SuggestionEvent{Type: "error", QueryID: queryID, Error: err.Error()}; return }
        eventChan <- outbound.SuggestionEvent{Type: "candidate", QueryID: queryID, Data: outbound.CandidateData{Rank: 1, Score: result.Confidence, Suggestion: result.Suggestion, Category: result.Category}}
        eventChan <- outbound.SuggestionEvent{Type: "end", QueryID: queryID, Data: outbound.EndData{TotalCandidates: 1, ElapsedMs: int64(0)}}
    }()
    return eventChan, nil
}

func (s *OpenAISuggestionService) ValidateProvider(ctx context.Context) error { return nil }

type OpenAIEmbeddingProvider struct { apiKey, model string; dimension int; baseURL string; httpClient *http.Client }

func (e *OpenAIEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
    reqBody := map[string]interface{}{"input": text, "model": e.model}
    b, _ := json.Marshal(reqBody)
    req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(b))
    if err != nil { return nil, fmt.Errorf("failed to create request: %w", err) }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+e.apiKey)
    resp, err := e.httpClient.Do(req)
    if err != nil { return nil, fmt.Errorf("failed to call OpenAI API: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { body, _ := io.ReadAll(resp.Body); return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body)) }
    var response struct { Data []struct { Embedding []float32 `json:"embedding"` } `json:"data"` }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil { return nil, fmt.Errorf("failed to decode response: %w", err) }
    if len(response.Data) == 0 { return nil, fmt.Errorf("no embedding data") }
    return response.Data[0].Embedding, nil
}

func (e *OpenAIEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    reqBody := map[string]interface{}{"input": texts, "model": e.model}
    b, _ := json.Marshal(reqBody)
    req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(b))
    if err != nil { return nil, fmt.Errorf("failed to create request: %w", err) }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+e.apiKey)
    resp, err := e.httpClient.Do(req)
    if err != nil { return nil, fmt.Errorf("failed to call OpenAI API: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { body, _ := io.ReadAll(resp.Body); return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body)) }
    var response struct { Data []struct { Embedding []float32 `json:"embedding"` } `json:"data"` }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil { return nil, fmt.Errorf("failed to decode response: %w", err) }
    embeddings := make([][]float32, len(response.Data))
    for i, d := range response.Data { embeddings[i] = d.Embedding }
    return embeddings, nil
}

func (e *OpenAIEmbeddingProvider) Dimension() int { return e.dimension }
func (e *OpenAIEmbeddingProvider) ValidateEmbedding(embedding []float32) bool { return len(embedding) == e.dimension }

type OpenAITrainingService struct { apiKey, baseURL string; httpClient *http.Client }
func (t *OpenAITrainingService) LearnFromResolved(ctx context.Context, ticket *outbound.TicketTrainingData) error { return nil }
func (t *OpenAITrainingService) LearnFromKnowledge(ctx context.Context, entry *outbound.KnowledgeTrainingData) error { return nil }
func (t *OpenAITrainingService) ValidateTraining(ctx context.Context, data interface{}) error { if data == nil { return fmt.Errorf("training data cannot be nil") }; return nil }

func calculateConfidence(content string, tokenCount int) float64 {
    c := strings.TrimSpace(content)
    if c == "" { return 0.0 }
    base := 0.6
    if tokenCount > 200 { base += 0.1 }
    if strings.Contains(strings.ToLower(c), "step") { base += 0.05 }
    if base > 0.95 { base = 0.95 }
    return base
}

func determineCategory(content, description string) string {
    c := strings.ToLower(content + " " + description)
    switch {
    case strings.Contains(c, "network"):
        return "NETWORK"
    case strings.Contains(c, "password") || strings.Contains(c, "login"):
        return "ACCOUNT"
    case strings.Contains(c, "printer") || strings.Contains(c, "driver"):
        return "HARDWARE"
    default:
        return "SOFTWARE"
    }
}

func oaGenerateTitle(desc string) string {
    d := strings.TrimSpace(desc)
    if len(d) > 60 { d = d[:60] }
    return d
}

func oaEstimatePriority(desc string) string {
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