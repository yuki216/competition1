# /speckit.tasks — Actionable Task Lists

Tujuan
- Mengubah rencana menjadi daftar pekerjaan yang dapat dieksekusi.

Backlog (Epics)
- Epic Auth: Login + Remember Me + reCAPTCHA (toggle)
  - Deskripsi: Membangun autentikasi berbasis JWT dengan refresh token; reCAPTCHA dapat diaktifkan/di-skip; Remember Me memperpanjang TTL.
  - Tujuan: Login aman, friksi adaptif terhadap risiko bot; akses token terisolasi.
  - Outcome: Endpoint login/refresh/logout/me berjalan; acceptance criteria hijau; observability aktif.

Stories
- ID: AUTH-01
  - Judul: Login tanpa reCAPTCHA (disabled/skip) dengan dukungan Remember Me
  - Acceptance Criteria:
    - Given reCAPTCHA disabled/skip, When login valid & remember_me=false, Then 200 dan refresh cookie TTL pendek.
    - Given reCAPTCHA disabled/skip, When login valid & remember_me=true, Then 200 dan refresh cookie TTL panjang.
  - Impact: tinggi
  - Dependencies: UserRepository, PasswordHasher, TokenService
  - Estimasi: 3 hari
  - Owner: <isi>

- ID: AUTH-02
  - Judul: Login dengan reCAPTCHA enabled (verifikasi siteverify)
  - Acceptance Criteria:
    - Given reCAPTCHA enabled, When token valid, Then 200 dan login berhasil; ada call `siteverify`.
    - Given reCAPTCHA enabled, When token invalid/missing, Then 400/422 dengan kode error sesuai.
  - Impact: sedang
  - Dependencies: RecaptchaVerifier port + konfigurasi `RECAPTCHA_SECRET`
  - Estimasi: 2 hari
  - Owner: <isi>

- ID: AUTH-03
  - Judul: Refresh access token berbasis refresh cookie
  - Acceptance Criteria:
    - Given refresh token valid, When call, Then 200 dan akses baru.
    - Given token revoked/expired, When call, Then 401 dengan kode error.
  - Impact: sedang
  - Dependencies: RefreshTokenRepository, TokenService
  - Estimasi: 2 hari
  - Owner: <isi>

- ID: AUTH-04
  - Judul: Logout (revoke semua refresh token via access token) + hapus cookie
  - Acceptance Criteria:
    - Given access token valid, When logout, Then 204 dan semua refresh token user direvoke & cookie dihapus.
  - Impact: rendah
  - Dependencies: RefreshTokenRepository, TokenService
  - Estimasi: 1 hari
  - Owner: <isi>

Langkah Eksekusi (TDD)
- [ ] Tulis test gagal pertama (unit di domain/use case)
- [ ] Implementasi minimal hingga test hijau
- [ ] Refactor menjaga hijau (ekstrak VO/push rule ke domain)
- [ ] Tambah edge cases dan negative tests
- [ ] Integration test adapter (DB/HTTP/message)
- [ ] Dokumentasi API & ADR singkat

Checklist Quality
- [ ] `golangci-lint` clean
- [ ] `go test ./... -race -cover` hijau
- [ ] Coverage domain/application ≥ 80%
- [ ] Error katalog diperbarui
- [ ] Observability baseline (log/metrics/traces) aktif

Timeline
- Iterasi/ Sprint: <rentang tanggal>
- Milestone: <daftar>

Risiko Operasional
- <daftar risiko>, mitigasi: <isi>