# Artefak Teknis — Migrasi & Ports

## Migrasi `refresh_tokens` (Sketsa)

### Up
```sql
-- Tabel refresh_tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID NOT NULL,
  token_hash BYTEA NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  device_id TEXT,
  revoked BOOLEAN NOT NULL DEFAULT FALSE,
  revoked_at TIMESTAMPTZ
);

-- Indeks
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked ON refresh_tokens(revoked);
```

### Down
```sql
DROP TABLE IF EXISTS refresh_tokens;
```

### Catatan
- Simpan hanya hash: `sha256(plain + REFRESH_TOKEN_SALT)`.
- Jalankan migrasi sebelum start aplikasi; backward-safe bila memungkinkan.
- Koneksi via `database/sql` + driver `pgx`; env `DATABASE_URL`.

## Query Operasi (Referensi)
- Create: `INSERT INTO refresh_tokens(user_id, token_hash, expires_at, device_id) VALUES ($1,$2,$3,$4)`
- Get: `SELECT id, user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash=$1`
- Revoke: `UPDATE refresh_tokens SET revoked=true, revoked_at=now() WHERE token_hash=$1`

## Ports (Kontrak Aplikasi)

### UserRepository
```go
// Cari user dan verifikasi password (hashing ada di adapter)
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (User, error)
    VerifyPassword(ctx context.Context, user User, password string) error
}
```

### RefreshTokenRepository
```go
type RefreshTokenRepository interface {
    Create(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time, deviceID string) error
    GetByHash(ctx context.Context, tokenHash []byte) (RefreshToken, error)
    RevokeByHash(ctx context.Context, tokenHash []byte) error
}
```

### TokenService (JWT)
```go
type TokenService interface {
    IssueAccessToken(ctx context.Context, userID string, email string) (token string, expiresAt time.Time, err error)
    ParseAccessToken(ctx context.Context, token string) (Claims, error)
}
```

### RecaptchaVerifier
```go
type RecaptchaVerifier interface {
    Verify(ctx context.Context, token string) (bool, error)
}
```

### EventPublisher (opsional)
```go
type EventPublisher interface {
    Publish(ctx context.Context, eventName string, payload map[string]any) error
}
```