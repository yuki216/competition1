package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	outbound "github.com/fixora/fixora/application/port/outbound"
	"github.com/fixora/fixora/domain"
)

// AIUseCase handles AI-related business logic
type AIUseCase struct {
	aiService     outbound.AISuggestionService
	embeddings    outbound.EmbeddingProvider
	knowledgeRepo outbound.KnowledgeRepository
	ticketRepo    outbound.TicketRepository
	training      outbound.AITrainingService
}

func NewAIUseCase(
	aiService outbound.AISuggestionService,
	embeddings outbound.EmbeddingProvider,
	knowledgeRepo outbound.KnowledgeRepository,
	ticketRepo outbound.TicketRepository,
	training outbound.AITrainingService,
) *AIUseCase {
	return &AIUseCase{aiService: aiService, embeddings: embeddings, knowledgeRepo: knowledgeRepo, ticketRepo: ticketRepo, training: training}
}

func (uc *AIUseCase) GetSuggestion(ctx context.Context, description string) (*outbound.SuggestionResult, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if uc.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	suggestion, err := uc.aiService.SuggestMitigation(ctx, description)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI suggestion: %w", err)
	}
	if suggestion.Confidence < 0.4 {
		return nil, fmt.Errorf("AI confidence too low: %.2f", suggestion.Confidence)
	}
	return &suggestion, nil
}

func (uc *AIUseCase) StreamSuggestion(ctx context.Context, description string) (<-chan outbound.SuggestionEvent, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if uc.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	return uc.aiService.StreamSuggestionMitigation(ctx, description)
}

func (uc *AIUseCase) SearchKnowledgeBase(ctx context.Context, query string, filter domain.KBChunkFilter) ([]*domain.KBChunk, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if uc.knowledgeRepo == nil {
		return nil, fmt.Errorf("knowledge repository not available")
	}
	chunks, err := uc.knowledgeRepo.SearchChunks(ctx, query, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge base: %w", err)
	}
	return chunks, nil
}

func (uc *AIUseCase) TrainFromResolvedTicket(ctx context.Context, ticketID string) error {
	if ticketID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	if uc.training == nil {
		return fmt.Errorf("AI training service not available")
	}
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}
	if ticket.Status != domain.TicketStatusResolved {
		return fmt.Errorf("ticket must be resolved before training")
	}
	var comments []string
	trainingData := &outbound.TicketTrainingData{TicketID: ticket.ID, Title: ticket.Title, Description: ticket.Description, Category: string(ticket.Category), Solution: extractSolutionFromComments(comments), Comments: comments}
	if err := uc.training.LearnFromResolved(ctx, trainingData); err != nil {
		return fmt.Errorf("failed to train from resolved ticket: %w", err)
	}
	return nil
}

func (uc *AIUseCase) TrainFromKnowledgeEntry(ctx context.Context, entryID string) error {
	if entryID == "" {
		return fmt.Errorf("entry ID is required")
	}
	if uc.training == nil {
		return fmt.Errorf("AI training service not available")
	}
	entry, err := uc.knowledgeRepo.FindEntryByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge entry: %w", err)
	}
	if !entry.IsActive() {
		return fmt.Errorf("knowledge entry must be active for training")
	}
	trainingData := &outbound.KnowledgeTrainingData{EntryID: entry.ID, Title: entry.Title, Content: entry.Content, Category: entry.Category, Tags: entry.Tags, Solution: entry.Content}
	if err := uc.training.LearnFromKnowledge(ctx, trainingData); err != nil {
		return fmt.Errorf("failed to train from knowledge entry: %w", err)
	}
	return nil
}

func (uc *AIUseCase) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if uc.embeddings == nil {
		return nil, fmt.Errorf("embedding service not available")
	}
	embedding, err := uc.embeddings.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	if !uc.embeddings.ValidateEmbedding(embedding) {
		return nil, fmt.Errorf("invalid embedding dimension")
	}
	return embedding, nil
}

func (uc *AIUseCase) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("at least one text is required")
	}
	if uc.embeddings == nil {
		return nil, fmt.Errorf("embedding service not available")
	}
	embeddings, err := uc.embeddings.EmbedBatch(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate batch embeddings: %w", err)
	}
	for i, embedding := range embeddings {
		if !uc.embeddings.ValidateEmbedding(embedding) {
			return nil, fmt.Errorf("invalid embedding dimension for text %d", i)
		}
	}
	return embeddings, nil
}

func (uc *AIUseCase) ValidateAIProvider(ctx context.Context) error {
	if uc.aiService == nil {
		return fmt.Errorf("AI suggestion service not configured")
	}
	if err := uc.aiService.ValidateProvider(ctx); err != nil {
		return fmt.Errorf("AI suggestion service validation failed: %w", err)
	}
	if uc.embeddings == nil {
		return fmt.Errorf("embedding service not configured")
	}
	if _, err := uc.embeddings.Embed(ctx, "test"); err != nil {
		return fmt.Errorf("embedding service validation failed: %w", err)
	}
	return nil
}

func (uc *AIUseCase) GetAIProviderInfo(ctx context.Context) map[string]interface{} {
	info := map[string]interface{}{"suggestion_service": uc.aiService != nil, "embedding_service": uc.embeddings != nil, "training_service": uc.training != nil}
	return info
}

func (uc *AIUseCase) AnalyzeTicketContent(ctx context.Context, title, description string) (*TicketAnalysis, error) {
	if title == "" || description == "" {
		return nil, fmt.Errorf("title and description are required")
	}
	var suggestion *outbound.SuggestionResult
	if uc.aiService != nil {
		res, err := uc.aiService.SuggestMitigation(ctx, description)
		if err == nil {
			suggestion = &res
		}
	}
	var embedding []float32
	if uc.embeddings != nil {
		e, err := uc.embeddings.Embed(ctx, description)
		if err == nil {
			embedding = e
		}
	}
	filter := domain.KBChunkFilter{TopK: 5}
	chunks, _ := uc.SearchKnowledgeBase(ctx, description, filter)
	analysis := &TicketAnalysis{Title: title, Description: description, AISuggestion: suggestion, Embedding: embedding, SimilarIssues: chunks, Category: string(domain.TicketCategoryOther), Priority: string(domain.TicketPriorityMedium), AnalyzedAt: time.Now()}
	return analysis, nil
}

type TicketAnalysis struct {
	Title         string                     `json:"title"`
	Description   string                     `json:"description"`
	AISuggestion  *outbound.SuggestionResult `json:"ai_suggestion,omitempty"`
	Embedding     []float32                  `json:"embedding,omitempty"`
	SimilarIssues []*domain.KBChunk          `json:"similar_issues,omitempty"`
	Category      string                     `json:"predicted_category,omitempty"`
	Priority      string                     `json:"predicted_priority,omitempty"`
	AnalyzedAt    time.Time                  `json:"analyzed_at"`
}

func extractSolutionFromComments(comments []string) string {
	if len(comments) == 0 {
		return ""
	}
	return comments[0]
}

func lower(s string) string          { return strings.ToLower(s) }
func contains(s, substr string) bool { return strings.Contains(s, substr) }

type AITicketIntakeRequest struct {
	Description     string                `json:"description"`
	Title           string                `json:"title,omitempty"`
	Category        domain.TicketCategory `json:"category,omitempty"`
	Priority        domain.TicketPriority `json:"priority,omitempty"`
	AutoCategorize  bool                  `json:"autoCategorize"`
	AutoPrioritize  bool                  `json:"autoPrioritize"`
	AutoTitleFromAI bool                  `json:"autoTitleFromAI"`
}

type OverrideMetaItem struct {
	Field      string  `json:"field"`
	Source     string  `json:"source"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence,omitempty"`
	Applied    bool    `json:"applied"`
}

type AITicketIntakeResponse struct {
	Ticket       *domain.Ticket             `json:"ticket"`
	AIInsight    *outbound.SuggestionResult `json:"ai_insight,omitempty"`
	OverrideMeta []OverrideMetaItem         `json:"override_meta"`
}

func (uc *AIUseCase) IntakeCreateTicket(ctx context.Context, req AITicketIntakeRequest, createdBy string) (*AITicketIntakeResponse, error) {
	if req.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	title := req.Title
	var insight *outbound.SuggestionResult
	var meta []OverrideMetaItem
	if req.AutoTitleFromAI || title == "" {
		title = defaultTitleFromDescription(req.Description)
		meta = append(meta, OverrideMetaItem{Field: "title", Source: "default", Value: title, Applied: true})
	}
	if req.AutoCategorize || req.Category == "" || req.AutoPrioritize || req.Priority == "" {
		if uc.aiService != nil {
			pred, err := uc.aiService.PredictAttributes(ctx, req.Description)
			if err == nil {
				if req.Category == "" || req.AutoCategorize {
					req.Category = domain.TicketCategory(pred.Category.Value)
					meta = append(meta, OverrideMetaItem{Field: "category", Source: "ai", Value: pred.Category.Value, Confidence: pred.Category.Confidence, Applied: true})
				}
				if req.Priority == "" || req.AutoPrioritize {
					req.Priority = domain.TicketPriority(pred.Priority.Value)
					meta = append(meta, OverrideMetaItem{Field: "priority", Source: "ai", Value: pred.Priority.Value, Confidence: pred.Priority.Confidence, Applied: true})
				}
			}
		}
	}
	ticket := domain.NewTicket(title, req.Description, req.Category, req.Priority, generateID())
	if uc.aiService != nil {
		res, err := uc.aiService.SuggestMitigation(ctx, req.Description)
		if err == nil {
			insight = &res
			ticket.SetAIInsight(res.Suggestion, res.Confidence)
			meta = append(meta, OverrideMetaItem{Field: "ai_insight", Source: "ai", Value: res.Suggestion, Confidence: res.Confidence, Applied: true})
		}
	}
	if err := uc.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}
	return &AITicketIntakeResponse{Ticket: ticket, AIInsight: insight, OverrideMeta: meta}, nil
}

func defaultTitleFromDescription(desc string) string {
	d := strings.TrimSpace(desc)
	if len(d) > 60 {
		d = d[:60]
	}
	return d
}
func normalizeCategory(s string) domain.TicketCategory {
	return domain.TicketCategory(strings.ToUpper(strings.TrimSpace(s)))
}
func normalizePriority(s string) domain.TicketPriority {
	return domain.TicketPriority(strings.ToUpper(strings.TrimSpace(s)))
}
