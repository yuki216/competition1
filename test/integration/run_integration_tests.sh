#!/bin/bash

# Integration Test Runner for Fixora API
# This script sets up the test environment and runs integration tests

set -e

echo "🧪 Starting Fixora API Integration Tests"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_DB_NAME="fixora_test"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5432"
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="postgres"

echo -e "${YELLOW}Setting up test environment...${NC}"

# Check if PostgreSQL is running
if ! pg_isready -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER; then
    echo -e "${RED}❌ PostgreSQL is not running or not accessible${NC}"
    echo "Please make sure PostgreSQL is running with:"
    echo "  Host: $POSTGRES_HOST"
    echo "  Port: $POSTGRES_PORT"
    echo "  User: $POSTGRES_USER"
    exit 1
fi

echo -e "${GREEN}✅ PostgreSQL is accessible${NC}"

# Check if we can connect to PostgreSQL
if ! PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d postgres -c '\q' 2>/dev/null; then
    echo -e "${RED}❌ Cannot connect to PostgreSQL with provided credentials${NC}"
    echo "Please check your PostgreSQL configuration and credentials"
    exit 1
fi

echo -e "${GREEN}✅ PostgreSQL connection successful${NC}"

# Set environment variables for tests
export DB_HOST=$POSTGRES_HOST
export DB_PORT=$POSTGRES_PORT
export DB_USER=$POSTGRES_USER
export DB_PASSWORD=$POSTGRES_PASSWORD
export DB_NAME=$TEST_DB_NAME
export ENVIRONMENT=test

echo -e "${YELLOW}Environment variables set:${NC}"
echo "  DB_HOST=$DB_HOST"
echo "  DB_PORT=$DB_PORT"
echo "  DB_USER=$DB_USER"
echo "  DB_NAME=$DB_NAME"
echo "  ENVIRONMENT=$ENVIRONMENT"

echo ""
echo -e "${YELLOW}Running integration tests...${NC}"

# Run the integration tests
cd ../../.. # Go to project root

# Run tests with verbose output
echo -e "${YELLOW}Executing: go test -v ./test/integration/...${NC}"
if go test -v ./test/integration/... -timeout=30s; then
    echo ""
    echo -e "${GREEN}🎉 All integration tests passed!${NC}"
else
    echo ""
    echo -e "${RED}❌ Some integration tests failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Integration tests completed successfully${NC}"