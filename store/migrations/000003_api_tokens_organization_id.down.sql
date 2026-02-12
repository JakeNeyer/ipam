DROP INDEX IF EXISTS idx_api_tokens_organization_id;
ALTER TABLE api_tokens DROP COLUMN IF EXISTS organization_id;
