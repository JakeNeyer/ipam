-- API tokens: optional organization_id for global-admin tokens scoped to one org.
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id);
CREATE INDEX IF NOT EXISTS idx_api_tokens_organization_id ON api_tokens(organization_id);
