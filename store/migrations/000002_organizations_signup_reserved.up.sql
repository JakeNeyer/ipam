-- Organizations, signup invites, and reserved blocks organization scoping (consolidated).

-- 1. Organizations (multi-tenancy; users and environments belong to an organization)
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id);
ALTER TABLE environments ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id);

INSERT INTO organizations (id, name, created_at)
SELECT '00000000-0000-0000-0000-000000000001'::uuid, 'Default', NOW()
WHERE EXISTS (SELECT 1 FROM users LIMIT 1)
ON CONFLICT (id) DO NOTHING;

UPDATE environments SET organization_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE organization_id IS NULL
  AND EXISTS (SELECT 1 FROM organizations WHERE id = '00000000-0000-0000-0000-000000000001'::uuid);

UPDATE users SET organization_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE organization_id IS NULL
  AND EXISTS (SELECT 1 FROM organizations WHERE id = '00000000-0000-0000-0000-000000000001'::uuid);

CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_environments_organization_id ON environments(organization_id);

-- 2. Signup invites (time-bound tokens for admin-created signup links; org and role from the start)
CREATE TABLE IF NOT EXISTS signup_invites (
    id UUID PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    used_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id),
    role TEXT NOT NULL DEFAULT 'user'
);

CREATE INDEX IF NOT EXISTS idx_signup_invites_token_hash ON signup_invites(token_hash);
CREATE INDEX IF NOT EXISTS idx_signup_invites_expires_at ON signup_invites(expires_at);
CREATE INDEX IF NOT EXISTS idx_signup_invites_created_by ON signup_invites(created_by);
CREATE INDEX IF NOT EXISTS idx_signup_invites_used_at ON signup_invites(used_at);

-- 3. Reserved blocks scoped to organization
ALTER TABLE reserved_blocks ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id);

UPDATE reserved_blocks SET organization_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE organization_id IS NULL
  AND EXISTS (SELECT 1 FROM organizations WHERE id = '00000000-0000-0000-0000-000000000001'::uuid);

ALTER TABLE reserved_blocks ALTER COLUMN organization_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_reserved_blocks_organization_id ON reserved_blocks(organization_id);

-- 4. OAuth: generic provider link (github, future: google, etc.). Password_hash may be empty for OAuth-only users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_provider TEXT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_provider_user_id TEXT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oauth_provider_uid ON users(oauth_provider, oauth_provider_user_id) WHERE oauth_provider IS NOT NULL AND oauth_provider_user_id IS NOT NULL;
