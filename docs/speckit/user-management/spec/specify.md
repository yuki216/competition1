# Specify — User Management

## 🎯 Tujuan
Menentukan kebutuhan dan kontrak fitur **User Management** agar admin dapat mengelola pengguna dalam sistem dengan aman, terukur, dan terpisah dari logika domain lain.  
Fitur ini berfokus pada pengelolaan data pengguna (CRUD), bukan pada proses autentikasi atau otorisasi token.

---

## 🧩 Feature / Scope

- **Nama Bounded Context:** User Management  
- **Ruang lingkup:** Create, Update, Delete (soft delete), List, dan Detail User  
- **Nilai bisnis:**  
  Memudahkan admin dalam mengelola akun pengguna di dalam sistem, termasuk pembuatan user baru, pembaruan data, menonaktifkan akun, dan melihat daftar user aktif.  
  Dengan fitur ini, pengelolaan data pengguna menjadi lebih efisien, terpusat, dan mudah diaudit.

### ⚙️ Asumsi & Batasan
1. Hanya **role `admin`** yang dapat melakukan operasi CRUD user.  
2. Proses login dan autentikasi **tidak termasuk** dalam cakupan (ditangani oleh fitur **Auth**).  
3. Password untuk user baru dapat dibuat otomatis oleh sistem atau dikirim terpisah oleh admin.  
4. **Email user harus unik** dan **tidak dapat diubah** setelah akun dibuat.  
5. Operasi **hapus user** dilakukan secara **soft delete** untuk menjaga jejak audit.

---

## 🧱 Model Domain

Gunakan model domain yang telah ditetapkan dalam konteks **Auth**, dengan perluasan pada entity `User` untuk mendukung operasi administratif.

**Entity: `User`**
| Field | Type | Keterangan |
|-------|------|------------|
| id | UUID | Primary Key |
| name | string | Nama lengkap pengguna |
| email | string | Email unik pengguna |
| password | string | Password terenkripsi |
| role | enum(`admin`, `user`, `superadmin`) | Hak akses pengguna |
| status | enum(`active`, `inactive`) | Status akun |
| created_at | timestamp | Waktu pembuatan |
| updated_at | timestamp | Waktu pembaruan |
| deleted_at | timestamp (nullable) | Penanda soft delete |

---

## 🧪 Use Cases

### 1️⃣ Create User
**Deskripsi:** Admin membuat akun user baru.

**Input DTO:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "role": "user",
  "status": "active",
  "password": "Secret123!"
}
```

**Aturan Validasi:**
- Email harus unik.
- Password minimal 8 karakter.
- Role hanya dapat diisi dengan nilai valid.

**Response (Success):**
```json
{
  "status": true,
  "message": "success",
  "data": null
}
```

**Error:**
- `email_already_exists`
- `invalid_role`

---

### 2️⃣ Update User
**Deskripsi:** Admin memperbarui data user yang sudah ada.

**Input DTO:**
```json
{
  "name": "John D.",
  "role": "admin",
  "status": "inactive"
}
```

**Aturan Validasi:**
- Hanya field tertentu yang boleh diubah (`name`, `role`, `status`).
- Email tidak dapat diubah.

**Response (Success):**
```json
{
  "status": true,
  "message": "success",
  "data": null
}
```

**Error:**
- `user_not_found`
- `invalid_role`

---

### 3️⃣ Delete User
**Deskripsi:** Admin menonaktifkan atau menghapus user (soft delete).

**Langkah:**
1. Sistem menandai `deleted_at` pada data user.
2. Data tetap tersimpan untuk kebutuhan audit.

**Response (Success):**
```json
{
  "status": true,
  "message": "deleted",
  "data": null
}
```

**Error:**
- `user_not_found`

---

### 4️⃣ Get User Detail
**Deskripsi:** Mendapatkan informasi detail user berdasarkan ID.

**Input:**
- `user_id` (UUID)

**Response (Success):**
```json
{
  "status": true,
  "message": "success",
  "data": {
    "id": "uuid",
    "name": "John D.",
    "email": "john@example.com",
    "role": "admin",
    "status": "active",
    "created_at": "2025-11-06T00:00:00Z"
  }
}
```

**Error:**
- `user_not_found`

---

### 5️⃣ List Users
**Deskripsi:** Mendapatkan daftar user aktif dengan pagination dan filter.

**Input:**
```json
{
  "page": 1,
  "limit": 10,
  "filter": {
    "name": "John",
    "role": "admin",
    "status": "active"
  }
}
```

**Response (Success):**
```json
{
  "status": true,
  "message": "success",
  "data": {
    "users": [
      {
        "id": "uuid",
        "name": "John Doe",
        "email": "john@example.com",
        "role": "user",
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

**Error:**
- Tidak ada error khusus (kosong = hasil pencarian 0).

---

## 🧭 Catatan Tambahan
- Domain ini **terintegrasi langsung dengan Auth** untuk validasi role admin.  
- Semua aksi CRUD harus terekam dalam audit log.  
- Setiap endpoint wajib dilindungi middleware `AdminOnly`.  
- Domain disarankan memiliki `UserRepository` dan `UserService` dengan test coverage minimal 80%.