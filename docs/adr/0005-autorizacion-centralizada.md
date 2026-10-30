# ADR-0005 — Autorización centralizada basada en políticas

- **Estado:** Propuesto
- **Fecha:** 2026-10-30
- **Sprint:** 1

## Contexto

La Definition of Done exige autorización aplicada en backend. Con varios
contextos y tenants, repartir comprobaciones `if role == ...` por los handlers
produce agujeros difíciles de auditar.

## Decisión

- Un único punto de decisión: `internal/platform/authz`, con la interfaz
  `Authorizer.Authorize(ctx, principal, action, resource)`.
- **Principal** = usuario + tenant activo + roles en ese tenant, resuelto desde la sesión por el middleware `RequireAuth`.
- **Acciones** con nombre `contexto:verbo` (`identity:read_self`, `catalog:manage`…), declaradas como constantes.
- **Recurso** con `TenantID` y `OwnerID`. Regla transversal: **nunca** se permite actuar sobre un recurso de otro tenant.
- **Denegar por defecto:** una acción sin regla se rechaza.
- Roles iniciales: `student`, `instructor`, `admin`, `owner`.
- El frontend puede ocultar opciones, pero nunca es la autoridad.

## Consecuencias

- Las reglas se prueban en una tabla (`authz_test.go`) y se revisan en un solo archivo.
- La implementación actual es RBAC + propiedad en código. Si las reglas crecen se puede sustituir por un motor de políticas (p. ej. Cedar u OPA) sin cambiar a los consumidores.
- Los futuros agentes de IA actuarán como principals con permisos limitados y pasarán por el mismo `Authorizer` (ADR-0007).
