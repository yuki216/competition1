package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/ports"
)

// CreateCommentRequest represents the request to create a comment
type CreateCommentRequest struct {
	TicketID string            `json:"ticket_id" validate:"required"`
	AuthorID string            `json:"author_id" validate:"required"`
	Role     domain.CommentRole `json:"role" validate:"required"`
	Body     string            `json:"body" validate:"required,min=1,max=5000"`
}

// UpdateCommentRequest represents the request to update a comment
type UpdateCommentRequest struct {
	Body string `json:"body" validate:"required,min=1,max=5000"`
}

// CommentResponse represents the response for a comment
type CommentResponse struct {
	Comment *domain.Comment `json:"comment"`
}

// ListCommentsResponse represents the response for listing comments
type ListCommentsResponse struct {
	Comments []*domain.Comment `json:"comments"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PerPage  int               `json:"per_page"`
}

// CommentUseCase handles comment-related business logic
type CommentUseCase struct {
	commentRepo   ports.CommentRepository
	ticketRepo    ports.TicketRepository
	eventPublisher ports.EventPublisher
	notifyService ports.NotificationService
}

// NewCommentUseCase creates a new comment use case
func NewCommentUseCase(
	commentRepo ports.CommentRepository,
	ticketRepo ports.TicketRepository,
	eventPublisher ports.EventPublisher,
	notifyService ports.NotificationService,
) *CommentUseCase {
	return &CommentUseCase{
		commentRepo:   commentRepo,
		ticketRepo:    ticketRepo,
		eventPublisher: eventPublisher,
		notifyService: notifyService,
	}
}

// CreateComment creates a new comment on a ticket
func (uc *CommentUseCase) CreateComment(ctx context.Context, req CreateCommentRequest) (*CommentResponse, error) {
	// Validate request
	if err := uc.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if ticket exists
	ticket, err := uc.ticketRepo.FindByID(ctx, req.TicketID)
	if err != nil {
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}
	if ticket == nil {
		return nil, fmt.Errorf("ticket not found")
	}

	// Create comment
	comment := domain.NewComment(req.TicketID, req.AuthorID, req.Role, req.Body)

	// Validate comment
	if err := comment.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid comment: %w", err)
	}

	// Save comment
	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			"comment.created",
			"comment",
			comment.ID,
			map[string]interface{}{
				"ticket_id":   comment.TicketID,
				"author_id":   comment.AuthorID,
				"role":        comment.Role,
				"body":        comment.Body,
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

    // Send notification (align with ports.NotificationService interface)
    if uc.notifyService != nil {
        _ = uc.notifyService.NotifyCommentAdded(ctx, comment, ticket)
    }

	return &CommentResponse{
		Comment: comment,
	}, nil
}

// GetCommentsByTicket retrieves comments for a ticket with pagination
func (uc *CommentUseCase) GetCommentsByTicket(ctx context.Context, ticketID string, page, perPage int) (*ListCommentsResponse, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}

	// Validate pagination
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Check if ticket exists
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}
	if ticket == nil {
		return nil, fmt.Errorf("ticket not found")
	}

    // Fetch all comments then paginate in-memory since repository doesn't provide pagination/count
    allComments, err := uc.commentRepo.ListByTicket(ctx, ticketID)
    if err != nil {
        return nil, fmt.Errorf("failed to list comments: %w", err)
    }

    total := len(allComments)
    start := (page - 1) * perPage
    if start < 0 {
        start = 0
    }
    if start > total {
        start = total
    }
    end := start + perPage
    if end > total {
        end = total
    }
    comments := allComments[start:end]

	return &ListCommentsResponse{
		Comments: comments,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
	}, nil
}

// UpdateComment updates an existing comment
func (uc *CommentUseCase) UpdateComment(ctx context.Context, commentID string, req UpdateCommentRequest) (*CommentResponse, error) {
	if commentID == "" {
		return nil, fmt.Errorf("comment ID is required")
	}

	// Validate request
	if err := uc.validateUpdateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing comment
	comment, err := uc.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find comment: %w", err)
	}
	if comment == nil {
		return nil, fmt.Errorf("comment not found")
	}

	// Check if comment can be edited (15-minute window)
	if time.Since(comment.CreatedAt) > 15*time.Minute {
		return nil, fmt.Errorf("comment can only be edited within 15 minutes of creation")
	}

	// Update comment body
	comment.Body = req.Body

	// Validate updated comment
	if err := comment.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid comment: %w", err)
	}

	// Save changes
	if err := uc.commentRepo.Update(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			"comment.updated",
			"comment",
			comment.ID,
			map[string]interface{}{
				"ticket_id": comment.TicketID,
				"author_id": comment.AuthorID,
				"body":      comment.Body,
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return &CommentResponse{
		Comment: comment,
	}, nil
}

// DeleteComment deletes a comment
func (uc *CommentUseCase) DeleteComment(ctx context.Context, commentID string) error {
	if commentID == "" {
		return fmt.Errorf("comment ID is required")
	}

	// Get existing comment
	comment, err := uc.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return fmt.Errorf("failed to find comment: %w", err)
	}
	if comment == nil {
		return fmt.Errorf("comment not found")
	}

	// Check if comment can be deleted (15-minute window)
	if time.Since(comment.CreatedAt) > 15*time.Minute {
		return fmt.Errorf("comment can only be deleted within 15 minutes of creation")
	}

	// Delete comment
	if err := uc.commentRepo.Delete(ctx, commentID); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			"comment.deleted",
			"comment",
			commentID,
			map[string]interface{}{
				"ticket_id": comment.TicketID,
				"author_id": comment.AuthorID,
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return nil
}

// Helper functions

func (uc *CommentUseCase) validateCreateRequest(req CreateCommentRequest) error {
	if req.TicketID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	if req.AuthorID == "" {
		return fmt.Errorf("author ID is required")
	}
	if req.Body == "" {
		return fmt.Errorf("comment body is required")
	}
	if len(req.Body) > 5000 {
		return fmt.Errorf("comment body must not exceed 5000 characters")
	}

	// Validate role
	validRoles := map[domain.CommentRole]bool{
		domain.CommentRoleEmployee: true,
		domain.CommentRoleAdmin:    true,
		domain.CommentRoleAI:       true,
	}
	if !validRoles[req.Role] {
		return fmt.Errorf("invalid comment role: %s", req.Role)
	}

	return nil
}

func (uc *CommentUseCase) validateUpdateRequest(req UpdateCommentRequest) error {
	if req.Body == "" {
		return fmt.Errorf("comment body is required")
	}
	if len(req.Body) > 5000 {
		return fmt.Errorf("comment body must not exceed 5000 characters")
	}
	return nil
}