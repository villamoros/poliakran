-- +goose Up
-- Catálogo de cursos (ADR-0001: tabla propia del contexto de Catálogo;
-- ADR-0004: todo curso pertenece a un tenant).

CREATE TABLE courses (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid        NOT NULL REFERENCES tenants (id) ON DELETE RESTRICT,
    slug            text        NOT NULL,
    title           text        NOT NULL,
    summary         text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    category        text        NOT NULL DEFAULT '',
    level           text        NOT NULL DEFAULT '' CHECK (level IN ('', 'basico', 'intermedio', 'avanzado')),
    duration_hours  integer     NOT NULL DEFAULT 0 CHECK (duration_hours BETWEEN 0 AND 500),
    objectives      text[]      NOT NULL DEFAULT '{}',
    prerequisites   text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'withdrawn')),
    published_at    timestamptz,
    withdrawn_at    timestamptz,
    created_by      uuid        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    version         integer     NOT NULL DEFAULT 1,
    CONSTRAINT courses_tenant_slug_key UNIQUE (tenant_id, slug),
    -- Un curso publicado siempre tiene fecha de publicación.
    CONSTRAINT courses_published_at_check CHECK (status <> 'published' OR published_at IS NOT NULL)
);

-- Catálogo público: cursos publicados de un tenant, más recientes primero.
CREATE INDEX courses_tenant_published_idx ON courses (tenant_id, published_at DESC) WHERE status = 'published';
-- Administración: todos los cursos de un tenant.
CREATE INDEX courses_tenant_updated_idx ON courses (tenant_id, updated_at DESC);

-- +goose Down
DROP TABLE courses;
