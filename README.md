# Auth Service

Authentication service implementing Clean Architecture with TDD approach.

## Features

- User registration and login
- JWT-based authentication
- Refresh token mechanism
- Secure password hashing with bcrypt
- PostgreSQL database
- Clean Architecture implementation
- Comprehensive testing

## Prerequisites

- Go 1.21+
- PostgreSQL 12+
- Make (optional)

## Setup

1. Clone the repository
2. Copy the environment configuration:
   ```bash
   cp .env.example .env
   ```

3. Update the `.env` file with your database credentials and JWT secret

4. Create the database:
   ```bash
   createdb auth_service
   ```

5. Run migrations:
   ```bash
   psql -d auth_service -f migrations/001_create_auth_tables.sql
   ```

6. Install dependencies:
   ```bash
   go mod download
   ```

7. Run the server:
   ```bash
   go run cmd/server/main.go
   ```

## API Endpoints

### Authentication

- `POST /v1/auth/login` - Login user
- `POST /v1/auth/refresh` - Refresh access token
- `POST /v1/auth/logout` - Logout user (requires auth)
- `GET /v1/auth/me` - Get current user (requires auth)

### Health Check

- `GET /health` - Health check endpoint

## Testing

Run unit tests:
```bash
go test ./test/unit/...
```

Run integration tests:
```bash
go test ./test/integration/...
```

Run all tests:
```bash
go test ./...
```

## Architecture

This project follows Clean Architecture principles with the following layers:

- **Domain Layer**: Contains business entities and rules
- **Application Layer**: Contains use cases and application logic
- **Infrastructure Layer**: Contains external dependencies (database, HTTP handlers, etc.)
- **Interface Layer**: Contains API contracts and DTOs

## Configuration

The service can be configured through environment variables:

- `DATABASE_URL`: PostgreSQL connection string
- `SERVER_HOST`: Server host (default: 0.0.0.0)
- `SERVER_PORT`: Server port (default: 8080)
- `JWT_SECRET`: Secret key for JWT signing
- `JWT_ALGORITHM`: JWT algorithm (default: HS256)
- `ACCESS_TOKEN_TTL`: Access token time-to-live (default: 15m)
- `REFRESH_TOKEN_TTL`: Refresh token time-to-live (default: 7d)

## Security

- Passwords are hashed using bcrypt with cost factor 10
- JWT tokens are signed with HS256 algorithm
- Refresh tokens are stored in the database and can be revoked
- All endpoints use HTTPS in production

## License

MIT