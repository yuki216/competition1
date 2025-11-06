package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/ports"
)

// PostgresCommentRepository implements CommentRepository using PostgreSQL
type PostgresCommentRepository struct {
	db *sql.DB
}

// NewPostgresCommentRepository creates a new PostgreSQL comment repository
func NewPostgresCommentRepository(db *sql.DB) ports.CommentRepository {
	return &PostgresCommentRepository{db: db}
}

// Create saves a new comment
func (r *PostgresCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	query := `
		INSERT INTO comments (id, ticket_id, author_id, role, body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		comment.ID,
		comment.TicketID,
		comment.AuthorID,
		string(comment.Role),
		comment.Body,
		comment.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// FindByID retrieves a comment by its ID
func (r *PostgresCommentRepository) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE id = $1
	`

	var comment domain.Comment

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&comment.ID,
		&comment.TicketID,
		&comment.AuthorID,
		&comment.Role,
		&comment.Body,
		&comment.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("comment not found")
		}
		return nil, fmt.Errorf("failed to find comment: %w", err)
	}

	return &comment, nil
}

// ListByTicket retrieves all comments for a ticket
func (r *PostgresCommentRepository) ListByTicket(ctx context.Context, ticketID string) ([]*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.Role,
			&comment.Body,
			&comment.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// Update updates an existing comment
func (r *PostgresCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	query := `
		UPDATE comments
		SET body = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, comment.ID, comment.Body)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

// Delete removes a comment
func (r *PostgresCommentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM comments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

// ListByTicketWithPagination retrieves comments for a ticket with pagination
func (r *PostgresCommentRepository) ListByTicketWithPagination(ctx context.Context, ticketID string, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE ticket_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, ticketID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments with pagination: %w", err)
	}
	defer rows.Close()

	var comments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.Role,
			&comment.Body,
			&comment.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// CountByTicket returns the number of comments for a ticket
func (r *PostgresCommentRepository) CountByTicket(ctx context.Context, ticketID string) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE ticket_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, ticketID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return count, nil
}

// ListByAuthor retrieves comments by a specific author
func (r *PostgresCommentRepository) ListByAuthor(ctx context.Context, authorID string, limit int) ([]*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE author_id = $1
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryContext(ctx, query, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments by author: %w", err)
	}
	defer rows.Close()

	var comments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.Role,
			&comment.Body,
			&comment.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// ListByRole retrieves comments by role
func (r *PostgresCommentRepository) ListByRole(ctx context.Context, role domain.CommentRole, limit int) ([]*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE role = $1
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryContext(ctx, query, string(role))
	if err != nil {
		return nil, fmt.Errorf("failed to query comments by role: %w", err)
	}
	defer rows.Close()

	var comments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.Role,
			&comment.Body,
			&comment.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// SearchInComments searches for text within comment bodies
func (r *PostgresCommentRepository) SearchInComments(ctx context.Context, searchTerm string, limit int) ([]*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE body ILIKE $1
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	rows, err := r.db.QueryContext(ctx, query, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search comments: %w", err)
	}
	defer rows.Close()

	var comments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.Role,
			&comment.Body,
			&comment.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

// GetRecentComments retrieves recent comments across all tickets
func (r *PostgresCommentRepository) GetRecentComments(ctx context.Context, hours int, limit int) ([]*domain.Comment, error) {
	query := `
		SELECT id, ticket_id, author_id, role, body, created_at
		FROM comments
		WHERE created_at >= $1
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	cutoffTime := time.Now().Add(time.Duration(-hours) * time.Hour)

	rows, err := r.db.QueryContext(ctx, query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent comments: %w", err)
	}
	defer rows.Close()

	var comments []*domain.Comment

	for rows.Next() {
		var comment domain.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.Role,
			&comment.Body,
			&comment.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}