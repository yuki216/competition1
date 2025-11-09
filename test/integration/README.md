# Fixora API Integration Tests

This directory contains comprehensive integration tests for the Fixora IT Ticketing System API.

## Overview

The integration tests cover all major API endpoints of the Fixora application:

### Ticket Management API
- **POST** `/api/v1/tickets` - Create new tickets
- **GET** `/api/v1/tickets` - List tickets with filters
- **GET** `/api/v1/tickets/{id}` - Get specific ticket details
- **PATCH** `/api/v1/tickets/{id}` - Update ticket information
- **POST** `/api/v1/tickets/{id}/assign` - Assign ticket to admin
- **POST** `/api/v1/tickets/{id}/resolve` - Mark ticket as resolved
- **POST** `/api/v1/tickets/{id}/close` - Close resolved ticket
- **GET** `/api/v1/tickets/stats` - Get ticket statistics

### AI Services API
- **POST** `/api/v1/ai/suggest` - Get AI-powered suggestions
- **GET** `/api/v1/ai/suggest/stream` - Stream AI suggestions (SSE)
- **POST** `/api/v1/ai/kb/search` - Search knowledge base with AI
- **POST** `/api/v1/ai/embedding` - Generate text embeddings
- **POST** `/api/v1/ai/analyze` - Analyze ticket content with AI
- **GET** `/api/v1/ai/health` - Check AI service health
- **GET** `/api/v1/ai/info` - Get AI provider information
- **POST** `/api/v1/tickets/ai-intake` - AI-driven ticket creation

### Knowledge Base API
- **POST** `/api/v1/kb/entries` - Create knowledge base entries
- **GET** `/api/v1/kb/entries` - List knowledge base entries
- **GET** `/api/v1/kb/entries/{id}` - Get specific entry
- **PATCH** `/api/v1/kb/entries/{id}` - Update knowledge base entry
- **POST** `/api/v1/kb/entries/{id}/publish` - Publish knowledge base entry
- **DELETE** `/api/v1/kb/entries/{id}` - Delete knowledge base entry
- **POST** `/api/v1/kb/search` - Search knowledge base
- **POST** `/api/v1/kb/upload-text` - Upload text content to knowledge base

### System API
- **GET** `/health` - Health check endpoint

## Test Structure

### Test Suite Architecture

The integration tests use a test suite pattern that provides:

1. **Database Setup**: Automatic creation and configuration of test database
2. **Mock Services**: AI services run in mock mode for consistent testing
3. **Isolation**: Each test runs in a clean environment
4. **Cleanup**: Automatic cleanup of test data

### Key Components

- **FixoraIntegrationTestSuite**: Main test suite structure
- **setupFixoraIntegrationTest()**: Initializes test environment
- **cleanup()**: Performs post-test cleanup
- **ensureTestDatabaseAndSchemaForFixora()**: Creates test database and schema

## Prerequisites

### Database Requirements

- PostgreSQL 12+ running on localhost:5432
- User: `postgres` with password: `postgres`
- Database creation privileges

### Environment Setup

The tests expect the following environment variables (can be overridden):

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=fixora_test

# Application Configuration
ENVIRONMENT=test
AI_MOCK_MODE=true
```

## Running Tests

### Using the Test Runner Script

The easiest way to run integration tests:

```bash
# From the test/integration directory
./run_integration_tests.sh
```

This script will:
1. Verify PostgreSQL connectivity
2. Set up environment variables
3. Run all integration tests with verbose output

### Manual Test Execution

```bash
# From project root
go test -v ./test/integration/... -timeout=30s

# Run specific test
go test -v ./test/integration/... -run TestTicketAPI -timeout=30s

# Run with coverage
go test -v -cover ./test/integration/... -timeout=30s
```

## Test Cases

### 1. Health Endpoint Tests
- Verify API health check functionality
- Test response format and status codes

### 2. Ticket API Tests
- **CreateTicket**: Test ticket creation with valid/invalid data
- **GetTicket**: Test ticket retrieval by ID
- **ListTickets**: Test ticket listing with filters and pagination
- **AssignTicket**: Test ticket assignment workflow
- **ResolveTicket**: Test ticket resolution process
- **CloseTicket**: Test ticket closure (requires resolved status)

### 3. AI API Tests
- **GetSuggestion**: Test AI-powered categorization and prioritization
- **HealthCheck**: Test AI service health monitoring
- **GetProviderInfo**: Test AI provider information retrieval

### 4. Knowledge Base API Tests
- **CreateEntry**: Test knowledge base entry creation
- **SearchEntries**: Test knowledge base search functionality

### 5. Error Handling Tests
- **InvalidTicketID**: Test 404 responses for non-existent tickets
- **InvalidRequestBody**: Test 400 responses for malformed JSON
- **MissingRequiredFields**: Test validation for missing required fields

## Database Schema

The integration tests create the following database tables:

### knowledge_entries
```sql
CREATE TABLE knowledge_entries (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    tags TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### knowledge_chunks
```sql
CREATE TABLE knowledge_chunks (
    id VARCHAR(255) PRIMARY KEY,
    entry_id VARCHAR(255) NOT NULL,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (entry_id) REFERENCES knowledge_entries(id) ON DELETE CASCADE
);
```

### tickets
```sql
CREATE TABLE tickets (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
    category VARCHAR(100) NOT NULL,
    priority VARCHAR(50) NOT NULL DEFAULT 'MEDIUM',
    created_by VARCHAR(255) NOT NULL,
    assigned_to VARCHAR(255),
    ai_insight JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### comments
```sql
CREATE TABLE comments (
    id VARCHAR(255) PRIMARY KEY,
    ticket_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE
);
```

## Configuration

### Test Configuration

The integration tests use a dedicated test configuration:

- **Server**: Runs on port 8081 (different from main application)
- **Database**: Uses `fixora_test` database
- **AI Services**: Mock mode enabled for consistent testing
- **Logging**: Minimal logging for test output

### Mock AI Services

AI services run in mock mode, providing predictable responses:
- **Suggestions**: Always return valid suggestions with test data
- **Embeddings**: Return mock embedding vectors
- **Health Checks**: Always return healthy status

## Best Practices

### Test Isolation
- Each test creates its own data
- Database is cleaned between tests
- No shared state between tests

### Error Scenarios
- Test both success and failure cases
- Verify proper HTTP status codes
- Validate error message formats

### Data Validation
- Verify response structures
- Check data integrity
- Validate business logic

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Verify PostgreSQL is running
   - Check connection credentials
   - Ensure test database exists

2. **Permission Errors**
   - Grant database creation privileges
   - Check user permissions

3. **Port Conflicts**
   - Ensure port 8081 is available
   - Check for running services

### Debug Mode

For debugging, you can run tests with additional logging:

```bash
# Enable debug logging
DEBUG=true go test -v ./test/integration/... -timeout=60s

# Run single test with detailed output
go test -v -test.run TestCreateTicket ./test/integration/... -timeout=60s
```

## Contributing

When adding new integration tests:

1. Follow the existing test suite pattern
2. Use the provided setup/teardown functions
3. Test both success and error scenarios
4. Validate response structures
5. Clean up any created data
6. Update this README with new test cases

## Future Enhancements

- Performance testing endpoints
- Load testing with concurrent requests
- API version compatibility tests
- Authentication/authorization integration tests
- WebSocket/SSE streaming tests
- File upload/download tests