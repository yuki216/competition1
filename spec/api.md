# /speckit.api — API Specification Template (Go 1.24, DDD + Clean Architecture)

Tujuan
- Mendefinisikan kontrak API yang jelas, konsisten, dan teruji untuk setiap bounded context.

Prinsip Desain
- Resource-first, selaras dengan aggregates di domain (hindari RPC tersembunyi).
- Use case–aware: endpoint merepresentasikan intent bisnis yang valid.
- Stabilitas kontrak: versioning eksplisit, backward compatibility bila memungkinkan.
- Idempotensi untuk operasi yang berpotensi diulang (POST/PUT tertentu).
- Observability: setiap request memiliki correlation ID; error terstruktur.

Standar Global
- Base URL: `https://<host>/<version>` (contoh: `/v1`).
- Content-Type: `application/json; charset=utf-8`.
- Waktu: RFC3339 UTC (contoh: `2025-01-02T15:04:05Z`).
- Header wajib: `X-Correlation-ID` (request/response), opsi `Idempotency-Key`.
- Response Envelope
- Semua response menggunakan amplop konsisten:
```json
{"status": true, "message": "success", "data": {}}
```
- Error response:
```json
{"status": false, "message": "error message", "data": null}
```
- `data` wajib bernilai `null` untuk semua status selain `200`/`201`.
- `message` berisi ringkas status keberhasilan/penyebab kegagalan.


Versioning
- URL versioning: `/v1` → `/v2` saat kontrak berubah tidak kompatibel.
- Deprecation: gunakan header `Sunset` dan dokumen jadwal.

Keamanan
- Auth: JWT Bearer; `token_type` di-hardcode sebagai "Bearer" di klien/server; field `token_type` tidak dikembalikan dalam response.
- Algoritma JWT: produksi disarankan `RS256/ES256` (kunci via env `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`), pengembangan boleh `HS256` (secret via env `JWT_SECRET`); pilih via env `JWT_ALG`.
- Scopes/roles per endpoint: definisikan matrix akses.
- Rate limit: kembalikan `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`.

Pagination & Query
- Skema: `?page=1&per_page=20&sort=-created_at&filter[field]=value`.
- Response pagination:
```json
{
  "data": [/* items */],
  "page": 1,
  "per_page": 20,
  "total": 120
}
```

Caching & Konsistensi
- GET mendukung `ETag` dan `If-None-Match`.
- Optimistic concurrency: header `If-Match` dengan `ETag` pada update.

Template Endpoint
- Resource: <nama resource/aggregate>
- Method: <GET|POST|PUT|PATCH|DELETE>
- Path: `/<version>/<resource>`
- Ringkasan: <isi>
- Deskripsi: <isi>
- Auth: <skema & scope>
- Headers (request): <daftar>
- Params
  - Path: <daftar>
  - Query: <daftar>
- Request Body (contoh):
```json
{
  "field": "value"
}
```
- Response 200/201 (contoh):
```json
{
  "id": "uuid",
  "field": "value",
  "created_at": "2025-01-02T15:04:05Z"
}
```
- Status Codes:
  - 200 OK / 201 Created / 202 Accepted
  - 400 Bad Request / 401 Unauthorized / 403 Forbidden
  - 404 Not Found / 409 Conflict / 422 Unprocessable Entity
  - 429 Too Many Requests / 500 Internal Server Error
- Error Examples:
```json
{
  "code": "invalid_email",
  "message": "Email format is invalid",
  "details": {"email": "bad@"},
  "traceId": "c861f6a8-..."
}
```
- Idempotensi: gunakan header `Idempotency-Key` untuk POST yang menciptakan resource.
- Observability: balas `X-Correlation-ID` yang sama dengan request.

Contoh Sederhana — Users
- Create User
  - Method: POST
  - Path: `/v1/users`
  - Request:
```json
{"email": "a@b.c"}
```
  - Response 201:
```json
{"id": "uuid", "email": "a@b.c", "created_at": "2025-01-02T15:04:05Z"}
```
  - Error 422:
```json
{"code": "invalid_email", "message": "Email format is invalid"}
```
- Get User by ID
  - Method: GET
  - Path: `/v1/users/{id}`
  - Response 200:
```json
{"id": "uuid", "email": "a@b.c", "created_at": "2025-01-02T15:04:05Z"}
```

Webhooks (Opsional)
- Endpoint: `<callback url>`
- Event: `<domain_event_name>`
- Signature: `X-Signature` (HMAC SHA-256)
- Retries: exponential backoff; idempotensi berbasis `event_id`.
- Payload contoh:
```json
{
  "event_id": "uuid",
  "type": "user.created",
  "occurred_at": "2025-01-02T15:04:05Z",
  "data": {"id": "uuid", "email": "a@b.c"}
}
```

Message Contracts (Event-driven)
- Topic: `<bc>.<aggregate>.<event>` (contoh: `users.user.created`).
- Schema: Avro/JSON Schema; versi: `schema_version`.
- Semantics: at-least-once; dedup per `event_id`.

Test Template (TDD)
- Unit (handler tanpa I/O): validasi DTO, mapping status codes.
- Integration (adapter): bersihkan DB/MQ, seed data, jalankan request.
- Table-driven contoh:
```go
cases := []struct{
  name string
  req  map[string]any
  want int
}{
  {"valid", map[string]any{"email":"a@b.c"}, 201},
  {"invalid", map[string]any{"email":"bad"}, 422},
}
```

Contoh — Auth
- Login
  - Method: POST
  - Path: `/v1/auth/login`
  - Ringkasan: Login user dengan opsi Remember Me dan reCAPTCHA.
  - Deskripsi: Mengembalikan `access_token` (JWT) dan menyetel cookie `refresh_token` (HttpOnly, Secure, SameSite=Lax). Bila `remember_me=true`, TTL refresh diperpanjang.
  - Auth: None
  - Headers (request): `X-Correlation-ID` (wajib)
  - Request Body (contoh):
```json
{
  "email": "user@example.com",
  "password": "Str0ngP@ss",
  "remember_me": true,
  "recaptcha_token": "rct-123" // opsional; wajib jika reCAPTCHA enabled & tidak di-skip
}
```
  - Response 200 (contoh):
```json
{
  "status": true,
  "message": "success",
  "data": {
    "access_token": "jwt...",
    "expires_in": 900
  }
}
```
  - Status Codes: 200 OK; 400 Bad Request (`recaptcha_required`); 401 Unauthorized (`invalid_credentials`); 422 Unprocessable Entity (`recaptcha_invalid`); 429 Too Many Requests (`too_many_attempts`)
  - Error Examples:
```json
{"status": false, "message": "Email atau password salah", "data": null}
```

- Refresh Access Token
  - Method: POST
  - Path: `/v1/auth/refresh`
  - Ringkasan: Mengeluarkan `access_token` baru berbasis refresh token.
  - Auth: Cookie `refresh_token` (HttpOnly) atau header `Refresh-Token`.
  - Response 200 (contoh):
```json
{
  "status": true,
  "message": "success",
  "data": {
    "access_token":"jwt...",
    "expires_in":900
  }
}
```
  - Status Codes: 200 OK; 401 Unauthorized (`token_invalid`|`token_expired`|`token_revoked`)

- Logout
  - Method: POST
  - Path: `/v1/auth/logout`
  - Ringkasan: Mencabut semua refresh token milik user terautentikasi dan menghapus cookie.
  - Auth: JWT Bearer
  - Response 204: tanpa body (No Content). Cookie `refresh_token` dihapus (`Max-Age=0`).
- Catatan: untuk status selain 200/201, `data` harus `null`. Untuk 204, tidak ada body.

- Me (profil saat ini)
  - Method: GET
  - Path: `/v1/auth/me`
  - Ringkasan: Mendapatkan data user dari access token.
  - Auth: JWT Bearer
  - Response 200 (contoh):
```json
{
  "status": true,
  "message": "success",
  "data": {"id":"uuid","email":"user@example.com"}
}
```
  - Status Codes: 200 OK; 401 Unauthorized

Catatan reCAPTCHA
- Jika `RECAPTCHA_ENABLED=true` dan `RECAPTCHA_SKIP=false`, field `recaptcha_token` wajib dan diverifikasi ke `https://www.google.com/recaptcha/api/siteverify` menggunakan `secret` konfigurasi.
- Jika `RECAPTCHA_ENABLED=false` atau `RECAPTCHA_SKIP=true`, server tidak memvalidasi reCAPTCHA dan login berjalan normal.

Cookie & Keamanan
- `refresh_token`: HttpOnly, Secure, SameSite=Lax; TTL pendek (mis. 7 hari) atau panjang untuk Remember Me (mis. 30 hari).
- JWT: RS256/ES256; claim minimal (`sub`, `exp`, `iat`), hindari data sensitif.

Dokumentasi & Generasi
- OpenAPI 3.0: simpan definisi di `api/openapi.yaml` (opsional).
- Contoh `curl` untuk setiap endpoint.
- Versi dokumen & riwayat perubahan.