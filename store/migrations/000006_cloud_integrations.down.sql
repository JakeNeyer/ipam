-- Revert cloud integrations schema changes (including soft-delete columns).

-- Allocations: soft delete
DROP INDEX IF EXISTS idx_allocations_deleted_at;
ALTER TABLE allocations DROP COLUMN IF EXISTS deleted_at;

-- Allocations: provider/external_id/connection_id
DROP INDEX IF EXISTS idx_allocations_connection_external_id;
DROP INDEX IF EXISTS idx_allocations_connection_id;
ALTER TABLE allocations DROP COLUMN IF EXISTS connection_id;
ALTER TABLE allocations DROP COLUMN IF EXISTS external_id;
ALTER TABLE allocations DROP COLUMN IF EXISTS provider;

-- Blocks: soft delete
DROP INDEX IF EXISTS idx_blocks_deleted_at;
ALTER TABLE blocks DROP COLUMN IF EXISTS deleted_at;

-- Blocks: provider/external_id/connection_id
DROP INDEX IF EXISTS idx_blocks_connection_id;
DROP INDEX IF EXISTS idx_blocks_connection_external_id;
ALTER TABLE blocks DROP COLUMN IF EXISTS connection_id;
ALTER TABLE blocks DROP COLUMN IF EXISTS external_id;
ALTER TABLE blocks DROP COLUMN IF EXISTS provider;

-- Pools: soft delete
DROP INDEX IF EXISTS idx_pools_deleted_at;
ALTER TABLE pools DROP COLUMN IF EXISTS deleted_at;

-- Pools: provider/external_id/connection_id/parent_pool_id
DROP INDEX IF EXISTS idx_pools_parent_pool_id;
DROP INDEX IF EXISTS idx_pools_connection_id;
DROP INDEX IF EXISTS idx_pools_connection_external_id;
ALTER TABLE pools DROP COLUMN IF EXISTS parent_pool_id;
ALTER TABLE pools DROP COLUMN IF EXISTS connection_id;
ALTER TABLE pools DROP COLUMN IF EXISTS external_id;
ALTER TABLE pools DROP COLUMN IF EXISTS provider;

-- Cloud connections
ALTER TABLE cloud_connections DROP CONSTRAINT IF EXISTS chk_conflict_resolution;
ALTER TABLE cloud_connections DROP CONSTRAINT IF EXISTS chk_sync_mode;
ALTER TABLE cloud_connections DROP CONSTRAINT IF EXISTS chk_sync_interval_minutes;
DROP INDEX IF EXISTS idx_cloud_connections_provider;
DROP INDEX IF EXISTS idx_cloud_connections_organization_id;
DROP TABLE IF EXISTS cloud_connections;
