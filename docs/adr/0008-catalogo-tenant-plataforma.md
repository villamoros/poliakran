# ADR-0008 — Catálogo por tenant y tenant de plataforma

- **Estado:** Propuesto
- **Fecha:** 2026-10-30
- **Sprint:** 2
- **Relacionado con:** ADR-0001, ADR-0004, ADR-0005

## Contexto

En el Sprint 2 nace el contexto de Catálogo. ADR-0004 exige que todo dato de
dominio lleve `tenant_id`, pero el catálogo tiene que ser visible para un
visitante sin sesión, que no pertenece a ningún tenant. Además, hasta ahora todo
usuario iniciaba sesión en su tenant personal, donde nadie es administrador.

## Decisión

1. **Los cursos pertenecen a un tenant** (`courses.tenant_id NOT NULL`). Las
   operaciones de administración solo leen y escriben cursos del tenant activo
   de la sesión; un curso de otro tenant se responde como inexistente (404).
2. **Tenant de plataforma.** Se añade `tenants.kind = 'platform'`, con una única
   fila (`slug = 'poliakran'`, creada por la migración 00002). Es el propio
   Politécnico Akran.
3. **Catálogo público = cursos publicados del tenant de plataforma.** El
   composition root obtiene su id de Identidad al arrancar y se lo pasa al
   Catálogo, que así no consulta tablas de otro contexto.
4. **Primer owner por línea de comandos** (LMS-US-014):
   `go run ./cmd/api grant-role <correo> <owner|admin>`. Requiere acceso al
   servidor; no existe ninguna API pública para concederse roles.
5. **Tenant activo al iniciar sesión.** Si el usuario es `owner` o `admin` de un
   tenant, entra en ese tenant; si no, en su tenant personal (o la membresía más
   antigua). Esto completa el punto 4 de ADR-0004 hasta que llegue el cambio
   explícito de organización activa (LMS-US-016).
6. **Acciones de autorización:** `catalog:manage` (crear, listar, consultar,
   editar) y `catalog:publish` (publicar, retirar), ambas para `admin` y `owner`.
   La decisión se toma en los casos de uso (`internal/catalog/app`), no en los
   handlers, para que futuros agentes pasen por las mismas reglas (ADR-0007).
7. **Ciclo de vida:** borrador → publicado → retirado, y un retirado puede
   volver a publicarse. Publicar y retirar son idempotentes. Las ediciones usan
   bloqueo optimista (`version`): una edición basada en una versión antigua
   recibe 409 en lugar de sobrescribir cambios.

## Alternativas consideradas

- **Catálogo global sin `tenant_id`:** simple, pero rompe ADR-0004 e impide
  que una organización tenga cursos propios más adelante.
- **Marcar cada curso como "público":** más flexible, pero abre la puerta a que
  una organización publique en el catálogo de la plataforma sin revisión. Se
  puede añadir después sin migrar datos.

## Consecuencias

- Las organizaciones podrán tener catálogos privados con el mismo modelo; solo
  habrá que decidir cómo los ven sus miembros (junto con LMS-US-016).
- Un owner/admin de la plataforma que también estudia entra en la plataforma;
  verá sus cursos de estudiante cuando exista el cambio de tenant (LMS-US-016).
- El 404 de un curso no publicado en la web se muestra con la página "Curso no
  disponible"; con Partial Prerendering el código HTTP del documento es 200 con
  `noindex`, mientras que la API sí responde 404.
