package postgres

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "strings"

    "github.com/fixora/fixora/domain"
    outbound "github.com/fixora/fixora/application/port/outbound"
)

// PostgresKnowledgeRepository implements KnowledgeRepository using PostgreSQL with pgvector
type PostgresKnowledgeRepository struct { db *sql.DB; embeddings outbound.EmbeddingProvider }

func NewPostgresKnowledgeRepository(db *sql.DB, embeddings outbound.EmbeddingProvider) outbound.KnowledgeRepository {
    return &PostgresKnowledgeRepository{db: db, embeddings: embeddings}
}

func (r *PostgresKnowledgeRepository) CreateEntry(ctx context.Context, entry *domain.KnowledgeEntry) error {
    query := `
        INSERT INTO knowledge_entries (id, title, content, status, category, tags, source_type, version, created_by, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `
    tagsJSON, err := json.Marshal(entry.Tags)
    if err != nil { return fmt.Errorf("failed to marshal tags: %w", err) }
    _, err = r.db.ExecContext(ctx, query, entry.ID, entry.Title, entry.Content, string(entry.Status), entry.Category, tagsJSON, string(entry.SourceType), entry.Version, entry.CreatedBy, entry.CreatedAt, entry.UpdatedAt)
    if err != nil { return fmt.Errorf("failed to create knowledge entry: %w", err) }
    return nil
}

func (r *PostgresKnowledgeRepository) FindEntryByID(ctx context.Context, id string) (*domain.KnowledgeEntry, error) {
    query := `
        SELECT id, title, content, status, category, tags, source_type, version, created_by, created_at, updated_at
        FROM knowledge_entries
        WHERE id = $1
    `
    var entry domain.KnowledgeEntry
    var tagsJSON []byte
    var category sql.NullString
    err := r.db.QueryRowContext(ctx, query, id).Scan(&entry.ID, &entry.Title, &entry.Content, &entry.Status, &category, &tagsJSON, &entry.SourceType, &entry.Version, &entry.CreatedBy, &entry.CreatedAt, &entry.UpdatedAt)
    if err != nil { if err == sql.ErrNoRows { return nil, domain.ErrKBEntryNotFound }; return nil, fmt.Errorf("failed to find knowledge entry: %w", err) }
    if category.Valid { entry.Category = category.String }
    if len(tagsJSON) > 0 { if err := json.Unmarshal(tagsJSON, &entry.Tags); err != nil { return nil, fmt.Errorf("failed to unmarshal tags: %w", err) } }
    return &entry, nil
}

func (r *PostgresKnowledgeRepository) UpdateEntry(ctx context.Context, entry *domain.KnowledgeEntry) error {
    query := `
        UPDATE knowledge_entries
        SET title = $2, content = $3, status = $4, category = $5, tags = $6, version = $7, updated_at = $8
        WHERE id = $1
    `
    tagsJSON, err := json.Marshal(entry.Tags)
    if err != nil { return fmt.Errorf("failed to marshal tags: %w", err) }
    result, err := r.db.ExecContext(ctx, query, entry.ID, entry.Title, entry.Content, string(entry.Status), entry.Category, tagsJSON, entry.Version, entry.UpdatedAt)
    if err != nil { return fmt.Errorf("failed to update knowledge entry: %w", err) }
    rowsAffected, err := result.RowsAffected()
    if err != nil { return fmt.Errorf("failed to get rows affected: %w", err) }
    if rowsAffected == 0 { return domain.ErrKBEntryNotFound }
    return nil
}

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
    if filter.Status != "" { conditions = append(conditions, fmt.Sprintf("ke.status = $%d", argIndex)); args = append(args, filter.Status); argIndex++ }
    if filter.Category != "" { conditions = append(conditions, fmt.Sprintf("ke.category = $%d", argIndex)); args = append(args, filter.Category); argIndex++ }
    if len(filter.Tags) > 0 { conditions = append(conditions, fmt.Sprintf("ke.tags @> $%d", argIndex)); args = append(args, filter.Tags); argIndex++ }
    if len(conditions) > 0 { query += " AND " + strings.Join(conditions, " AND ") }
    query += " ORDER BY ke.updated_at DESC"
    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil { return nil, fmt.Errorf("failed to query knowledge entries: %w", err) }
    defer rows.Close()
    var entries []*domain.KnowledgeEntry
    for rows.Next() {
        var entry domain.KnowledgeEntry
        var tagsJSON []byte
        var category sql.NullString
        err := rows.Scan(&entry.ID, &entry.Title, &entry.Content, &entry.Status, &category, &tagsJSON, &entry.SourceType, &entry.Version, &entry.CreatedBy, &entry.CreatedAt, &entry.UpdatedAt)
        if err != nil { return nil, fmt.Errorf("failed to scan knowledge entry: %w", err) }
        if category.Valid { entry.Category = category.String }
        if len(tagsJSON) > 0 { if err := json.Unmarshal(tagsJSON, &entry.Tags); err != nil { return nil, fmt.Errorf("failed to unmarshal tags: %w", err) } }
        entries = append(entries, &entry)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("error iterating knowledge entries: %w", err) }
    return entries, nil
}

func (r *PostgresKnowledgeRepository) DeleteEntry(ctx context.Context, id string) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return fmt.Errorf("failed to begin transaction: %w", err) }
    defer tx.Rollback()
    if _, err = tx.ExecContext(ctx, "DELETE FROM kb_chunks WHERE entry_id = $1", id); err != nil { return fmt.Errorf("failed to delete knowledge chunks: %w", err) }
    if _, err = tx.ExecContext(ctx, "DELETE FROM knowledge_entries WHERE id = $1", id); err != nil { return fmt.Errorf("failed to delete knowledge entry: %w", err) }
    if err := tx.Commit(); err != nil { return fmt.Errorf("failed to commit transaction: %w", err) }
    return nil
}

func (r *PostgresKnowledgeRepository) CreateChunk(ctx context.Context, chunk *domain.KBChunk) error {
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO kb_chunks (id, entry_id, chunk_index, content, embedding, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, chunk.ID, chunk.EntryID, chunk.ChunkIndex, chunk.Content, chunk.Embedding, chunk.CreatedAt)
    if err != nil { return fmt.Errorf("failed to create knowledge chunk: %w", err) }
    return nil
}

func (r *PostgresKnowledgeRepository) FindChunksByEntry(ctx context.Context, entryID string) ([]*domain.KBChunk, error) {
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, entry_id, chunk_index, content, embedding, created_at
        FROM kb_chunks
        WHERE entry_id = $1
        ORDER BY chunk_index
    `, entryID)
    if err != nil { return nil, fmt.Errorf("failed to query knowledge chunks: %w", err) }
    defer rows.Close()
    var chunks []*domain.KBChunk
    for rows.Next() {
        var chunk domain.KBChunk
        var embedding []byte
        if err := rows.Scan(&chunk.ID, &chunk.EntryID, &chunk.ChunkIndex, &chunk.Content, &embedding, &chunk.CreatedAt); err != nil { return nil, fmt.Errorf("failed to scan knowledge chunk: %w", err) }
        chunk.Embedding = bytesToFloat32Slice(embedding)
        chunks = append(chunks, &chunk)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("error iterating knowledge chunks: %w", err) }
    return chunks, nil
}

func (r *PostgresKnowledgeRepository) SearchChunks(ctx context.Context, queryText string, filter domain.KBChunkFilter) ([]*domain.KBChunk, error) {
    // Simplified semantic search stub: in production, use pgvector similarity operators
    rows, err := r.db.QueryContext(ctx, `
        SELECT id, entry_id, chunk_index, content, embedding, created_at
        FROM kb_chunks
        WHERE content ILIKE '%' || $1 || '%'
        ORDER BY created_at DESC
        LIMIT COALESCE($2, 10)
    `, queryText, filter.TopK)
    if err != nil { return nil, fmt.Errorf("failed to search knowledge chunks: %w", err) }
    defer rows.Close()
    var chunks []*domain.KBChunk
    for rows.Next() {
        var chunk domain.KBChunk
        var embedding []byte
        if err := rows.Scan(&chunk.ID, &chunk.EntryID, &chunk.ChunkIndex, &chunk.Content, &embedding, &chunk.CreatedAt); err != nil { return nil, fmt.Errorf("failed to scan knowledge chunk: %w", err) }
        chunk.Embedding = bytesToFloat32Slice(embedding)
        chunks = append(chunks, &chunk)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("error iterating knowledge search results: %w", err) }
    return chunks, nil
}

func (r *PostgresKnowledgeRepository) UpdateChunk(ctx context.Context, chunk *domain.KBChunk) error {
    result, err := r.db.ExecContext(ctx, `
        UPDATE kb_chunks SET content = $2, embedding = $3 WHERE id = $1
    `, chunk.ID, chunk.Content, float32SliceToBytes(chunk.Embedding))
    if err != nil { return fmt.Errorf("failed to update knowledge chunk: %w", err) }
    rowsAffected, err := result.RowsAffected()
    if err != nil { return fmt.Errorf("failed to get rows affected: %w", err) }
    if rowsAffected == 0 { return fmt.Errorf("chunk not found") }
    return nil
}

func (r *PostgresKnowledgeRepository) DeleteChunksByEntry(ctx context.Context, entryID string) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM kb_chunks WHERE entry_id = $1`, entryID)
    if err != nil { return fmt.Errorf("failed to delete chunks by entry: %w", err) }
    return nil
}

func bytesToFloat32Slice(data []byte) []float32 {
    // Placeholder for converting pgvector binary to float32 slice
    return []float32{}
}

func float32SliceToBytes(floats []float32) []byte {
    // Placeholder for converting float32 slice to pgvector binary
    return []byte{}
}