package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fixora/fixora/internal/domain"
	"github.com/fixora/fixora/internal/ports"
)

// PostgresTicketRepository implements TicketRepository using PostgreSQL
type PostgresTicketRepository struct {
	db *sql.DB
}

// NewPostgresTicketRepository creates a new PostgreSQL ticket repository
func NewPostgresTicketRepository(db *sql.DB) ports.TicketRepository {
	return &PostgresTicketRepository{db: db}
}

// Create saves a new ticket
func (r *PostgresTicketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	query := `
		INSERT INTO tickets (id, title, description, status, category, priority, created_by, assigned_to, ai_insight, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	var aiInsightJSON []byte
	var err error

	if ticket.AIInsight != nil {
		aiInsightJSON, err = json.Marshal(ticket.AIInsight)
		if err != nil {
			return fmt.Errorf("failed to marshal AI insight: %w", err)
		}
	}

	var assignedTo *string
	if ticket.AssignedTo != nil {
		assignedTo = ticket.AssignedTo
	}

	_, err = r.db.ExecContext(ctx, query,
		ticket.ID,
		ticket.Title,
		ticket.Description,
		string(ticket.Status),
		string(ticket.Category),
		string(ticket.Priority),
		ticket.CreatedBy,
		assignedTo,
		aiInsightJSON,
		ticket.CreatedAt,
		ticket.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	return nil
}

// FindByID retrieves a ticket by its ID
func (r *PostgresTicketRepository) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	query := `
		SELECT id, title, description, status, category, priority, created_by, assigned_to, ai_insight, created_at, updated_at
		FROM tickets
		WHERE id = $1
	`

	var ticket domain.Ticket
	var assignedTo sql.NullString
	var aiInsightJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ticket.ID,
		&ticket.Title,
		&ticket.Description,
		&ticket.Status,
		&ticket.Category,
		&ticket.Priority,
		&ticket.CreatedBy,
		&assignedTo,
		&aiInsightJSON,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	if assignedTo.Valid {
		ticket.AssignedTo = &assignedTo.String
	}

	if len(aiInsightJSON) > 0 {
		var aiInsight domain.AIInsight
		if err := json.Unmarshal(aiInsightJSON, &aiInsight); err != nil {
			return nil, fmt.Errorf("failed to unmarshal AI insight: %w", err)
		}
		ticket.AIInsight = &aiInsight
	}

	return &ticket, nil
}

// Update updates an existing ticket
func (r *PostgresTicketRepository) Update(ctx context.Context, ticket *domain.Ticket) error {
	query := `
		UPDATE tickets
		SET title = $2, description = $3, status = $4, category = $5, priority = $6,
			assigned_to = $7, ai_insight = $8, updated_at = $9
		WHERE id = $1
	`

	var aiInsightJSON []byte
	var err error

	if ticket.AIInsight != nil {
		aiInsightJSON, err = json.Marshal(ticket.AIInsight)
		if err != nil {
			return fmt.Errorf("failed to marshal AI insight: %w", err)
		}
	}

	var assignedTo *string
	if ticket.AssignedTo != nil {
		assignedTo = ticket.AssignedTo
	}

	result, err := r.db.ExecContext(ctx, query,
		ticket.ID,
		ticket.Title,
		ticket.Description,
		string(ticket.Status),
		string(ticket.Category),
		string(ticket.Priority),
		assignedTo,
		aiInsightJSON,
		ticket.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrTicketNotFound
	}

	return nil
}

// List retrieves tickets based on filter criteria
func (r *PostgresTicketRepository) List(ctx context.Context, filter domain.TicketFilter) ([]*domain.Ticket, error) {
	query := `
		SELECT id, title, description, status, category, priority, created_by, assigned_to, ai_insight, created_at, updated_at
		FROM tickets
		WHERE 1=1
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE conditions
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*filter.Status))
		argIndex++
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, string(*filter.Category))
		argIndex++
	}

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, string(*filter.Priority))
		argIndex++
	}

	if filter.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIndex))
		args = append(args, *filter.CreatedBy)
		argIndex++
	}

	if filter.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, *filter.AssignedTo)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	// Add LIMIT and OFFSET
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*domain.Ticket

	for rows.Next() {
		var ticket domain.Ticket
		var assignedTo sql.NullString
		var aiInsightJSON []byte

		err := rows.Scan(
			&ticket.ID,
			&ticket.Title,
			&ticket.Description,
			&ticket.Status,
			&ticket.Category,
			&ticket.Priority,
			&ticket.CreatedBy,
			&assignedTo,
			&aiInsightJSON,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		if assignedTo.Valid {
			ticket.AssignedTo = &assignedTo.String
		}

		if len(aiInsightJSON) > 0 {
			var aiInsight domain.AIInsight
			if err := json.Unmarshal(aiInsightJSON, &aiInsight); err != nil {
				return nil, fmt.Errorf("failed to unmarshal AI insight: %w", err)
			}
			ticket.AIInsight = &aiInsight
		}

		tickets = append(tickets, &ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tickets: %w", err)
	}

	return tickets, nil
}

// Delete removes a ticket (soft delete - update status to CLOSED)
func (r *PostgresTicketRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE tickets SET status = 'CLOSED', updated_at = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrTicketNotFound
	}

	return nil
}

// Count returns the number of tickets matching the filter
func (r *PostgresTicketRepository) Count(ctx context.Context, filter domain.TicketFilter) (int, error) {
	query := `SELECT COUNT(*) FROM tickets WHERE 1=1`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE conditions (same as List method)
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*filter.Status))
		argIndex++
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, string(*filter.Category))
		argIndex++
	}

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, string(*filter.Priority))
		argIndex++
	}

	if filter.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIndex))
		args = append(args, *filter.CreatedBy)
		argIndex++
	}

	if filter.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, *filter.AssignedTo)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tickets: %w", err)
	}

	return count, nil
}

// Helper method to map SQL null types
func mapStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Helper method to build WHERE clause for complex filters
func (r *PostgresTicketRepository) buildWhereClause(filter domain.TicketFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*filter.Status))
		argIndex++
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, string(*filter.Category))
		argIndex++
	}

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, string(*filter.Priority))
		argIndex++
	}

	if filter.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIndex))
		args = append(args, *filter.CreatedBy)
		argIndex++
	}

	if filter.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argIndex))
		args = append(args, *filter.AssignedTo)
		argIndex++
	}

	var whereClause string
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}