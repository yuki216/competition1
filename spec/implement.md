# /speckit.implement — Execute All Tasks According to the Plan

## Daftar Isi
- Tujuan
- Persiapan
- Log Eksekusi (Harian)
- Metodologi TDD
- Implementasi Auth
- Validasi Akhir
- Release Notes
- Retrospective
- Adapter Postgres — RefreshTokenRepository
- Error Handling & Mapping
- Checklist Implementasi

## Tujuan
- Mengeksekusi seluruh tasks untuk membangun fitur sesuai plan.

## Persiapan
- Prasyarat terpenuhi (DoR): Go 1.24, Postgres 16, CLI migrasi (opsional).
- Rencana teknis final: merujuk ke `spec/plan.md`, `spec/constitution.md`, dan `spec/specify.md`.
- Struktur folder: mengikuti base di `spec/constitution.md` (gunakan `migrations/` untuk SQL).
- Variabel lingkungan: gunakan daftar diselaraskan (`DATABASE_URL`, `JWT_ALG`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` atau `JWT_SECRET`, `REFRESH_TOKEN_SALT`, `RECAPTCHA_ENABLED`, `RECAPTCHA_SKIP`, `RECAPTCHA_SECRET`, `PORT`).

## Log Eksekusi (Harian)
- Tanggal: <YYYY-MM-DD>
- Pekerjaan: <isi>
- Test yang ditulis: <daftar>
- Hasil `go test ./... -race -cover`: <ringkas>
- Catatan refactor: <isi>
- Masalah & solusi: <isi>

## Metodologi TDD
1) Tulis test yang merepresentasikan perilaku dan error.
2) Implementasi minimal hingga test hijau.
3) Refactor, dorong aturan ke domain, jaga test hijau.
4) Tambah integration test untuk adapter penting (DB/HTTP/message).
5) Dokumentasi & quality gates; pastikan coverage domain/application ≥ 80%.

## Implementasi Auth (Login/Remember Me/reCAPTCHA)
- Unit: `LoginUseCase` — skenario reCAPTCHA enabled vs disabled/skip; kredensial valid/invalid; `remember_me` true/false; verifikasi hasil TTL.
- Integration: adapter `RecaptchaVerifier` (mock/real), `JWT` issuer, `RefreshTokenRepository`; verifikasi cookie HttpOnly+SameSite=Lax di layer REST; endpoint `/v1/auth/refresh` menerima header `Refresh-Token` sebagai alternatif; endpoint `/v1/auth/logout` memakai JWT Bearer dan menghapus cookie.
- Negative tests: `recaptcha_invalid`, `recaptcha_required`, `token_revoked/expired`; throttling percobaan.
- Observability: periksa `X-Correlation-ID` mengalir ke log; event `UserLoggedIn/LoggedOut` tercatat.

## Validasi Akhir
- [ ] Semua acceptance criteria terpenuhi
- [ ] Semua quality gates hijau
- [ ] Observability baseline aktif
- [ ] Error katalog diperbarui
- [ ] ADR ditulis bila ada keputusan arsitektur
- [ ] reCAPTCHA toggle (enabled/skip) diverifikasi di e2e

## Release Notes
- Versi: <x.y.z>
- Perubahan utama: <isi>
- Migrasi/kompatibilitas: <isi>

## Retrospective
- Apa yang berjalan baik: <isi>
- Apa yang perlu ditingkatkan: <isi>
- Aksi perbaikan: <isi>

## Adapter Postgres — RefreshTokenRepository

### Migrasi
- Buat tabel `refresh_tokens` (lihat `plan.md`) dan pastikan indeks tersedia.
- Indeks disarankan: `unique(token_hash)`, `idx(user_id)`, `idx(expires_at)`, `idx(revoked)`.

### Implementasi
- Simpan hanya hash dari refresh token (`sha256(token_plain + SALT)`), bukan plaintext.
- Env: `DATABASE_URL`, `REFRESH_TOKEN_SALT`.

### Operasi
- Create: `INSERT INTO refresh_tokens(user_id, token_hash, expires_at, device_id) VALUES ($1,$2,$3,$4)`
- Get: `SELECT id, user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash=$1`
- Revoke: `UPDATE refresh_tokens SET revoked=true, revoked_at=now() WHERE token_hash=$1`

### Integration Tests
- Create→Get: menyimpan token dan membaca kembali; verifikasi expiry dan `user_id`.
- Revoke: idempotensi; revoke dua kali tetap konsisten.
- Expired: data yang expired ditolak pada use case refresh.

### Observability
- Log terstruktur query error dengan `X-Correlation-ID`.

### Keamanan
- Token disimpan hashed (bytea), lindungi dari kebocoran; gunakan parameterized queries.
- Cookie: `HttpOnly`, `Secure`, `SameSite=Lax`; domain/path disesuaikan kebutuhan.

## Error Handling & Mapping
- Gunakan katalog error dari `spec/specify.md` untuk pemetaan REST.
- Prinsip:
  - Domain/application memakai error sentinel; adapter membungkus dengan konteks.
  - Mapping contoh: `ErrUnauthorized`→`401`, `ErrBadRequest`→`400`, `ErrRateLimited`→`429`, `ErrInternal`→`500`.
- Pastikan envelope respons mengikuti standar di `spec/api.md` (`status/message/data`).

## Checklist Implementasi
- [ ] Struktur proyek mengikuti `spec/constitution.md` (domain/application/delivery/infrastructure).
- [ ] Variabel lingkungan diset dan tervalidasi di startup.
- [ ] Migrasi `refresh_tokens` berjalan up/down; indeks sesuai rencana.
- [ ] Adapter `RefreshTokenRepository` mendukung create/get/revoke dan idempotensi.
- [ ] Login flow: reCAPTCHA (toggle enabled/skip) diuji, JWT issuer bekerja, cookie aman.
- [ ] Integration tests untuk DB dan handler REST disiapkan (happy/error paths).
- [ ] Logging terstruktur dengan `X-Correlation-ID` konsisten.
- [ ] Release notes dan retro diperbarui sebelum rilis.