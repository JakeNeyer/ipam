-- Reverse organizations, signup invites, reserved blocks, and OAuth columns.

DROP INDEX IF EXISTS idx_users_oauth_provider_uid;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider;

DROP INDEX IF EXISTS idx_reserved_blocks_organization_id;
ALTER TABLE reserved_blocks DROP COLUMN IF EXISTS organization_id;

DROP INDEX IF EXISTS idx_signup_invites_used_at;
DROP INDEX IF EXISTS idx_signup_invites_created_by;
DROP INDEX IF EXISTS idx_signup_invites_expires_at;
DROP INDEX IF EXISTS idx_signup_invites_token_hash;
DROP TABLE IF EXISTS signup_invites;

DROP INDEX IF EXISTS idx_environments_organization_id;
DROP INDEX IF EXISTS idx_users_organization_id;
ALTER TABLE environments DROP COLUMN IF EXISTS organization_id;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;

DROP TABLE IF EXISTS organizations;
