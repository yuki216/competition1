-- Initial schema for Fixora IT Ticketing System
-- Version: 001
-- Created: 2024-01-01

-- Enable pgvector extension for vector search
CREATE EXTENSION IF NOT EXISTS vector;

-- Tickets table
CREATE TABLE IF NOT EXISTS tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'IN_PROGRESS', 'RESOLVED', 'CLOSED')),
    category TEXT NOT NULL CHECK (category IN ('NETWORK', 'SOFTWARE', 'HARDWARE', 'ACCOUNT', 'OTHER')),
    priority TEXT NOT NULL CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    created_by UUID NOT NULL,
    assigned_to UUID,
    ai_insight JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Comments table
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('EMPLOYEE', 'ADMIN', 'AI')),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Knowledge entries table
CREATE TABLE IF NOT EXISTS knowledge_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'archived')),
    category TEXT,
    tags TEXT[],
    source_type TEXT NOT NULL CHECK (source_type IN ('MANUAL', 'LEARNED')),
    version INT NOT NULL DEFAULT 1,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Knowledge chunks table for vector search
CREATE TABLE IF NOT EXISTS kb_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID NOT NULL REFERENCES knowledge_entries(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    embedding VECTOR(1536), -- Default dimension, will be adjusted based on embedding model
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL,
    resource_id UUID NOT NULL,
    action TEXT NOT NULL,
    actor_id UUID NOT NULL,
    actor_role TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Basic indexes for tickets
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_category ON tickets(category);
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);
CREATE INDEX IF NOT EXISTS idx_tickets_created_by ON tickets(created_by);
CREATE INDEX IF NOT EXISTS idx_tickets_assigned_to ON tickets(assigned_to);
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at);

-- Indexes for comments
CREATE INDEX IF NOT EXISTS idx_comments_ticket_id ON comments(ticket_id);
CREATE INDEX IF NOT EXISTS idx_comments_author_id ON comments(author_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at);

-- Indexes for knowledge entries
CREATE INDEX IF NOT EXISTS idx_kb_entries_status ON knowledge_entries(status);
CREATE INDEX IF NOT EXISTS idx_kb_entries_category ON knowledge_entries(category);
CREATE INDEX IF NOT EXISTS idx_kb_entries_tags ON knowledge_entries USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_kb_entries_source_type ON knowledge_entries(source_type);

-- Indexes for knowledge chunks
CREATE INDEX IF NOT EXISTS idx_kb_chunks_entry_id ON kb_chunks(entry_id);
CREATE INDEX IF NOT EXISTS idx_kb_chunks_chunk_index ON kb_chunks(entry_id, chunk_index);

-- Vector index for similarity search (ivfflat with cosine distance)
CREATE INDEX IF NOT EXISTS idx_kb_chunks_embedding_ivfflat
    ON kb_chunks USING ivfflat (embedding vector_cosine_ops);

-- Indexes for audit logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type_id ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_tickets_updated_at
    BEFORE UPDATE ON tickets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_entries_updated_at
    BEFORE UPDATE ON knowledge_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add RLS (Row Level Security) policies if needed
-- These policies will be implemented based on specific security requirements

-- Create sequences for any custom ID generation if needed
CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START 1001;

-- Create function to generate ticket numbers
CREATE OR REPLACE FUNCTION generate_ticket_number()
RETURNS TEXT AS $$
DECLARE
    ticket_number TEXT;
    year_month TEXT;
BEGIN
    year_month := TO_CHAR(NOW(), 'YYYYMM');
    ticket_number := 'TKT-' || year_month || '-' || LPAD(nextval('ticket_number_seq')::TEXT, 4, '0');
    RETURN ticket_number;
END;
$$ LANGUAGE plpgsql;