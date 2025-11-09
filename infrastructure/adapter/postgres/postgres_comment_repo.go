package postgres

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/fixora/fixora/domain"
    outbound "github.com/fixora/fixora/application/port/outbound"
)

// PostgresCommentRepository implements CommentRepository using PostgreSQL
type PostgresCommentRepository struct { db *sql.DB }

func NewPostgresCommentRepository(db *sql.DB) outbound.CommentRepository { return &PostgresCommentRepository{db: db} }

func (r *PostgresCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO comments (id, ticket_id, author_id, role, body, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, comment.ID, comment.TicketID, comment.AuthorID, string(comment.Role), comment.Body, comment.CreatedAt)
    if err != nil { return fmt.Errorf("failed to create comment: %w", err) }
    return nil
}

func (r *PostgresCommentRepository) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
    var comment domain.Comment
    err := r.db.QueryRowContext(ctx, `
        SELECT id, ticket_id, author_id, role, body, created_at FROM comments WHERE id = $1
    `, id).Scan(&comment.ID, &comment.TicketID, &comment.AuthorID, &comment.Role, &comment.Body, &comment.CreatedAt)
    if err != nil { if err == sql.ErrNoRows { return nil, fmt.Errorf("comment not found") }; return nil, fmt.Errorf("failed to find comment: %w", err) }
    return &comment, nil
}

func (r *PostgresCommentRepository) ListByTicket(ctx context.Context, ticketID string) ([]*domain.Comment, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, ticket_id, author_id, role, body, created_at FROM comments WHERE ticket_id = $1 ORDER BY created_at ASC
    `, ticketID)
    if err != nil { return nil, fmt.Errorf("failed to query comments: %w", err) }
    defer rows.Close()
    var comments []*domain.Comment
    for rows.Next() {
        var c domain.Comment
        if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Role, &c.Body, &c.CreatedAt); err != nil { return nil, fmt.Errorf("failed to scan comment: %w", err) }
        comments = append(comments, &c)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("error iterating comments: %w", err) }
    return comments, nil
}

func (r *PostgresCommentRepository) ListByTicketWithPagination(ctx context.Context, ticketID string, limit, offset int) ([]*domain.Comment, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, ticket_id, author_id, role, body, created_at FROM comments
        WHERE ticket_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3
    `, ticketID, limit, offset)
    if err != nil { return nil, fmt.Errorf("failed to query comments with pagination: %w", err) }
    defer rows.Close()
    var comments []*domain.Comment
    for rows.Next() {
        var c domain.Comment
        if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Role, &c.Body, &c.CreatedAt); err != nil { return nil, fmt.Errorf("failed to scan comment: %w", err) }
        comments = append(comments, &c)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("error iterating comments: %w", err) }
    return comments, nil
}

func (r *PostgresCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
    result, err := r.db.ExecContext(ctx, `UPDATE comments SET body = $2 WHERE id = $1`, comment.ID, comment.Body)
    if err != nil { return fmt.Errorf("failed to update comment: %w", err) }
    rowsAffected, err := result.RowsAffected()
    if err != nil { return fmt.Errorf("failed to get rows affected: %w", err) }
    if rowsAffected == 0 { return fmt.Errorf("comment not found") }
    return nil
}

func (r *PostgresCommentRepository) Delete(ctx context.Context, id string) error {
    result, err := r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
    if err != nil { return fmt.Errorf("failed to delete comment: %w", err) }
    rowsAffected, err := result.RowsAffected()
    if err != nil { return fmt.Errorf("failed to get rows affected: %w", err) }
    if rowsAffected == 0 { return fmt.Errorf("comment not found") }
    return nil
}

func (r *PostgresCommentRepository) CountByTicket(ctx context.Context, ticketID string) (int, error) {
    var count int
    err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE ticket_id = $1`, ticketID).Scan(&count)
    if err != nil { return 0, fmt.Errorf("failed to count comments: %w", err) }
    return count, nil
}