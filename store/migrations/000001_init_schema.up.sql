-- IPAM schema: environments, blocks, allocations, reserved_blocks, users, sessions, api_tokens

CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS blocks (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    cidr TEXT NOT NULL,
    environment_id UUID REFERENCES environments(id) ON DELETE SET NULL,
    total_ips BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS allocations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    block_name TEXT NOT NULL,
    block_cidr TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reserved_blocks (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    cidr TEXT NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    tour_completed BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_blocks_environment_id ON blocks(environment_id);
CREATE INDEX IF NOT EXISTS idx_allocations_block_name ON allocations(block_name);
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry);
CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
