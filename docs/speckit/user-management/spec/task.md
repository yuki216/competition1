# Task — User Management

## Ceklist Task
[x] implementasi code berdasarkan specify
[x] buatkan dokumentasi api menggunakan curl, simpan di docs/api/curl

## 📝 Implementasi Selesai

### ✅ Komponen yang Diimplementasikan:

1. **Entity Updates**
   - `domain/entity/user.go`: Ditambahkan field `name`, `status`, `deleted_at`
   - Constructor dan helper methods untuk soft delete dan update

2. **Database Migration**
   - `migrations/005_add_user_management_fields.up.sql`: Menambah kolom baru
   - `migrations/005_add_user_management_fields.down.sql`: Rollback migration

3. **Repository Layer**
   - Update interface `UserRepository` dengan method baru:
     - `Update()`: Update user data
     - `SoftDelete()`: Soft delete user
     - `FindAll()`: List users dengan pagination & filter
     - `ExistsByEmail()`: Check email uniqueness
     - `FindByRole()`: Filter by role
   - Implementasi lengkap di kedua repository adapter

4. **Use Case Layer**
   - `CreateUserUseCase`: Validasi & pembuatan user baru
   - `UpdateUserUseCase`: Update data user
   - `DeleteUserUseCase`: Soft delete user
   - `GetUserDetailUseCase`: Get user detail
   - `ListUsersUseCase`: List users dengan pagination
   - `UserManagementUseCaseImpl`: Composite use case

5. **Middleware**
   - `RequireAdmin()`: Validasi admin role untuk proteksi endpoint
   - Support untuk `admin` dan `superadmin` role

6. **HTTP Handler**
   - `UserManagementHandler`: Handler lengkap untuk semua endpoints
   - Validasi input, error handling, response formatting
   - Route registration dengan admin middleware

7. **API Documentation**
   - `docs/api/curl/user-management.md`: Dokumentasi lengkap dengan cURL examples
   - Semua endpoints terdokumentasi dengan request/response samples

### 🔍 Fitur yang Tersedia:

- **Create User**: `POST /v1/admin/users`
- **Update User**: `PUT /v1/admin/users/{id}`
- **Delete User**: `DELETE /v1/admin/users/{id}` (soft delete)
- **Get User Detail**: `GET /v1/admin/users/{id}`
- **List Users**: `GET /v1/admin/users` dengan pagination & filter

### 🛡️ Security Features:
- Admin-only access control
- Bearer token authentication
- Input validation & sanitization
- SQL injection prevention dengan parameterized queries

### 📊 Quality:
- Clean Architecture (DDD + Ports & Adapters)
- SOLID principles compliance
- Error handling yang konsisten
- Type safety dengan Go's strong typing