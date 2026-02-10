-- Signup invites: time-bound tokens for admin-created signup links

CREATE TABLE IF NOT EXISTS signup_invites (
    id UUID PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    used_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_signup_invites_token_hash ON signup_invites(token_hash);
CREATE INDEX IF NOT EXISTS idx_signup_invites_expires_at ON signup_invites(expires_at);
CREATE INDEX IF NOT EXISTS idx_signup_invites_created_by ON signup_invites(created_by);
CREATE INDEX IF NOT EXISTS idx_signup_invites_used_at ON signup_invites(used_at);
