-- Cloud provider integrations: cloud_connections table and provider/external_id/connection_id on pools, blocks, allocations.
-- Includes sync_interval_minutes, sync_mode, and conflict_resolution on cloud_connections.
-- Also adds soft-delete (deleted_at) for pools, blocks, and allocations for IPAM conflict resolution.

-- 1. Cloud connections (per-organization link to AWS/Azure/GCP)
CREATE TABLE IF NOT EXISTS cloud_connections (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('aws', 'azure', 'gcp')),
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    credentials_ref TEXT,
    last_sync_at TIMESTAMPTZ,
    last_sync_status TEXT,
    last_sync_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cloud_connections_organization_id ON cloud_connections(organization_id);
CREATE INDEX IF NOT EXISTS idx_cloud_connections_provider ON cloud_connections(provider);

-- Sync interval, mode, and conflict resolution (idempotent: no-op if already applied)
ALTER TABLE cloud_connections ADD COLUMN IF NOT EXISTS sync_interval_minutes INTEGER NOT NULL DEFAULT 5;
ALTER TABLE cloud_connections DROP CONSTRAINT IF EXISTS chk_sync_interval_minutes;
ALTER TABLE cloud_connections ADD CONSTRAINT chk_sync_interval_minutes CHECK (sync_interval_minutes >= 0 AND sync_interval_minutes <= 1440);

ALTER TABLE cloud_connections ADD COLUMN IF NOT EXISTS sync_mode TEXT NOT NULL DEFAULT 'read_only';
ALTER TABLE cloud_connections DROP CONSTRAINT IF EXISTS chk_sync_mode;
ALTER TABLE cloud_connections ADD CONSTRAINT chk_sync_mode CHECK (sync_mode IN ('read_only', 'read_write'));

ALTER TABLE cloud_connections ADD COLUMN IF NOT EXISTS conflict_resolution TEXT NOT NULL DEFAULT 'cloud';
UPDATE cloud_connections SET conflict_resolution = 'cloud' WHERE conflict_resolution IN ('last-write-wins', 'integration', 'aws');
UPDATE cloud_connections SET conflict_resolution = 'ipam' WHERE conflict_resolution = 'app';
ALTER TABLE cloud_connections ALTER COLUMN conflict_resolution SET DEFAULT 'cloud';
ALTER TABLE cloud_connections DROP CONSTRAINT IF EXISTS chk_conflict_resolution;
ALTER TABLE cloud_connections ADD CONSTRAINT chk_conflict_resolution CHECK (conflict_resolution IN ('cloud', 'ipam'));

-- 2. Pools: provider, external_id, connection_id, parent_pool_id (for sub-pools)
ALTER TABLE pools ADD COLUMN IF NOT EXISTS provider TEXT DEFAULT 'native';
ALTER TABLE pools ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE pools ADD COLUMN IF NOT EXISTS connection_id UUID REFERENCES cloud_connections(id) ON DELETE SET NULL;
ALTER TABLE pools ADD COLUMN IF NOT EXISTS parent_pool_id UUID REFERENCES pools(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_pools_connection_external_id ON pools(connection_id, external_id) WHERE external_id IS NOT NULL AND connection_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pools_connection_id ON pools(connection_id);
CREATE INDEX IF NOT EXISTS idx_pools_parent_pool_id ON pools(parent_pool_id);

-- Pools: soft delete (deleted_at) for IPAM conflict resolution
ALTER TABLE pools ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_pools_deleted_at ON pools(deleted_at) WHERE deleted_at IS NOT NULL;

-- 3. Blocks: provider, external_id, connection_id (for cloud-synced blocks e.g. VPCs)
ALTER TABLE blocks ADD COLUMN IF NOT EXISTS provider TEXT DEFAULT 'native';
ALTER TABLE blocks ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE blocks ADD COLUMN IF NOT EXISTS connection_id UUID REFERENCES cloud_connections(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_blocks_connection_external_id ON blocks(connection_id, external_id) WHERE external_id IS NOT NULL AND connection_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_blocks_connection_id ON blocks(connection_id);

-- Blocks: soft delete (deleted_at) for IPAM conflict resolution
ALTER TABLE blocks ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_blocks_deleted_at ON blocks(deleted_at) WHERE deleted_at IS NOT NULL;

-- 4. Allocations: provider, external_id, connection_id (for cloud-synced allocations e.g. AWS subnets)
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS provider TEXT DEFAULT 'native';
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS connection_id UUID REFERENCES cloud_connections(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_allocations_connection_id ON allocations(connection_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_allocations_connection_external_id ON allocations(connection_id, external_id) WHERE external_id IS NOT NULL AND connection_id IS NOT NULL;

-- Allocations: soft delete (deleted_at) for IPAM conflict resolution
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_allocations_deleted_at ON allocations(deleted_at) WHERE deleted_at IS NOT NULL;
