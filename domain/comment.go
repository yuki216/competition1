package domain

import (
    "time"
)

// CommentRole represents the role of the comment author
type CommentRole string

const (
    CommentRoleEmployee CommentRole = "EMPLOYEE"
    CommentRoleAdmin    CommentRole = "ADMIN"
    CommentRoleAI       CommentRole = "AI"
)

// Comment represents a comment on a ticket
type Comment struct {
    ID        string      `json:"id"`
    TicketID  string      `json:"ticket_id"`
    AuthorID  string      `json:"author_id"`
    Role      CommentRole `json:"role"`
    Body      string      `json:"body"`
    CreatedAt time.Time   `json:"created_at"`
}

// NewComment creates a new comment
func NewComment(ticketID, authorID string, role CommentRole, body string) *Comment {
    return &Comment{
        ID:        generateCommentID(),
        TicketID:  ticketID,
        AuthorID:  authorID,
        Role:      role,
        Body:      body,
        CreatedAt: time.Now(),
    }
}

// IsValid checks if the comment is valid
func (c *Comment) IsValid() error {
    if c.TicketID == "" {
        return ErrEmptyTicketID
    }
    if c.AuthorID == "" {
        return ErrEmptyAuthorID
    }
    if c.Body == "" {
        return ErrEmptyCommentBody
    }
    if c.Role != CommentRoleEmployee && c.Role != CommentRoleAdmin && c.Role != CommentRoleAI {
        return ErrInvalidCommentRole
    }
    return nil
}

// Comment errors
var (
    ErrEmptyTicketID      = NewDomainError("ticket ID cannot be empty")
    ErrEmptyAuthorID      = NewDomainError("author ID cannot be empty")
    ErrEmptyCommentBody   = NewDomainError("comment body cannot be empty")
    ErrInvalidCommentRole = NewDomainError("invalid comment role")
)

// Helper function for generating comment IDs
func generateCommentID() string {
    return "comment_" + time.Now().Format("20060102150405")
}