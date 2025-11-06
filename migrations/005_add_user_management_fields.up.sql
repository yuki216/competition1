-- Add name, status, and deleted_at columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Update existing users to have default values
UPDATE users SET
  name = COALESCE(name, ''),
  status = COALESCE(status, 'active')
WHERE name = '' OR status = '' OR name IS NULL OR status IS NULL;

-- Add check constraint for valid statuses
ALTER TABLE users ADD CONSTRAINT chk_user_status
  CHECK (status IN ('active', 'inactive'));

-- Update role constraint to include 'user' and 'superadmin' as specified in spec
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_user_role;
ALTER TABLE users ADD CONSTRAINT chk_user_role
  CHECK (role IN ('admin', 'user', 'superadmin', 'employee'));