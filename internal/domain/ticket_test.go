package domain

import (
	"testing"
	"time"
)

func TestNewTicket(t *testing.T) {
	title := "Test Ticket"
	description := "This is a test ticket description"
	category := TicketCategorySoftware
	priority := TicketPriorityMedium
	createdBy := "user123"

	ticket := NewTicket(title, description, category, priority, createdBy)

	if ticket.Title != title {
		t.Errorf("Expected title %s, got %s", title, ticket.Title)
	}

	if ticket.Description != description {
		t.Errorf("Expected description %s, got %s", description, ticket.Description)
	}

	if ticket.Category != category {
		t.Errorf("Expected category %s, got %s", category, ticket.Category)
	}

	if ticket.Priority != priority {
		t.Errorf("Expected priority %s, got %s", priority, ticket.Priority)
	}

	if ticket.Status != TicketStatusOpen {
		t.Errorf("Expected status %s, got %s", TicketStatusOpen, ticket.Status)
	}

	if ticket.CreatedBy != createdBy {
		t.Errorf("Expected createdBy %s, got %s", createdBy, ticket.CreatedBy)
	}

	if ticket.AssignedTo != nil {
		t.Errorf("Expected AssignedTo to be nil, got %v", ticket.AssignedTo)
	}
}

func TestTicket_Assign(t *testing.T) {
	ticket := NewTicket("Test", "Description", TicketCategoryNetwork, TicketPriorityLow, "user1")
	adminID := "admin1"

	err := ticket.Assign(adminID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ticket.AssignedTo == nil {
		t.Error("Expected AssignedTo to be set")
	} else if *ticket.AssignedTo != adminID {
		t.Errorf("Expected AssignedTo %s, got %s", adminID, *ticket.AssignedTo)
	}

	if ticket.Status != TicketStatusInProgress {
		t.Errorf("Expected status %s, got %s", TicketStatusInProgress, ticket.Status)
	}
}

func TestTicket_AssignClosedTicket(t *testing.T) {
	ticket := NewTicket("Test", "Description", TicketCategoryHardware, TicketPriorityHigh, "user1")
	ticket.Status = TicketStatusClosed

	err := ticket.Assign("admin1")
	if err == nil {
		t.Error("Expected error when assigning closed ticket")
	}

	if err != ErrTicketClosed {
		t.Errorf("Expected ErrTicketClosed, got %v", err)
	}
}

func TestTicket_Resolve(t *testing.T) {
	ticket := NewTicket("Test", "Description", TicketCategoryAccount, TicketPriorityCritical, "user1")

	err := ticket.Resolve()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ticket.Status != TicketStatusResolved {
		t.Errorf("Expected status %s, got %s", TicketStatusResolved, ticket.Status)
	}
}

func TestTicket_Close(t *testing.T) {
	ticket := NewTicket("Test", "Description", TicketCategoryOther, TicketPriorityLow, "user1")
	ticket.Status = TicketStatusResolved

	err := ticket.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ticket.Status != TicketStatusClosed {
		t.Errorf("Expected status %s, got %s", TicketStatusClosed, ticket.Status)
	}
}

func TestTicket_CloseUnresolvedTicket(t *testing.T) {
	ticket := NewTicket("Test", "Description", TicketCategorySoftware, TicketPriorityMedium, "user1")

	err := ticket.Close()
	if err == nil {
		t.Error("Expected error when closing unresolved ticket")
	}

	if err != ErrTicketNotResolved {
		t.Errorf("Expected ErrTicketNotResolved, got %v", err)
	}
}

func TestTicket_SetAIInsight(t *testing.T) {
	ticket := NewTicket("Test", "Description", TicketCategoryNetwork, TicketPriorityHigh, "user1")
	insightText := "Try restarting your router"
	confidence := 0.85

	ticket.SetAIInsight(insightText, confidence)

	if ticket.AIInsight == nil {
		t.Error("Expected AIInsight to be set")
	} else {
		if ticket.AIInsight.Text != insightText {
			t.Errorf("Expected AIInsight text %s, got %s", insightText, ticket.AIInsight.Text)
		}
		if ticket.AIInsight.Confidence != confidence {
			t.Errorf("Expected AIInsight confidence %f, got %f", confidence, ticket.AIInsight.Confidence)
		}
	}
}

func TestTicketFilter(t *testing.T) {
	filter := TicketFilter{
		Limit:  10,
		Offset: 0,
	}

	if filter.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", filter.Limit)
	}

	if filter.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", filter.Offset)
	}
}

func TestTicketStatusValues(t *testing.T) {
	tests := []struct {
		status   TicketStatus
		expected string
	}{
		{TicketStatusOpen, "OPEN"},
		{TicketStatusInProgress, "IN_PROGRESS"},
		{TicketStatusResolved, "RESOLVED"},
		{TicketStatusClosed, "CLOSED"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

func TestTicketCategoryValues(t *testing.T) {
	tests := []struct {
		category TicketCategory
		expected string
	}{
		{TicketCategoryNetwork, "NETWORK"},
		{TicketCategorySoftware, "SOFTWARE"},
		{TicketCategoryHardware, "HARDWARE"},
		{TicketCategoryAccount, "ACCOUNT"},
		{TicketCategoryOther, "OTHER"},
	}

	for _, tt := range tests {
		if string(tt.category) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.category))
		}
	}
}

func TestTicketPriorityValues(t *testing.T) {
	tests := []struct {
		priority  TicketPriority
		expected  string
	}{
		{TicketPriorityLow, "LOW"},
		{TicketPriorityMedium, "MEDIUM"},
		{TicketPriorityHigh, "HIGH"},
		{TicketPriorityCritical, "CRITICAL"},
	}

	for _, tt := range tests {
		if string(tt.priority) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.priority))
		}
	}
}

func TestTicketTimestamps(t *testing.T) {
	before := time.Now()
	ticket := NewTicket("Test", "Description", TicketCategorySoftware, TicketPriorityMedium, "user1")
	after := time.Now()

	if ticket.CreatedAt.Before(before) || ticket.CreatedAt.After(after) {
		t.Error("CreatedAt timestamp is not within expected range")
	}

	if ticket.UpdatedAt.Before(before) || ticket.UpdatedAt.After(after) {
		t.Error("UpdatedAt timestamp is not within expected range")
	}

	// CreatedAt and UpdatedAt should be equal initially
	if !ticket.CreatedAt.Equal(ticket.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should be equal initially")
	}

	// UpdatedAt should change when ticket is modified
	oldUpdatedAt := ticket.UpdatedAt
	ticket.SetAIInsight("Test insight", 0.5)

	if !ticket.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated when ticket is modified")
	}
}

func BenchmarkNewTicket(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTicket("Test Ticket", "Description", TicketCategorySoftware, TicketPriorityMedium, "user1")
	}
}

func BenchmarkTicket_Assign(b *testing.B) {
	ticket := NewTicket("Test", "Description", TicketCategoryNetwork, TicketPriorityLow, "user1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset status for each iteration
		ticket.Status = TicketStatusOpen
		ticket.Assign("admin1")
	}
}