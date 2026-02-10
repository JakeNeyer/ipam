-- Drop in reverse dependency order

DROP INDEX IF EXISTS idx_api_tokens_user_id;
DROP INDEX IF EXISTS idx_sessions_expiry;
DROP INDEX IF EXISTS idx_users_email_lower;
DROP INDEX IF EXISTS idx_allocations_block_name;
DROP INDEX IF EXISTS idx_blocks_environment_id;

DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS reserved_blocks;
DROP TABLE IF EXISTS allocations;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS environments;
