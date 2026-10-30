-- +goose Up
-- Identidad y organizaciones (ADR-0003, ADR-0004).
-- Los usuarios son identidades globales; la pertenencia a un tenant se expresa
-- con memberships. Toda tabla de dominio futura llevará tenant_id.

CREATE TABLE tenants (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        text        NOT NULL UNIQUE,
    name        text        NOT NULL,
    kind        text        NOT NULL CHECK (kind IN ('personal', 'organization')),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email             text        NOT NULL,
    email_normalized  text        NOT NULL,
    display_name      text        NOT NULL,
    password_hash     text        NOT NULL,
    status            text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_normalized_key UNIQUE (email_normalized)
);

CREATE TABLE memberships (
    tenant_id   uuid        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role        text        NOT NULL CHECK (role IN ('student', 'instructor', 'admin', 'owner')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, user_id)
);

CREATE INDEX memberships_user_id_idx ON memberships (user_id);

-- Los tokens de sesión nunca se guardan en claro: solo su hash SHA-256.
CREATE TABLE sessions (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash  bytea       NOT NULL UNIQUE,
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id   uuid        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_agent  text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE memberships;
DROP TABLE users;
DROP TABLE tenants;
