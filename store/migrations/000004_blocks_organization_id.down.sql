DROP INDEX IF EXISTS idx_blocks_organization_id;
ALTER TABLE blocks DROP COLUMN IF EXISTS organization_id;
