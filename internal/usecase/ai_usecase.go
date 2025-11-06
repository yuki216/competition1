package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/ports"
)

// AIUseCase handles AI-related business logic
type AIUseCase struct {
    aiService     ports.AISuggestionService
    embeddings    ports.EmbeddingProvider
    knowledgeRepo ports.KnowledgeRepository
    ticketRepo    ports.TicketRepository
    training      ports.AITrainingService
}

// NewAIUseCase creates a new AI use case
func NewAIUseCase(
	aiService ports.AISuggestionService,
	embeddings ports.EmbeddingProvider,
	knowledgeRepo ports.KnowledgeRepository,
	ticketRepo ports.TicketRepository,
	training ports.AITrainingService,
) *AIUseCase {
	return &AIUseCase{
		aiService:     aiService,
		embeddings:    embeddings,
		knowledgeRepo: knowledgeRepo,
		ticketRepo:    ticketRepo,
		training:      training,
	}
}

// GetSuggestion provides AI suggestion for a ticket description
func (uc *AIUseCase) GetSuggestion(ctx context.Context, description string) (*ports.SuggestionResult, error) {
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

	// Filter suggestions with low confidence
	if suggestion.Confidence < 0.4 {
		return nil, fmt.Errorf("AI confidence too low: %.2f", suggestion.Confidence)
	}

	return &suggestion, nil
}

// StreamSuggestion provides streaming AI suggestions
func (uc *AIUseCase) StreamSuggestion(ctx context.Context, description string) (<-chan ports.SuggestionEvent, error) {
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}

	if uc.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	return uc.aiService.StreamSuggestionMitigation(ctx, description)
}

// SearchKnowledgeBase performs semantic search in the knowledge base
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

// TrainFromResolvedTicket trains the AI model using resolved ticket data
func (uc *AIUseCase) TrainFromResolvedTicket(ctx context.Context, ticketID string) error {
	if ticketID == "" {
		return fmt.Errorf("ticket ID is required")
	}

	if uc.training == nil {
		return fmt.Errorf("AI training service not available")
	}

	// Get ticket details
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	if ticket.Status != domain.TicketStatusResolved {
		return fmt.Errorf("ticket must be resolved before training")
	}

	// Get comments for additional context
	var comments []string
	if uc.ticketRepo != nil {
		// Note: In a real implementation, you would inject CommentRepository
		// commentRepo := someCommentRepo
		// commentList, err := commentRepo.ListByTicket(ctx, ticketID)
		// if err == nil {
		//     for _, comment := range commentList {
		//         if comment.Role != domain.CommentRoleAI {
		//             comments = append(comments, comment.Body)
		//         }
		//     }
		// }
	}

	// Create training data
	trainingData := &ports.TicketTrainingData{
		TicketID:    ticket.ID,
		Title:       ticket.Title,
		Description: ticket.Description,
		Category:    string(ticket.Category),
		Solution:    extractSolutionFromComments(comments), // Simplified
		Comments:    comments,
	}

	// Train the model
	if err := uc.training.LearnFromResolved(ctx, trainingData); err != nil {
		return fmt.Errorf("failed to train from resolved ticket: %w", err)
	}

	return nil
}

// TrainFromKnowledgeEntry trains the AI model using knowledge base entry
func (uc *AIUseCase) TrainFromKnowledgeEntry(ctx context.Context, entryID string) error {
	if entryID == "" {
		return fmt.Errorf("entry ID is required")
	}

	if uc.training == nil {
		return fmt.Errorf("AI training service not available")
	}

	// Get knowledge entry
	entry, err := uc.knowledgeRepo.FindEntryByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge entry: %w", err)
	}

	if !entry.IsActive() {
		return fmt.Errorf("knowledge entry must be active for training")
	}

	// Create training data
	trainingData := &ports.KnowledgeTrainingData{
		EntryID:  entry.ID,
		Title:    entry.Title,
		Content:  entry.Content,
		Category: entry.Category,
		Tags:     entry.Tags,
		Solution: entry.Content, // Simplified - in real implementation, extract solution
	}

	// Train the model
	if err := uc.training.LearnFromKnowledge(ctx, trainingData); err != nil {
		return fmt.Errorf("failed to train from knowledge entry: %w", err)
	}

	return nil
}

// GenerateEmbedding generates embedding for given text
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

	// Validate embedding dimension
	if !uc.embeddings.ValidateEmbedding(embedding) {
		return nil, fmt.Errorf("invalid embedding dimension")
	}

	return embedding, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
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

	// Validate all embeddings
	for i, embedding := range embeddings {
		if !uc.embeddings.ValidateEmbedding(embedding) {
			return nil, fmt.Errorf("invalid embedding dimension for text %d", i)
		}
	}

	return embeddings, nil
}

// ValidateAIProvider checks if AI services are healthy
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

	// Test embedding service with simple text
	_, err := uc.embeddings.Embed(ctx, "test")
	if err != nil {
		return fmt.Errorf("embedding service validation failed: %w", err)
	}

	return nil
}

// GetAIProviderInfo returns information about the AI provider
func (uc *AIUseCase) GetAIProviderInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	if uc.aiService != nil {
		info["suggestion_service"] = "available"
	} else {
		info["suggestion_service"] = "not_available"
	}

	if uc.embeddings != nil {
		info["embedding_service"] = "available"
		info["embedding_dimension"] = uc.embeddings.Dimension()
	} else {
		info["embedding_service"] = "not_available"
	}

	if uc.training != nil {
		info["training_service"] = "available"
	} else {
		info["training_service"] = "not_available"
	}

	if uc.knowledgeRepo != nil {
		info["knowledge_repository"] = "available"
	} else {
		info["knowledge_repository"] = "not_available"
	}

	info["last_validation"] = time.Now().Format(time.RFC3339)

	return info
}

// AnalyzeTicketContent analyzes ticket content and provides insights
func (uc *AIUseCase) AnalyzeTicketContent(ctx context.Context, title, description string) (*TicketAnalysis, error) {
	if title == "" || description == "" {
		return nil, fmt.Errorf("title and description are required")
	}

	analysis := &TicketAnalysis{
		Title:       title,
		Description: description,
		AnalyzedAt:  time.Now(),
	}

	// Combine title and description for analysis
	fullText := fmt.Sprintf("%s %s", title, description)

	// Get AI suggestion
	if uc.aiService != nil {
		suggestion, err := uc.aiService.SuggestMitigation(ctx, fullText)
		if err == nil {
			analysis.AISuggestion = &suggestion
		}
	}

	// Generate embedding for similarity search
	if uc.embeddings != nil {
		embedding, err := uc.embeddings.Embed(ctx, fullText)
		if err == nil {
			analysis.Embedding = embedding
		}
	}

	// Search for similar issues in knowledge base
	if uc.knowledgeRepo != nil && analysis.Embedding != nil {
		filter := domain.KBChunkFilter{
			TopK: 5,
		}
		similarChunks, err := uc.knowledgeRepo.SearchChunks(ctx, fullText, filter)
		if err == nil {
			analysis.SimilarIssues = similarChunks
		}
	}

	return analysis, nil
}

// TicketAnalysis represents the analysis result of a ticket
type TicketAnalysis struct {
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	AISuggestion  *ports.SuggestionResult   `json:"ai_suggestion,omitempty"`
	Embedding     []float32                 `json:"embedding,omitempty"`
	SimilarIssues []*domain.KBChunk         `json:"similar_issues,omitempty"`
	Category      string                    `json:"predicted_category,omitempty"`
	Priority      string                    `json:"predicted_priority,omitempty"`
	AnalyzedAt    time.Time                 `json:"analyzed_at"`
}

// Helper functions

func extractSolutionFromComments(comments []string) string {
	// Simplified implementation - in a real system, you would use NLP techniques
	// to extract the actual solution from the conversation
	if len(comments) == 0 {
		return ""
	}

	// Look for resolution keywords in comments
	resolutionKeywords := []string{
		"resolved", "fixed", "solved", "completed", "done",
		"the issue was", "the problem was", "solution is",
	}

	for _, comment := range comments {
		lowerComment := lower(comment)
		for _, keyword := range resolutionKeywords {
			if contains(lowerComment, keyword) {
				return comment
			}
		}
	}

	// Return the last comment as fallback
	return comments[len(comments)-1]
}

// Simple string helpers (in a real implementation, use strings package)
func lower(s string) string {
	// Implementation would use strings.ToLower()
	return s
}

func contains(s, substr string) bool {
    // Implementation would use strings.Contains()
    return true
}

// AI Intake DTOs and logic

// AITicketIntakeRequest represents the request payload for AI-driven ticket intake
type AITicketIntakeRequest struct {
    Description     string `json:"description"`
    Title           string `json:"title,omitempty"`
    Category        domain.TicketCategory `json:"category,omitempty"`
    Priority        domain.TicketPriority `json:"priority,omitempty"`
    AutoCategorize  bool   `json:"autoCategorize"`
    AutoPrioritize  bool   `json:"autoPrioritize"`
    AutoTitleFromAI bool   `json:"autoTitleFromAI"`
}

// OverrideMetaItem describes how a field was set during intake
type OverrideMetaItem struct {
    Field      string  `json:"field"`
    Source     string  `json:"source"` // user | ai | default
    Value      string  `json:"value"`
    Confidence float64 `json:"confidence,omitempty"`
    Applied    bool    `json:"applied"`
}

// AITicketIntakeResponse represents the result of AI-driven ticket intake
type AITicketIntakeResponse struct {
    Ticket      *domain.Ticket        `json:"ticket"`
    AIInsight   *ports.SuggestionResult `json:"ai_insight,omitempty"`
    OverrideMeta []OverrideMetaItem   `json:"override_meta"`
}

// IntakeCreateTicket creates a ticket using AI predictions to auto-fill fields
func (uc *AIUseCase) IntakeCreateTicket(ctx context.Context, req AITicketIntakeRequest, createdBy string) (*AITicketIntakeResponse, error) {
    if req.Description == "" {
        return nil, fmt.Errorf("description is required")
    }
    if createdBy == "" {
        return nil, fmt.Errorf("created_by is required from auth context")
    }

    // Fetch AI predictions
    var preds ports.PredictedAttributes
    if uc.aiService != nil {
        p, err := uc.aiService.PredictAttributes(ctx, req.Description)
        if err == nil {
            preds = p
        }
    }

    // Confidence thresholds
    const titleThreshold = 0.60
    const catThreshold = 0.70
    const prioThreshold = 0.70

    overrideMeta := make([]OverrideMetaItem, 0, 3)

    // Title resolution
    var title string
    if req.Title != "" {
        title = req.Title
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:   "title",
            Source:  "user",
            Value:   title,
            Applied: true,
        })
    } else if req.AutoTitleFromAI && preds.Title.Value != "" && preds.Title.Confidence >= titleThreshold {
        title = preds.Title.Value
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:      "title",
            Source:     "ai",
            Value:      title,
            Confidence: preds.Title.Confidence,
            Applied:    true,
        })
    } else {
        title = defaultTitleFromDescription(req.Description)
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:   "title",
            Source:  "default",
            Value:   title,
            Applied: true,
        })
    }

    // Category resolution
    var category domain.TicketCategory
    if req.Category != "" {
        category = req.Category
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:   "category",
            Source:  "user",
            Value:   string(category),
            Applied: true,
        })
    } else if req.AutoCategorize && preds.Category.Value != "" && preds.Category.Confidence >= catThreshold {
        category = normalizeCategory(preds.Category.Value)
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:      "category",
            Source:     "ai",
            Value:      string(category),
            Confidence: preds.Category.Confidence,
            Applied:    true,
        })
    } else {
        category = domain.TicketCategoryOther
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:   "category",
            Source:  "default",
            Value:   string(category),
            Applied: true,
        })
    }

    // Priority resolution
    var priority domain.TicketPriority
    if req.Priority != "" {
        priority = req.Priority
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:   "priority",
            Source:  "user",
            Value:   string(priority),
            Applied: true,
        })
    } else if req.AutoPrioritize && preds.Priority.Value != "" && preds.Priority.Confidence >= prioThreshold {
        priority = normalizePriority(preds.Priority.Value)
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:      "priority",
            Source:     "ai",
            Value:      string(priority),
            Confidence: preds.Priority.Confidence,
            Applied:    true,
        })
    } else {
        priority = domain.TicketPriorityMedium
        overrideMeta = append(overrideMeta, OverrideMetaItem{
            Field:   "priority",
            Source:  "default",
            Value:   string(priority),
            Applied: true,
        })
    }

    // Create ticket
    ticket := domain.NewTicket(title, req.Description, category, priority, createdBy)

    // Optionally attach AI insight
    var insight *ports.SuggestionResult
    if uc.aiService != nil {
        if s, err := uc.aiService.SuggestMitigation(ctx, req.Description); err == nil && s.Confidence >= 0.4 {
            ticket.SetAIInsight(s.Suggestion, s.Confidence)
            insight = &s
        }
    }

    // Persist ticket
    if uc.ticketRepo == nil {
        return nil, fmt.Errorf("ticket repository not available")
    }
    if err := uc.ticketRepo.Create(ctx, ticket); err != nil {
        return nil, fmt.Errorf("failed to create ticket: %w", err)
    }

    return &AITicketIntakeResponse{
        Ticket:       ticket,
        AIInsight:    insight,
        OverrideMeta: overrideMeta,
    }, nil
}

// Helpers for normalization
func defaultTitleFromDescription(desc string) string {
    if len(desc) == 0 {
        return "Issue reported"
    }
    if len(desc) <= 60 {
        return desc
    }
    return desc[:60] + "..."
}

func normalizeCategory(s string) domain.TicketCategory {
    switch lower(s) {
    case "network":
        return domain.TicketCategoryNetwork
    case "software":
        return domain.TicketCategorySoftware
    case "hardware":
        return domain.TicketCategoryHardware
    case "account":
        return domain.TicketCategoryAccount
    default:
        return domain.TicketCategoryOther
    }
}

func normalizePriority(s string) domain.TicketPriority {
    switch lower(s) {
    case "low":
        return domain.TicketPriorityLow
    case "medium":
        return domain.TicketPriorityMedium
    case "high":
        return domain.TicketPriorityHigh
    case "critical":
        return domain.TicketPriorityCritical
    default:
        return domain.TicketPriorityMedium
    }
}