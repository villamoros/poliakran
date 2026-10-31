# Product Backlog

Ordenado por prioridad. P0 = necesario para el MVP; P1 = importante, alcance y
momento ajustables. Estimación en puntos relativos (pendiente para historias
futuras; se estimarán en su Sprint Planning).

## Épicas

| ID | Épica | Descripción | Prioridad | Estado |
|---|---|---|---|---|
| LMS-EP01 | Identity & Access | Registro, autenticación y autorización | P0 | En curso (Sprints 1–2) |
| LMS-EP11 | Platform Quality | Pruebas, seguridad y observabilidad | P0 | En curso |
| LMS-EP02 | Course Catalog | Catálogo y consulta de cursos | P0 | Hecho (Sprint 2) |
| LMS-EP03 | Course Management | Creación y publicación de cursos | P0 | Hecho (Sprint 2); exigir contenido al publicar en Sprint 3 |
| LMS-EP04 | Content Management | Gestión de módulos y lecciones | P0 | Sprint 3 |
| LMS-EP05 | Enrollment | Inscripción y acceso a cursos | P0 | Sprint 4 |
| LMS-EP06 | Learning Progress | Seguimiento del progreso | P0 | Sprint 4 |
| LMS-EP08 | Student Dashboard | Experiencia y panel del estudiante | P0 | Sprint 4 |
| LMS-EP07 | Assessment | Evaluaciones básicas y resultados | P0 | Sprint 5 |
| LMS-EP09 | Admin Dashboard | Panel administrativo | P0 | Sprint 6 |
| LMS-EP10 | File Storage | Gestión de recursos y archivos | P1 | Sprint 3 |
| LMS-EP12 | AI Readiness | Preparación arquitectónica para IA | P1 | Continuo (ADR-0007) |

## Historias de usuario

| ID | Historia | Épica | Prioridad | Sprint | Estado |
|---|---|---|---|---|---|
| LMS-US-001 | Registro de estudiante | EP01 | P0 | 1 | Hecho |
| LMS-US-002 | Inicio de sesión | EP01 | P0 | 1 | Hecho |
| LMS-US-013 | Cierre de sesión | EP01 | P0 | 1 | Hecho |
| LMS-US-003 | Consultar catálogo | EP02 | P0 | 2 | Hecho |
| LMS-US-004 | Crear curso | EP03 | P0 | 2 | Hecho |
| LMS-US-006 | Publicar curso | EP03 | P0 | 2 | Hecho |
| LMS-US-014 | Asignar rol de administrador | EP01 | P0 | 2 | Parcial: primer owner/admin por comando (Sprint 2); conceder `admin` desde la app, pendiente |
| LMS-US-005 | Administrar módulos y lecciones | EP04 | P0 | 3 | Por hacer |
| LMS-US-012 | Gestionar recursos educativos | EP10 | P1 | 3 | Por hacer |
| LMS-US-007 | Inscribirse en curso | EP05 | P0 | 4 | Por hacer |
| LMS-US-008 | Completar lección | EP06 | P0 | 4 | Por hacer |
| LMS-US-009 | Consultar progreso | EP06 | P0 | 4 | Por hacer |
| LMS-US-010 | Presentar evaluación | EP07 | P0 | 5 | Por hacer |
| LMS-US-011 | Consultar resultados | EP07 | P0 | 5 | Por hacer |
| LMS-US-016 | Cambiar de organización activa | EP01 | P1 | Al llegar organizaciones | Por refinar |
| LMS-US-015 | Recuperar acceso (contraseña olvidada) | EP01 | P0 | Por decidir | Por refinar: requiere proveedor de correo |

Los criterios de aceptación de LMS-US-001 a LMS-US-012 están en el documento
maestro del proyecto. Las historias nuevas:

### LMS-US-013 — Cierre de sesión
Como usuario autenticado quiero cerrar sesión para que nadie más use mi cuenta en este dispositivo.
- La sesión se revoca en el servidor, no solo en el navegador.
- Repetir el cierre no produce error.
- Tras cerrar, las funciones protegidas exigen volver a iniciar sesión.

### LMS-US-014 — Asignar rol de administrador
Como propietario de la plataforma quiero conceder el rol de administrador para que otra persona gestione el catálogo.
- Solo un `owner` puede conceder o retirar el rol `admin`.
- Existe una forma segura de crear el primer `owner` (comando de arranque, no API pública). ✅ Sprint 2: `go run ./cmd/api grant-role <correo> owner`.

### LMS-US-015 — Recuperar acceso
Como usuario registrado quiero restablecer mi contraseña si la olvido.
- Requiere elegir proveedor de correo transaccional.
- El enlace caduca y es de un solo uso; la respuesta no revela si el correo existe.

### LMS-US-016 — Cambiar de organización activa
Como estudiante que pertenece a varias organizaciones quiero elegir en cuál estoy trabajando para ver sus cursos y mi progreso en ella.
- La sesión guarda la organización activa; solo se puede elegir una con membresía vigente.
- Cada organización ve únicamente el progreso generado en ella.
- El tenant personal sigue disponible.

## Deuda técnica y mejoras

| ID | Descripción | Prioridad |
|---|---|---|
| LMS-TD-001 | Verificar automáticamente que ningún contexto importa el dominio de otro (linter de dependencias) | ✅ Hecho en Sprint 2 (`internal/architecture_test.go`) |
| LMS-TD-002 | Pruebas end-to-end automatizadas con Playwright en CI (registro → panel → cierre) | P0 |
| LMS-TD-003 | Limpieza periódica de sesiones caducadas o revocadas | P1 |
| LMS-TD-004 | Limitador de intentos compartido cuando haya más de una réplica del backend | P1 |
| LMS-TD-005 | Métricas (Prometheus/OpenTelemetry) y trazas, además de los logs estructurados | P1 |
| LMS-TD-006 | Dockerfile del backend y del frontend para despliegue | P0 (Sprint 7) |
| LMS-TD-007 | Imagen de portada del curso (requiere almacenamiento de archivos, EP10) | P1 (Sprint 3) |
| LMS-TD-008 | Exigir contenido (módulos y lecciones) para publicar un curso | P0 (Sprint 3) |

## Decisiones pendientes del Product Owner

| ID | Pregunta | Recomendación | Impacto si cambia |
|---|---|---|---|
| LMS-DEC-001 | ¿Base de datos compartida con `tenant_id` o BD/esquema por organización? | ✅ Decidido 2026-10-30: compartida + `tenant_id` | — |
| LMS-DEC-002 | ¿Un estudiante independiente puede pertenecer a varias organizaciones? | ✅ Decidido 2026-10-30: sí, con organización activa | Genera LMS-US-016 |
| LMS-DEC-003 | ¿Proveedor de correo para verificación y recuperación de acceso? | Por definir | Bloquea LMS-US-015 |
| LMS-DEC-004 | ¿El catálogo público son los cursos del tenant de plataforma (ADR-0008)? | Sí; las organizaciones tendrán catálogos privados más adelante | Cambia qué cursos ve un visitante |
| LMS-DEC-005 | ¿Se puede editar un curso ya publicado? | Sí, si sigue cumpliendo las condiciones de publicación (implementado así) | Bloquear la edición obligaría a retirar y volver a publicar |
