package ports

import (
	"context"
	"time"
	"github.com/fixora/fixora/internal/domain"
)

// TicketRepository defines the interface for ticket persistence
type TicketRepository interface {
	// Create saves a new ticket
	Create(ctx context.Context, ticket *domain.Ticket) error

	// FindByID retrieves a ticket by its ID
	FindByID(ctx context.Context, id string) (*domain.Ticket, error)

	// Update updates an existing ticket
	Update(ctx context.Context, ticket *domain.Ticket) error

	// List retrieves tickets based on filter criteria
	List(ctx context.Context, filter domain.TicketFilter) ([]*domain.Ticket, error)

	// Delete removes a ticket (soft delete recommended)
	Delete(ctx context.Context, id string) error

	// Count returns the number of tickets matching the filter
	Count(ctx context.Context, filter domain.TicketFilter) (int, error)
}

// CommentRepository defines the interface for comment persistence
type CommentRepository interface {
    // Create saves a new comment
    Create(ctx context.Context, comment *domain.Comment) error

	// FindByID retrieves a comment by its ID
	FindByID(ctx context.Context, id string) (*domain.Comment, error)

    // ListByTicket retrieves all comments for a ticket
    ListByTicket(ctx context.Context, ticketID string) ([]*domain.Comment, error)

    // ListByTicketWithPagination retrieves comments for a ticket with pagination
    // limit specifies the maximum number of comments to return, and offset specifies
    // the number of comments to skip before starting to collect the result set.
    ListByTicketWithPagination(ctx context.Context, ticketID string, limit, offset int) ([]*domain.Comment, error)

    // Update updates an existing comment
    Update(ctx context.Context, comment *domain.Comment) error

    // Delete removes a comment
    Delete(ctx context.Context, id string) error

    // CountByTicket returns the total number of comments for a ticket
    CountByTicket(ctx context.Context, ticketID string) (int, error)
}

// KnowledgeRepository defines the interface for knowledge base persistence
type KnowledgeRepository interface {
	// CreateEntry saves a new knowledge base entry
	CreateEntry(ctx context.Context, entry *domain.KnowledgeEntry) error

	// FindEntryByID retrieves a knowledge base entry by its ID
	FindEntryByID(ctx context.Context, id string) (*domain.KnowledgeEntry, error)

	// UpdateEntry updates an existing knowledge base entry
	UpdateEntry(ctx context.Context, entry *domain.KnowledgeEntry) error

	// ListEntries retrieves knowledge base entries based on filter
	ListEntries(ctx context.Context, filter domain.KBChunkFilter) ([]*domain.KnowledgeEntry, error)

	// DeleteEntry removes a knowledge base entry
	DeleteEntry(ctx context.Context, id string) error

	// CreateChunk saves a knowledge base chunk
	CreateChunk(ctx context.Context, chunk *domain.KBChunk) error

	// FindChunksByEntry retrieves all chunks for an entry
	FindChunksByEntry(ctx context.Context, entryID string) ([]*domain.KBChunk, error)

	// SearchChunks performs similarity search on chunks
	SearchChunks(ctx context.Context, query string, filter domain.KBChunkFilter) ([]*domain.KBChunk, error)

	// UpdateChunk updates an existing chunk
	UpdateChunk(ctx context.Context, chunk *domain.KBChunk) error

	// DeleteChunksByEntry removes all chunks for an entry
	DeleteChunksByEntry(ctx context.Context, entryID string) error
}

// MetricRepository defines the interface for metrics persistence
type MetricRepository interface {
	// CalculateMetrics generates metrics based on the given filter
	CalculateMetrics(ctx context.Context, filter domain.MetricFilter) (*domain.Metric, error)

	// GetResolutionTimes retrieves resolution times for SLA calculation
	GetResolutionTimes(ctx context.Context, filter domain.MetricFilter) ([]time.Duration, error)

	// GetTicketCounts retrieves ticket counts by status/category
	GetTicketCounts(ctx context.Context, filter domain.MetricFilter) (map[string]int, error)
}

// AuditRepository defines the interface for audit log persistence
type AuditRepository interface {
	// Create creates a new audit entry
	Create(ctx context.Context, audit *domain.AuditEntry) error

	// List retrieves audit entries based on filter
	List(ctx context.Context, resourceType, resourceID string, limit int) ([]*domain.AuditEntry, error)

	// FindByID retrieves an audit entry by its ID
	FindByID(ctx context.Context, id string) (*domain.AuditEntry, error)
}