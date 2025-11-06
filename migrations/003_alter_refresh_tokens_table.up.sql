-- Alter refresh_tokens: replace token with token_hash BYTEA and add revoked boolean
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS token_hash BYTEA;

-- Migrate existing hex-encoded hashes from token (VARCHAR) to token_hash (BYTEA)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_name = 'refresh_tokens' AND column_name = 'token'
  ) THEN
    UPDATE refresh_tokens SET token_hash = decode(token, 'hex') WHERE token_hash IS NULL;
  END IF;
END $$;

-- Drop old token column if exists
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns 
    WHERE table_name = 'refresh_tokens' AND column_name = 'token'
  ) THEN
    ALTER TABLE refresh_tokens DROP COLUMN token;
  END IF;
END $$;

-- Add revoked boolean flag
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS revoked BOOLEAN DEFAULT FALSE;

-- Ensure constraints and indexes
DO $$
BEGIN
  -- Unique index on token_hash
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_refresh_tokens_token_hash'
  ) THEN
    CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
  END IF;
  
  -- Index on revoked
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_refresh_tokens_revoked'
  ) THEN
    CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens(revoked);
  END IF;
END $$;