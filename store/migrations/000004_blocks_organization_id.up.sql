-- Orphan blocks (environment_id IS NULL) are scoped to an organization.
ALTER TABLE blocks ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_blocks_organization_id ON blocks(organization_id);
