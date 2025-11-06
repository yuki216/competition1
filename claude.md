# Fixora IT Ticketing System

## Overview

Fixora is an AI-assisted IT ticketing system designed to streamline IT support processes. The system enables employees to submit tickets, receive AI-powered suggestions, and allows IT administrators to manage, assign, and resolve tickets efficiently while building a knowledge base for continuous AI learning.

## Project Structure

This is a Go-based web application following Clean Architecture principles with Domain-Driven Design (DDD) patterns.

```
/cmd/fixora/
/internal
  /domain
    ticket.go
    comment.go
    kb.go
  /usecase
    ticket_usecase.go
    ai_usecase.go
  /ports
    repository.go
    ai.go
    notification.go
  /adapter
    /http
      handler_ticket.go
      handler_ai_suggest.go
      handler_kb.go
    /persistence
      postgres_ticket_repo.go
      postgres_kb_repo.go
      pgvector_chunks_repo.go
    /ai
      openai_adapter.go
      zai_adapter.go
      embedding_adapter.go
    /notification
      slack_adapter.go
  /infra
    events/broker.go
    sse/streamer.go
  /test
    fixtures
/migrations/
/docs/
```

## Core Components

### 1. Domain Model

#### Ticket (Aggregate Root)
- id: UUID
- title: string
- description: string
- status: OPEN | IN_PROGRESS | RESOLVED | CLOSED
- category: NETWORK | SOFTWARE | HARDWARE | ACCOUNT | OTHER
- priority: LOW | MEDIUM | HIGH | CRITICAL
- createdBy: EmployeeID
- assignedTo: AdminID | null
- aiInsight: { text: string, confidence: float } | null
- createdAt, updatedAt

#### Comment
- id, ticketId, authorId, role: EMPLOYEE|ADMIN|AI, body, createdAt

#### KnowledgeBaseEntry
- id, contextLabel, solutionSteps, tags[], source: MANUAL|LEARNED, active:boolean, updatedAt

### 2. AI Integration

#### AI Providers
The system supports multiple AI providers through abstraction:
- **OpenAI**: GPT models for suggestions and embeddings
- **Z.ai**: Alternative AI provider
- Configuration via `AI_PROVIDER` environment variable

#### Key AI Services
- **AISuggestionService**: Provides mitigation suggestions based on ticket descriptions
- **AITrainingService**: Learns from resolved tickets to improve suggestions
- **EmbeddingProvider**: Generates text embeddings for knowledge base search

### 3. Knowledge Base with Vector Search

The knowledge base uses **pgvector** for semantic search:

#### Database Schema
```sql
-- Knowledge entries
CREATE TABLE knowledge_entries (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  status TEXT CHECK (status IN ('draft','active','archived')),
  category TEXT,
  tags TEXT[],
  source_type TEXT,
  version INT DEFAULT 1,
  created_by UUID NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Text chunks with embeddings
CREATE TABLE kb_chunks (
  id UUID PRIMARY KEY,
  entry_id UUID REFERENCES knowledge_entries(id) ON DELETE CASCADE,
  chunk_index INT NOT NULL,
  content TEXT NOT NULL,
  embedding VECTOR(1536), -- Dimension varies by embedding model
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Vector index for similarity search
CREATE INDEX idx_kb_chunks_embedding_ivfflat
  ON kb_chunks USING ivfflat (embedding vector_cosine_ops);
```

#### Knowledge Base Operations
- **Upload text content**: Support for plain text and markdown
- **Automatic chunking**: Breaks large content into searchable chunks (500-1000 chars)
- **Embedding generation**: Creates vector representations for semantic search
- **Similarity search**: Finds relevant content using cosine similarity

### 4. API Endpoints

#### Ticket Management
- `POST /v1/tickets` - Create ticket
- `GET /v1/tickets/{id}` - Get ticket details with comments
- `POST /v1/tickets/{id}/comments` - Add comment
- `POST /v1/tickets/{id}/assign` - Assign ticket to admin
- `POST /v1/tickets/{id}/resolve` - Resolve ticket

#### Knowledge Base
- `POST /v1/kb/entries` - Create KB entry (draft)
- `POST /v1/kb/entries/upload-text` - Upload text/markdown content
- `PATCH /v1/kb/entries/{id}` - Update KB entry
- `POST /v1/kb/entries/{id}/publish` - Publish entry with chunking and embedding

#### AI Services
- `POST /v1/ai/suggest` - Get AI suggestion (JSON response)
- `GET /v1/ai/suggest/stream` - Streaming suggestions via Server-Sent Events (SSE)

#### Reporting
- `GET /v1/reports` - Get system metrics and reports

### 5. Real-time Features

#### Server-Sent Events (SSE)
The AI suggest endpoint provides streaming responses for better user experience:

```
GET /v1/ai/suggest/stream?query=Wi-Fi+connection+issues

Events:
- init: Stream initialization with query ID
- candidate: Individual suggestion results with relevance scores
- progress: Search progress updates
- end: Stream completion
- error: Error events with appropriate codes
```

**Performance Targets:**
- First event: ≤ 500ms
- Event cadence: ≤ 250ms flush interval
- Heartbeat: Every 15 seconds

### 6. Key Workflows

#### Ticket Creation with AI Assistance
1. Employee submits issue description
2. AI analyzes and suggests mitigation (if confidence > 0.7)
3. Employee can create ticket if issue persists
4. System stores ticket with AI insights

#### Knowledge Base Publishing
1. Admin creates/uploads content (draft status)
2. Text normalization and chunking
3. Batch embedding generation
4. Vector index creation/refresh
5. Atomic publish to active status

#### Ticket Resolution Loop
1. Admin assigns and works on ticket
2. Comments and updates are tracked
3. Final resolution triggers AI training
4. Resolution becomes part of knowledge base

### 7. Non-Functional Requirements

#### Performance
- API response: ≤ 300ms (non-AI operations)
- AI inference: ≤ 2s for suggestions
- Vector search: ≤ 200ms for Top-K=10 on 100k+ chunks
- SSE streaming: First event ≤ 500ms

#### Security
- Role-based access control (RBAC)
- Ticket data accessible only to creator and IT staff
- PII redaction options
- API key management for AI providers
- Comprehensive audit trails

#### Reliability
- 99.9% uptime target
- Graceful provider fallback
- Idempotent operations
- Connection pooling and circuit breakers

#### Observability
- Structured JSON logging
- Correlation IDs for request tracing
- Metrics for latency, error rates, and AI accuracy
- Vector search performance monitoring

### 8. Configuration

#### Environment Variables
```bash
# AI Provider Configuration
AI_PROVIDER=openai                    # or zai
OPENAI_API_KEY=your_openai_key
ZAI_API_KEY=your_zai_key

# Embedding Configuration
EMBEDDING_DIM=1536                   # Match model dimension
TOP_K=10                            # Default search results
CHUNK_SIZE=800                      # Text chunk size
CHUNK_OVERLAP=150                  # Overlap between chunks

# SSE Configuration
SSE_FLUSH_MS=250                   # Event flush interval
SSE_HEARTBEAT_MS=15000             # Heartbeat interval

# Database
DATABASE_URL=postgres_connection_string
```

### 9. Development Guidelines

#### Testing Strategy
- **Domain tests**: Invariants and business rules
- **Use case tests**: Mock ports for deterministic testing
- **Integration tests**: Real database and AI services
- **SSE tests**: Stream latency and ordering validation
- **Vector tests**: Cosine similarity accuracy

#### Code Standards
- Clean Architecture with clear separation of concerns
- Domain-first design with no infrastructure dependencies
- Test-driven development with >80% coverage
- Structured logging and error handling
- Comprehensive API documentation (OpenAPI)

### 10. Deployment Considerations

#### Database Requirements
- PostgreSQL with pgvector extension
- Regular VACUUM/ANALYZE for vector tables
- Connection pooling for high concurrency
- Daily backups with index restoration testing

#### Monitoring
- Application metrics (latency, throughput, errors)
- AI provider performance and costs
- Vector search accuracy and latency
- Knowledge base size and query patterns
- SLA compliance tracking

## Getting Started

### Prerequisites
- Go 1.21+
- PostgreSQL 15+ with pgvector
- AI provider API keys (OpenAI or Z.ai)

### Database Setup
```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Run migrations
migrate up
```

### Local Development
```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Start development server
go run cmd/fixora/main.go
```

### Testing AI Integration
The system provides mock AI services for testing without consuming API quotas:
```bash
# Enable mock mode
AI_MOCK=true go run cmd/fixora/main.go
```

## Current Status

The project is currently in the specification phase with detailed domain models, API contracts, and implementation plans defined. The next steps involve:

1. Setting up the project structure and dependencies
2. Implementing core domain models and ports
3. Creating database migrations
4. Building AI service adapters
5. Implementing HTTP handlers and SSE streaming
6. Setting up comprehensive testing

## Documentation

- **Specifications**: `spec/specify.md` - Business requirements and use cases
- **Technical Plan**: `spec/plan.md` - Detailed implementation guide
- **API Documentation**: `api/openapi.yaml` - REST API specifications
- **Database Schema**: `migrations/` - Database migration files

This project follows the speckit constitution emphasizing Domain-Driven Design, Clean Architecture, and Test-Driven Development principles.