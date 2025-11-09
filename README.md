# Fixora IT Ticketing System

Fixora is an AI-assisted IT ticketing system designed to streamline IT support processes. The system enables employees to submit tickets, receive AI-powered suggestions, and allows IT administrators to manage, assign, and resolve tickets efficiently while building a knowledge base for continuous AI learning.

## Features

### Core Functionality
- **Ticket Management**: Create, assign, update, and resolve IT support tickets
- **AI-Powered Suggestions**: Get intelligent mitigation suggestions based on ticket descriptions
- **Knowledge Base**: Build and search a comprehensive knowledge base with vector search
- **Real-time Updates**: Server-Sent Events (SSE) for live ticket updates
- **Multi-Provider AI Support**: Support for OpenAI, Z.ai, and mock AI providers

### Advanced Features
- **Vector Search**: Semantic search using pgvector and embeddings
- **Streaming AI Suggestions**: Real-time AI suggestion streaming
- **Audit Trail**: Complete audit logging for compliance
- **Metrics & Reporting**: Track ticket resolution times and AI accuracy
- **Role-Based Access**: Employee, Admin, and AI roles with appropriate permissions

## Architecture

Fixora follows Clean Architecture principles with Domain-Driven Design (DDD):

```
/cmd/server/                 # Application entry point (main server)
/domain/                     # Domain entities, value objects, errors
/application/
  /port/
    /inbound/               # Use case inputs/contracts
    /outbound/              # Interfaces for external services (repositories, AI, tokens)
  /usecase/                 # Application use cases (business orchestration)
/infrastructure/
  /adapter/
    /postgres/              # Database repositories (Postgres adapters)
  /http/
    /handler/               # HTTP handlers (transport layer)
    /middleware/            # HTTP middleware
    /response/              # Response envelope helpers
  /service/
    /ai/                    # AI provider adapters (OpenAI, mock)
    /jwt/                   # Token service (JWT)
    /logger/                # Structured logger
    /password/              # Password hashing/verification
    /ratelimit/             # Rate limiting utilities
    /recaptcha/             # reCAPTCHA verification client
/migrations/                # Database migrations
/api/                       # Swagger UI & OpenAPI spec
/docs/                      # Documentation
/test/                      # Tests & mocks
```

## Technology Stack

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 15+ with pgvector extension
- **HTTP Framework**: Gorilla Mux
- **AI Services**: OpenAI, Z.ai (with mock implementation for testing)
- **Real-time**: Server-Sent Events (SSE)
- **Architecture**: Clean Architecture + Domain-Driven Design

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+ with pgvector extension
- (Optional) OpenAI API key or Z.ai API key for AI features

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/fixora/fixora.git
   cd fixora
   ```

2. **Set up the environment**:
   ```bash
   make setup
   ```

3. **Configure the application**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Set up the database**:
   ```bash
   # Create database
   make db-create

   # Run migrations
   make migrate

   # (Optional) Seed with sample data
   make seed
   ```

5. **Run the application**:
   ```bash
   make dev
   ```

The application will start on `http://localhost:8080`

## Configuration

The application is configured via environment variables. See `.env.example` for all available options:

### Key Configuration

- `AI_PROVIDER`: Set to `mock` (default), `openai`, or `zai`
- `DATABASE_URL`: PostgreSQL connection string
- `SERVER_PORT`: HTTP server port (default: 8080)
- `JWT_SECRET`: Secret for JWT authentication

### AI Configuration

For production AI features:

**OpenAI**:
```bash
AI_PROVIDER=openai
OPENAI_API_KEY=your_openai_api_key_here
```

**Z.ai**:
```bash
AI_PROVIDER=zai
ZAI_API_KEY=your_zai_api_key_here
```

## API Documentation

### Ticket Management

- `POST /api/v1/tickets` - Create a new ticket
- `GET /api/v1/tickets` - List tickets with filters
- `GET /api/v1/tickets/{id}` - Get ticket details
- `POST /api/v1/tickets/{id}/assign` - Assign ticket to admin
- `POST /api/v1/tickets/{id}/resolve` - Resolve ticket
- `POST /api/v1/tickets/{id}/close` - Close ticket

### AI Services

- `POST /api/v1/ai/suggest` - Get AI suggestion for ticket description
- `GET /api/v1/ai/suggest/stream` - Stream AI suggestions (SSE)
- `POST /api/v1/ai/kb/search` - Search knowledge base
- `POST /api/v1/ai/analyze` - Analyze ticket content
- `GET /api/v1/ai/health` - Check AI service health

### Knowledge Base

- `POST /api/v1/kb/entries` - Create knowledge base entry
- `GET /api/v1/kb/entries` - List knowledge base entries
- `POST /api/v1/kb/entries/{id}/publish` - Publish entry with embeddings
- `POST /api/v1/kb/search` - Search knowledge base
- `POST /api/v1/kb/upload-text` - Upload text content

## Development

### Available Commands

```bash
# Build the application
make build

# Run in development mode
make dev

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Run all checks
make check

# Watch for changes and rebuild
make watch

# Generate mocks
make generate-mocks
```

### Testing

The project uses table-driven tests and mocks:

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# View coverage report
open coverage.html
```

### Database Operations

```bash
# Create database
make db-create

# Run migrations
make migrate

# Seed with sample data
make seed

# Reset database
make db-reset

# Backup database
make db-backup
```

## Docker Support

### Using Docker Compose

```bash
# Build and start with Docker Compose
make docker-run

# View logs
make docker-logs

# Stop containers
make docker-stop
```

### Manual Docker

```bash
# Build Docker image
docker build -t fixora:latest .

# Run container
docker run -p 8080:8080 --env-file .env fixora:latest
```

## Production Deployment

### Environment Setup

1. Set production environment variables:
   ```bash
   ENVIRONMENT=production
   DEBUG=false
   REQUIRE_HTTPS=true
   ```

2. Use production AI provider:
   ```bash
   AI_PROVIDER=openai
   OPENAI_API_KEY=your_production_api_key
   AI_MOCK_MODE=false
   ```

3. Set secure secrets:
   ```bash
   JWT_SECRET=your_production_secret_key
   ENCRYPTION_KEY=your_encryption_key
   ```

### Build Production Binary

```bash
make prod-build
```

### Database Setup

1. Create PostgreSQL database with pgvector extension
2. Set up connection pooling
3. Run migrations: `./build/fixora -migrate`
4. Configure backup strategy

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `make test`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Code Style

- Follow Go formatting standards: `make fmt`
- Run linter: `make lint`
- Write tests for new features
- Update documentation

## Architecture Documentation

### Domain Models

- **Ticket**: Core entity representing IT support tickets
- **Comment**: Comments and discussions on tickets
- **KnowledgeBaseEntry**: Knowledge base articles with vector embeddings
- **KBChunk**: Text chunks for vector search
- **Metrics**: System performance and SLA metrics

### Use Cases

- **TicketUseCase**: Ticket lifecycle management
- **AIUseCase**: AI suggestion and analysis services
- **KnowledgeUseCase**: Knowledge base management

### Adapters

- **HTTP**: REST API handlers and SSE streaming
- **Persistence**: PostgreSQL repositories with vector search
- **AI**: Multi-provider AI service adapters

## Performance

### Vector Search

- Uses pgvector with ivfflat indexing
- Supports cosine similarity search
- Configurable top-K results
- Optimized for 100k+ chunks

### AI Services

- Streaming responses for better UX
- Caching for improved performance
- Fallback providers for reliability
- Configurable timeouts and retries

### Database

- Optimized indexes for common queries
- Connection pooling
- Materialized views for metrics
- Regular maintenance scripts

## Security

- JWT-based authentication
- Role-based access control
- Audit logging for compliance
- PII redaction options
- Rate limiting and request validation

## Monitoring

- Structured JSON logging
- Performance metrics collection
- Health check endpoints
- Error tracking and alerting

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:

1. Check the [documentation](docs/)
2. Search existing [issues](https://github.com/fixora/fixora/issues)
3. Create a new issue for bugs or feature requests
4. Join our community discussions

## Roadmap

- [ ] Web-based administration interface
- [ ] Advanced analytics dashboard
- [ ] Multi-tenant support
- [ ] Integration with external IT systems
- [ ] Mobile application
- [ ] Advanced AI features (auto-categorization, priority prediction)
- [ ] Email notification templates
- [ ] API rate limiting and quotas
- [ ] Advanced search and filtering
- [ ] Custom workflow automation
