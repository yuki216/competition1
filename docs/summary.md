# Ringkasan Eksekusi Spec (Auth, Go 1.24)

- Arsitektur: DDD + Clean Architecture; layer domain/application/delivery/infrastructure (lihat `spec/constitution.md`).
- Ports: `UserRepository`, `RefreshTokenRepository`, `TokenService`, `RecaptchaVerifier`, `EventPublisher` (kontrak di application).
- Data: tabel `refresh_tokens` dengan indeks `unique(token_hash)`, `idx(user_id)`, `idx(expires_at)`, `idx(revoked)` (lihat `spec/specify.md`).
- Keamanan: JWT Bearer; refresh token disimpan hashed (`sha256(plain + SALT)`); cookie `HttpOnly+Secure+SameSite`.
- API: login/refresh/logout/me; envelope konsisten `{status,message,data}`; versioning `/v1` (lihat `spec/api.md`).
- Env: `DATABASE_URL`, `JWT_ALG`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` atau `JWT_SECRET`, `REFRESH_TOKEN_SALT`, `RECAPTCHA_ENABLED`, `RECAPTCHA_SKIP`, `RECAPTCHA_SECRET`, `PORT`.
- Observability: header `X-Correlation-ID` melekat di request/response dan log.
- TDD: test dulu (unit + integration), implementasi minimal, refactor, jaga green; coverage domain/app ≥80%.
- Migrasi: jalankan sebelum app start; backward-safe bila memungkinkan; format `YYYYMMDDHHMMSS_<name>.sql`.
- DoD: tiap layer mengikuti checklist di `spec/constitution.md` dan validasi akhir di `spec/implement.md`.