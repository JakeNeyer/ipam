-- Environment pools: a range of CIDRs that network blocks in an environment can draw from.
-- Hierarchy: Environment -> Pool(s) -> Network blocks -> Allocations

CREATE TABLE IF NOT EXISTS pools (
    id UUID PRIMARY KEY,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cidr TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pools_environment_id ON pools(environment_id);
CREATE INDEX IF NOT EXISTS idx_pools_organization_id ON pools(organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pools_env_name ON pools(environment_id, LOWER(TRIM(name)));

ALTER TABLE blocks ADD COLUMN IF NOT EXISTS pool_id UUID REFERENCES pools(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_blocks_pool_id ON blocks(pool_id);
