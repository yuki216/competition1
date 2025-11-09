# Rencana Mini Iterasi (Auth)

## Iterasi 1 — Baseline & Login Use Case
- Tujuan
  - Fondasi arsitektur (struktur folder), entity/VO domain untuk Auth.
  - Implementasi `LoginUseCase` dengan reCAPTCHA mock, JWT issuer mock.
  - Unit tests untuk happy path + error mapping dasar.
- Langkah
  - Buat kontrak ports (`application/repository`): UserRepository, TokenService, RecaptchaVerifier, RefreshTokenRepository.
  - Implementasi domain entities/VO minimal (User, RefreshToken VO bila perlu).
  - Tulis unit test table-driven untuk skenario: valid/invalid credentials, reCAPTCHA toggle enabled/skip.
  - Implementasi minimal use case hingga test hijau.
- Risiko & Mitigasi
  - Ketidakselarasan kontrak — mitigasi: kunci signature ports dari `spec/constitution.md`.
  - Coverage rendah — mitigasi: target ≥80% domain/app.

## Iterasi 2 — Adapter Nyata & Endpoints REST
- Tujuan
  - Adapter nyata: PasswordHasher, JWT issuer, RefreshToken persistence (Postgres).
  - Endpoint REST: `POST /v1/auth/login`, `POST /v1/auth/refresh`, `POST /v1/auth/logout`, `GET /v1/auth/me`.
  - Remember Me TTL.
- Langkah
  - Siapkan koneksi Postgres (`DATABASE_URL`), migrasi `refresh_tokens`.
  - Implementasi `RefreshTokenRepository` (create/get/revoke) dengan parameterized queries.
  - Implementasi JWT issuer sesuai `JWT_ALG` (dev: HS256, prod: RS/ES).
  - Handler REST dengan envelope `status/message/data` dan cookie aman.
  - Integration tests untuk adapter DB dan handler REST.
- Risiko & Mitigasi
  - Kebocoran token — mitigasi: simpan hanya hash, audit log.
  - Konsistensi cookie — mitigasi: `HttpOnly`, `Secure`, `SameSite` sesuai `spec/api.md`.

## Iterasi 3 — Integrasi reCAPTCHA & Observability
- Tujuan
  - Integrasi eksternal reCAPTCHA `siteverify`.
  - Rate limiting/throttling.
  - Observability (log/trace) dan error katalog lengkap.
- Langkah
  - Implementasi client reCAPTCHA (`RECAPTCHA_SECRET`), toggle `RECAPTCHA_ENABLED`/`RECAPTCHA_SKIP`.
  - Tambahkan throttling percobaan login.
  - Tambah logging terstruktur dengan `X-Correlation-ID`.
  - Lengkapi error katalog & acceptance tests end-to-end.
- Risiko & Mitigasi
  - Ketergantungan eksternal — mitigasi: fallback & timeout; test dengan mock.
  - Performa logging — mitigasi: sampling dan format terstruktur.