# Sprint 2 — Catálogo y administración inicial

- **Fechas:** 2026-10-30 → 2026-11-13 (2 semanas; arranca al cerrar la Review del Sprint 1)
- **Cierre:** 2026-10-31, por decisión del Product Owner al cumplirse el Sprint Goal.
- **Sprint Goal:** permitir que el administrador gestione cursos y que el visitante pueda consultar el catálogo.
- **Incremento esperado:** el administrador puede crear cursos y publicar contenido descriptivo que los visitantes pueden consultar.

## Elementos seleccionados

| ID | Elemento | Estado |
|---|---|---|
| LMS-US-003 | Consultar catálogo | ✅ Hecho |
| LMS-US-004 | Crear curso | ✅ Hecho |
| LMS-US-006 | Publicar curso (incluye retirar) | ✅ Hecho |
| LMS-US-014 | Asignar rol de administrador — parte necesaria: crear el primer owner/admin de forma segura | ◐ Parcial (ver abajo) |
| LMS-TD-001 | Verificación automática de límites entre contextos | ✅ Hecho (acción de la Retrospectiva del Sprint 1) |

**LMS-US-014, refinamiento:** para cumplir el Sprint Goal hacía falta que existiera
un administrador. Se implementó el arranque seguro (comando `grant-role`, sin API
pública). Queda para un sprint posterior que un `owner` conceda o retire `admin`
desde la aplicación.

## Plan del Sprint (tareas técnicas)

| Tarea | Estado |
|---|---|
| ADR-0008: catálogo por tenant y tenant de plataforma | ✅ Propuesto |
| Migración 00002: tenant de plataforma (`kind = 'platform'`, único) | ✅ |
| Migración 00003: tabla `courses` con `tenant_id`, estados, índices y `version` | ✅ |
| Contexto `internal/catalog` (domain, app, adapters postgres/httpapi) | ✅ |
| Modelo de curso y reglas: validación, condiciones de publicación, ciclo de vida, slug | ✅ |
| Acciones `catalog:manage` y `catalog:publish` en la política central | ✅ |
| Autorización en los casos de uso, no en los handlers | ✅ |
| Endpoints públicos: listado (paginación, categoría) y detalle por slug | ✅ |
| Endpoints de administración: listar, crear, consultar, editar, publicar, retirar | ✅ |
| Bloqueo optimista en la edición; publicar/retirar idempotentes | ✅ |
| Comando `grant-role` y tenant activo para owner/admin al iniciar sesión | ✅ |
| Prueba de arquitectura (`internal/architecture_test.go`) | ✅ |
| Vistas públicas: `/cursos` (con filtro por categoría) y `/cursos/[slug]` | ✅ |
| Vistas de administración: `/admin/cursos`, `/admin/cursos/nuevo`, `/admin/cursos/[id]` | ✅ |
| Acceso a la gestión desde el panel y navegación en la cabecera | ✅ |
| Pruebas unitarias, de casos de uso, de API con PostgreSQL y de componentes | ✅ |
| Documentación: API, guía de desarrollo, ADR, backlog | ✅ |

## Verificación de criterios de aceptación

### LMS-US-003 — Consultar catálogo
| Criterio | Cómo se cumple | Prueba |
|---|---|---|
| Se muestran los cursos publicados | `GET /api/v1/catalog/courses`, página `/cursos` | `TestCatalogAdministrationAndPublicCatalog`, `TestPublicCatalogShowsOnlyPublishedPlatformCourses` |
| Se presenta su información principal | Tarjeta con categoría, título, descripción breve, nivel y duración | `CourseCard.test.tsx` |
| Retirados y borradores no aparecen | Filtro `status = 'published'` en la consulta; detalle 404 | `TestCatalogAdministrationAndPublicCatalog`, `TestPublicCatalogShowsOnlyPublishedPlatformCourses` |
| Se puede consultar el detalle | `GET /api/v1/catalog/courses/{slug}`, página `/cursos/[slug]` | `TestCatalogAdministrationAndPublicCatalog` |

### LMS-US-004 — Crear curso
| Criterio | Cómo se cumple | Prueba |
|---|---|---|
| El administrador autorizado puede crear | `POST /api/v1/admin/courses` con `catalog:manage` (admin/owner) | `TestCatalogAdministrationAndPublicCatalog`, `TestPolicy` |
| Se validan los campos obligatorios | `domain.NewDetails` + validación espejo en cliente; 422 por campo | `TestNewDetailsRequiresTitleAndSummaryOnly`, `CourseForm.test.tsx`, `validation.test.ts` |
| Se crea como borrador | Estado inicial `draft` | `TestCreateStartsAsDraftWithUniqueSlug` |
| Se puede editar antes de publicar | `PUT /api/v1/admin/courses/{id}` con `version` | `TestUpdateUsesOptimisticLocking`, `CourseEditor.test.tsx` |

### LMS-US-006 — Publicar curso
| Criterio | Cómo se cumple | Prueba |
|---|---|---|
| Solo usuarios autorizados publican | `catalog:publish` (admin/owner) en el caso de uso; estudiantes 403, sin sesión 401 | `TestAdministrativeOperationsAreProtected`, `TestOnlyAuthorizedPrincipalsManageCourses` |
| Se comprueban las condiciones mínimas | `CheckPublishable`: descripción, categoría, nivel, duración, objetivos | `TestPublishRequiresMinimumConditions`, `CourseEditor.test.tsx` |
| El estado cambia a publicado | `published` + `published_at` | `TestLifecycle` |
| Aparece en el catálogo público | Listado y detalle públicos | `TestCatalogAdministrationAndPublicCatalog` |

## Definition of Done del Incremento

| # | Punto | Estado |
|---|---|---|
| 1 | Criterios de aceptación | ✅ |
| 2 | Lógica implementada | ✅ |
| 3 | Autorización en backend | ✅ Política central, aplicada en los casos de uso; aislamiento por tenant |
| 4 | Validaciones | ✅ Backend (autoridad) y cliente |
| 5–6 | Pruebas automatizadas en verde | ✅ `go test -p 1 ./...` con PostgreSQL y `npm test` (39 pruebas) |
| 7 | Revisión de código | ✅ Aceptado por el Product Owner al cerrar el sprint (2026-10-31) |
| 8 | Integración con el producto | ✅ Recorrido navegador → Next.js → Go → PostgreSQL verificado |
| 9 | Sin errores bloqueantes conocidos | ✅ |
| 10 | Documentación | ✅ API, guía local, ADR-0008, backlog |
| 11 | Sin secretos en logs | ✅ Eventos de catálogo solo con ids (`TestAuditLogsContainOnlyIdentifiers`) |
| 12 | Demostrable en el entorno acordado | ✅ Local con `docker compose` |
| — | Consistencia ante repetición | ✅ Edición con versión antigua → 409; publicar/retirar idempotentes; slug único por tenant |

## Impedimentos y decisiones abiertas

- **ADR-0008 (propuesto):** el catálogo público son los cursos publicados del tenant de plataforma. Revisar en la Review.
- **Tenant activo de owner/admin:** al iniciar sesión entran en el tenant que gestionan. Se sustituirá por la elección explícita de LMS-US-016.
- **Por decidir:** ¿quién puede editar un curso ya publicado? Hoy admin/owner pueden, siempre que siga cumpliendo las condiciones de publicación.
- **LMS-DEC-003** (proveedor de correo) sigue abierta; no afecta a este sprint.

## Sprint Review

**Resultado:** Sprint Goal cumplido. El administrador gestiona cursos y el
visitante consulta el catálogo sin sesión. LMS-US-003, LMS-US-004 y LMS-US-006
quedan terminadas; LMS-US-014 queda parcial, como se acordó en la planificación.

**Demostración (navegador → Next.js → Go → PostgreSQL, build de producción):**
1. Registro de una cuenta y `grant-role <correo> owner` desde la línea de comandos.
2. Nuevo inicio de sesión: el panel muestra **Gestionar cursos** y la sesión entra en el tenant de plataforma.
3. Curso creado solo con título y descripción breve → borrador.
4. Intento de publicarlo → el formulario marca descripción, categoría, nivel, duración y objetivos.
5. Datos completados, guardado y publicación → aparece en `/cursos` para un visitante.
6. Detalle público del curso; la URL de un borrador muestra "Curso no disponible".
7. Visitante en `/admin/cursos` → redirigido a iniciar sesión; estudiante → "sin permiso".
8. Curso retirado → desaparece del catálogo y sigue en la gestión.
9. Catálogo revisado también en ancho de móvil (390 px).

**Evidencia:** `go test -p 1 -race ./...` con PostgreSQL real, 39 pruebas de
frontend, lint, typecheck y build de producción en verde. Commit `b5dc55f`
integrado en el repositorio local del Product Owner (rama `sprint-02`).

**Feedback para el backlog:**
- Siguen abiertas para el Sprint Planning 3: LMS-DEC-004 (catálogo público = tenant de plataforma, ADR-0008) y LMS-DEC-005 (edición de cursos publicados).
- Publicar exigirá contenido cuando existan módulos y lecciones (LMS-TD-008, Sprint 3).
- Imagen de portada pendiente del almacenamiento de archivos (LMS-TD-007).

**Historias no completadas en el sprint:** ninguna de las seleccionadas. De
LMS-US-014 queda conceder o retirar `admin` desde la aplicación.

## Sprint Retrospective

**Qué salió bien**
- La prueba de arquitectura (LMS-TD-001) llegó antes que el segundo contexto, como se acordó en la Retro del Sprint 1.
- La autorización vive en los casos de uso: las pruebas de rutas confirman 401/403 en todas las operaciones de administración.
- El recorrido real en navegador destapó un error que las pruebas unitarias no veían.

**Qué mejorar**
- Un enlace a `/panel` en la cabecera rompía el registro: Next.js precargaba la redirección a `/ingresar` hecha sin sesión y la reutilizaba tras registrarse. Se resolvió con `prefetch={false}`, pero no hay prueba automática que lo cubra.
- El historial Git local y el de GitHub eran distintos; la entrega necesitó un bundle. Ya están unificados (rama `sprint-02` sobre `origin/main`).
- La CI sigue sin ejecutarse en GitHub mientras la rama no se suba.

**Acciones para el Sprint 3**
1. Subir `sprint-02`, abrir el pull request y comprobar la CI en GitHub; borrar `tools/ci/` (el workflow ya está en `.github/`).
2. Pruebas E2E con Playwright (LMS-TD-002) que cubran registro → panel y la publicación de un curso.
3. Decidir LMS-DEC-004 y LMS-DEC-005 en el Sprint Planning.
