package usecase

import (
	"context"
	"fmt"
	"strings"

	"fixora/internal/domain"
	"fixora/internal/ports"
)

// KnowledgeUseCase handles knowledge base business logic
type KnowledgeUseCase struct {
	knowledgeRepo ports.KnowledgeRepository
	embeddings    ports.EmbeddingProvider
	eventPublisher ports.EventPublisher
}

// NewKnowledgeUseCase creates a new knowledge use case
func NewKnowledgeUseCase(
	knowledgeRepo ports.KnowledgeRepository,
	embeddings ports.EmbeddingProvider,
	eventPublisher ports.EventPublisher,
) *KnowledgeUseCase {
	return &KnowledgeUseCase{
		knowledgeRepo: knowledgeRepo,
		embeddings:    embeddings,
		eventPublisher: eventPublisher,
	}
}

// CreateEntry creates a new knowledge base entry
func (uc *KnowledgeUseCase) CreateEntry(ctx context.Context, req CreateKnowledgeEntryRequest) (*domain.KnowledgeEntry, error) {
	if err := uc.validateCreateEntryRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	entry := domain.NewKnowledgeEntry(
		req.Title,
		req.Content,
		req.Category,
		req.Tags,
		req.CreatedBy,
	)

	if err := uc.knowledgeRepo.CreateEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create knowledge entry: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeKBEntryCreated,
			"knowledge_entry",
			entry.ID,
			map[string]interface{}{
				"title":      entry.Title,
				"category":   entry.Category,
				"created_by": entry.CreatedBy,
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return entry, nil
}

// PublishEntry processes and publishes a knowledge base entry
func (uc *KnowledgeUseCase) PublishEntry(ctx context.Context, entryID string) error {
	if entryID == "" {
		return fmt.Errorf("entry ID is required")
	}

	// Get entry
	entry, err := uc.knowledgeRepo.FindEntryByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge entry: %w", err)
	}

	if entry.Status != domain.KnowledgeEntryStatusDraft {
		return fmt.Errorf("only draft entries can be published")
	}

	// Process content: normalize and chunk
	normalizedContent := uc.normalizeContent(entry.Content)
	chunks := uc.chunkContent(normalizedContent)

	// Generate embeddings for chunks
	if uc.embeddings != nil {
		chunkTexts := make([]string, len(chunks))
		for i, chunk := range chunks {
			chunkTexts[i] = chunk.Content
		}

		embeddings, err := uc.embeddings.EmbedBatch(ctx, chunkTexts)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Set embeddings for chunks
		for i, embedding := range embeddings {
			chunks[i].SetEmbedding(embedding)
		}
	}

	// Save chunks
	for _, chunk := range chunks {
		if err := uc.knowledgeRepo.CreateChunk(ctx, &chunk); err != nil {
			return fmt.Errorf("failed to create knowledge chunk: %w", err)
		}
	}

	// Publish entry
	if err := entry.Publish(); err != nil {
		return fmt.Errorf("failed to publish entry: %w", err)
	}

	// Update entry
	if err := uc.knowledgeRepo.UpdateEntry(ctx, entry); err != nil {
		return fmt.Errorf("failed to update knowledge entry: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeKBEntryPublished,
			"knowledge_entry",
			entry.ID,
			map[string]interface{}{
				"title":    entry.Title,
				"category": entry.Category,
				"version":  entry.Version,
				"chunks":   len(chunks),
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return nil
}

// GetEntry retrieves a knowledge base entry
func (uc *KnowledgeUseCase) GetEntry(ctx context.Context, entryID string) (*domain.KnowledgeEntry, error) {
	if entryID == "" {
		return nil, fmt.Errorf("entry ID is required")
	}

	entry, err := uc.knowledgeRepo.FindEntryByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge entry: %w", err)
	}

	return entry, nil
}

// ListEntries retrieves knowledge base entries based on filters
func (uc *KnowledgeUseCase) ListEntries(ctx context.Context, filter domain.KBChunkFilter) ([]*domain.KnowledgeEntry, error) {
	entries, err := uc.knowledgeRepo.ListEntries(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge entries: %w", err)
	}

	return entries, nil
}

// UpdateEntry updates a knowledge base entry
func (uc *KnowledgeUseCase) UpdateEntry(ctx context.Context, entryID string, req UpdateKnowledgeEntryRequest) (*domain.KnowledgeEntry, error) {
	if entryID == "" {
		return nil, fmt.Errorf("entry ID is required")
	}

	if err := uc.validateUpdateEntryRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get current entry
	entry, err := uc.knowledgeRepo.FindEntryByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge entry: %w", err)
	}

	// Update content
	entry.UpdateContent(req.Title, req.Content)
	entry.Category = req.Category
	entry.Tags = req.Tags

	// Save updated entry
	if err := uc.knowledgeRepo.UpdateEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to update knowledge entry: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeKBEntryUpdated,
			"knowledge_entry",
			entry.ID,
			map[string]interface{}{
				"title":    entry.Title,
				"category": entry.Category,
				"tags":     entry.Tags,
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return entry, nil
}

// DeleteEntry deletes a knowledge base entry
func (uc *KnowledgeUseCase) DeleteEntry(ctx context.Context, entryID string) error {
	if entryID == "" {
		return fmt.Errorf("entry ID is required")
	}

	// Get entry to ensure it exists
	entry, err := uc.knowledgeRepo.FindEntryByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge entry: %w", err)
	}

	// Archive instead of hard delete
	if err := entry.Archive(); err != nil {
		return fmt.Errorf("failed to archive entry: %w", err)
	}

	// Update entry
	if err := uc.knowledgeRepo.UpdateEntry(ctx, entry); err != nil {
		return fmt.Errorf("failed to update knowledge entry: %w", err)
	}

	return nil
}

// SearchEntries performs semantic search in the knowledge base
func (uc *KnowledgeUseCase) SearchEntries(ctx context.Context, query string, filter domain.KBChunkFilter) ([]*domain.KBChunk, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	chunks, err := uc.knowledgeRepo.SearchChunks(ctx, query, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge base: %w", err)
	}

	return chunks, nil
}

// Request/Response types

type CreateKnowledgeEntryRequest struct {
	Title     string   `json:"title" validate:"required,min=3,max=200"`
	Content   string   `json:"content" validate:"required,min=10"`
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	CreatedBy string   `json:"created_by" validate:"required"`
}

type UpdateKnowledgeEntryRequest struct {
	Title    string   `json:"title" validate:"required,min=3,max=200"`
	Content  string   `json:"content" validate:"required,min=10"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// Helper methods

func (uc *KnowledgeUseCase) validateCreateEntryRequest(req CreateKnowledgeEntryRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) < 3 {
		return fmt.Errorf("title must be at least 3 characters")
	}
	if len(req.Title) > 200 {
		return fmt.Errorf("title must not exceed 200 characters")
	}

	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	if len(req.Content) < 10 {
		return fmt.Errorf("content must be at least 10 characters")
	}

	if req.CreatedBy == "" {
		return fmt.Errorf("created by is required")
	}

	return nil
}

func (uc *KnowledgeUseCase) validateUpdateEntryRequest(req UpdateKnowledgeEntryRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) < 3 {
		return fmt.Errorf("title must be at least 3 characters")
	}
	if len(req.Title) > 200 {
		return fmt.Errorf("title must not exceed 200 characters")
	}

	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	if len(req.Content) < 10 {
		return fmt.Errorf("content must be at least 10 characters")
	}

	return nil
}

func (uc *KnowledgeUseCase) normalizeContent(content string) string {
	// Normalize whitespace
	content = strings.ReplaceAll(content, "\t", " ")
	content = strings.ReplaceAll(content, "\n\n", "\n")
	content = strings.Join(strings.Fields(content), " ")

	// Simple markdown stripping (in production, use a proper markdown parser)
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "*", "")
	content = strings.ReplaceAll(content, "#", "")

	return strings.TrimSpace(content)
}

func (uc *KnowledgeUseCase) chunkContent(content string) []domain.KBChunk {
	chunkSize := 800
	overlap := 150

	var chunks []domain.KBChunk

	// Simple chunking algorithm
	for i := 0; i < len(content); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunkContent := content[i:end]
		chunk := domain.NewKBChunk("temp_id", len(chunks), chunkContent)
		chunks = append(chunks, *chunk)
	}

	return chunks
}