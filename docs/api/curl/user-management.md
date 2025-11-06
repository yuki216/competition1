# User Management API Documentation

## Base URL
```
http://localhost:8080/v1/admin
```

## Authentication
All endpoints require **Bearer Token** authentication with **Admin** or **Super Admin** role.

### Authorization Header
```
Authorization: Bearer <access_token>
```

---

## 1. Create User

Membuat user baru dalam sistem.

**Endpoint:** `POST /v1/admin/users`

**Request Headers:**
```bash
Content-Type: application/json
Authorization: Bearer <admin_access_token>
```

**Request Body:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "Secret123!",
  "role": "user",
  "status": "active"
}
```

**cURL Example:**
```bash
curl -X POST http://localhost:8080/v1/admin/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "Secret123!",
    "role": "user",
    "status": "active"
  }'
```

**Success Response (201):**
```json
{
  "status": true,
  "message": "User created successfully",
  "data": null
}
```

**Error Responses:**
- `400` - Bad Request: Invalid input data
- `401` - Unauthorized: Invalid or missing token
- `403` - Forbidden: User is not admin
- `409` - Conflict: Email already exists
- `422` - Unprocessable Entity: Validation errors

---

## 2. Update User

Memperbarui data user yang sudah ada.

**Endpoint:** `PUT /v1/admin/users/{id}`

**Request Headers:**
```bash
Content-Type: application/json
Authorization: Bearer <admin_access_token>
```

**Request Body:**
```json
{
  "name": "John D.",
  "role": "admin",
  "status": "inactive"
}
```

**cURL Example:**
```bash
curl -X PUT http://localhost:8080/v1/admin/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_access_token>" \
  -d '{
    "name": "John D.",
    "role": "admin",
    "status": "inactive"
  }'
```

**Success Response (200):**
```json
{
  "status": true,
  "message": "User updated successfully",
  "data": null
}
```

**Error Responses:**
- `400` - Bad Request: Invalid user ID
- `401` - Unauthorized: Invalid or missing token
- `403` - Forbidden: User is not admin
- `404` - Not Found: User not found
- `422` - Unprocessable Entity: Validation errors

---

## 3. Delete User

Menghapus user (soft delete).

**Endpoint:** `DELETE /v1/admin/users/{id}`

**Request Headers:**
```bash
Authorization: Bearer <admin_access_token>
```

**cURL Example:**
```bash
curl -X DELETE http://localhost:8080/v1/admin/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <admin_access_token>"
```

**Success Response (200):**
```json
{
  "status": true,
  "message": "User deleted successfully",
  "data": null
}
```

**Error Responses:**
- `400` - Bad Request: Invalid user ID
- `401` - Unauthorized: Invalid or missing token
- `403` - Forbidden: User is not admin
- `404` - Not Found: User not found

---

## 4. Get User Detail

Mendapatkan detail user berdasarkan ID.

**Endpoint:** `GET /v1/admin/users/{id}`

**Request Headers:**
```bash
Authorization: Bearer <admin_access_token>
```

**cURL Example:**
```bash
curl -X GET http://localhost:8080/v1/admin/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <admin_access_token>"
```

**Success Response (200):**
```json
{
  "status": true,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "John Doe",
    "email": "john@example.com",
    "role": "user",
    "status": "active",
    "created_at": "2025-11-06T10:30:00Z"
  }
}
```

**Error Responses:**
- `400` - Bad Request: Invalid user ID
- `401` - Unauthorized: Invalid or missing token
- `403` - Forbidden: User is not admin
- `404` - Not Found: User not found

---

## 5. List Users

Mendapatkan daftar user dengan pagination dan filter.

**Endpoint:** `GET /v1/admin/users`

**Request Headers:**
```bash
Authorization: Bearer <admin_access_token>
```

**Query Parameters:**
- `page` (integer, optional): Page number (default: 1)
- `limit` (integer, optional): Items per page (default: 10, max: 100)
- `name` (string, optional): Filter by name (partial match)
- `role` (string, optional): Filter by role (admin, user, superadmin, employee)
- `status` (string, optional): Filter by status (active, inactive)

**cURL Examples:**

**Basic List:**
```bash
curl -X GET "http://localhost:8080/v1/admin/users" \
  -H "Authorization: Bearer <admin_access_token>"
```

**With Pagination:**
```bash
curl -X GET "http://localhost:8080/v1/admin/users?page=2&limit=5" \
  -H "Authorization: Bearer <admin_access_token>"
```

**With Filters:**
```bash
curl -X GET "http://localhost:8080/v1/admin/users?name=John&role=admin&status=active" \
  -H "Authorization: Bearer <admin_access_token>"
```

**Success Response (200):**
```json
{
  "status": true,
  "message": "success",
  "data": {
    "users": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "John Doe",
        "email": "john@example.com",
        "role": "user",
        "status": "active"
      },
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "name": "Jane Smith",
        "email": "jane@example.com",
        "role": "admin",
        "status": "active"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 25
    }
  }
}
```

**Error Responses:**
- `401` - Unauthorized: Invalid or missing token
- `403` - Forbidden: User is not admin
- `500` - Internal Server Error: Server error

---

## Data Validation Rules

### Name
- Required
- Min length: 2 characters
- Max length: 255 characters

### Email
- Required
- Valid email format
- Unique (case-insensitive)

### Password
- Required for create user
- Min length: 8 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 digit
- At least 1 special character

### Role
- Required
- Valid values: `admin`, `user`, `superadmin`, `employee`

### Status
- Optional for create user (defaults to `active`)
- Valid values: `active`, `inactive`

---

## Error Response Format

All error responses follow this format:

```json
{
  "status": false,
  "message": "Error description",
  "data": null
}
```

## Common HTTP Status Codes

- `200` - OK: Request successful
- `201` - Created: Resource created successfully
- `400` - Bad Request: Invalid input data
- `401` - Unauthorized: Authentication required
- `403` - Forbidden: Insufficient permissions
- `404` - Not Found: Resource not found
- `409` - Conflict: Resource already exists
- `422` - Unprocessable Entity: Validation failed
- `500` - Internal Server Error: Server error