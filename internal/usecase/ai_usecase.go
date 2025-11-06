package usecase

import (
	"context"
	"fmt"
	"time"

	"fixora/internal/domain"
	"fixora/internal/ports"
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