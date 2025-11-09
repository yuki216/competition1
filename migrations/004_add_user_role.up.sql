-- Add role column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'employee';

-- Create index for role column
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Update existing users to have 'employee' role by default
UPDATE users SET role = 'employee' WHERE role IS NULL OR role = '';

-- Add check constraint for valid roles
ALTER TABLE users ADD CONSTRAINT chk_user_role
  CHECK (role IN ('admin', 'employee'));