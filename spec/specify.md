# /speckit.specify — Define Requirements & User Stories

Tujuan
- Mendefinisikan apa yang dibangun: kebutuhan, user stories, dan kontrak.

Feature/Scope
- Nama fitur/BC: Auth — Login, Remember Me, reCAPTCHA (toggle)
- Ringkasan nilai bisnis: Login aman dengan friksi terukur; bot dicegah; pengalaman bisa diaktifkan/di-skip sesuai konteks.
- Asumsi & batasan: Auth berbasis JWT (access token singkat); refresh token disimpan di cookie HttpOnly; reCAPTCHA opsional (dapat diaktifkan atau di-skip via konfigurasi/env/role).

Bounded Context
- Bahasa ubiquitous (istilah utama): login, access_token, refresh_token, remember_me, recaptcha_token, verifier, revoke
- Integrasi ke BC lain: Users (sync via repository), Anti-bot reCAPTCHA (HTTP external, sync), Session/RefreshToken storage (persistence adapter)

Model Domain
- Entities: User (id, email, password_hash, status); RefreshToken (id, user_id, token_hash, expires_at, revoked); Session (id, user_id, created_at, expires_at).
- Value Objects: Email (validasi format); Password (hash + policy); RecaptchaToken (string + masa berlaku).
- Aggregates: Auth (root: User; konsistensi: refresh token per perangkat; aturan revoke).
- Domain Services: AuthService (verifikasi kredensial), TokenService (issue/revoke), RecaptchaVerifier (verify ke eksternal via port).
- Commands: Login(email, password, remember_me, recaptcha_token?), Refresh(refresh_token), Logout(access_token context).
- Domain Events: UserLoggedIn, TokenRefreshed, UserLoggedOut; payload: user_id, device_id; idempotensi berbasis event_id.
- Policies/Saga: Wajib verifikasi reCAPTCHA bila enabled; boleh skip bila flag skip aktif atau environment dev/test.

Use Cases
- Login: Input DTO {email, password, remember_me, recaptcha_token?}; Output DTO {access_token, expires_in}; Langkah: (opsional) verifikasi reCAPTCHA → verifikasi kredensial → issue access+refresh token → set cookie HttpOnly → catat event. Error: invalid_credentials, recaptcha_required, recaptcha_invalid, account_locked.
- Refresh Access Token: Input DTO {refresh_token}; Output DTO {access_token, expires_in}; Langkah: validasi refresh → issue access baru → rotasi/pertahankan refresh sesuai kebijakan. Error: token_invalid, token_revoked, token_expired.
- Logout: Input DTO {refresh_token}; Output: none; Langkah: revoke refresh token → catat event. Error: token_missing, token_invalid.

Kontrak API
- Login
  - Method: POST
  - Path: `/v1/auth/login`
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
  "data" : {
    "access_token": "jwt...",
    "expires_in": 900
  }
}
```
  - Status Codes: 200 OK; 400 Bad Request (`recaptcha_required`); 401 Unauthorized (`invalid_credentials`); 422 Unprocessable Entity (`recaptcha_invalid`); 429 Too Many Requests (`too_many_attempts`)

- Refresh Access Token
  - Method: POST
  - Path: `/v1/auth/refresh`
  - Auth: Cookie `refresh_token` (HttpOnly) atau header `Refresh-Token`
  - Request: tidak ada body; ambil refresh token dari cookie/header
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
  - Status Codes: 200 OK; 401 Unauthorized (`token_invalid`|`token_expired`|`token_revoked`)

- Logout
  - Method: POST
  - Path: `/v1/auth/logout`
  - Auth: JWT Bearer (Authorization header)
  - Request: tidak ada body
  - Response 204: tanpa body (No Content). Cookie `refresh_token` dihapus (`Max-Age=0`).
- Catatan: untuk semua status selain 200/201, `data` harus `null`. Untuk 204, tidak ada body.

- Me (profil saat ini)
  - Method: GET
  - Path: `/v1/auth/me`
  - Auth: JWT Bearer
  - Response 200 (contoh):
```json
{
  "status": true,
  "message": "success",
  "data": {
    "id": "uuid",
    "email": "user@example.com"
  }
}
```
  - Status Codes: 200 OK; 401 Unauthorized

Ports (Kontrak Aplikasi)
- Repository Port: operasi, pre/post-condition, error.
- Message Port: topik, schema, idempotensi, retry.

Invariants & Aturan Bisnis
- Password di-hash; verifikasi menggunakan hasher yang konsisten (unit test).
- Refresh token unik per perangkat; revoke membatalkan akses (integration test pada repository).
- Remember Me memperpanjang TTL refresh token (mis. 30 hari); default tanpa remember Me lebih pendek (mis. 7 hari).
- reCAPTCHA: jika `RECAPTCHA_ENABLED=true` dan `RECAPTCHA_SKIP=false`, login wajib membawa `recaptcha_token` valid; jika `RECAPTCHA_ENABLED=false` atau `RECAPTCHA_SKIP=true`, server tidak memverifikasi reCAPTCHA.
- Access token TTL singkat (mis. 15 menit); tidak disimpan di server (stateless JWT). Pilihan algoritma `JWT_ALG` (`RS256/ES256` produksi dengan `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`; `HS256` dev dengan `JWT_SECRET`).
- Semua operasi mencatat event dan correlation ID untuk observability.

Konfigurasi Lingkungan (diselaraskan)
- `DATABASE_URL` — koneksi Postgres.
- `JWT_ALG` — algoritma JWT (`RS256`/`ES256`/`HS256`).
- `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` — kunci JWT untuk RS/ES.
- `JWT_SECRET` — secret JWT untuk HS (dev).
- `REFRESH_TOKEN_SALT` — salt untuk hashing refresh token.
- `RECAPTCHA_ENABLED`, `RECAPTCHA_SKIP`, `RECAPTCHA_SECRET` — kontrol & secret reCAPTCHA.
- `PORT` — port HTTP.

Error Catalog
- invalid_credentials — 401 Unauthorized — "Email atau password salah" — mitigasi: throttling/lockout.
- recaptcha_required — 400 Bad Request — "Token reCAPTCHA wajib" — mitigasi: tampilkan challenge.
- recaptcha_invalid — 422 Unprocessable Entity — "Token reCAPTCHA tidak valid" — mitigasi: ulangi challenge.
- token_invalid — 401 Unauthorized — "Refresh token tidak valid" — mitigasi: login ulang.
- token_revoked — 401 Unauthorized — "Refresh token sudah dicabut" — mitigasi: login ulang.
- token_expired — 401 Unauthorized — "Refresh token kedaluwarsa" — mitigasi: login ulang.
- account_locked — 403 Forbidden — "Akun dikunci" — mitigasi: prosedur unlock.
- too_many_attempts — 429 Too Many Requests — "Percobaan login berlebih" — mitigasi: cooldown.

Skenario & Acceptance Criteria
- Skenario utama:
  - Given reCAPTCHA disabled, When login dengan kredensial valid & remember_me=false, Then 200 dan access_token dikembalikan, cookie refresh diset TTL pendek.
  - Given reCAPTCHA enabled, When login dengan kredensial valid & remember_me=true & recaptcha_token valid, Then 200 dan access_token dikembalikan, cookie refresh diset TTL panjang.
  - Given reCAPTCHA enabled, When login tanpa/recaptcha_token invalid, Then 400/422 dengan kode error sesuai.
  - Given kredensial invalid, When login, Then 401 invalid_credentials.
  - Given refresh token valid, When refresh, Then 200 dengan access_token baru.
  - Given access token valid, When logout, Then 204; semua refresh token user direvoke dan cookie `refresh_token` dihapus.
- Edge cases: refresh setelah revoke; akun locked; banyak percobaan login; skip reCAPTCHA pada env dev/test atau role trusted.
- Kriteria penerimaan:
  - Response codes sesuai spesifikasi; bentuk error konsisten.
  - Cookie `refresh_token` HttpOnly+Secure+SameSite=Lax di-set/di-hapus sesuai aksi.
  - TTL berbeda untuk remember_me=true vs false; diukur dengan expiry.
  - reCAPTCHA diverifikasi saat enabled; tidak ada call saat disabled/skip.
  - Observability: `X-Correlation-ID` tercermin; event audit dicatat.

Test Data
- User valid: {id:"uuid-1", email:"user@example.com", password_hash:"$2a$..."}
- Kredensial: valid {email:"user@example.com", password:"Str0ngP@ss"}; invalid {password:"wrong"}, {email:"bad@"}.
- Refresh token: valid {token_hash:"hash1", expires:+7d/+30d}; invalid {revoked:true}, {expired:true}.
- reCAPTCHA: token valid {"token":"rct-123"}; token invalid {"token":"bad"}; kosong.

NFR Spesifik Fitur
- Keamanan: password hash kuat (bcrypt/argon2), JWT RS256/ES256, cookie HttpOnly+Secure+SameSite=Lax.
- Kinerja: verifikasi reCAPTCHA ≤ 200ms rata-rata; login end-to-end ≤ 500ms p95.
- Observability: log terstruktur; tracing untuk login/refresh; audit event UserLoggedIn/LoggedOut.
- Compliance: rotasi secret secara berkala; enkripsi at-rest untuk refresh token.
- Konfigurasi toggle: `RECAPTCHA_ENABLED` (bool), `RECAPTCHA_SKIP` (bool/role/env), `REMEMBER_ME_TTL_DAYS` (int).

Persistence (Postgres)
- Skema Tabel:
  - `users`: id uuid pk, email citext unique, password_hash text, status varchar(16), created_at timestamptz.
  - `refresh_tokens`: id uuid pk, user_id uuid fk users(id), token_hash bytea unique, device_id text, issued_at timestamptz, expires_at timestamptz, revoked boolean, revoked_at timestamptz, reason text.
- Keamanan Token:
  - Refresh token disimpan sebagai `token_hash` (hasil `sha256(token_plain + REFRESH_TOKEN_SALT)`), bukan plaintext.
  - Cookie `refresh_token` membawa plaintext; server melakukan hash saat validasi.
- Operasi Utama:
  - Create: `INSERT INTO refresh_tokens(user_id, token_hash, expires_at, device_id) VALUES ($1,$2,$3,$4)`
  - Get: `SELECT id, user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash=$1`
  - Revoke: `UPDATE refresh_tokens SET revoked=true, revoked_at=now() WHERE token_hash=$1`
- Indeks:
  - unique(token_hash), idx(user_id), idx(expires_at), idx(revoked).
- Env & Koneksi:
  - `DATABASE_URL`, `REFRESH_TOKEN_SALT`; koneksi via `database/sql` + `pgx`.
- TTL & Remember Me:
  - `remember_me=true` memperpanjang `expires_at` (mis. 30 hari) dibanding default (mis. 7 hari).
- Catatan Cookie:
  - `HttpOnly`, `Secure`, `SameSite=Lax`; domain/path disesuaikan kebutuhan.