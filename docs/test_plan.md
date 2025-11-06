# Test Plan — Auth Flow

## Tujuan
- Memverifikasi alur login/refresh/logout/me sesuai kontrak `spec/api.md` dan aturan arsitektur `spec/constitution.md`.
- Menjaga kualitas dengan cakupan domain/application ≥80% dan integration tests adapters.

## Unit Tests
- LoginUseCase
  - Kredensial valid menghasilkan JWT + opsi remember_me mempengaruhi TTL refresh.
  - Kredensial invalid mengembalikan error terstruktur (401) tanpa kebocoran detail.
  - reCAPTCHA toggle:
    - `RECAPTCHA_ENABLED=true` → token wajib; invalid → `recaptcha_invalid`.
    - `RECAPTCHA_SKIP=true` → lewati verifikasi.
- RefreshTokenUseCase
  - Token hashed ditemukan dan belum revoked → akses baru dikeluarkan.
  - Token expired/revoked → error sesuai katalog.
- LogoutUseCase
  - Idempotensi revoke: revoke dua kali konsisten.
- MeUseCase
  - Token akses valid → klaim user dikembalikan.

## Integration Tests
- Adapter RecaptchaVerifier
  - Mock: valid/invalid; Real: hit `siteverify` dengan secret; timeout & error path.
- JWT Issuer
  - HS dev (JWT_SECRET) dan RS/ES prod (PRIVATE/PUBLIC KEY) → issue+parse.
- RefreshTokenRepository (Postgres)
  - Create→Get: simpan hash, verifikasi expiry dan user_id.
  - Revoke: idempotensi; dua kali tetap konsisten.
  - Expired: ditolak pada use case.
- REST Handlers
  - Envelope `{status,message,data}` konsisten.
  - Cookie aman (`HttpOnly`,`Secure`,`SameSite=Lax`).

## Negative & Edge Cases
- Throttling percobaan login berulang.
- `X-Correlation-ID` harus ada pada setiap respons (observability baseline).
- Error mapping: sentinel → HTTP codes (`401`,`400`,`429`,`500`).

## Data & Test Fixtures
- Buat user test dengan password hash valid.
- Buat device_id untuk variasi multi-perangkat.
- Gunakan testdata standar di `test/testdata/` (nama folder sesuai base).

## Pelaporan
- Jalankan `go test ./... -race -cover` dan catat ringkasan.
- Laporkan coverage per paket untuk domain & application.