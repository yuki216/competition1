-- Remove check constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_user_role;

-- Remove index
DROP INDEX IF EXISTS idx_users_role;

-- Remove role column
ALTER TABLE users DROP COLUMN IF EXISTS role;