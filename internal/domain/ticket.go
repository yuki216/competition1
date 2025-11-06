package domain

import (
	"time"
)

// TicketStatus represents the status of a ticket
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "OPEN"
	TicketStatusInProgress TicketStatus = "IN_PROGRESS"
	TicketStatusResolved   TicketStatus = "RESOLVED"
	TicketStatusClosed     TicketStatus = "CLOSED"
)

// TicketCategory represents the category of a ticket
type TicketCategory string

const (
	TicketCategoryNetwork  TicketCategory = "NETWORK"
	TicketCategorySoftware TicketCategory = "SOFTWARE"
	TicketCategoryHardware TicketCategory = "HARDWARE"
	TicketCategoryAccount  TicketCategory = "ACCOUNT"
	TicketCategoryOther    TicketCategory = "OTHER"
)

// TicketPriority represents the priority of a ticket
type TicketPriority string

const (
	TicketPriorityLow      TicketPriority = "LOW"
	TicketPriorityMedium   TicketPriority = "MEDIUM"
	TicketPriorityHigh     TicketPriority = "HIGH"
	TicketPriorityCritical TicketPriority = "CRITICAL"
)

// AIInsight represents AI-generated insights for a ticket
type AIInsight struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// Ticket represents an IT support ticket
type Ticket struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      TicketStatus    `json:"status"`
	Category    TicketCategory  `json:"category"`
	Priority    TicketPriority  `json:"priority"`
	CreatedBy   string          `json:"created_by"`
	AssignedTo  *string         `json:"assigned_to,omitempty"`
	AIInsight   *AIInsight      `json:"ai_insight,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// NewTicket creates a new ticket
func NewTicket(title, description string, category TicketCategory, priority TicketPriority, createdBy string) *Ticket {
	now := time.Now()
	return &Ticket{
		ID:          generateID(),
		Title:       title,
		Description: description,
		Status:      TicketStatusOpen,
		Category:    category,
		Priority:    priority,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Assign assigns the ticket to an admin
func (t *Ticket) Assign(adminID string) error {
	if t.Status == TicketStatusClosed {
		return ErrTicketClosed
	}
	t.AssignedTo = &adminID
	t.Status = TicketStatusInProgress
	t.UpdatedAt = time.Now()
	return nil
}

// Resolve marks the ticket as resolved
func (t *Ticket) Resolve() error {
	if t.Status == TicketStatusClosed {
		return ErrTicketClosed
	}
	t.Status = TicketStatusResolved
	t.UpdatedAt = time.Now()
	return nil
}

// Close closes the ticket
func (t *Ticket) Close() error {
	if t.Status != TicketStatusResolved {
		return ErrTicketNotResolved
	}
	t.Status = TicketStatusClosed
	t.UpdatedAt = time.Now()
	return nil
}

// SetAIInsight sets the AI insight for the ticket
func (t *Ticket) SetAIInsight(text string, confidence float64) {
	t.AIInsight = &AIInsight{
		Text:       text,
		Confidence: confidence,
	}
	t.UpdatedAt = time.Now()
}

// TicketFilter represents filters for listing tickets
type TicketFilter struct {
	Status     *TicketStatus     `json:"status,omitempty"`
	Category   *TicketCategory   `json:"category,omitempty"`
	Priority   *TicketPriority   `json:"priority,omitempty"`
	CreatedBy  *string           `json:"created_by,omitempty"`
	AssignedTo *string           `json:"assigned_to,omitempty"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// Custom errors
var (
	ErrTicketNotFound     = NewDomainError("ticket not found")
	ErrTicketClosed       = NewDomainError("cannot modify closed ticket")
	ErrTicketNotResolved  = NewDomainError("ticket must be resolved before closing")
	ErrInvalidAssignment  = NewDomainError("invalid assignment")
	ErrInvalidStatus      = NewDomainError("invalid status transition")
)

// DomainError represents a domain-specific error
type DomainError struct {
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

func NewDomainError(message string) *DomainError {
	return &DomainError{Message: message}
}

// Helper function for generating IDs (in production, use proper UUID)
func generateID() string {
	// In a real implementation, use github.com/google/uuid
	return "ticket_" + time.Now().Format("20060102150405")
}