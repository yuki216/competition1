# Comment on Ticket API Documentation

## Overview
This document describes the REST API endpoints for managing comments on tickets in the application. The Comment on Ticket feature allows users to add, view, update, and delete comments on support tickets.

## Base URL
```
https://api.example.com/v1
```

## Authentication
All comment endpoints require authentication via JWT Bearer token or appropriate user context headers:
- `X-User-ID`: User identifier
- `X-User-Role`: User role (EMPLOYEE, ADMIN)

## Endpoints

### 1. Create Comment
Creates a new comment on a ticket.

**Endpoint:** `POST /v1/tickets/{id}/comments`

**Path Parameters:**
- `id` (string, required): The ID of the ticket

**Headers:**
- `Content-Type: application/json`
- `X-User-ID` (string, required): User ID
- `X-User-Role` (string, optional): User role (defaults to EMPLOYEE)

**Request Body:**
```json
{
  "ticket_id": "ticket-123",
  "author_id": "user-123",
  "role": "EMPLOYEE",
  "body": "This is a test comment"
}
```

**Response 201 Created:**
```json
{
  "status": true,
  "message": "Comment created successfully",
  "data": {
    "id": "comment-456",
    "ticket_id": "ticket-123",
    "author_id": "user-123",
    "role": "EMPLOYEE",
    "body": "This is a test comment",
    "created_at": "2025-01-02T15:04:05Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body or missing required fields
- `404 Not Found`: Ticket not found
- `422 Unprocessable Entity`: Validation failed (empty body, too long, invalid role)
- `500 Internal Server Error`: Server error

### 2. Get Comments by Ticket
Retrieves comments for a specific ticket with pagination.

**Endpoint:** `GET /v1/tickets/{id}/comments`

**Path Parameters:**
- `id` (string, required): The ID of the ticket

**Query Parameters:**
- `page` (integer, optional): Page number (default: 1)
- `per_page` (integer, optional): Items per page (default: 20, max: 100)

**Response 200 OK:**
```json
{
  "status": true,
  "message": "Comments retrieved successfully",
  "data": {
    "comments": [
      {
        "id": "comment-1",
        "ticket_id": "ticket-123",
        "author_id": "user-123",
        "role": "EMPLOYEE",
        "body": "This is the first comment",
        "created_at": "2025-01-02T15:04:05Z"
      },
      {
        "id": "comment-2",
        "ticket_id": "ticket-123",
        "author_id": "admin-456",
        "role": "ADMIN",
        "body": "This is an admin response",
        "created_at": "2025-01-02T15:10:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "per_page": 20
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid ticket ID
- `404 Not Found`: Ticket not found
- `500 Internal Server Error`: Server error

### 3. Update Comment
Updates an existing comment. Comments can only be edited within 15 minutes of creation.

**Endpoint:** `PATCH /v1/comments/{id}`

**Path Parameters:**
- `id` (string, required): The ID of the comment

**Headers:**
- `Content-Type: application/json`

**Request Body:**
```json
{
  "body": "This is the updated comment content"
}
```

**Response 200 OK:**
```json
{
  "status": true,
  "message": "Comment updated successfully",
  "data": {
    "id": "comment-456",
    "ticket_id": "ticket-123",
    "author_id": "user-123",
    "role": "EMPLOYEE",
    "body": "This is the updated comment content",
    "created_at": "2025-01-02T15:04:05Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body or empty comment body
- `403 Forbidden`: Edit window expired (more than 15 minutes since creation)
- `404 Not Found`: Comment not found
- `422 Unprocessable Entity`: Comment body exceeds 5000 characters
- `500 Internal Server Error`: Server error

### 4. Delete Comment
Deletes a comment. Comments can only be deleted within 15 minutes of creation.

**Endpoint:** `DELETE /v1/comments/{id}`

**Path Parameters:**
- `id` (string, required): The ID of the comment

**Response 204 No Content:**
No response body.

**Error Responses:**
- `400 Bad Request`: Invalid comment ID
- `403 Forbidden`: Delete window expired (more than 15 minutes since creation)
- `404 Not Found`: Comment not found
- `500 Internal Server Error`: Server error

## Data Models

### Comment Object
```json
{
  "id": "string",
  "ticket_id": "string",
  "author_id": "string",
  "role": "EMPLOYEE|ADMIN|AI",
  "body": "string",
  "created_at": "ISO8601 datetime"
}
```

### Comment Roles
- `EMPLOYEE`: Regular user comment
- `ADMIN`: Administrator comment
- `AI`: System-generated AI comment

## Validation Rules

### Comment Body
- Required: Yes
- Minimum length: 1 character
- Maximum length: 5000 characters
- Cannot be empty or whitespace only

### Edit/Delete Window
- Comments can be edited or deleted within 15 minutes of creation
- After 15 minutes, comments become read-only
- This prevents abuse while allowing for typo corrections

## Rate Limiting
- Comment creation: 10 comments per minute per user
- Comment updates: 5 updates per minute per comment
- Comment deletion: 5 deletions per minute per user

## Security Considerations
- All endpoints require proper authentication
- Users can only edit/delete their own comments (except admins)
- Comment content is sanitized to prevent XSS attacks
- SQL injection protection via parameterized queries
- Audit logging for all comment operations

## Error Code Reference

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `ticket_id` | 400 | Ticket ID is required |
| `comment_id` | 400 | Comment ID is required |
| `invalid_request` | 400 | Invalid request body |
| `ticket_not_found` | 404 | Ticket not found |
| `comment_not_found` | 404 | Comment not found |
| `empty_comment_body` | 422 | Comment body is required |
| `comment_too_long` | 422 | Comment body exceeds 5000 characters |
| `invalid_role` | 422 | Invalid comment role |
| `edit_window_expired` | 403 | Comment can only be edited within 15 minutes |
| `delete_window_expired` | 403 | Comment can only be deleted within 15 minutes |
| `internal_error` | 500 | Internal server error |

## Examples

### Example 1: Creating a comment on a ticket
```bash
curl -X POST https://api.example.com/v1/tickets/ticket-123/comments \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-456" \
  -H "X-User-Role: EMPLOYEE" \
  -d '{
    "body": "I'm experiencing the same issue. My internet connection drops every few minutes."
  }'
```

### Example 2: Getting comments for a ticket
```bash
curl -X GET "https://api.example.com/v1/tickets/ticket-123/comments?page=1&per_page=10"
```

### Example 3: Updating a comment
```bash
curl -X PATCH https://api.example.com/v1/comments/comment-789 \
  -H "Content-Type: application/json" \
  -d '{
    "body": "Updated: The connection drops every 5 minutes, not every few minutes."
  }'
```

### Example 4: Deleting a comment
```bash
curl -X DELETE https://api.example.com/v1/comments/comment-789
```

## Integration Notes

### Events Published
- `comment.created`: When a new comment is created
- `comment.updated`: When a comment is updated
- `comment.deleted`: When a comment is deleted

### Notifications Sent
- `NotifyCommentCreated`: Sent to relevant parties when a comment is added to a ticket they're involved in

### Integration with Ticket Flow
- Comments are automatically added when tickets are resolved (system comments)
- Comments are included in ticket detail responses
- AI-generated comments can be created when `role=AI` is used

## Version History

### v1.0.0 (2025-01-06)
- Initial implementation of Comment on Ticket API
- CRUD operations for comments
- 15-minute edit/delete window
- Pagination support for comment lists
- Role-based comment system
- Full integration with existing ticket system