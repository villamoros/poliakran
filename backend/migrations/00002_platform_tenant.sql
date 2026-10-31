-- +goose Up
-- Tenant de plataforma (ADR-0008): el propio Politécnico Akran. Sus cursos
-- publicados forman el catálogo público y sus owner/admin gestionan ese catálogo.
ALTER TABLE tenants DROP CONSTRAINT tenants_kind_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_kind_check
    CHECK (kind IN ('personal', 'organization', 'platform'));

-- Solo puede existir un tenant de plataforma.
CREATE UNIQUE INDEX tenants_single_platform_idx ON tenants (kind) WHERE kind = 'platform';

INSERT INTO tenants (slug, name, kind) VALUES ('poliakran', 'Politécnico Akran', 'platform');

-- +goose Down
DELETE FROM tenants WHERE kind = 'platform';
DROP INDEX tenants_single_platform_idx;
ALTER TABLE tenants DROP CONSTRAINT tenants_kind_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_kind_check
    CHECK (kind IN ('personal', 'organization'));
