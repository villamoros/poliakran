# Sprint 2 — Catálogo y administración inicial

- **Fechas:** 2026-10-30 → 2026-11-13 (2 semanas; arranca al cerrar la Review del Sprint 1)
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
| 7 | Revisión de código | ⏳ Pendiente: revisión del Product Owner / pull request |
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

## Sprint Review (por completar)

Demostración propuesta (verificada en el entorno local antes de la Review):
1. Registrar una cuenta y concederle `owner` con `go run ./cmd/api grant-role <correo> owner`.
2. Volver a entrar: el panel muestra **Gestionar cursos**.
3. Crear un curso solo con título y descripción breve → queda en borrador.
4. Intentar publicarlo → el formulario marca lo que falta.
5. Completar los datos, guardar y publicar → aparece en `/cursos` sin sesión.
6. Abrir el detalle público; un borrador muestra "Curso no disponible".
7. Con una cuenta de estudiante, `/admin/cursos` responde "sin permiso".
8. Retirar el curso → desaparece del catálogo, pero sigue en la gestión.

## Sprint Retrospective (por completar)

Notas para la conversación:
- Un enlace a `/panel` en la cabecera rompía el registro: Next.js precargaba la redirección a `/ingresar` sin sesión y la reutilizaba. Se resolvió con `prefetch={false}`; conviene cubrirlo con la prueba E2E de LMS-TD-002.
- La prueba de arquitectura llegó antes que el segundo contexto, como se acordó.
