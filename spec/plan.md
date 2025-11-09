# /speckit.plan — Technical Implementation Plan (Go 1.24)

Tujuan
- Membuat rencana teknis implementasi sesuai stack yang dipilih.

Arsitektur & Struktur
- Pola: DDD + Clean Architecture; ports & adapters.
- Struktur proyek (diselaraskan dengan base):
```
your-project/
  api/
  build/
  cmd/app/
  docs/
  domain/
    aggregate/
    enum/
    entity/
    event/
    valueobject/
    repository/
    service/
  application/
    usecase/
    repository/
  infrastructure/
    repository/
    service/
  delivery/
    http/
    grpc/
  migrations/
  pkg/
    middleware/
    errorcodes/
  proto/
  test/
    testdata/
    infrastructure/
  web/
  go.mod
```

Tooling & Dependensi
- Go toolchain: 1.24; `go mod init <module>`.
- Linter/formatter: `golangci-lint`, `go fmt`.
- Testing: standar `testing`, table-driven (boleh `testify` bila perlu).
- DB & migrasi: PostgreSQL 16; tool migrasi: golang-migrate.
  - Direktori migrasi: `migrations` dengan format `YYYYMMDDHHMMSS_<name>.sql`.
  - Koneksi: `database/sql` + driver `pgx`.
  - Env (diselaraskan): `DATABASE_URL`, `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`, `JWT_ALG`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` (RS/ES), `JWT_SECRET` (HS dev), `REFRESH_TOKEN_SALT`, `RECAPTCHA_ENABLED`, `RECAPTCHA_SKIP`, `RECAPTCHA_SECRET`, `PORT`.
  - Strategi deploy: migrasi dijalankan sebelum start aplikasi; backward-safe bila memungkinkan.
- Messaging: <kafka/nats/redis> (opsional).

Data & Transaksi
- Skema awal (Postgres):
  - `users` (id uuid pk, email citext unique, password_hash text not null, status varchar(16) default 'active', created_at timestamptz default now()).
  - `refresh_tokens` (id uuid pk, user_id uuid fk users(id) on delete cascade, token_hash bytea unique not null, device_id text null, issued_at timestamptz default now(), expires_at timestamptz not null, revoked boolean default false, revoked_at timestamptz null, reason text null).
  - Opsional `sessions` bila perlu agregasi perangkat.
- Kunci & indeks:
  - unique(email), unique(token_hash), idx(refresh_tokens.user_id), idx(refresh_tokens.expires_at), idx(refresh_tokens.revoked).
- Isolasi transaksi: Read Committed; operasi single-row (insert/update/select) tanpa long transactions.
- Idempotensi:
  - Login: insert refresh token baru; jika duplikasi token_hash hampir mustahil (random + salt).
  - Logout: `UPDATE ... SET revoked=true` aman idempotent.
  - Refresh: lookup by token_hash; tidak mengubah refresh token (atau rotasi di iterasi lanjut).

Observability
- Logging (adapter): JSON, korelasi request id.
- Metrics: <daftar>; tracing: <alat/standar>.

Security
- AuthN/AuthZ: JWT Bearer; refresh token via cookie HttpOnly (SameSite=Lax) atau header `Refresh-Token` untuk endpoint `/v1/auth/refresh`; roles/scopes per endpoint; Remember Me memperpanjang TTL refresh.
- Secrets & Algoritma: pilih `JWT_ALG` (`RS256/ES256` produksi dengan `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY`; `HS256` dev dengan `JWT_SECRET`); `RECAPTCHA_SECRET` via env/Vault; rotasi berkala.
- Validasi input: email/password (format & policy), `recaptcha_token` (bila reCAPTCHA enabled); throttle percobaan login.
- Logout: menggunakan JWT Bearer (`Authorization` header) dan menghapus cookie `refresh_token` pada respons 204 (Max-Age=0).

CI/CD
- Pipeline: lint → unit → integration → build → image → deploy.
- Artifacts: laporan coverage, SBOM, dok API.

Rencana Iterasi
- Iterasi 1: baseline + domain Auth (entities/VO) + Login use case + TokenService + RecaptchaVerifier mock + unit tests.
- Iterasi 2: adapter nyata (PasswordHasher bcrypt/argon2, JWT issuer, persistence RefreshToken) + endpoints REST (login/refresh/logout/me) + Remember Me TTL.
- Iterasi 3: integrasi reCAPTCHA eksternal (`siteverify`) + rate limiting/throttling + observability (log/traces) + lengkapi error katalog & acceptance tests.

Risiko & Mitigasi
- Risiko: <daftar>; probabilitas/dampak: <isi>; mitigasi: <isi>.

Kriteria Sukses
- Semua quality gates hijau; feature berjalan sesuai acceptance criteria.