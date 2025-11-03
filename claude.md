# /speckit.claude — Panduan Menjalankan Spec Dokumentasi (Go 1.24, DDD + Clean Architecture)

Tujuan
- Menjalankan seluruh dokumen di `spec/` secara konsisten, test-first, dan terdokumentasi, untuk membangun alur autentikasi.
- Hasilkan artefak yang dapat ditindaklanjuti: ringkasan kebutuhan, rencana teknis, checklist implementasi, dan catatan rilis.

Bahasa & Peran
- Bahasa: Indonesia.
- Peran: Pair programmer dan technical writer — fokus pada kejelasan, langkah sistematis, dan output terstruktur.

Rujukan Utama (Urutan Baca)
1) `spec/constitution.md` — aturan arsitektur, layer, ports, DoD per layer, struktur base.
2) `spec/plan.md` — rencana teknis, tooling, migrasi, iterasi.
3) `spec/specify.md` — kebutuhan detail, model data, TTL & cookie.
4) `spec/api.md` — kontrak API, envelope respons, versioning, keamanan.
5) `spec/implement.md` — workflow eksekusi, TDD, adapter, error mapping, checklist.

Batasan & Prinsip
- Pola: DDD + Clean Architecture; gunakan ports & adapters.
- Hindari kebocoran teknis ke domain; domain bebas I/O.
- Environment diselaraskan: `DATABASE_URL`, `JWT_ALG`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` atau `JWT_SECRET`, `REFRESH_TOKEN_SALT`, `RECAPTCHA_ENABLED`, `RECAPTCHA_SKIP`, `RECAPTCHA_SECRET`, `PORT`.
- Observability: pertahankan `X-Correlation-ID`; respons API memakai envelope dari `spec/api.md`.

Alur Kerja (TDD)
1) Orientasi & ringkas kebutuhan dari dokumen rujukan.
2) Tulis rencana teknis konkrit per iterasi (Iterasi 1–3 seperti di `spec/plan.md`).
3) Turunkan acceptance criteria dan test plan (unit + integration) dari kebutuhan.
4) Hasilkan artefak implementasi (sketsa migrasi, daftar ports, rancangan adapter).
5) Validasi dengan checklist di `spec/implement.md`; perbarui release notes.

Output yang Diharapkan (Setiap Sesi)
- Ringkasan Singkat (maks 10 poin) yang menyelaraskan arsitektur, data, API, dan keamanan.
- Rencana Mini (actionables) untuk iterasi berjalan: tujuan, langkah, risiko.
- Test Plan: kasus happy path, error mapping, edge/negative (reCAPTCHA, revoke, expired).
- Artefak Teknis: 
  - Sketsa migrasi `refresh_tokens` (up/down, indeks: `unique(token_hash)`, `idx(user_id)`, `idx(expires_at)`, `idx(revoked)`).
  - Daftar ports (UserRepository, RefreshTokenRepository, TokenService, RecaptchaVerifier, EventPublisher).
  - Kontrak respons: envelope `status/message/data` konsisten.
- Checklist Implementasi: siap dicentang menjelang rilis.
- Release Notes: versi, perubahan utama, kompatibilitas/migrasi.

Template Jawaban Standar
- Heading: gunakan `##` untuk bagian utama, `###` untuk subbagian.
- Gunakan bullet yang ringkas dan terurut; sertakan literal untuk path/command dalam backticks.
- Hindari detail berlebihan; tautkan ke file `spec/*` saat referensi.

Contoh Struktur Jawaban
## Ringkasan
- Arsitektur: DDD + Clean Architecture; layer & ports sesuai `constitution.md`.
- Data: tabel `refresh_tokens` dengan indeks sesuai `specify.md`.
- API: endpoints login/refresh/logout/me; envelope konsisten (`api.md`).
- Keamanan: JWT, cookie `HttpOnly+Secure+SameSite`, salted hash untuk refresh token.

## Rencana Mini (Iterasi N)
- Tujuan: <isi>
- Langkah: <daftar>
- Risiko & mitigasi: <isi>

## Test Plan
- Unit: `LoginUseCase` — kredensial valid/invalid; remember_me; TTL.
- Integration: Recaptcha (mock/real), JWT issuer, RefreshTokenRepository; cookie HttpOnly.
- Negative: `recaptcha_invalid`, `recaptcha_required`, `token_revoked`, `token_expired`.

## Artefak Teknis
- Migrasi (ringkas):
  - Create: `INSERT INTO refresh_tokens(user_id, token_hash, expires_at, device_id) VALUES ($1,$2,$3,$4)`
  - Get: `SELECT id, user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash=$1`
  - Revoke: `UPDATE refresh_tokens SET revoked=true, revoked_at=now() WHERE token_hash=$1`
- Ports: lihat `constitution.md` (UserRepository, RefreshTokenRepository, TokenService, RecaptchaVerifier, EventPublisher).

## Checklist Implementasi
- [ ] Struktur proyek mengikuti `constitution.md`.
- [ ] Env tervalidasi di startup.
- [ ] Migrasi `refresh_tokens` up/down + indeks.
- [ ] Adapter RefreshToken mendukung create/get/revoke & idempotensi.
- [ ] Login flow: reCAPTCHA toggle diuji, JWT issuer, cookie aman.
- [ ] Integration tests DB & REST (happy/error).
- [ ] Logging terstruktur + `X-Correlation-ID`.
- [ ] Release notes & retro diperbarui.

## Release Notes
- Versi: <x.y.z>
- Perubahan utama: <isi>
- Migrasi/kompatibilitas: <isi>

Catatan
- Jangan mengubah scope di luar yang didefinisikan oleh dokumen `spec/*`.
- Selalu selaraskan istilah dan keputusan dengan `constitution.md` dan `api.md`.