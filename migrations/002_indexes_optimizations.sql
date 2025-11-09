-- Additional indexes and optimizations for Fixora IT Ticketing System
-- Version: 002
-- Created: 2024-01-01

-- Composite indexes for common query patterns

-- Tickets: For dashboard queries (status + assigned_to + created_at)
CREATE INDEX IF NOT EXISTS idx_tickets_status_assigned_created
ON tickets(status, assigned_to, created_at DESC);

-- Tickets: For filtering by category and status
CREATE INDEX IF NOT EXISTS idx_tickets_category_status
ON tickets(category, status);

-- Tickets: For user ticket history (created_by + created_at)
CREATE INDEX IF NOT EXISTS idx_tickets_created_by_created_at
ON tickets(created_by, created_at DESC);

-- Comments: For recent comments on specific tickets
CREATE INDEX IF NOT EXISTS idx_comments_ticket_created_at
ON comments(ticket_id, created_at DESC);

-- Knowledge entries: For active entries search
CREATE INDEX IF NOT EXISTS idx_kb_entries_status_category
ON knowledge_entries(status, category) WHERE status = 'active';

-- Knowledge chunks: Optimized vector search with filters
-- Create separate indexes for different search patterns
-- Note: PostgreSQL does not support subqueries in partial index predicates.
-- We fall back to a standard index on entry_id.
CREATE INDEX IF NOT EXISTS idx_kb_chunks_entry_status
ON kb_chunks(entry_id);

-- Full-text search index for knowledge content (optional)
CREATE INDEX IF NOT EXISTS idx_kb_chunks_content_gin
ON kb_chunks USING gin(to_tsvector('english', content));

-- Full-text search index for ticket descriptions
CREATE INDEX IF NOT EXISTS idx_tickets_description_gin
ON tickets USING gin(to_tsvector('english', description));

-- Full-text search index for ticket titles
CREATE INDEX IF NOT EXISTS idx_tickets_title_gin
ON tickets USING gin(to_tsvector('english', title));

-- Audit logs: For recent activity tracking
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at_desc
ON audit_logs(created_at DESC);

-- Audit logs: For resource-specific audit trails
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type_id_created
ON audit_logs(resource_type, resource_id, created_at DESC);

-- Partial indexes for better performance

-- Only index open tickets for active work queries
CREATE INDEX IF NOT EXISTS idx_tickets_open_assigned
ON tickets(assigned_to, created_at)
WHERE status IN ('OPEN', 'IN_PROGRESS');

-- Only index resolved tickets for metrics calculations
CREATE INDEX IF NOT EXISTS idx_tickets_resolved_created
ON tickets(created_at)
WHERE status = 'RESOLVED';

-- Only index active knowledge chunks for search
-- Note: PostgreSQL does not support subqueries in partial index predicates.
-- We fall back to an unfiltered vector index on embedding.
CREATE INDEX IF NOT EXISTS idx_kb_chunks_active_embedding
ON kb_chunks USING ivfflat (embedding vector_cosine_ops);

-- Function to update vector index statistics
CREATE OR REPLACE FUNCTION update_vector_index_stats()
RETURNS void AS $$
BEGIN
    ANALYZE kb_chunks;
    ANALYZE knowledge_entries;
END;
$$ LANGUAGE plpgsql;

-- Materialized view for ticket metrics (refresh as needed)
CREATE MATERIALIZED VIEW IF NOT EXISTS ticket_metrics AS
SELECT
    status,
    category,
    priority,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))/3600) as avg_hours_to_close,
    MIN(created_at) as oldest_ticket,
    MAX(created_at) as newest_ticket
FROM tickets
GROUP BY status, category, priority;

-- Index for materialized view
CREATE INDEX IF NOT EXISTS idx_ticket_metrics_status_category
ON ticket_metrics(status, category);

-- Function to refresh metrics view
CREATE OR REPLACE FUNCTION refresh_ticket_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY ticket_metrics;
END;
$$ LANGUAGE plpgsql;

-- Monitoring and optimization views

-- View for monitoring index usage
CREATE OR REPLACE VIEW index_usage_stats AS
SELECT
    schemaname,
    relname AS tablename,
    indexrelname AS indexname,
    idx_tup_read,
    idx_tup_fetch,
    idx_scan
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- View for table sizes
CREATE OR REPLACE VIEW table_sizes AS
SELECT
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
    pg_total_relation_size(schemaname||'.'||tablename) as size_bytes
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY size_bytes DESC;

-- View for ticket aging (stale tickets)
CREATE OR REPLACE VIEW aging_tickets AS
SELECT
    id,
    title,
    status,
    category,
    priority,
    created_by,
    created_at,
    updated_at,
    EXTRACT(EPOCH FROM (NOW() - created_at))/3600 as hours_open,
    EXTRACT(EPOCH FROM (NOW() - updated_at))/3600 as hours_since_update
FROM tickets
WHERE status IN ('OPEN', 'IN_PROGRESS')
ORDER BY created_at;

-- Cleanup function for old audit logs (keep last 2 years)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_logs
    WHERE created_at < NOW() - INTERVAL '2 years';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust based on your security requirements)
-- These are basic grants - customize based on your application needs

-- GRANT CONNECT ON DATABASE fixora TO fixora_app;
-- GRANT USAGE ON SCHEMA public TO fixora_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fixora_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO fixora_app;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO fixora_app;

-- Set default permissions for future tables
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO fixora_app;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO fixora_app;