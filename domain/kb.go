package domain

import (
    "time"
)

// KnowledgeEntryStatus represents the status of a knowledge base entry
type KnowledgeEntryStatus string

const (
    KnowledgeEntryStatusDraft    KnowledgeEntryStatus = "draft"
    KnowledgeEntryStatusActive   KnowledgeEntryStatus = "active"
    KnowledgeEntryStatusArchived KnowledgeEntryStatus = "archived"
)

// KnowledgeSourceType represents the source type of knowledge
type KnowledgeSourceType string

const (
    KnowledgeSourceTypeManual  KnowledgeSourceType = "MANUAL"
    KnowledgeSourceTypeLearned KnowledgeSourceType = "LEARNED"
)

// KnowledgeEntry represents a knowledge base entry
type KnowledgeEntry struct {
    ID         string               `json:"id"`
    Title      string               `json:"title"`
    Content    string               `json:"content"`
    Status     KnowledgeEntryStatus `json:"status"`
    Category   string               `json:"category,omitempty"`
    Tags       []string             `json:"tags,omitempty"`
    SourceType KnowledgeSourceType  `json:"source_type"`
    Version    int                  `json:"version"`
    CreatedBy  string               `json:"created_by"`
    CreatedAt  time.Time            `json:"created_at"`
    UpdatedAt  time.Time            `json:"updated_at"`
}

// NewKnowledgeEntry creates a new knowledge base entry
func NewKnowledgeEntry(title, content, category string, tags []string, createdBy string) *KnowledgeEntry {
    now := time.Now()
    return &KnowledgeEntry{
        ID:         generateKBEntryID(),
        Title:      title,
        Content:    content,
        Status:     KnowledgeEntryStatusDraft,
        Category:   category,
        Tags:       tags,
        SourceType: KnowledgeSourceTypeManual,
        Version:    1,
        CreatedBy:  createdBy,
        CreatedAt:  now,
        UpdatedAt:  now,
    }
}

// Publish marks the entry as active and increments version
func (k *KnowledgeEntry) Publish() error {
    if k.Status == KnowledgeEntryStatusArchived {
        return ErrCannotPublishArchived
    }
    k.Status = KnowledgeEntryStatusActive
    k.Version++
    k.UpdatedAt = time.Now()
    return nil
}

// Archive archives the entry
func (k *KnowledgeEntry) Archive() error {
    if k.Status == KnowledgeEntryStatusArchived {
        return ErrAlreadyArchived
    }
    k.Status = KnowledgeEntryStatusArchived
    k.UpdatedAt = time.Now()
    return nil
}

// UpdateContent updates the entry content and sets status to draft
func (k *KnowledgeEntry) UpdateContent(title, content string) {
    k.Title = title
    k.Content = content
    k.Status = KnowledgeEntryStatusDraft
    k.UpdatedAt = time.Now()
}

// IsActive checks if the entry is active
func (k *KnowledgeEntry) IsActive() bool {
    return k.Status == KnowledgeEntryStatusActive
}

// KBChunk represents a chunk of knowledge base content with embedding
type KBChunk struct {
    ID         string    `json:"id"`
    EntryID    string    `json:"entry_id"`
    ChunkIndex int       `json:"chunk_index"`
    Content    string    `json:"content"`
    Embedding  []float32 `json:"embedding,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
}

// NewKBChunk creates a new knowledge base chunk
func NewKBChunk(entryID string, chunkIndex int, content string) *KBChunk {
    return &KBChunk{
        ID:         generateChunkID(),
        EntryID:    entryID,
        ChunkIndex: chunkIndex,
        Content:    content,
        CreatedAt:  time.Now(),
    }
}

// SetEmbedding sets the embedding vector for the chunk
func (k *KBChunk) SetEmbedding(embedding []float32) {
    k.Embedding = embedding
}

// KBChunkFilter represents filters for searching knowledge base chunks
type KBChunkFilter struct {
    Tags     []string `json:"tags,omitempty"`
    Category string   `json:"category,omitempty"`
    Status   string   `json:"status,omitempty"`
    TopK     int      `json:"top_k"`
}

// Knowledge base errors
var (
    ErrKBEntryNotFound        = NewDomainError("knowledge base entry not found")
    ErrCannotPublishArchived  = NewDomainError("cannot publish archived entry")
    ErrAlreadyArchived        = NewDomainError("entry is already archived")
    ErrInvalidEmbedding       = NewDomainError("invalid embedding dimension")
    ErrEmptyKBContent         = NewDomainError("knowledge base content cannot be empty")
    ErrDuplicateKBEntry       = NewDomainError("knowledge base entry already exists")
)

// Helper functions for generating IDs
func generateKBEntryID() string {
    return "kb_" + time.Now().Format("20060102150405")
}

func generateChunkID() string {
    return "chunk_" + time.Now().Format("20060102150405")
}