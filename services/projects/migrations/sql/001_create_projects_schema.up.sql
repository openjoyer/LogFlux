CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_valid CHECK (
        char_length(email) BETWEEN 1 AND 254 AND email = lower(email)
    ),
    CONSTRAINT users_password_hash_not_empty CHECK (password_hash <> '')
);

CREATE TABLE user_sessions (
    token_hash BYTEA PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX user_sessions_expires_at_idx ON user_sessions (expires_at);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT teams_name_valid CHECK (
        char_length(name) <= 255 AND btrim(name) <> ''
    )
);

CREATE TABLE team_memberships (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX team_memberships_user_id_idx ON team_memberships (user_id);

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT projects_name_valid CHECK (
        char_length(name) <= 255 AND btrim(name) <> ''
    )
);

CREATE INDEX projects_team_created_idx
    ON projects (team_id, created_at DESC, id DESC);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash BYTEA NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT api_keys_name_valid CHECK (
        char_length(name) <= 255 AND btrim(name) <> ''
    ),
    CONSTRAINT api_keys_key_hash_size CHECK (octet_length(key_hash) = 32),
    CONSTRAINT api_keys_scopes_not_empty CHECK (cardinality(scopes) > 0)
);

CREATE INDEX api_keys_project_created_idx
    ON api_keys (project_id, created_at DESC, id DESC);
