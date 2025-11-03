# Checklist Implementasi (Auth)

## Struktur & Arsitektur
- [ ] Struktur folder mengikuti base (`domain/`, `application/`, `delivery/`, `infrastructure/`, `migrations/`, `pkg/`, `proto/`, `test/`, `web/`).
- [ ] Aturan import dipatuhi (domain bebas I/O, application mengimpor domain + ports, delivery mengimpor application, infrastructure mengimplementasikan ports).
- [ ] Ports didefinisikan di application: UserRepository, RefreshTokenRepository, TokenService, RecaptchaVerifier, EventPublisher.

## Environment & Konfigurasi
- [ ] `DATABASE_URL`, `JWT_ALG`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` atau `JWT_SECRET`, `REFRESH_TOKEN_SALT`, `RECAPTCHA_ENABLED`, `RECAPTCHA_SKIP`, `RECAPTCHA_SECRET`, `PORT` diset.
- [ ] Validasi env di startup; default aman untuk dev.

## Migrasi & Persistence
- [ ] Migrasi `refresh_tokens` up/down berjalan; indeks tersedia (`unique(token_hash)`, `idx(user_id)`, `idx(expires_at)`, `idx(revoked)`).
- [ ] `RefreshTokenRepository` mendukung create/get/revoke dengan parameterized queries.
- [ ] Token refresh disimpan dalam bentuk hashed (`sha256(plain + SALT)`), bukan plaintext.

## Implementasi Use Case & API
- [ ] LoginUseCase mendukung reCAPTCHA toggle (enabled/skip), credentials valid/invalid, remember_me TTL.
- [ ] Refresh/Logout/Me sesuai kontrak dan idempotensi revoke.
- [ ] REST handlers memakai envelope `{status,message,data}`; cookie `HttpOnly`,`Secure`,`SameSite=Lax`.

## Testing & Quality Gates
- [ ] Unit tests untuk setiap use case (happy + error mapping).
- [ ] Integration tests untuk adapter DB/JWT/reCAPTCHA dan REST handlers.
- [ ] Coverage domain/application ≥80%; `go test ./... -race -cover` hijau.

## Observability & Keamanan
- [ ] `X-Correlation-ID` dilewatkan di request/response dan log.
- [ ] Header keamanan & cookie policy konsisten.
- [ ] Error katalog diperbarui dan dipetakan ke HTTP status.

## Dokumentasi & Rilis
- [ ] Release Notes ditulis (versi, perubahan utama, kompatibilitas/migrasi).
- [ ] Retrospective dibuat (yang baik/yang perlu ditingkatkan/aksi perbaikan).