package usecase

import (
	"context"
	"fmt"
	"time"

	"fixora/internal/domain"
	"fixora/internal/ports"
)

// CreateTicketRequest represents the request to create a ticket
type CreateTicketRequest struct {
	Title       string                `json:"title" validate:"required,min=3,max=200"`
	Description string                `json:"description" validate:"required,min=10,max=2000"`
	Category    domain.TicketCategory `json:"category" validate:"required"`
	Priority    domain.TicketPriority `json:"priority" validate:"required"`
	CreatedBy   string                `json:"created_by" validate:"required"`
	UseAI       bool                  `json:"use_ai"`
}

// CreateTicketResponse represents the response after creating a ticket
type CreateTicketResponse struct {
	Ticket    *domain.Ticket        `json:"ticket"`
	AIInsight *ports.SuggestionResult `json:"ai_insight,omitempty"`
}

// TicketUseCase handles ticket-related business logic
type TicketUseCase struct {
	ticketRepo    ports.TicketRepository
	commentRepo   ports.CommentRepository
	aiService     ports.AISuggestionService
	eventPublisher ports.EventPublisher
	notifyService ports.NotificationService
}

// NewTicketUseCase creates a new ticket use case
func NewTicketUseCase(
	ticketRepo ports.TicketRepository,
	commentRepo ports.CommentRepository,
	aiService ports.AISuggestionService,
	eventPublisher ports.EventPublisher,
	notifyService ports.NotificationService,
) *TicketUseCase {
	return &TicketUseCase{
		ticketRepo:    ticketRepo,
		commentRepo:   commentRepo,
		aiService:     aiService,
		eventPublisher: eventPublisher,
		notifyService: notifyService,
	}
}

// CreateTicket creates a new ticket with optional AI suggestion
func (uc *TicketUseCase) CreateTicket(ctx context.Context, req CreateTicketRequest) (*CreateTicketResponse, error) {
	// Validate request
	if err := uc.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create ticket
	ticket := domain.NewTicket(req.Title, req.Description, req.Category, req.Priority, req.CreatedBy)

	// Get AI suggestion if requested
	var aiInsight *ports.SuggestionResult
	if req.UseAI && uc.aiService != nil {
		suggestion, err := uc.aiService.SuggestMitigation(ctx, req.Description)
		if err == nil && suggestion.Confidence >= 0.4 {
			ticket.SetAIInsight(suggestion.Suggestion, suggestion.Confidence)
			aiInsight = &suggestion
		}
		// Log AI suggestion failure but don't fail ticket creation
	}

	// Save ticket
	if err := uc.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeTicketCreated,
			"ticket",
			ticket.ID,
			map[string]interface{}{
				"title":       ticket.Title,
				"category":    ticket.Category,
				"priority":    ticket.Priority,
				"created_by":  ticket.CreatedBy,
				"ai_insight":  ticket.AIInsight,
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event) // Log error but don't fail
	}

	// Send notification
	if uc.notifyService != nil {
		_ = uc.notifyService.NotifyTicketCreated(ctx, ticket) // Log error but don't fail
	}

	return &CreateTicketResponse{
		Ticket:    ticket,
		AIInsight: aiInsight,
	}, nil
}

// GetTicket retrieves a ticket by ID
func (uc *TicketUseCase) GetTicket(ctx context.Context, ticketID string) (*domain.Ticket, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}

	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return ticket, nil
}

// ListTickets retrieves tickets based on filter criteria
func (uc *TicketUseCase) ListTickets(ctx context.Context, filter domain.TicketFilter) ([]*domain.Ticket, int, error) {
	// Set default pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	tickets, err := uc.ticketRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tickets: %w", err)
	}

	count, err := uc.ticketRepo.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tickets: %w", err)
	}

	return tickets, count, nil
}

// AssignTicket assigns a ticket to an admin
func (uc *TicketUseCase) AssignTicket(ctx context.Context, ticketID, adminID string) (*domain.Ticket, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}
	if adminID == "" {
		return nil, fmt.Errorf("admin ID is required")
	}

	// Get ticket
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// Assign ticket
	if err := ticket.Assign(adminID); err != nil {
		return nil, fmt.Errorf("failed to assign ticket: %w", err)
	}

	// Save changes
	if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeTicketAssigned,
			"ticket",
			ticket.ID,
			map[string]interface{}{
				"assigned_to": adminID,
				"assigned_by": "system", // In real implementation, get from context
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	// Send notification
	if uc.notifyService != nil {
		_ = uc.notifyService.NotifyTicketAssigned(ctx, ticket, adminID)
	}

	return ticket, nil
}

// ResolveTicket marks a ticket as resolved
func (uc *TicketUseCase) ResolveTicket(ctx context.Context, ticketID, resolution string) (*domain.Ticket, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}
	if resolution == "" {
		return nil, fmt.Errorf("resolution is required")
	}

	// Get ticket
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// Resolve ticket
	if err := ticket.Resolve(); err != nil {
		return nil, fmt.Errorf("failed to resolve ticket: %w", err)
	}

	// Save changes
	if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Add resolution comment
	if uc.commentRepo != nil {
		comment := domain.NewComment(
			ticketID,
			"system", // In real implementation, get from context
			domain.CommentRoleAdmin,
			fmt.Sprintf("Ticket resolved: %s", resolution),
		)
		_ = uc.commentRepo.Create(ctx, comment)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeTicketResolved,
			"ticket",
			ticket.ID,
			map[string]interface{}{
				"resolution": resolution,
				"resolved_by": "system", // In real implementation, get from context
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	// Send notification
	if uc.notifyService != nil {
		_ = uc.notifyService.NotifyTicketResolved(ctx, ticket)
	}

	return ticket, nil
}

// CloseTicket closes a ticket (must be resolved first)
func (uc *TicketUseCase) CloseTicket(ctx context.Context, ticketID string) (*domain.Ticket, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}

	// Get ticket
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// Close ticket
	if err := ticket.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ticket: %w", err)
	}

	// Save changes
	if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeTicketUpdated,
			"ticket",
			ticket.ID,
			map[string]interface{}{
				"status":      ticket.Status,
				"updated_by": "system", // In real implementation, get from context
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return ticket, nil
}

// UpdateTicket updates ticket information
func (uc *TicketUseCase) UpdateTicket(ctx context.Context, ticketID string, updates map[string]interface{}) (*domain.Ticket, error) {
	if ticketID == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}

	// Get current ticket
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// Apply updates
	if title, ok := updates["title"].(string); ok && title != "" {
		ticket.Title = title
	}

	if description, ok := updates["description"].(string); ok && description != "" {
		ticket.Description = description
	}

	if category, ok := updates["category"].(domain.TicketCategory); ok {
		ticket.Category = category
	}

	if priority, ok := updates["priority"].(domain.TicketPriority); ok {
		ticket.Priority = priority
	}

	// Save changes
	if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Publish event
	if uc.eventPublisher != nil {
		event := ports.NewEvent(
			ports.EventTypeTicketUpdated,
			"ticket",
			ticket.ID,
			map[string]interface{}{
				"updates":    updates,
				"updated_by": "system", // In real implementation, get from context
			},
			1,
		)
		_ = uc.eventPublisher.Publish(ctx, *event)
	}

	return ticket, nil
}

// GetTicketStats retrieves ticket statistics for dashboard
func (uc *TicketUseCase) GetTicketStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	// Get counts by status
	statusFilters := []domain.TicketStatus{
		domain.TicketStatusOpen,
		domain.TicketStatusInProgress,
		domain.TicketStatusResolved,
		domain.TicketStatusClosed,
	}

	for _, status := range statusFilters {
		filter := domain.TicketFilter{Status: &status}
		count, err := uc.ticketRepo.Count(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to count tickets by status %s: %w", status, err)
		}
		stats[string(status)] = count
	}

	// Get total count
	totalCount, err := uc.ticketRepo.Count(ctx, domain.TicketFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total tickets: %w", err)
	}
	stats["total"] = totalCount

	return stats, nil
}

// Helper functions

func (uc *TicketUseCase) validateCreateRequest(req CreateTicketRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) < 3 {
		return fmt.Errorf("title must be at least 3 characters")
	}
	if len(req.Title) > 200 {
		return fmt.Errorf("title must not exceed 200 characters")
	}

	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(req.Description) < 10 {
		return fmt.Errorf("description must be at least 10 characters")
	}
	if len(req.Description) > 2000 {
		return fmt.Errorf("description must not exceed 2000 characters")
	}

	if req.CreatedBy == "" {
		return fmt.Errorf("created by is required")
	}

	// Validate category
	validCategories := map[domain.TicketCategory]bool{
		domain.TicketCategoryNetwork:  true,
		domain.TicketCategorySoftware: true,
		domain.TicketCategoryHardware: true,
		domain.TicketCategoryAccount:  true,
		domain.TicketCategoryOther:    true,
	}
	if !validCategories[req.Category] {
		return fmt.Errorf("invalid category: %s", req.Category)
	}

	// Validate priority
	validPriorities := map[domain.TicketPriority]bool{
		domain.TicketPriorityLow:      true,
		domain.TicketPriorityMedium:   true,
		domain.TicketPriorityHigh:     true,
		domain.TicketPriorityCritical: true,
	}
	if !validPriorities[req.Priority] {
		return fmt.Errorf("invalid priority: %s", req.Priority)
	}

	return nil
}