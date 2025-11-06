package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fixora/internal/domain"
	"fixora/internal/ports"
)

// PostgresKnowledgeRepository implements KnowledgeRepository using PostgreSQL with pgvector
type PostgresKnowledgeRepository struct {
	db         *sql.DB
	embeddings ports.EmbeddingProvider
}

// NewPostgresKnowledgeRepository creates a new PostgreSQL knowledge repository
func NewPostgresKnowledgeRepository(db *sql.DB, embeddings ports.EmbeddingProvider) ports.KnowledgeRepository {
	return &PostgresKnowledgeRepository{
		db:         db,
		embeddings: embeddings,
	}
}

// CreateEntry saves a new knowledge base entry
func (r *PostgresKnowledgeRepository) CreateEntry(ctx context.Context, entry *domain.KnowledgeEntry) error {
	query := `
		INSERT INTO knowledge_entries (id, title, content, status, category, tags, source_type, version, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		entry.ID,
		entry.Title,
		entry.Content,
		string(entry.Status),
		entry.Category,
		tagsJSON,
		string(entry.SourceType),
		entry.Version,
		entry.CreatedBy,
		entry.CreatedAt,
		entry.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create knowledge entry: %w", err)
	}

	return nil
}

// FindEntryByID retrieves a knowledge base entry by its ID
func (r *PostgresKnowledgeRepository) FindEntryByID(ctx context.Context, id string) (*domain.KnowledgeEntry, error) {
	query := `
		SELECT id, title, content, status, category, tags, source_type, version, created_by, created_at, updated_at
		FROM knowledge_entries
		WHERE id = $1
	`

	var entry domain.KnowledgeEntry
	var tagsJSON []byte
	var category sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.Title,
		&entry.Content,
		&entry.Status,
		&category,
		&tagsJSON,
		&entry.SourceType,
		&entry.Version,
		&entry.CreatedBy,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrKBEntryNotFound
		}
		return nil, fmt.Errorf("failed to find knowledge entry: %w", err)
	}

	if category.Valid {
		entry.Category = category.String
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &entry.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	return &entry, nil
}

// UpdateEntry updates an existing knowledge base entry
func (r *PostgresKnowledgeRepository) UpdateEntry(ctx context.Context, entry *domain.KnowledgeEntry) error {
	query := `
		UPDATE knowledge_entries
		SET title = $2, content = $3, status = $4, category = $5, tags = $6, version = $7, updated_at = $8
		WHERE id = $1
	`

	tagsJSON, err := json.Marshal(entry.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.Title,
		entry.Content,
		string(entry.Status),
		entry.Category,
		tagsJSON,
		entry.Version,
		entry.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update knowledge entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrKBEntryNotFound
	}

	return nil
}

// ListEntries retrieves knowledge base entries based on filter
func (r *PostgresKnowledgeRepository) ListEntries(ctx context.Context, filter domain.KBChunkFilter) ([]*domain.KnowledgeEntry, error) {
	query := `
		SELECT DISTINCT ke.id, ke.title, ke.content, ke.status, ke.category, ke.tags, ke.source_type, ke.version, ke.created_by, ke.created_at, ke.updated_at
		FROM knowledge_entries ke
		LEFT JOIN kb_chunks kc ON ke.id = kc.entry_id
		WHERE 1=1
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE conditions
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("ke.status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("ke.category = $%d", argIndex))
		args = append(args, filter.Category)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("ke.tags @> $%d", argIndex))
		args = append(args, filter.Tags)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY ke.updated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge entries: %w", err)
	}
	defer rows.Close()

	var entries []*domain.KnowledgeEntry

	for rows.Next() {
		var entry domain.KnowledgeEntry
		var tagsJSON []byte
		var category sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&entry.Content,
			&entry.Status,
			&category,
			&tagsJSON,
			&entry.SourceType,
			&entry.Version,
			&entry.CreatedBy,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan knowledge entry: %w", err)
		}

		if category.Valid {
			entry.Category = category.String
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &entry.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating knowledge entries: %w", err)
	}

	return entries, nil
}

// DeleteEntry removes a knowledge base entry
func (r *PostgresKnowledgeRepository) DeleteEntry(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete chunks first (foreign key constraint should handle this, but we'll be explicit)
	_, err = tx.ExecContext(ctx, "DELETE FROM kb_chunks WHERE entry_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge chunks: %w", err)
	}

	// Delete the entry
	_, err = tx.ExecContext(ctx, "DELETE FROM knowledge_entries WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge entry: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateChunk saves a knowledge base chunk
func (r *PostgresKnowledgeRepository) CreateChunk(ctx context.Context, chunk *domain.KBChunk) error {
	query := `
		INSERT INTO kb_chunks (id, entry_id, chunk_index, content, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		chunk.ID,
		chunk.EntryID,
		chunk.ChunkIndex,
		chunk.Content,
		chunk.Embedding,
		chunk.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create knowledge chunk: %w", err)
	}

	return nil
}

// FindChunksByEntry retrieves all chunks for an entry
func (r *PostgresKnowledgeRepository) FindChunksByEntry(ctx context.Context, entryID string) ([]*domain.KBChunk, error) {
	query := `
		SELECT id, entry_id, chunk_index, content, embedding, created_at
		FROM kb_chunks
		WHERE entry_id = $1
		ORDER BY chunk_index
	`

	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*domain.KBChunk

	for rows.Next() {
		var chunk domain.KBChunk
		var embedding []byte // PostgreSQL returns vector as byte array

		err := rows.Scan(
			&chunk.ID,
			&chunk.EntryID,
			&chunk.ChunkIndex,
			&chunk.Content,
			&embedding,
			&chunk.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan knowledge chunk: %w", err)
		}

		// Convert byte array to float32 slice (this might need adjustment based on pgvector driver)
		if len(embedding) > 0 {
			// Note: This conversion depends on how pgvector returns vectors
			// You might need to use a proper pgvector driver that handles vectors correctly
			chunk.Embedding = bytesToFloat32Slice(embedding)
		}

		chunks = append(chunks, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating knowledge chunks: %w", err)
	}

	return chunks, nil
}

// SearchChunks performs similarity search on chunks
func (r *PostgresKnowledgeRepository) SearchChunks(ctx context.Context, queryText string, filter domain.KBChunkFilter) ([]*domain.KBChunk, error) {
	if r.embeddings == nil {
		return nil, fmt.Errorf("embedding provider not configured")
	}

	// Generate embedding for query
	queryEmbedding, err := r.embeddings.Embed(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Build the similarity search query
	sqlQuery := `
		SELECT kc.id, kc.entry_id, kc.chunk_index, kc.content, kc.embedding, kc.created_at,
			   1 - (kc.embedding <=> $1) as score
		FROM kb_chunks kc
		JOIN knowledge_entries ke ON ke.id = kc.entry_id
		WHERE ke.status = 'active'
	`

	var args []interface{}
	args = append(args, queryEmbedding)
	argIndex := 2

	// Add additional filters
	if filter.Category != "" {
		sqlQuery += fmt.Sprintf(" AND ke.category = $%d", argIndex)
		args = append(args, filter.Category)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		sqlQuery += fmt.Sprintf(" AND ke.tags @> $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	// Order by similarity score and limit results
	topK := filter.TopK
	if topK <= 0 {
		topK = 10
	}

	sqlQuery += fmt.Sprintf(" ORDER BY kc.embedding <=> $1 LIMIT $%d", argIndex)
	args = append(args, topK)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*domain.KBChunk

	for rows.Next() {
		var chunk domain.KBChunk
		var embedding []byte
		var score float64

		err := rows.Scan(
			&chunk.ID,
			&chunk.EntryID,
			&chunk.ChunkIndex,
			&chunk.Content,
			&embedding,
			&chunk.CreatedAt,
			&score,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		if len(embedding) > 0 {
			chunk.Embedding = bytesToFloat32Slice(embedding)
		}

		chunks = append(chunks, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return chunks, nil
}

// UpdateChunk updates an existing chunk
func (r *PostgresKnowledgeRepository) UpdateChunk(ctx context.Context, chunk *domain.KBChunk) error {
	query := `
		UPDATE kb_chunks
		SET content = $2, embedding = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		chunk.ID,
		chunk.Content,
		chunk.Embedding,
	)

	if err != nil {
		return fmt.Errorf("failed to update knowledge chunk: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("knowledge chunk not found")
	}

	return nil
}

// DeleteChunksByEntry removes all chunks for an entry
func (r *PostgresKnowledgeRepository) DeleteChunksByEntry(ctx context.Context, entryID string) error {
	query := `DELETE FROM kb_chunks WHERE entry_id = $1`

	_, err := r.db.ExecContext(ctx, query, entryID)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge chunks: %w", err)
	}

	return nil
}

// Helper function to convert byte array to float32 slice
func bytesToFloat32Slice(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}

	// This is a basic implementation - you might need to adjust based on pgvector encoding
	floats := make([]float32, len(data)/4)
	for i := 0; i < len(data); i += 4 {
		// This assumes little-endian byte order
		bits := uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16 | uint32(data[i+3])<<24
		floats[i/4] = float32(bits)
	}
	return floats
}

// Helper function to convert float32 slice to byte array
func float32SliceToBytes(floats []float32) []byte {
	if len(floats) == 0 {
		return nil
	}

	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := uint32(f)
		data[i*4] = byte(bits)
		data[i*4+1] = byte(bits >> 8)
		data[i*4+2] = byte(bits >> 16)
		data[i*4+3] = byte(bits >> 24)
	}
	return data
}